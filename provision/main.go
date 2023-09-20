package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"gopkg.in/gcfg.v1"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/client"
	"github.com/timapril/go-registrar/epp"
	eppclient "github.com/timapril/go-registrar/epp/client"
	"github.com/timapril/go-registrar/keychain"
	"github.com/timapril/go-registrar/lib"

	superclient "github.com/timapril/go-registrar/epp/superClient"
)

// Will be filled in during the build process
var version = "undefined"

var log = logging.MustGetLogger("registrar")
var format = logging.MustStringFormatter(
	"%{color}%{level:.4s} %{time:15:04:05.000000} %{shortfile} %{callpath} %{longfunc} %{id:03x}%{color:reset} %{message}",
)

var (
	crid = flag.Int("cr", -1, "Change request ID")

	verbose     = flag.Bool("v", false, "Verbose logging")
	veryverbose = flag.Bool("vv", false, "Very Verbose logging")

	passphraseIn = flag.Bool("passin", false, "Set if the EPP passphrase will be provided via stdin")

	confpath = flag.String("conf", "./conf", "The path to the configuration file")
)

// Config is used to load the configuration file information from a correctly
// formatted file
type Config struct {
	Registrar struct {
		Server      string
		Port        int64
		UseHTTPS    bool
		TrustAnchor []string
	}

	Certs struct {
		CACertPath string
		CertPath   string
		KeyPath    string
	}
	Transfer struct {
		AuthInfoIn  string
		AuthInfoOut string
	}

	Mac keychain.Conf

	Testing struct {
		SpoofCert  string
		CertHeader string
	}

	VerisignEPP eppclient.Config

	Passphrase struct {
		Base64Command  string
		DecryptCommand string
		PassphraseName string
	}

	CacheConfig client.DiskCacheConfig
}

var runReportTemplate = `==========================================================================
===                          Begin Run Report                          ===
==========================================================================

EPP Run ID:    {{.RunID}}
TXID Prefix:   {{.TransactionIDPrefix}}
Start Time:    {{.StartTime}}
End Time:      {{.EndTime}}
Duration:      {{.Duration}}
Run Completed: {{.CompletedRun}}

Local Transfer Actions:
	Approved transfer out:{{ range $val := .TransfersLocalApproved}}
		{{$val}}{{end}}

	Rejected transfer out:{{ range $val := .TransfersLocalRejected}}
		{{$val}}{{end}}

Remote Transfer Actions:
	Approved transfer in:{{ range $val := .TransfersRemoteApproved}}
		{{$val}}{{end}}

	Rejected transfer in:{{ range $val := .TransfersRemoteRejected}}
		{{$val}}{{end}}

	Cancelled transfer in:{{ range $val := .TransfersRemoteCancelled}}
		{{$val}}{{end}}

Verified Work:
	Domains:{{ range $val := .VerifiedDomains}}
		{{$val}}{{end}}

	Hosts:{{ range $val := .VerifiedHosts}}
		{{$val}}{{end}}

Registered Domains:{{ range $val := .DomainsRegistered}}
	{{$val}}{{end}}

Work Done:{{range $val := .WorkLog}}
	{{$val}}{{end}}

Registry Changes Needed:{{range $val := .RegistryLockChanges}}
	{{$val}}{{end}}

==========================================================================
===                           End Run Report                           ===
==========================================================================
`

// RunReport is a data structure that collects infomration about the run to be
// displayed at the end of the session
type RunReport struct {
	RunID               int64
	TransactionIDPrefix string
	StartTime           time.Time
	EndTime             time.Time
	Duration            time.Duration
	CompletedRun        bool

	TransfersLocalRejected []string
	TransfersLocalApproved []string

	TransfersRemoteRejected  []string
	TransfersRemoteApproved  []string
	TransfersRemoteCancelled []string

	VerifiedDomains []string
	VerifiedHosts   []string

	DomainsRegistered        []string
	DomainsTransferRequested []string

	RegistryLockChanges []string

	WorkLog []string
}

// AddWorkItem will append a line with the current timestamp and the work item
// that was completed to the list of work items to log
func (r *RunReport) AddWorkItem(item string) {
	logLine := fmt.Sprintf("%s: %s", time.Now().Format("Mon Jan 2 15:04:05 -0700 MST"), item)
	r.WorkLog = append(r.WorkLog, logLine)
}

// DisplayReport is used to generate a text version of the run report to be
// saved to displayed
func (r RunReport) DisplayReport() (string, error) {
	r.Duration = r.EndTime.Sub(r.StartTime)

	t := template.Must(template.New("rr").Parse(runReportTemplate))

	var doc bytes.Buffer
	err := t.Execute(&doc, r)
	if err != nil {
		return "", err
	}
	s := doc.String()

	return s, nil
}

// GetConnectionURL will return the URL that can be used to connect to the
// Registrar server as defined by the parameters in the configuartion
func (c Config) GetConnectionURL() string {
	if c.Registrar.UseHTTPS {
		return fmt.Sprintf("https://%s:%d", c.Registrar.Server, c.Registrar.Port)
	}
	return fmt.Sprintf("http://%s:%d", c.Registrar.Server, c.Registrar.Port)
}

// GetTrustAnchor will use the information in the config object to create the
// trust anchor set for the application and return the trustanchors or an error
// if an error occurs.
func (c Config) GetTrustAnchor() (client.TrustAnchors, error) {
	ta := client.TrustAnchors{}
	for _, anchor := range c.Registrar.TrustAnchor {
		pubkey, readErr := os.ReadFile(anchor)
		if readErr != nil {
			return ta, readErr
		}
		err := ta.AddKey(string(pubkey))
		if err != nil {
			return ta, err
		}
	}
	return ta, nil
}

// GetRegistrarClient will use the configuration object and generate and
// return an Registrar client object. If an error occurs when generating
// the client, the error is returned
func (c Config) GetRegistrarClient() (cli client.Client, err error) {
	cli.TrustAnchor, err = c.GetTrustAnchor()
	if err != nil {
		return
	}

	if c.Testing.SpoofCert != "" {
		log.Debug("Using the testing credentials rather than production")
		cli.Prepare(c.GetConnectionURL(), log, c.CacheConfig)
		spoofCert, readErr := os.ReadFile(c.Testing.SpoofCert)
		if readErr != nil {
			err = readErr
			return
		}
		cli.SpoofCertificateForTesting(string(spoofCert), string(c.Testing.CertHeader))
	}
	if c.Registrar.UseHTTPS {
		cli.PrepareSSL(c.GetConnectionURL(), c.Certs.CertPath, c.Certs.KeyPath, c.Certs.CACertPath, c.Mac, log, c.CacheConfig)
	} else {
		cli.Prepare(c.GetConnectionURL(), log, c.CacheConfig)
	}

	return cli, nil
}

func decryptPassphrase(encryptedPassphrase string, conf Config) (passphrase []byte, err error) {
	command := fmt.Sprintf("echo %s | %s | %s", strings.TrimSpace(encryptedPassphrase), conf.Passphrase.Base64Command, conf.Passphrase.DecryptCommand)
	log.Debug(command)
	return exec.Command("/bin/sh", "-c", command).Output()
}

func main() {
	rr := RunReport{}
	rr.StartTime = time.Now()
	rr.CompletedRun = false

	flag.Parse()

	conf := Config{}

	ll := logging.ERROR
	if *verbose {
		ll = logging.INFO
	}
	if *veryverbose {
		ll = logging.DEBUG
	}

	prepareLogging(ll)

	confErr := gcfg.ReadFileInto(&conf, *confpath)
	if confErr != nil {
		log.Error(confErr)
		return
	}

	log.Infof("Git Version: %s\n", version)

	var passphrase string
	if *passphraseIn {
		log.Debug("Passphrase passed in via command line")
		reader := bufio.NewReader(os.Stdin)
		passphraseInRaw, _ := reader.ReadString('\n')
		passphrase = strings.TrimSpace(passphraseInRaw)
		if len(strings.TrimSpace(passphrase)) == 0 {
			log.Error("Passphrase was empty")
			return
		}
	}

	cli, cliErr := conf.GetRegistrarClient()
	if cliErr != nil {
		log.Error(cliErr)
		return
	}
	errs := cli.PrepareObjectDirectory()
	if len(errs) != 0 {
		for _, err := range errs {
			log.Error(err)
		}
		return
	}

	runID, errs := cli.RequestNewEPPRunID()
	if len(errs) != 0 {
		for _, err := range errs {
			log.Error(err)
		}
		return
	}

	rr.RunID = runID

	if len(passphrase) == 0 {
		encPassphrase, err := cli.GetEncryptedPassphrase(conf.Passphrase.PassphraseName)
		if err != nil {
			fmt.Println(err)
			return
		}
		rawPassphrase, decryptErr := decryptPassphrase(encPassphrase, conf)
		if decryptErr != nil {
			log.Errorf("Error decrypting passphrase %s", decryptErr)
			fmt.Println(decryptErr)
			return
		}
		passphrase = strings.TrimSpace(string(rawPassphrase))
	}

	conf.VerisignEPP.Password = passphrase

	conf.VerisignEPP.TransactionPrefix = fmt.Sprintf("%s-%d-%s-", conf.VerisignEPP.TransactionPrefix, runID, time.Now().Format("020106"))
	rr.TransactionIDPrefix = conf.VerisignEPP.TransactionPrefix

	sc, scCreateErr := superclient.NewSuperClient(conf.VerisignEPP, log, func(action lib.EPPAction) {
		cli.PushEPPActionLog(action)
	}, func(action lib.EPPAction) {
		cli.PushEPPActionLog(action)
	})
	if scCreateErr != nil {
		log.Error(scCreateErr)
		return
	}

	if crid != nil && *crid > 0 {
		fmt.Printf("looking up cr id: %d\n", *crid)
		cr, err := cli.GetChangeRequest(int64(*crid))
		if err != nil {
			log.Error(err)
			return
		}

		for _, app := range cr.Approvals {
			if app.IsSigned {
				fmt.Println(app.ID)

				VerifyApproval(cli, app, cr)
			}
		}
	}

	// ips, errs := cli.GetHostIPAllowList()
	// if len(errs) != 0 {
	// 	fmt.Println(errs)
	// }
	// fmt.Println(ips)
	// return

	var phaseName string
	var eppPhase int64

	verifiedDomains := make(map[string]*lib.DomainExport)
	domainAvailability := make(map[string]bool)
	domainInfoResponses := make(map[string]*epp.Response)

	verifiedHosts := make(map[string]*lib.HostExport)
	hostAvailability := make(map[string]bool)
	hostInfoResponses := make(map[string]*epp.Response)

	var shouldKeepGoingHostInfo bool
	var skipHostnamems []string

	hostDomains := make(map[string]bool)

	domainIds, hostIds, _, errs := GetAllWork(cli)
	if len(errs) != 0 {
		for _, err := range errs {
			log.Error(err)
			goto Cleanup
		}
	}

	// Do the expensive part of verifying all of the hosts
	log.Info("Begin: Host verification")
	for _, host := range hostIds {
		log.Infof("\tStart: Host ID %d", host)
		verified, errs, object := cli.GetVerifiedHost(host, time.Now().Unix())
		if len(errs) != 0 {
			for _, err := range errs {
				log.Errorf("\tError: %s", err)
			}
		} else {
			if verified {
				verifiedHosts[object.HostName] = object
				rr.VerifiedHosts = append(rr.VerifiedHosts, object.HostName)
				hostInfoResponses[object.HostName] = nil
				parentDomain, parentDomainErr := GetParentDomain(object.HostName)
				if parentDomainErr == nil {
					log.Infof("Adding %s to the list of domains to get INFO responses for", parentDomain)
					hostDomains[parentDomain] = true
				} else {
					log.Errorf("Error extracting parent domain from %s: %s", object.HostName, parentDomainErr)
					if strings.HasPrefix(parentDomainErr.Error(), "Unhandled TLD") {
						err := HostUnsetCheck(cli, object.ID)
						if err != nil {
							log.Errorf("Error unsetting check for %s: %s", object.HostName, err.Error())
						}
					}
				}
				log.Infof("\t\t%s verified", object.HostName)
			} else {
				log.Errorf("\t\tError verifying %s", object.HostName)
			}
		}
		log.Infof("\tEnd: Host ID %d", host)
	}
	log.Info("End: Host verification")

	// TODO: Get domains for all hosts on record
	// Lets get all of the verified domains first
	log.Info("Begin: Domain verification")
	for dom := range hostDomains {
		if _, ok := verifiedDomains[dom]; !ok {
			id, errs := cli.GetDomainIDFromName(dom)
			if len(errs) != 0 {
				if errs[0].Error() != "record not found" {
					log.Errorf("%s", errs[0].Error())
					goto Cleanup
				}
			}
			found := false
			for _, dom := range domainIds {
				if dom == id {
					found = true
					break
				}
			}
			if !found {
				domainIds = append(domainIds, id)
			}
		}
	}
	for _, domain := range domainIds {
		log.Infof("\tStart: Domain ID %d", domain)
		verified, errs, object := cli.GetVerifiedDomain(domain, time.Now().Unix())
		if len(errs) != 0 {
			for _, err := range errs {
				log.Errorf("\tError: %s", err)
			}
		} else {
			if verified {
				verifiedDomains[object.DomainName] = object
				rr.VerifiedDomains = append(rr.VerifiedDomains, object.DomainName)
				domainInfoResponses[object.DomainName] = nil
				log.Infof("\t\t%s verified", object.DomainName)
			} else {
				log.Errorf("\t\tError verifying %s", object.DomainName)
			}
		}
		log.Infof("\tEnd: Domain ID %d", domain)
	}

	log.Info("End: Domain verification")

	phaseName = "Poll Request Processing"
	eppPhase++

	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := HandlePolls(&rr, cli, sc, &verifiedDomains); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Domain Existance Check"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := GetDomainExistance(&rr, cli, sc, &verifiedDomains, &domainInfoResponses, &domainAvailability); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Host Existance Check"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := GetHostExistance(&rr, cli, sc, &verifiedHosts, &hostInfoResponses, &hostAvailability); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Domain infomration gathering"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := GetDomainInfo(&rr, cli, sc, &verifiedDomains, &domainAvailability, &domainInfoResponses, false); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Host infomration gathering"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	shouldKeepGoingHostInfo, skipHostnamems = GetHostInfo(&rr, cli, sc, &verifiedHosts, &hostAvailability, &verifiedDomains, &hostInfoResponses, false)
	if !shouldKeepGoingHostInfo {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	if len(skipHostnamems) != 0 {
		fmt.Println(skipHostnamems)
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Domain Transfer In"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := DomainTransferInRequests(conf, &rr, cli, sc, &verifiedDomains, &domainAvailability, &domainInfoResponses); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Host Updates"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := HostUpdate(&rr, cli, sc, &verifiedHosts, &verifiedDomains, &hostAvailability, &hostInfoResponses, &skipHostnamems); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Domain Updates"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := DomainUpdate(conf, &rr, cli, sc, &verifiedDomains, &domainAvailability, &domainInfoResponses); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Update Registrar with Domains"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := PushDomains(cli, &verifiedDomains, &domainInfoResponses); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	eppPhase++
	phaseName = "Update Registrar with Hosts"
	log.Infof("Starting Phase %d: %s", eppPhase, phaseName)
	if shouldKeepGoing := PushHosts(cli, &verifiedHosts, &hostInfoResponses); !shouldKeepGoing {
		log.Errorf("Terminal Phase %d: %s", eppPhase, phaseName)
		goto Cleanup
	}
	log.Infof("Ending Phase %d: %s", eppPhase, phaseName)

	// TODO: Check for renewals

	rr.CompletedRun = true

Cleanup:
	log.Info("Done, logging out of verisign")
	_, err := sc.Logout()
	if err != nil {
		log.Error(err)
		return
	}

	rr.EndTime = time.Now()
	time.Sleep(1 * time.Second)
	endRunErrs := cli.EndEPPRun(runID)
	if len(endRunErrs) != 0 {
		for _, err := range endRunErrs {
			log.Error(err)
		}
		return
	}
	fmt.Println(rr.DisplayReport())

}

// GetParentDomain will inspect the provided hostname and extract the parent
// domain name that is one level down from the TLD. If the hostname is not valid
// or has an unsupported TLD, an error will be returned otherwise the doamin
// and TLD will be returned
func GetParentDomain(hostname string) (parentDomain string, err error) {
	tokens := strings.Split(hostname, ".")
	if len(tokens) > 2 {
		domainLen := 0
		switch tokens[len(tokens)-1] {
		case "COM":
			domainLen = 2
		case "NET":
			domainLen = 2
		default:
			return "", fmt.Errorf("Unhandled TLD %s", tokens[len(tokens)-1])
		}

		domainTokens := tokens[len(tokens)-domainLen:]
		return strings.Join(domainTokens, "."), nil
	}
	return "", errors.New("Invalid Hostname")
}

// HandlePolls will issue poll requests to the EPP Server and process poll
// requests that are pending for the server
func HandlePolls(rr *RunReport, client client.Client, eppClient *superclient.SuperClient, verifiedDomains *map[string]*lib.DomainExport) (shouldContinue bool) {
	log.Info("Begin: Poll request processing")
	for {
		rr.AddWorkItem("EPP POLL QUERY")
		pollMessageExists, pollMessage, action, pollErr := eppClient.PollRequest()
		client.PushEPPActionLog(action)
		if pollErr != nil {
			log.Errorf("\tError: %s", pollErr)
			return false
		}
		okToDequeue := false

		if pollMessageExists {
			if pollMessage.IsDomainTransferRequested {
				log.Infof("\tTransfer Request for %s", pollMessage.DomainTransfer.Name)
				transferAllowed, transferAllowedErr := handleDomainTransferOutRequest(pollMessage.DomainTransfer.Name, verifiedDomains)
				if transferAllowedErr != nil {
					log.Errorf("Error processing poll request, exiting %s", transferAllowedErr)
					return false
				}

				if transferAllowed {
					log.Infof("\tTransfer of %s has been approved", pollMessage.DomainTransfer.Name)
					rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN TRANSFER APPROVE %s", pollMessage.DomainTransfer.Name))
					transferRespCode, action, transferErr := eppClient.ApproveDomainTransfer(pollMessage.DomainTransfer.Name)
					client.PushEPPActionLog(action)
					if transferErr != nil {
						log.Errorf("Error occured when approving transfer - (%d) %s", transferRespCode, transferErr)
						return false
					}
					rr.TransfersLocalApproved = append(rr.TransfersLocalApproved, pollMessage.DomainTransfer.Name)
					okToDequeue = true
				} else {
					log.Infof("\tTransfer of %s has been rejected", pollMessage.DomainTransfer.Name)
					rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN TRANSFER REJECT %s", pollMessage.DomainTransfer.Name))
					transferRespCode, action, transferErr := eppClient.RejectDomainTransfer(pollMessage.DomainTransfer.Name)
					client.PushEPPActionLog(action)
					if transferErr != nil {
						log.Errorf("Error occured when rejecting transfer - (%d) %s", transferRespCode, transferErr)
						return
					}
					rr.TransfersLocalRejected = append(rr.TransfersLocalRejected, pollMessage.DomainTransfer.Name)
					okToDequeue = true
				}

			}
			if pollMessage.IsDomainTransferRejected {
				log.Infof("\t Transfer request for %s has been rejected", pollMessage.DomainTransfer.Name)
				rr.TransfersRemoteRejected = append(rr.TransfersRemoteRejected, pollMessage.DomainTransfer.Name)
				okToDequeue = true
			}
			if pollMessage.IsDomainTransferApproved {
				log.Infof("\t Transfer request for %s has been approved", pollMessage.DomainTransfer.Name)
				rr.TransfersRemoteApproved = append(rr.TransfersRemoteApproved, pollMessage.DomainTransfer.Name)
				okToDequeue = true
			}
			if pollMessage.IsDomainTransferCancelled {
				log.Infof("\t Transfer request for %s has been cancelled", pollMessage.DomainTransfer.Name)
				rr.TransfersRemoteCancelled = append(rr.TransfersRemoteCancelled, pollMessage.DomainTransfer.Name)
				okToDequeue = true
			}
			if pollMessage.IsUnusedObjectsPolicy {
				log.Infof("\t Unused objects policy message received for host %s", pollMessage.HostInf.Name)
				log.Infof("\t Would you like to ACK this unused objects policy message? [y/N]")
				var response string
				_, err := fmt.Scanln(&response)
				if err != nil {
					log.Errorf("Error reading in user response whether to ACK or not, exiting")
					return false
				}
				response = strings.ToLower(strings.TrimSpace(response))
				if response == "y" {
					okToDequeue = true
				}
			}

			if okToDequeue {
				rr.AddWorkItem(fmt.Sprintf("EPP POLL ACK - %s", pollMessage.ID))
				action, pollErr := eppClient.AckPoll(pollMessage.ID)
				client.PushEPPActionLog(action)
				if pollErr != nil {
					log.Errorf("Error acking poll message - %s", pollErr)
				}
			} else {
				log.Error("Not dequeueing message, this will result in a loop, exiting")
				return false
			}

		} else {
			log.Info("\tNo more poll requests found")
			break
		}
	}
	log.Info("End: Poll request processing")

	return true
}

// GetDomainExistance will iterate through the list of domains that have
// been verifed to make sure that they all exist. If a domain does not exist
// and its status is Active then it will be registered.
func GetDomainExistance(rr *RunReport, client client.Client, eppClient *superclient.SuperClient, verifiedDomains *map[string]*lib.DomainExport, domainMap *map[string]*epp.Response, da *map[string]bool) (shouldContinue bool) {
	for domainName := range *domainMap {
		rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN CHECK %s", domainName))
		domAvail, _, action, domAvailErr := eppClient.DomainAvailable(domainName)
		client.PushEPPActionLog(action)
		if domAvailErr != nil {
			log.Errorf("\tError checking if %s is available - %s", domainName, domAvailErr)
			return false
		}
		if !domAvail {
			log.Infof("\tDomain %s exists", domainName)
			(*da)[domainName] = false
		} else {
			log.Infof("\tDomain %s does not exist yet", domainName)
			if domainObject, ok := (*verifiedDomains)[domainName]; ok {
				if domainObject.CurrentRevision.DesiredState == lib.StateActive {
					log.Infof("\tDomain %s should be created", domainName)
					rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN CREATE %s - %d yrs", domainName, 1))
					rc, action, err := eppClient.DomainCreate(domainName, 1)
					client.PushEPPActionLog(action)
					if err != nil {
						log.Errorf("\tError creating domain %s - (%d) %s", domainName, rc, err)
						return false
					}
					(*da)[domainName] = false
					rr.DomainsRegistered = append(rr.DomainsRegistered, domainName)
				} else {
					(*da)[domainName] = true
				}
			} else {
				log.Errorf("Domain %s is not currently listed as requiring work so it was not registred", domainName)
				return false
			}
		}
	}
	return true
}

// GetHostExistance will iterate through the list of hosts that have
// been verifed to make sure that they all exist. If a host does not exist
// it will not be created, not enough information is availabe at this point
// to register domains yet
func GetHostExistance(rr *RunReport, client client.Client, eppClient *superclient.SuperClient, verifiedHosts *map[string]*lib.HostExport, hostMap *map[string]*epp.Response, ha *map[string]bool) (shouldContinue bool) {
	for hostName := range *hostMap {
		rr.AddWorkItem(fmt.Sprintf("EPP HOST CHECK %s", hostName))
		hosAvail, action, hosAvailErr := eppClient.HostAvailable(hostName)
		client.PushEPPActionLog(action)
		if hosAvailErr != nil {
			log.Errorf("\tError checking if %s is available - %s", hostName, hosAvailErr)
			return false
		}
		if !hosAvail {
			log.Infof("\tHost %s exits", hostName)
			(*ha)[hostName] = false
		} else {
			log.Infof("\tHost %s does not exist yet", hostName)
			(*ha)[hostName] = true
		}
	}
	return true
}

// GetDomainInfo will iterate over the list of domains in the doaminMap that
// does not have a response object associated yet and if  the domain is
// implemented, it will request the domain infomration otherwise it will skip
// the domain
func GetDomainInfo(rr *RunReport, client client.Client, eppClient *superclient.SuperClient, verifiedDomains *map[string]*lib.DomainExport, da *map[string]bool, domainMap *map[string]*epp.Response, refresh bool) (shouldContinue bool) {
	log.Info("Iterating through all domains to get their infomration from the registry")
	for domainName, val := range *domainMap {
		// In the case where you want to fill out missing entries you want to skip
		// existing entries
		if !refresh && val != nil {
			continue
		}
		domainAvailable, domainAvailableFound := (*da)[domainName]
		log.Debugf("Domain Available: %t", domainAvailable)
		log.Debugf("Domain Available Found: %t", domainAvailableFound)
		if !domainAvailableFound || domainAvailable {
			log.Infof("\tDomain %s is not registred, skipping", domainName)
			continue
		}
		log.Infof("\tStart Domain %s", domainName)
		if val, ok := (*domainMap)[domainName]; val == nil || !ok {
			log.Infof("\tGetting domain info for %s", domainName)
			domainInfoErr := UpdateDomainInfo(rr, client, eppClient, domainName, domainMap)
			if domainInfoErr != nil {
				log.Errorf("\tError getting domain info for %s - %s", domainName, domainInfoErr)
				return false
			}
			log.Infof("\tDone getting domain info for %s", domainName)
		}
	}
	return true
}

// GetHostInfo will iterate over the list of domains in the hostMap that
// does not have a response object associated yet and if the host is
// implemented, it will request the host infomration otherwise it will skip
// the host
func GetHostInfo(rr *RunReport, client client.Client, eppClient *superclient.SuperClient, verifiedHosts *map[string]*lib.HostExport, ha *map[string]bool, verifiedDomains *map[string]*lib.DomainExport, hostMap *map[string]*epp.Response, refresh bool) (shouldContinue bool, skipHostnames []string) {
	log.Info("Iterating through all hosts to get their information from the registry")
	for hostname, val := range *hostMap {
		// In the case where you want to fill missing entries, you want to skip
		// existing entries
		if !refresh && val != nil {
			continue
		}

		parentDomain, parentDomainErr := GetParentDomain(hostname)
		if parentDomainErr != nil {
			log.Errorf("Unable to get parent domain for %s - %s", hostname, parentDomainErr)
			return false, skipHostnames
		}
		parentDomainInControl := false
		if _, ok := (*verifiedDomains)[parentDomain]; ok {
			parentDomainInControl = true
		}

		if !parentDomainInControl {
			log.Debugf("Skipping info on hostname %s, parent domain %s is not registrared by this registrar", hostname, parentDomain)
		} else {
			hostAvailable, hostAvailableFound := (*ha)[hostname]
			log.Debugf("Host Available: %t", hostAvailable)
			log.Debugf("Host Available Found: %t", hostAvailableFound)
			if !hostAvailableFound || hostAvailable {
				log.Infof("\tHost %s is not registred, skipping", hostname)
				continue
			}

			log.Infof("\tStarting Host %s", hostname)
			if val, ok := (*hostMap)[hostname]; val == nil || !ok {
				log.Infof("\tGetting host info for %s", hostname)
				hostInfoErr := UpdateHostInfo(rr, client, eppClient, hostname, hostMap)
				if hostInfoErr != nil {
					log.Errorf("\tError getting host info for %s - %s", hostname, hostInfoErr)
					skipHostnames = append(skipHostnames, hostname)
				}
				log.Infof("\tDone getting host info for %s", hostname)
			}
		}
	}
	return true, skipHostnames
}

// DomainUpdate will iterate over the domains that require work and will update
// or delete the domains as defined by the object status
func DomainUpdate(conf Config, rr *RunReport, client client.Client, eppClient *superclient.SuperClient, verifiedDomains *map[string]*lib.DomainExport, da *map[string]bool, domainMap *map[string]*epp.Response) (shouldContinue bool) {
	for domainName, domainRegObject := range *verifiedDomains {
		log.Infof("Domain %s: Starting to process domain", domainName)

		domainChangeMade := false

		domainAvailable, domainAvailableFound := (*da)[domainName]
		if !domainAvailableFound {
			log.Errorf("Domain %s: No availability information found, skipping domain", domainName)
			continue
		}

		if domainAvailable {
			if domainRegObject.State == lib.StateInactive {
				log.Infof("Domain %s: doamin is available but is inactive", domainName)
				continue
			} else {
				log.Errorf("Domain %s: domain is available to register, this is a mistake", domainName)
				continue
			}
		} else {
			if domainRegObject.CurrentRevision.ID == 0 {
				log.Errorf("Domain %s: No current revision found, skipping", domainName)
				continue
			}
			switch domainRegObject.CurrentRevision.DesiredState {
			case lib.StateInactive:
				// TODO: Check for server status flags that prevent delete
				// TODO: Unlock domain (update and delete status flags)

				log.Infof("Domain %s: Domain is marked as inactive, preparing to delete the object", domainName)
				allowDelete, promptErr := PromptToConfirm(fmt.Sprintf("Has the CSO, EVP of Platform *OR* the CEO approved the deletion of %s (may result in someone else registering it)?", domainName))
				if promptErr != nil {
					log.Errorf("Domain %s: Error prompting for deletion out confirmation, skipping - %s", domainName, promptErr)
					continue
				}
				if allowDelete {
					log.Infof("Domain %s: Delete has been approved", domainName)
					rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN DELETE %s", domainName))
					_, action, domDelErr := eppClient.DomainDelete(domainName)
					client.PushEPPActionLog(action)
					if domDelErr != nil {
						log.Errorf("\tError deleting domain %s - %s", domainName, domDelErr)
						return false
					}
					domainChangeMade = true
				}
			case lib.StateActive:
				eppResponse, eppDomainFound := (*domainMap)[domainName]
				if !eppDomainFound {
					log.Errorf("Domain %s: Domain information not found, skipping", domainName)
					continue
				}
				if eppResponse == nil || eppResponse.ResultData == nil || eppResponse.ResultData.DomainInfDataResp == nil {
					log.Errorf("Domain %s: EPP Info response is not valid, skipping", domainName)
					continue
				}
				// eppDomain := eppResponse.ResultData.DomainInfDataResp
				clientUpdate, clientDelete, clientTransfer, clientRenew, clientHold, flagErr := DiffDomainStatuses(eppResponse, domainRegObject)
				if flagErr != nil {
					log.Errorf("Domain %s: Error diffing statuses - %s", domainName, flagErr)
					continue
				}

				addHosts, remHosts, hostDiffErr := DiffHostList(eppResponse, domainRegObject)
				if hostDiffErr != nil {
					log.Errorf("Domain %s: Error diffing hosts - %s", domainName, hostDiffErr)
					continue
				}

				addDS, removeDS, dsDiffErr := lib.DiffDomainDSData(eppResponse, domainRegObject.CurrentRevision.DSDataEntries)
				if dsDiffErr != nil {
					log.Errorf("Domain %s: Error diffing DS records - %s", domainName, dsDiffErr)
				}

				if clientUpdate != nil || clientDelete != nil || clientTransfer != nil || clientRenew != nil || clientHold != nil || len(addHosts) != 0 || len(remHosts) != 0 || len(addDS) != 0 || len(removeDS) != 0 {
					log.Infof("Domain %s: Changes are required", domainName)
					unlockDone, unlockErr := DomainUnlockForChange(rr, client, domainName, eppClient, eppResponse)
					if unlockErr != nil {
						log.Errorf("Domain %s: Error unlocking domain - %s", domainName, unlockErr)
					}
					// in the likely event that client update was not changing but
					// something else was, we should relock the domain when the change
					// is complete
					if clientUpdate == nil && unlockDone {
						trueVal := true
						clientUpdate = &trueVal
					}

					if len(addHosts) != 0 {
						rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN UPDATE %s - add host(s)", domainName))
						_, action, eppErr := eppClient.DomainAddHosts(domainName, addHosts)
						client.PushEPPActionLog(action)
						if eppErr != nil {
							log.Errorf("Domain %s: error adding host(s) - %s", domainName, eppErr)
						} else {
							domainChangeMade = true
						}
					}
					if len(remHosts) != 0 {
						rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN UPDATE %s - remove host(s)", domainName))
						_, action, eppErr := eppClient.DomainRemoveHosts(domainName, remHosts)
						client.PushEPPActionLog(action)
						if eppErr != nil {
							log.Errorf("Domain %s: error removing host(s) - %s", domainName, eppErr)
						} else {
							domainChangeMade = true
						}
					}

					if len(addDS) != 0 {
						rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN UPDATE %s - DS Addition", domainName))
						_, action, eppErr := eppClient.DomainAddDSRecords(domainName, addDS)
						client.PushEPPActionLog(action)
						if eppErr != nil {
							log.Errorf("Domain %s: error adding DS record(s) - %s", domainName, eppErr)
						} else {
							domainChangeMade = true
						}
					}
					if len(removeDS) != 0 {
						rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN UPDATE %s - DS Deletion", domainName))
						_, action, eppErr := eppClient.DomainRemoveDSRecords(domainName, removeDS)
						client.PushEPPActionLog(action)
						if eppErr != nil {
							log.Errorf("Domain %s: error removing DS record(s) - %s", domainName, eppErr)
						} else {
							domainChangeMade = true
						}
					}

					addStatus := []string{}
					remStatus := []string{}

					binStatus(clientUpdate, epp.StatusClientUpdateProhibited, &addStatus, &remStatus)
					binStatus(clientDelete, epp.StatusClientDeleteProhibited, &addStatus, &remStatus)
					binStatus(clientTransfer, epp.StatusClientTransferProhibited, &addStatus, &remStatus)
					binStatus(clientRenew, epp.StatusClientRenewProhibited, &addStatus, &remStatus)
					binStatus(clientHold, epp.StatusClientHold, &addStatus, &remStatus)

					// remove statuses first, since the update lock will likely be added
					// in the next step
					if len(remStatus) != 0 {
						rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN UPDATE %s - Removing Status Flags", domainName))
						_, action, eppErr := eppClient.DomainRemoveStatuses(domainName, remStatus)
						client.PushEPPActionLog(action)
						if eppErr != nil {
							log.Errorf("Domain %s: error removing status flags - %s", domainName, eppErr)
						} else {
							domainChangeMade = true
						}
					}
					if len(addStatus) != 0 {
						rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN UPDATE %s - Adding Status Flags", domainName))
						_, action, eppErr := eppClient.DomainAddStatuses(domainName, addStatus)
						client.PushEPPActionLog(action)
						if eppErr != nil {
							log.Errorf("Domain %s: error removing status flags - %s", domainName, eppErr)
						} else {
							domainChangeMade = true
						}
					}
				}

				// Domain renewal
				blockRenew := false
				for _, status := range eppResponse.ResultData.DomainInfDataResp.Status {
					if status.StatusFlag == epp.StatusClientRenewProhibited || status.StatusFlag == epp.StatusServerRenewProhibited {
						blockRenew = true
					}
				}
				if !blockRenew {
					if domainRegObject.ExpireDate.Before(time.Now().Add(time.Hour * 24 * 365)) {
						rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN RENEW %s", domainName))
						renewalPeriod := epp.DomainPeriod{}
						renewalPeriod.Unit = epp.DomainPeriodYear
						renewalPeriod.Value = 1
						_, action, eppErr := eppClient.DomainRenew(domainName, domainRegObject.ExpireDate.Format("2006-01-02"), renewalPeriod)
						client.PushEPPActionLog(action)
						if eppErr != nil {
							log.Errorf("Domain %s: error renewing domain", domainName)
						} else {
							domainChangeMade = true
						}
					}
				}

			case lib.StateExternal:
				eppResponse, eppDomainFound := (*domainMap)[domainName]
				if !eppDomainFound {
					log.Errorf("Domain %s: Domain information not found, skipping", domainName)
					continue
				}
				if eppResponse == nil || eppResponse.ResultData == nil || eppResponse.ResultData.DomainInfDataResp == nil {
					log.Errorf("Domain %s: EPP Info response is not valid, skipping", domainName)
					continue
				}
				eppDomain := eppResponse.ResultData.DomainInfDataResp

				if eppDomain.ClientID != conf.VerisignEPP.RegistrarID {
					log.Infof("Domain %s: Already external, nothing to do", domainName)
					continue
				}
				allowTransferOut, promptErr := PromptToConfirm(fmt.Sprintf("Has the CSO, EVP of Platform *OR* the CEO approved the transfer of %s out of our control?", domainName))
				if promptErr != nil {
					log.Errorf("Domain %s: Error prompting for transfer out confirmation, skipping - %s", domainName, promptErr)
					continue
				}
				if allowTransferOut {
					log.Infof("Domain %s: Transfer out has been approved", domainName)

					clientTransferLocked := false
					serverTransferLocked := false
					for _, flag := range eppDomain.Status {
						if flag.StatusFlag == epp.StatusClientTransferProhibited {
							clientTransferLocked = true
						}
						if flag.StatusFlag == epp.StatusServerTransferProhibited {
							serverTransferLocked = true
						}
					}
					if serverTransferLocked {
						log.Infof("Domain %s: Domain transfer cannot begin, domain has %s flag set", domainName, epp.StatusServerTransferProhibited)
						rr.RegistryLockChanges = append(rr.RegistryLockChanges, fmt.Sprintf("DOMAIN %s - REMOVE - %s", domainName, epp.StatusServerTransferProhibited))
					}
					if clientTransferLocked {
						log.Infof("Doamin %s: Removing domain's client transfer lock", domainName)
						rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN UPDATE %s - remove client transfer lock", domainName))
						_, action, eppErr := eppClient.DomainRemoveStatuses(domainName, []string{epp.StatusClientTransferProhibited})
						client.PushEPPActionLog(action)
						if eppErr != nil {
							log.Errorf("Domain %s: error removing status flags - %s", domainName, eppErr)
						} else {
							domainChangeMade = true
						}
					}
					writeErr := PutAuthInfo(domainName, eppDomain.AuthPW.Password, conf)
					if writeErr != nil {
						log.Errorf("Domain %s: Error writing auth info to file - %s", domainName, writeErr)
					}
				} else {
					log.Infof("Domain %s: Transfer out has been rejected", domainName)
				}
			}
		}

		if domainChangeMade {
			domainUpdateErr := UpdateDomainInfo(rr, client, eppClient, domainName, domainMap)
			if domainUpdateErr != nil {
				log.Errorf("Domain %s: Error updating doamin info - %s", domainName, domainUpdateErr)
			}
		}
	}

	return true
}

func binStatus(status *bool, key string, addStatus, remStatus *[]string) {
	if status != nil {
		if *status {
			*addStatus = append(*addStatus, key)
		} else {
			*remStatus = append(*remStatus, key)
		}
	}
}

// HostUpdate will iterate over the hosts that require work and either create,
// update or delete the hosts as defined by the object status
func HostUpdate(rr *RunReport, client client.Client, eppClient *superclient.SuperClient, verifiedHosts *map[string]*lib.HostExport, verifiedDomains *map[string]*lib.DomainExport, ha *map[string]bool, hostMap *map[string]*epp.Response, skipHostnamems *[]string) (shouldContinue bool) {
	for hostname, hostRegObject := range *verifiedHosts {
		log.Infof("Host %s: Starting to process host", hostname)

		hostChangeMade := false

		parentDomain, parentDomainErr := GetParentDomain(hostname)
		if parentDomainErr != nil {
			log.Errorf("Unable to get parent domain for %s - %s", hostname, parentDomainErr)
			return false
		}
		dom, ok := (*verifiedDomains)[parentDomain]
		if !ok {
			continue
		}
		if dom.CurrentRevision.RevisionState == lib.StateExternal {
			log.Noticef("Domains %s is currently external, cannout modify hostname %s. Skipping", dom.DomainName, hostname)
			continue
		}

		// Check to see if the host exists or not
		hostAvailable, hostAvailableFound := (*ha)[hostname]
		if !hostAvailableFound {
			log.Errorf("Host %s: No availability information found, skipping host", hostname)
			continue
		}

		if hostAvailable {
			// The host has not been created yet, lets go and create it
			if hostRegObject.CurrentRevision.DesiredState != lib.StateActive {
				log.Infof("Host %s: Host is not set to the Active state, skipping creation (desired state = %s)", hostname, hostRegObject.CurrentRevision.DesiredState)
				continue
			}

			var ipv4Addresses, ipv6Addresses []string
			for _, ip := range hostRegObject.CurrentRevision.HostAddresses {
				switch ip.Protocol {
				case 4:
					ipv4Addresses = append(ipv4Addresses, ip.IPAddress)
				case 6:
					ipv6Addresses = append(ipv6Addresses, ip.IPAddress)
				}
			}

			rr.AddWorkItem(fmt.Sprintf("EPP HOST CREATE %s", hostname))
			_, action, hosCreateErr := eppClient.HostCreate(hostname, ipv4Addresses, ipv6Addresses)
			client.PushEPPActionLog(action)
			if hosCreateErr != nil {
				log.Errorf("\tError creating host %s - %s", hostname, hosCreateErr)
				return false
			}
			hostChangeMade = true
		} else {
			// The host exists already, we should update it
			if hostRegObject.CurrentRevision.DesiredState == lib.StateInactive {
				// TODO: Check for server status flags that prevent delete
				// TODO: Unlock host (update and delete status flags)

				log.Infof("Host %s: Host is marked as inactive, deleting the object", hostname)
				rr.AddWorkItem(fmt.Sprintf("EPP HOST DELETE %s", hostname))
				_, action, hosDelErr := eppClient.HostDelete(hostname)
				client.PushEPPActionLog(action)
				if hosDelErr != nil {
					log.Errorf("\tError deleting host %s - %s", hostname, hosDelErr)
					return false
				}
				hostChangeMade = true
			} else if hostRegObject.CurrentRevision.DesiredState == lib.StateActive {
				// Check for server locks
				log.Infof("Host %s: Host is marked as active, requires updating", hostname)

				var isUpdateUnlocked bool

				ipv4Add, ipv4Rem, ipv6Add, ipv6Rem, diffErr := lib.DiffIPsExport((*hostMap)[hostname], hostRegObject)
				if diffErr != nil {
					log.Errorf("Host %s: Error diffing ips - %s", hostname, diffErr)
					continue
				}

				clientUpdate, clientDelete, diffStatusErr := DiffHostStatuses((*hostMap)[hostname], hostRegObject)
				if diffStatusErr != nil {
					log.Errorf("Host %s: Error diffing statuses - %s", hostname, diffStatusErr)
					continue
				}

				// Check if any changes need to be made and unlock if needed
				if len(ipv4Add) > 0 || len(ipv4Rem) > 0 || len(ipv6Add) > 0 || len(ipv6Rem) > 0 || clientUpdate != nil || clientDelete != nil {
					unlockErr := HostUnlockForChange(rr, client, hostname, eppClient, (*hostMap)[hostname])
					if unlockErr != nil {
						log.Errorf("Host %s: Error unlocking domain for change - %s", hostname, unlockErr)
						fmt.Println(*hostMap)
						continue
					}
					hostChangeMade = true
					isUpdateUnlocked = true
				} else {
					log.Infof("Host %s: No changes are required, skipping", hostname)
					continue
				}

				if len(ipv4Add) > 0 || len(ipv4Rem) > 0 || len(ipv6Add) > 0 || len(ipv6Rem) > 0 {
					log.Infof("Host %s: Update host IP addresses", hostname)
					rr.AddWorkItem(fmt.Sprintf("EPP HOST UPDATE %s - ip update", hostname))
					_, action, changeErr := eppClient.HostChangeIPAddresses(hostname, ipv4Add, ipv4Rem, ipv6Add, ipv6Rem)
					client.PushEPPActionLog(action)
					if changeErr != nil {
						log.Errorf("Host %s: Error changing ip addresses - %s", hostname, changeErr)
					}
					hostChangeMade = true
				} else {
					log.Infof("Host %s: No IP changes to make", hostname)
				}

				if hostRegObject.CurrentRevision.ClientUpdateProhibitedStatus && isUpdateUnlocked {
					// Host was unlocked for changes but should be relocked
					log.Infof("Host %s: Host was unlocked for changes, it needs to be relocked", hostname)
					trueVal := true
					clientUpdate = &trueVal
				}

				if !hostRegObject.CurrentRevision.ClientUpdateProhibitedStatus && isUpdateUnlocked {
					// Host was unlocked so we dont need to unlock the host again
					log.Infof("Host %s: Host is already unlocked for changes, client update change not needed any more", hostname)
					clientUpdate = nil
				}

				if clientDelete != nil || clientUpdate != nil {
					log.Infof("Host %s: Update host client status flags", hostname)
					rr.AddWorkItem(fmt.Sprintf("EPP HOST UPDATE %s - status update", hostname))
					_, action, hostUpdateErr := eppClient.HostStatusUpdate(hostname, clientUpdate, clientDelete)
					client.PushEPPActionLog(action)
					if hostUpdateErr != nil {
						log.Errorf("\tError updating host %s statuses - %s", hostname, hostUpdateErr)
						return false
					}
					hostChangeMade = true
				}
			}
		}

		if hostChangeMade {
			hostUpdateErr := UpdateHostInfo(rr, client, eppClient, hostname, hostMap)
			if hostUpdateErr != nil {
				log.Errorf("Host %s: Error updating host info - %s", hostname, hostUpdateErr)
			}
		}
		log.Infof("Host %s: Done processing host", hostname)
	}
	return true
}

// UpdateHostInfo will attempt to query the registry for the host information.
// If an error is encountered, it will be returned.
func UpdateHostInfo(rr *RunReport, client client.Client, eppClient *superclient.SuperClient, hostname string, hostMap *map[string]*epp.Response) error {
	rr.AddWorkItem(fmt.Sprintf("EPP HOST INFO %s", hostname))
	_, fullResponse, action, hostInfoErr := eppClient.HostInfo(hostname)
	client.PushEPPActionLog(action)
	(*hostMap)[hostname] = fullResponse
	return hostInfoErr
}

// UpdateDomainInfo will attempt to quert the registry for the domain
// information. If an error is encountered, it will be returned.
func UpdateDomainInfo(rr *RunReport, client client.Client, eppClient *superclient.SuperClient, domainName string, domainMap *map[string]*epp.Response) error {
	rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN INFO %s", domainName))
	_, fullResponse, action, domainInfoErr := eppClient.DomainInfo(domainName, epp.DomainInfoHostsDelegated, nil)
	client.PushEPPActionLog(action)
	(*domainMap)[domainName] = fullResponse
	return domainInfoErr
}

// DomainUnlockForChange will attempt to unlock a domain to allow changes to be
// made to the domain. If an error occurs unlocking the domain, the error will
// be returned, otherwise a flag indicating that the if domain has been unlocked
// will be returned
func DomainUnlockForChange(rr *RunReport, client client.Client, domainName string, eppClient *superclient.SuperClient, registryState *epp.Response) (unlockRequired bool, err error) {
	if registryState != nil && registryState.ResultData != nil && registryState.ResultData.DomainInfDataResp != nil {
		isClientUpdateLocked := false
		for _, status := range registryState.ResultData.DomainInfDataResp.Status {
			if status.StatusFlag == epp.StatusServerUpdateProhibited {
				log.Errorf("Domain %s: Registry lock is in place, cannot unlock", domainName)
				return false, errors.New("Ojbect has the StatusServerUpdateProhibited flag set, cannot update")
			}
			if status.StatusFlag == epp.StatusClientUpdateProhibited {
				log.Infof("Domain %s: Client update lock is set, removing before change can be made", domainName)
				isClientUpdateLocked = true
			}
		}
		if isClientUpdateLocked {
			log.Infof("Domain %s: Removing client update lock", domainName)
			rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN UPDATE %s - status update", domainName))
			_, action, updateErr := eppClient.DomainRemoveStatuses(domainName, []string{epp.StatusClientUpdateProhibited})
			client.PushEPPActionLog(action)
			return true, updateErr
		}
		return false, nil
	}
	return false, errors.New("Unable to find host information from registry")
}

// HostUnlockForChange will attempt to unlock a host to allow changes to be
// made to the host. HostResetFlags should be called after the action is
// completed to ensure that all flags are restored
func HostUnlockForChange(rr *RunReport, client client.Client, hostname string, eppClient *superclient.SuperClient, registryState *epp.Response) error {
	if registryState != nil && registryState.ResultData != nil && registryState.ResultData.HostInfDataResp != nil {
		isClientUpdateLocked := false
		for _, status := range registryState.ResultData.HostInfDataResp.Status {
			if status.StatusFlag == epp.StatusServerUpdateProhibited {
				log.Errorf("Host %s: Registry lock is in place, cannot unlock", hostname)
				return errors.New("Object has the serverUpdateProhibited flag set, cannot update")
			}
			if status.StatusFlag == epp.StatusClientUpdateProhibited {
				log.Infof("Host %s: Client update lock set, removing before changes can be made", hostname)
				isClientUpdateLocked = true
			}
		}
		if isClientUpdateLocked {
			log.Infof("Host %s: Removing client update lock", hostname)
			rr.AddWorkItem(fmt.Sprintf("EPP HOST UPDATE %s - status update", hostname))
			falseFlag := false
			_, action, updateErr := eppClient.HostStatusUpdate(hostname, &falseFlag, nil)
			client.PushEPPActionLog(action)
			return updateErr
		}
		return nil
	}
	return errors.New("Unable to find host infomration from registry")
}

// DiffHostList will take an epp reponse for a Domain and a domain object from
// the Registrar system and generate a list of hosts that need to be added
// to the domain record at the registry and a list of hosts that need to be
// removed from the domain record at the registry to make the registry match
// the registrar server. If an error occurs during the process, it will be
// retruned
func DiffHostList(registry *epp.Response, registrar *lib.DomainExport) (addHosts, remHosts []string, err error) {
	if registry == nil || registry.ResultData == nil || registry.ResultData.DomainInfDataResp == nil {
		err = errors.New("Domain info section of registry response is empty")
		return
	}

	currentList := []string{}
	expectedList := []string{}

	for _, current := range registry.ResultData.DomainInfDataResp.NSHosts.Hosts {
		currentList = append(currentList, current.Value)
	}
	for _, expected := range registrar.CurrentRevision.Hostnames {
		expectedList = append(expectedList, expected.HostName)
	}

	for _, current := range currentList {
		found := false
		for _, expected := range expectedList {
			if current == expected {
				found = true
			}
		}
		if !found {
			remHosts = append(remHosts, current)
		}
	}

	for _, expected := range expectedList {
		found := false
		for _, current := range currentList {
			if current == expected {
				found = true
			}
		}
		if !found {
			addHosts = append(addHosts, expected)
		}
	}
	return
}

// DiffDomainStatuses will take an epp reponse for a domain and a domain object
// from the Registrar system and generate a pointers to bools that will be
// nil if no changes are required and nil otherwise. If an error occurs it will
// be returned.
func DiffDomainStatuses(registry *epp.Response, registrar *lib.DomainExport) (clientUpdate, clientDelete, clientTransfer, clientRenew, clientHold *bool, err error) {
	if registry == nil || registry.ResultData == nil || registry.ResultData.DomainInfDataResp == nil {
		err = errors.New("Domain info section of registry response is empty")
		return
	}

	registryClientDeleteProhibited := false
	registryClientUpdateProhibited := false
	registryClientTransferProhibited := false
	registryClientRenewProhibited := false
	registryClientHold := false

	for _, status := range registry.ResultData.DomainInfDataResp.Status {
		switch status.StatusFlag {
		case epp.StatusClientDeleteProhibited:
			registryClientDeleteProhibited = true
		case epp.StatusClientUpdateProhibited:
			registryClientUpdateProhibited = true
		case epp.StatusClientTransferProhibited:
			registryClientTransferProhibited = true
		case epp.StatusClientRenewProhibited:
			registryClientRenewProhibited = true
		case epp.StatusClientHold:
			registryClientHold = true
		}
	}

	if registrar.CurrentRevision.ClientDeleteProhibitedStatus != registryClientDeleteProhibited {
		clientDelete = &registrar.CurrentRevision.ClientDeleteProhibitedStatus
	}
	if registrar.CurrentRevision.ClientUpdateProhibitedStatus != registryClientUpdateProhibited {
		clientUpdate = &registrar.CurrentRevision.ClientUpdateProhibitedStatus
	}
	if registrar.CurrentRevision.ClientTransferProhibitedStatus != registryClientTransferProhibited {
		clientTransfer = &registrar.CurrentRevision.ClientTransferProhibitedStatus
	}
	if registrar.CurrentRevision.ClientRenewProhibitedStatus != registryClientRenewProhibited {
		clientRenew = &registrar.CurrentRevision.ClientRenewProhibitedStatus
	}
	if registrar.CurrentRevision.ClientHoldStatus != registryClientHold {
		clientHold = &registrar.CurrentRevision.ClientHoldStatus
	}

	return
}

// DiffHostStatuses will take an epp reponse for a host and a host object from
// the Registrar system and generate a pointers to bools that will be nil if
// no changes are required and nil otherwise. If an error occurs it will be
// returned.
func DiffHostStatuses(registry *epp.Response, registrar *lib.HostExport) (clientUpdate, clientDelete *bool, err error) {
	if registry == nil || registry.ResultData == nil || registry.ResultData.HostInfDataResp == nil {
		err = errors.New("Host info section of registry response is empty")
		return
	}

	registryClientDeleteProhibited := false
	registryClientUpdateProhibited := false

	for _, status := range registry.ResultData.HostInfDataResp.Status {
		switch status.StatusFlag {
		case epp.StatusClientDeleteProhibited:
			registryClientDeleteProhibited = true
		case epp.StatusClientUpdateProhibited:
			registryClientUpdateProhibited = true
		}
	}

	if registrar.CurrentRevision.ClientDeleteProhibitedStatus != registryClientDeleteProhibited {
		clientDelete = &registrar.CurrentRevision.ClientDeleteProhibitedStatus
	}
	if registrar.CurrentRevision.ClientUpdateProhibitedStatus != registryClientUpdateProhibited {
		clientUpdate = &registrar.CurrentRevision.ClientUpdateProhibitedStatus
	}

	return
}

// func DiffDomainDSData(registry *epp.Response, registrar *lib.DomainExport) (addDSRecords, remDSRecords []epp.DSData, err error) {
// 	if registry == nil {
// 		err = errors.New("Domain info section of registry response is empty")
// 		return
// 	}
//
// 	currentDSData := make(map[string]epp.DSData)
// 	expectedDSData := make(map[string]epp.DSData)
//
// 	if registry.Extension != nil && registry.Extension.SecDNSInfData != nil {
// 		for _, entry := range registry.Extension.SecDNSInfData.DSData {
// 			strRep := fmt.Sprintf("%d:%d:%d:%s", entry.KeyTag, entry.Alg, entry.DigestType, entry.Digest)
// 			currentDSData[strRep] = entry
// 		}
// 	}
//
// 	for _, entry := range registrar.CurrentRevision.DSDataEntries {
// 		strRep := fmt.Sprintf("%d:%d:%d:%s", entry.KeyTag, entry.Algorithm, entry.DigestType, entry.Digest)
// 		dsdata := epp.DSData{Alg: int(entry.Algorithm), Digest: entry.Digest, DigestType: int(entry.DigestType), KeyTag: int(entry.KeyTag)}
// 		expectedDSData[strRep] = dsdata
// 	}
//
// 	for current, currentVal := range currentDSData {
// 		found := false
// 		for expected := range expectedDSData {
// 			if current == expected {
// 				found = true
// 			}
// 		}
// 		if !found {
// 			remDSRecords = append(remDSRecords, currentVal)
// 		}
// 	}
//
// 	for expected, expectedVal := range expectedDSData {
// 		found := false
// 		for current := range currentDSData {
// 			if current == expected {
// 				found = true
// 			}
// 		}
// 		if !found {
// 			addDSRecords = append(addDSRecords, expectedVal)
// 		}
// 	}
//
// 	return
// }

// DomainTransferInRequests will inspect domains from Registrar and
// the domain information from the registry and deteremine if the a domain
// transfer request should be issued. If it is deteremined that a domain
// transfer is required, the authinfo will be retrieved from the directory
// defined in the config and then the transfer request will be submitted
func DomainTransferInRequests(conf Config, rr *RunReport, client client.Client, eppClient *superclient.SuperClient, verifiedDomains *map[string]*lib.DomainExport, da *map[string]bool, domainMap *map[string]*epp.Response) (shouldContinue bool) {
	for _, dom := range *verifiedDomains {
		domainName := dom.DomainName
		if dom.CurrentRevision.DesiredState == lib.StateActive {
			if val, ok := (*da)[domainName]; ok && !val {
				if domainInfo, diOK := (*domainMap)[domainName]; diOK {
					if domainInfo != nil && domainInfo.ResultData != nil &&
						domainInfo.ResultData.DomainInfDataResp != nil {
						if domainInfo.ResultData.DomainInfDataResp.ClientID != conf.VerisignEPP.RegistrarID {
							log.Infof("Domain %s is external (sponsored by %s) and should be transferred in", domainName, domainInfo.ResultData.DomainInfDataResp.ClientID)
							hasLockFlag := false
							for _, status := range domainInfo.ResultData.DomainInfDataResp.Status {
								if status.StatusFlag == "clientTransferProhibited" || status.StatusFlag == "serverTransferProhibited" {
									hasLockFlag = true
									break
								}
							}
							if hasLockFlag {
								log.Infof("The domain %s has a transfer lock enabled, skipping transfer", domainName)
								continue
							}

							ai, aierr := GetAuthInfo(domainName, conf)
							if aierr != nil {
								log.Errorf("\tUnable to find auth info for %s, %s", domainName, aierr)
								continue
							}

							log.Infof("\tAuthInfo found for transfer of Domain %s, requesting transfer", domainName)
							rr.AddWorkItem(fmt.Sprintf("EPP DOMAIN TRANSFER REQUEST %s", domainName))
							_, respCode, action, trerr := eppClient.RequestDomainTransfer(domainName, ai)
							client.PushEPPActionLog(action)
							if trerr != nil {
								log.Errorf("\tAn error occured trying to request a transfer for domain %s - (%d) %s", domainName, respCode, trerr)
								return false
							}
							log.Infof("\tDomain %s transfer has been requested", domainName)
							rr.DomainsTransferRequested = append(rr.DomainsTransferRequested, domainName)
						}
					}
				}
			}
		}
	}
	return true
}

// GetAuthInfo will look in the directory specified in the configuration for a
// file that will contain the auth info for a the given domain name, if the
// auth info is found, it will be returned, otherwise an error is returned
func GetAuthInfo(domainName string, conf Config) (ai string, err error) {
	data, err := os.ReadFile(path.Join(conf.Transfer.AuthInfoIn, domainName))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// PutAuthInfo will attempt to write a file with the auth info for the given
// domain name into a file with a name of the domain name. If there is an error
// creating the folder or the file, it will be returned
func PutAuthInfo(domainName string, authInfo string, conf Config) (err error) {
	if _, err := os.Stat(conf.Transfer.AuthInfoOut); err != nil {
		if os.IsNotExist(err) {
			createErr := os.MkdirAll(conf.Transfer.AuthInfoOut, 0700)
			if createErr != nil {
				return err
			}
		} else {
			return err
		}
	}

	return os.WriteFile(path.Join(conf.Transfer.AuthInfoOut, domainName), []byte(authInfo), 0700)
}

// VerifyApproval will take a client, approval and a change request object and
// will attempt to verify that the approval is authentic returning true iff it
// has been transitivly signed by one of the trust anchors and a list of errors
// if any occur.
func VerifyApproval(client client.Client, app lib.ApprovalExport, cr *lib.ChangeRequestExport) (pass bool, errs []error) {
	pass = false
	log.Infof("Approval %d: Starting verification", app.ID)

	as, getAppSetErrs := client.GetApproverSetAt(app.ApproverSetID, app.CreatedAt.Unix())
	if len(getAppSetErrs) != 0 {
		errs = getAppSetErrs
		log.Errorf("Approval %d: Error getting the approver set for verifying", app.ID)
		return
	}

	asVerified, ase, asErrs := client.VerifyApproverSet(as)
	if len(asErrs) != 0 {
		errs = asErrs
		log.Errorf("Approval %d: Error verifying approver set", app.ID)
		return
	}
	if !asVerified {
		errMsg := fmt.Sprintf("Approver Set %d was not verified", app.ApproverSetID)
		errs = append(errs, errors.New(errMsg))
		log.Errorf("Approval %d: Approver set did not verify", app.ID)
		return
	}

	wasSignedByAppSet, data := ase.IsSignedBy(app.Signature)
	if !wasSignedByAppSet {
		errorStr := fmt.Sprintf("Approval %d: Approval was not signed by the approval set", app.ID)
		log.Error(errorStr)
		errs = append(errs, errors.New(errorStr))
		return
	}
	log.Infof("Approval %d: request was signed by Approver Set %d - %s", app.ID, ase.ID, ase.Description)

	var aa lib.ApprovalAttestationUnmarshal
	unmarshalErr := json.Unmarshal(data, &aa)
	if unmarshalErr != nil {
		log.Errorf("Approval %d: Error unmarshaling approval assertion - %s", app.ID, unmarshalErr)
		errs = append(errs, unmarshalErr)
		return
	}

	signedData, err := aa.ExportRev.MarshalJSON()
	if err != nil {
		errs = append(errs, err)
		log.Errorf("Approval %d: Error converting the export revision to string - %s", app.ID, err)
		return
	}

	switch cr.RegistrarObjectType {
	case lib.DomainType:
		return DomainVerify(client, signedData, app, cr)
	case lib.HostType:
		return HostVerify(client, signedData, app, cr)
	case lib.ContactType:
		return ContactVerify(client, signedData, app, cr)

	case lib.APIUserType:
		return APIUserVerify(client, signedData, app, cr)
	case lib.ApproverType:
		return ApproverVerify(client, signedData, app, cr)
	case lib.ApproverSetType:
		return ApproverSetVerify(client, signedData, app, cr)
	}
	err = fmt.Errorf("Unhandled object type %s", cr.RegistrarObjectType)
	errs = append(errs, err)
	return false, errs
}

// ContactVerify will take a client, signed data, an approval and the change
// request and attempt to verify the contact object. Iff the contact is verified
// it will return true and if not false will be returned with a list of errors
// that have occured verifying the contact.
func ContactVerify(client client.Client, signedData []byte, app lib.ApprovalExport, cr *lib.ChangeRequestExport) (pass bool, errs []error) {
	log.Infof("Approval %d: Approval for a Contact", app.ID)
	ce := lib.ContactExport{}
	err := json.Unmarshal(signedData, &ce)
	if err != nil {
		errs = append(errs, err)
		log.Errorf("Approval %d: Error unmarshaling the signed approval - %s", app.ID, err)
		return
	}
	conr, errs3 := client.GetContactRevision(cr.ProposedRevisionID)
	if len(errs3) != 0 {
		errs = append(errs3, err)
		log.Errorf("Approval %d: Error getting the contact revision - %s", app.ID, err)
		return
	}
	comparePass, errs2 := conr.CompareExport(ce.PendingRevision)
	if len(errs2) != 0 {
		log.Errorf("Approval %d: The pending revision did not match the saved version", app.ID)
		for _, err := range errs2 {
			log.Errorf("       %s", err)
		}
		errs = append(errs, errs2...)
		return
	}
	if !comparePass {
		log.Errorf("Approval %d: The object comparision failed", app.ID)
	} else {
		pass = true
		log.Infof("Approval %d: The approval matched", app.ID)
	}
	return
}

// HostVerify will take a client, signed data, an approval and the change
// request and attempt to verify the host object. Iff the contact is verified
// it will return true and if not false will be returned with a list of errors
// that have occured verifying the host.
func HostVerify(client client.Client, signedData []byte, app lib.ApprovalExport, cr *lib.ChangeRequestExport) (pass bool, errs []error) {
	log.Infof("Approval %d: Approval for a Host", app.ID)
	he := lib.HostExport{}
	err := json.Unmarshal(signedData, &he)
	if err != nil {
		errs = append(errs, err)
		log.Errorf("Approval %d: Error unmarshaling the signed approval - %s", app.ID, err)
		return
	}
	hr, errs3 := client.GetHostRevision(cr.ProposedRevisionID)
	if len(errs3) != 0 {
		errs = append(errs3, err)
		log.Errorf("Approval %d: Error getting the host revision - %s", app.ID, err)
		return
	}
	comparePass, errs2 := hr.CompareExport(he.PendingRevision)
	if len(errs2) != 0 {
		log.Errorf("Approval %d: The pending revision did not match the saved version", app.ID)
		for _, err := range errs2 {
			log.Errorf("       %s", err)
		}
		errs = append(errs, errs2...)
		return
	}
	if !comparePass {
		log.Errorf("Approval %d: The object comparision failed", app.ID)
	} else {
		pass = true
		log.Infof("Approval %d: The approval matched", app.ID)
	}
	return
}

// DomainVerify will take a client, signed data, an approval and the change
// request and attempt to verify the host object. Iff the domain is verified
// it will return true and if not false will be returned with a list of errors
// that have occured verifying the domain.
func DomainVerify(client client.Client, signedData []byte, app lib.ApprovalExport, cr *lib.ChangeRequestExport) (pass bool, errs []error) {
	log.Infof("Approval %d: Approval for a Domain", app.ID)
	de := lib.DomainExport{}
	err := json.Unmarshal(signedData, &de)
	if err != nil {
		errs = append(errs, err)
		log.Errorf("Approval %d: Error unmarshaling the signed approval - %s", app.ID, err)
		return
	}
	dr, errs3 := client.GetDomainRevision(cr.ProposedRevisionID)
	if len(errs3) != 0 {
		errs = append(errs3, err)
		log.Errorf("Approval %d: Error getting the domain revision - %s", app.ID, err)
		return
	}
	comparePass, errs2 := dr.CompareExport(de.PendingRevision)
	if len(errs2) != 0 {
		log.Errorf("Approval %d: The pending revision did not match the saved version", app.ID)
		for _, err := range errs2 {
			log.Errorf("       %s", err)
		}
		errs = append(errs, errs2...)
		return
	}
	if !comparePass {
		log.Errorf("Approval %d: The object comparision failed", app.ID)
	} else {
		pass = true
		log.Infof("Approval %d: The approval matched", app.ID)
	}
	return
}

// PromptToConfirm will take a question request and prompt the user to answer
// yes or no to the prompt. If the user does not type yes or no the function
// will prompt again for the user to answer again until one of the valid
// answers is entered.
func PromptToConfirm(request string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\n\n%s (Yes/No)\n", request)
		text, readErr := reader.ReadString('\n')
		if readErr != nil {
			return false, readErr
		}

		if text == "Yes\n" {
			return true, nil
		}
		if text == "No\n" {
			return false, nil
		}

		fmt.Printf("Error: Invalid input your input must be \"Yes\" or \"No\"\n")
	}
}

func handleDomainTransferOutRequest(domainname string, verifiedDomains *map[string]*lib.DomainExport) (bool, error) {
	// default to not allow the transfer
	allowTransfer := false
	log.Infof("Begin: Handle transfer out request for %s", domainname)
	var workingDomain *lib.DomainExport
	for domainName, val := range *verifiedDomains {
		if domainName == domainname {
			workingDomain = val
			log.Infof("\tFound %s in the list of domains that require work", domainname)
			break
		}
	}
	if workingDomain == nil {
		log.Infof("\tUnable to find %s in the list of domains that require work", domainname)
		return false, nil
	}

	if workingDomain.CurrentRevision.DesiredState != lib.StateExternal {
		log.Infof("\t%s did not have the desired state of 'external'", domainname)
		return false, nil
	}
	log.Infof("\t%s did has the desired state of 'external', preparing for final verification", domainname)

	transferOutApproved, promptErr := PromptToConfirm(fmt.Sprintf("Has the CSO, EVP of Platform *OR* the CEO approved the transfer of %s out of our control?", domainname))
	if promptErr != nil {
		log.Errorf("\tError with final verification %s, ", promptErr)
		return false, promptErr
	}

	if transferOutApproved {
		log.Infof("\tThe transfer out of %s has been approved, proceeding with transfer approval", domainname)
		return true, nil
	}

	return allowTransfer, nil
}

func handleDomainWork(conf Config, client client.Client, eppClient *superclient.SuperClient, obj *lib.DomainExport) error {
	DomainName := strings.ToUpper(obj.DomainName)
	log.Infof("Domain %s: Starting to process", DomainName)

	// Start by checking if the domain exists or not
	log.Fatal("Is this still used?")
	domainAvail, _, action, domainAvailErr := eppClient.DomainAvailable(DomainName)
	client.PushEPPActionLog(action)
	if domainAvailErr != nil {
		log.Error(domainAvailErr)
		log.Infof("Domain %s: Error processing domain, skipping rest of domain processing", DomainName)
		return domainAvailErr
	}

	// If the domain is available and has the desired state of active, register
	// it otherwise throw an error.
	if domainAvail {
		log.Infof("Domain %s: The domain is not registered yet", DomainName)
		if obj.CurrentRevision.DesiredState == lib.StateActive {
			log.Infof("Domain %s: Domain needs to be created", DomainName)
			log.Fatal("Is this still used?")
			_, _, err := eppClient.DomainCreate(DomainName, 1)
			if err != nil {
				return err
			}
		} else {
			log.Errorf("Domain %s: Domain name does not exist but is marked as %s", DomainName, obj.CurrentRevision.DesiredState)
			log.Infof("Domain %s: Error processing domain, skipping rest of domain processing", DomainName)
			return nil
		}
	}

	// The domain should now be created or we should have exitied by now. Lets
	// grab the current state from the registry
	log.Fatal("Is this still used?")
	eppDomain, fullResponse, action, domainInfoErr := eppClient.DomainInfo(DomainName, epp.DomainInfoHostsDelegated, nil)
	client.PushEPPActionLog(action)
	if domainInfoErr != nil {
		log.Error(domainInfoErr)
		log.Infof("Domain %s: Error processing domain, skipping rest of domain processing", DomainName)
		return domainInfoErr
	}

	// Sanity check, the domain info object from the registry should not be nil,
	// this should never happen, but its worth checking
	if eppDomain == nil {
		log.Errorf("Domain %s: There was an error with the response from the registry, it was nil", DomainName)
		log.Infof("Domain %s: Error processing domain, skipping rest of domain processing", DomainName)
		return nil
	}

	// Sanity check, is the domain we got back the domain we are looking for, this
	// should never happen
	if strings.EqualFold(obj.DomainName, eppDomain.Name) {
		log.Errorf("Domain %s: Name mismatch %s - %s", DomainName, strings.ToUpper(obj.DomainName), strings.ToUpper(eppDomain.Name))
		log.Infof("Domain %s: Error processing domain, skipping rest of domain processing", DomainName)
		return nil
	}

	// For domains that are owned by some other regitry
	if eppDomain.ClientID != conf.VerisignEPP.RegistrarID {
		log.Infof("Domain %s: Owned by another registrar", DomainName)

		if obj.CurrentRevision.DesiredState == lib.StateExternal {
			// In this case, the domain is marked as being external anyway, so it
			// looks like we have nothing to do. Lets send the domain object back to
			// the registrar server just to keep it up to date
			log.Infof("Domain %s: Expected to be external, nothing to do", DomainName)
			pushErr := PushDomainInfo(client, obj.ID, fullResponse)
			if pushErr != nil {
				log.Error(pushErr)
				log.Infof("Domain %s: Error processing domain, skipping rest of domain processing", DomainName)
				return pushErr
			}
		} else {
			log.Infof("Domain %s: Should be transferred into this registrar", DomainName)
		}
	} else {
		log.Infof("Domain %s: Owned by this registrar", DomainName)
		if obj.CurrentRevision.DesiredState == lib.StateExternal {
			log.Infof("Domain %s: State set to external, allowed for transition out assuming approval", DomainName)
			transferOutApproved, promptErr := PromptToConfirm(fmt.Sprintf("Has the CSO, EVP of Platform or the CEO approved the transfer of %s out of our control?", DomainName))
			if promptErr != nil {
				log.Error(promptErr)
			} else {
				if transferOutApproved {
					log.Infof("Domain %s: Transfer out approved", DomainName)
				} else {
					log.Infof("Domain %s: Transfer out not approved", DomainName)
				}
			}
		}
	}

	pushErr := PushDomainInfo(client, obj.ID, fullResponse)
	if pushErr != nil {
		log.Error(pushErr)
		log.Infof("Domain %s: Error processing domain, skipping rest of domain processing", DomainName)
		return pushErr
	}
	log.Infof("Domain %s: Complete processing", DomainName)

	return nil
}

// PushDomains will iterate through the list of domain responses that have been
// gathered and push the domain objects to the Registrar server
func PushDomains(client client.Client, verifiedDomains *map[string]*lib.DomainExport, domainMap *map[string]*epp.Response) (shouldContinue bool) {
	for domainName, domainEPP := range *domainMap {
		if domainRegObject, ok := (*verifiedDomains)[domainName]; ok && domainRegObject != nil {
			if domainEPP == nil {
				pushErr := DomainUnsetCheck(client, domainRegObject.ID)
				if pushErr != nil {
					log.Errorf("Domain %s: Error unsetting check in Registrar - %s", domainName, pushErr)
				}
				log.Infof("Domain %s: Has been marked as no check required", domainName)
				continue
			}

			log.Infof("Domain %s: Pushing domain to Registrar", domainName)
			pushErr := PushDomainInfo(client, domainRegObject.ID, domainEPP)
			if pushErr != nil {
				log.Errorf("Domain %s: Error pushing domain to Registrar - %s", domainName, pushErr)
			}
		} else {
			log.Infof("Domain %s: Cannot find the Registrar object id", domainName)
		}
	}
	return true
}

// PushHosts will iterate through the list of host responses that have been
// gathered and push the host objects to the Registrar server
func PushHosts(client client.Client, verifiedHosts *map[string]*lib.HostExport, hostMap *map[string]*epp.Response) (shouldContinue bool) {
	for hostName, hostEPP := range *hostMap {
		if hostRegObject, ok := (*verifiedHosts)[hostName]; ok && hostRegObject != nil {
			if hostEPP == nil {
				pushErr := HostUnsetCheck(client, hostRegObject.ID)
				if pushErr != nil {
					log.Errorf("Host %s: Error unsetting check in Registrar - %s", hostName, pushErr)
				}
				log.Infof("Host %s: Has been marked as no check required", hostName)
				continue
			}

			log.Infof("Host %s: Pushing host to Registrar", hostName)
			pushErr := PushHostInfo(client, hostRegObject.ID, hostEPP)
			if pushErr != nil {
				log.Errorf("Host %s: Error pushing host to Registrar - %s", hostName, pushErr)
			}
		} else {
			log.Infof("Host %s: Cannot find the Registrar object id", hostName)
		}
	}
	return true
}

// PushDomainInfo will take the given domain info response object and push it
// to the registrar server
func PushDomainInfo(client client.Client, objectID int64, resp *epp.Response) error {
	pushErrs := client.PushInfoEPP(lib.DomainType, objectID, resp)
	if len(pushErrs) != 0 {
		for _, err := range pushErrs {
			log.Errorf("Error pushing domain %d - %s", objectID, err)
		}
		return pushErrs[0]
	}
	return nil
}

// DomainUnsetCheck will toggle the check required field for a domain when no
// epp data is available
func DomainUnsetCheck(client client.Client, objectID int64) error {
	pushErrs := client.UnsetEPPCheck(lib.DomainType, objectID)
	if len(pushErrs) != 0 {
		for _, err := range pushErrs {
			log.Errorf("Error pushing domain %d - %s", objectID, err)
		}
		return pushErrs[0]
	}
	return nil
}

// PushHostInfo will take the given host info reponse object and push it to the
// the registrar server
func PushHostInfo(client client.Client, objectID int64, resp *epp.Response) error {
	pushErrs := client.PushInfoEPP(lib.HostType, objectID, resp)
	if len(pushErrs) != 0 {
		for _, err := range pushErrs {
			log.Errorf("Error pushing host %d - %s", objectID, err)
		}
		return pushErrs[0]
	}
	return nil
}

// HostUnsetCheck will toggle the check required field for a host when no
// epp data is available
func HostUnsetCheck(client client.Client, objectID int64) error {
	pushErrs := client.UnsetEPPCheck(lib.HostType, objectID)
	if len(pushErrs) != 0 {
		for _, err := range pushErrs {
			log.Errorf("Error pushing host %d - %s", objectID, err)
		}
		return pushErrs[0]
	}
	return nil
}

// APIUserVerify will take a client, signed data, an approval and the change
// request and attempt to verify the api user object. Iff the domain is verified
// it will return true and if not false will be returned with a list of errors
// that have occured verifying the api user.
func APIUserVerify(client client.Client, signedData []byte, app lib.ApprovalExport, cr *lib.ChangeRequestExport) (pass bool, errs []error) {
	log.Infof("Approval %d: Approval for a API User", app.ID)
	ae := lib.APIUserExportFull{}
	err := json.Unmarshal(signedData, &ae)
	if err != nil {
		errs = append(errs, err)
		log.Errorf("Approval %d: Error unmarshaling the signed approval - %s", app.ID, err)
		return
	}
	ar, errs3 := client.GetAPIUserRevision(cr.ProposedRevisionID)
	if len(errs3) != 0 {
		errs = append(errs3, err)
		log.Errorf("Approval %d: Error getting the API user revision - %s", app.ID, err)
		return
	}
	comparePass, errs2 := ar.CompareExport(ae.PendingRevision)
	if len(errs2) != 0 {
		log.Errorf("Approval %d: The pending revision did not match the saved version", app.ID)
		for _, err := range errs2 {
			log.Errorf("       %s", err)
		}
		errs = append(errs, errs2...)
		return
	}
	if !comparePass {
		log.Errorf("Approval %d: The object comparision failed", app.ID)
	} else {
		pass = true
		log.Infof("Approval %d: The approval matched", app.ID)
	}
	return
}

// ApproverVerify will take a client, signed data, an approval and the change
// request and attempt to verify the approver object. Iff the domain is verified
// it will return true and if not false will be returned with a list of errors
// that have occured verifying the approver.
func ApproverVerify(client client.Client, signedData []byte, app lib.ApprovalExport, cr *lib.ChangeRequestExport) (pass bool, errs []error) {
	log.Infof("Approval %d: Approval for a Approver", app.ID)
	ae := lib.ApproverExportFull{}
	err := json.Unmarshal(signedData, &ae)
	if err != nil {
		errs = append(errs, err)
		log.Errorf("Approval %d: Error unmarshaling the signed approval - %s", app.ID, err)
		return
	}
	ar, errs3 := client.GetApproverRevision(cr.ProposedRevisionID)
	if len(errs3) != 0 {
		errs = append(errs3, err)
		log.Errorf("Approval %d: Error getting the Approver revision - %s", app.ID, err)
		return
	}
	comparePass, errs2 := ar.CompareExport(ae.PendingRevision)
	if len(errs2) != 0 {
		log.Errorf("Approval %d: The pending revision did not match the saved version", app.ID)
		for _, err := range errs2 {
			log.Errorf("       %s", err)
		}
		errs = append(errs, errs2...)
		return
	}
	if !comparePass {
		log.Errorf("Approval %d: The object comparision failed", app.ID)
	} else {
		pass = true
		log.Infof("Approval %d: The approval matched", app.ID)
	}
	return
}

// ApproverSetVerify will take a client, signed data, an approval and the change
// request and attempt to verify the approver set object. Iff the domain is
// verified it will return true and if not false will be returned with a list of
// errors that have occured verifying the approver set.
func ApproverSetVerify(client client.Client, signedData []byte, app lib.ApprovalExport, cr *lib.ChangeRequestExport) (pass bool, errs []error) {
	log.Infof("Approval %d: Approval for a Approver Set", app.ID)
	ae := lib.ApproverSetExportFull{}
	err := json.Unmarshal(signedData, &ae)
	if err != nil {
		errs = append(errs, err)
		log.Errorf("Approval %d: Error unmarshaling the signed approval - %s", app.ID, err)
		return
	}
	ar, errs3 := client.GetApproverSetRevision(cr.ProposedRevisionID)
	if len(errs3) != 0 {
		errs = append(errs3, err)
		log.Errorf("Approval %d: Error getting the Approver Set revision - %s", app.ID, err)
		return
	}
	comparePass, errs2 := ar.CompareExport(ae.PendingRevision)
	if len(errs2) != 0 {
		log.Errorf("Approval %d: The pending revision did not match the saved version", app.ID)
		for _, err := range errs2 {
			log.Errorf("       %s", err)
		}
		errs = append(errs, errs2...)
		return
	}
	if !comparePass {
		log.Errorf("Approval %d: The object comparision failed", app.ID)
	} else {
		pass = true
		log.Infof("Approval %d: The approval matched", app.ID)
	}
	return
}

// GetAllWork take an registrar client and attemp to query all registry
// objects that may require work to be done with the registry. Lists of domain,
// host ane contact ids will be returned along with a list of errors if any
// occured during the collection process
func GetAllWork(client client.Client) (domains, hosts, contacts []int64, errs []error) {
	domains, _, errs = client.GetWork(lib.DomainType)
	if len(errs) != 0 {
		return
	}
	hosts, _, errs = client.GetWork(lib.HostType)
	if len(errs) != 0 {
		return
	}
	contacts, _, errs = client.GetWork(lib.ContactType)
	return
}

// prepareLogging sets up logging so the application can have
// configurable logging
func prepareLogging(level logging.Level) {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLevel := logging.AddModuleLevel(backendFormatter)
	backendLevel.SetLevel(level, "")
	logging.SetBackend(backendLevel)
}
