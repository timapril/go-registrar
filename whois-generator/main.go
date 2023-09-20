package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/timapril/go-registrar/client"
	"github.com/timapril/go-registrar/epp"
	"github.com/timapril/go-registrar/keychain"
	"github.com/timapril/go-registrar/lib"

	logging "github.com/op/go-logging"
	"gopkg.in/gcfg.v1"
)

var (
	certFile   = flag.String("cert", "", "A PEM eoncoded certificate file.")
	keyFile    = flag.String("key", "", "A PEM encoded private key file.")
	caFile     = flag.String("CA", "", "A PEM eoncoded CA's certificate file.")
	configPath = flag.String("conf", "~/.registrar", "A configuration file to provide default values for the Registrar client application")

	// keychainEnabled = flag.Bool("keychain.enabled", false, "If keychain should be used for auth or not")
	// keychainName    = flag.String("keychain.name", "", "The name of the keychain entry holding the key passphrase")
	// keychainAccount = flag.String("keychain.account", "", "The account name for the keychain entry holding the key passphrase")

	appServer = flag.String("server", "", "The application server to connect to")

	outFilePath = flag.String("out", "", "The filename that the output will be written to")

	// getWhois       = flag.Bool("whois", false, "Should the WHOIS block be output")
	defaultContact = flag.Int64("default_contact", 1, "The ID of the default contact that should be set")

	verbose = flag.Bool("v", false, "Verbose logging")
)

// Config is an object that holds the configuration for the client
// application. gcfg parses the configuarion file into this format of
// object
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
	Defaults struct {
		ApproverID int64
		AppServer  string
	}

	Mac keychain.Conf

	TrustedKeys struct {
		Key []string
	}
	RegistrarInfo struct {
		WHOISServer       string
		URL               string
		Name              string
		IANAID            int64
		AbuseContactEmail string
		AbuseContactPhone string
		NoticeTextPath    string
	}
	Testing struct {
		SpoofCert  string
		CertHeader string
	}

	CacheConfig client.DiskCacheConfig
}

var conf Config

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
	} else {
		cli.PrepareSSL(c.GetConnectionURL(), c.Certs.CertPath, c.Certs.KeyPath, c.Certs.CACertPath, c.Mac, log, c.CacheConfig)
	}

	return cli, nil
}

var log = logging.MustGetLogger("registrar")
var format = logging.MustStringFormatter(
	"%{color}%{level:.4s} %{time:15:04:05.000000} %{shortfile} %{callpath} %{longfunc} %{id:03x}%{color:reset} %{message}",
)

func main() {
	flag.Parse()

	ll := logging.ERROR
	if *verbose {
		ll = logging.DEBUG
	}

	prepareLogging(ll)

	textTemplates, textTmplErr := template.New("templates").Funcs(template.FuncMap{
		"dict": dictify,
	}).Parse(whoisTemplate)
	if textTmplErr != nil {
		log.Error(textTmplErr)
		return
	}

	var outFile *os.File
	if *outFilePath != "" {
		var createErr error
		outFile, createErr = os.Create(*outFilePath)
		if createErr != nil {
			fmt.Println(createErr)
			return
		}
		defer outFile.Close()
	} else {
		outFile = os.Stdout
	}

	confErr := gcfg.ReadFileInto(&conf, *configPath)
	if confErr != nil {
		log.Fatal(confErr)
	}

	noticeText, noticeErr := os.ReadFile(conf.RegistrarInfo.NoticeTextPath)
	if noticeErr != nil {
		log.Fatal(noticeErr)
	}

	thisRegistrar := registrarInformation{}
	thisRegistrar.WHOISServer = conf.RegistrarInfo.WHOISServer
	thisRegistrar.URL = conf.RegistrarInfo.URL
	thisRegistrar.Name = conf.RegistrarInfo.Name
	thisRegistrar.IANAID = conf.RegistrarInfo.IANAID
	thisRegistrar.AbuseContactEmail = conf.RegistrarInfo.AbuseContactEmail
	thisRegistrar.AbuseContactPhone = conf.RegistrarInfo.AbuseContactPhone
	thisRegistrar.NoticeText = strings.TrimSpace(string(noticeText))

	startTime := time.Now().UTC()
	lastUpdateTime := startTime.Format("2006-01-02T15:04:05Z")

	cli, err := conf.GetRegistrarClient()
	if err != nil {
		log.Error(err)
		return
	}
	errs := cli.PrepareObjectDirectory()
	if len(errs) != 0 {
		for _, err := range errs {
			log.Error(err)
		}
		return
	}

	ver, contactErrs, defContact := cli.GetVerifiedContact(*defaultContact, startTime.Unix())
	if len(contactErrs) != 0 {
		for _, err := range contactErrs {
			log.Error(err)
		}
		return
	}
	if !ver {
		log.Error("Unable to verify default contact, exiting.")
		return
	}

	domOutput := []string{}

	for _, dom := range cli.ObjectDir.DomainIDs {
		log.Infof("Domain ID: %d", dom)

		whois := whoisDisplayObject{}

		errs := whois.BuildDomain(&cli, dom, defContact, thisRegistrar, startTime.Unix())
		if errs != nil {
			if len(errs) != 0 {
				log.Errorf("Domain object empty with ID %d - %s", dom, errs[0].Error())
			} else {
				log.Errorf("Domain object empty with ID %d", dom)
			}
		} else {
			var buf bytes.Buffer

			whois.Timestamp = lastUpdateTime

			whois.DomainStatuses = getDomainStatuses(whois)
			err := textTemplates.ExecuteTemplate(&buf, "domain", whois)
			if err != nil {
				log.Error(err)
			}

			domOutput = append(domOutput, strings.TrimSpace(buf.String()))
		}
	}

	_, err = outFile.WriteString(strings.Join(domOutput, "\n\n"))
	if err != nil {
		log.Error(err)
	}

}

func getIcannEPPStatusCodeLink(status string) string {
	return fmt.Sprintf("%s https://icann.org/epp#%s", status, status)
}

func getDomainStatuses(w whoisDisplayObject) []string {
	var ret []string

	if w.Domain.ClientDeleteProhibitedStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("clientDeleteProhibited"))
	}
	if w.Domain.ClientHoldStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("clientHold"))
	}
	if w.Domain.ClientRenewProhibitedStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("clientRenewProhibited"))
	}
	if w.Domain.ClientTransferProhibitedStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("clientTransferProhibited"))
	}
	if w.Domain.ClientUpdateProhibitedStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("clientUpdateProhibited"))
	}

	if w.Domain.ServerDeleteProhibitedStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("serverDeleteProhibited"))
	}
	if w.Domain.ServerHoldStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("serverHold"))
	}
	if w.Domain.ServerRenewProhibitedStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("serverRenewProhibited"))
	}
	if w.Domain.ServerTransferProhibitedStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("serverTransferProhibited"))
	}
	if w.Domain.ServerUpdateProhibitedStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("serverUpdateProhibited"))
	}

	if w.Domain.PendingUpdateStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("pendingUpdate"))
	}
	if w.Domain.PendingTransferStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("pendingTransfer"))
	}
	if w.Domain.PendingRenewStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("pendingRenew"))
	}
	if w.Domain.PendingDeleteStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("pendingDelete"))
	}
	if w.Domain.PendingCreateStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("pendingCreate"))
	}

	if w.Domain.OKStatus {
		ret = append(ret, getIcannEPPStatusCodeLink("ok"))
	}

	return ret

}

type registrarInformation struct {
	WHOISServer       string
	URL               string
	Name              string
	IANAID            int64
	AbuseContactEmail string
	AbuseContactPhone string
	NoticeText        string
}

type whoisDisplayObject struct {
	Domain *lib.DomainExport

	Registrant     *lib.ContactExport
	AdminContact   *lib.ContactExport
	TechContact    *lib.ContactExport
	BillingContact *lib.ContactExport

	Hosts []*lib.HostExport

	RegInfo registrarInformation

	DNSSECSigned   string
	DomainStatuses []string
	Timestamp      string
}

func (w whoisDisplayObject) UpdateDate() string {
	return w.Domain.UpdateDate.Format(epp.EPPTimeFormat)
}

func (w whoisDisplayObject) CreateDate() string {
	return w.Domain.CreateDate.Format(epp.EPPTimeFormat)
}

func (w whoisDisplayObject) ExpireDate() string {
	return w.Domain.ExpireDate.Format(epp.EPPTimeFormat)
}

func (w whoisDisplayObject) DomainLower() string {
	return strings.ToLower(w.Domain.DomainName)
}

func (w whoisDisplayObject) DomainName() string {
	return strings.ToUpper(w.Domain.DomainName)
}

func (w whoisDisplayObject) DoaminRegistryID() string {
	return w.Domain.DomainROID
}

func (w *whoisDisplayObject) BuildDomain(client *client.Client, domainID int64, defaultContact *lib.ContactExport, regInfo registrarInformation, startTime int64) (errs []error) {
	log.Infof("Domain ID: %d", domainID)

	w.RegInfo = regInfo

	ver, verErr, dom := client.GetVerifiedDomain(domainID, startTime)

	if len(verErr) != 0 {
		errs = append(errs, verErr...)
		return
	}
	if dom == nil {
		errs = append(errs, fmt.Errorf("Unable to verify doamin with ID %d", domainID))
		return
	}
	w.Domain = dom

	if !ver {
		errs = append(errs, fmt.Errorf("Unable to find verified domain with ID %d", domainID))
		return
	}

	if len(w.Domain.CurrentRevision.DSDataEntries) != 0 {
		w.DNSSECSigned = "signed"
	} else {
		w.DNSSECSigned = "unsigned"
	}

	registrantContactID := w.Domain.CurrentRevision.DomainRegistrant.ID
	adminContactID := w.Domain.CurrentRevision.DomainAdminContact.ID
	techContactID := w.Domain.CurrentRevision.DomainTechContact.ID
	billingContactID := w.Domain.CurrentRevision.DomainBillingContact.ID

	var tempCon *lib.ContactExport

	ver, verErr, tempCon = client.GetVerifiedContact(registrantContactID, startTime)
	if len(verErr) != 0 {
		errs = append(errs, verErr...)
		return
	}
	if ver {
		w.Registrant = tempCon
	} else {
		errs = append(errs, fmt.Errorf("Unable to find verified Registrant Contact with ID %d, using default", registrantContactID))
		w.Registrant = defaultContact
	}

	ver, verErr, tempCon = client.GetVerifiedContact(adminContactID, startTime)
	if len(verErr) != 0 {
		errs = append(errs, verErr...)
		return
	}
	if ver {
		w.AdminContact = tempCon
	} else {
		errs = append(errs, fmt.Errorf("Unable to find verified Admin Contact with ID %d, using default", adminContactID))
		w.AdminContact = defaultContact
	}

	ver, verErr, tempCon = client.GetVerifiedContact(techContactID, startTime)
	if len(verErr) != 0 {
		errs = append(errs, verErr...)
		return
	}
	if ver {
		w.TechContact = tempCon
	} else {
		errs = append(errs, fmt.Errorf("Unable to find verified Tech Contact with ID %d, using default", techContactID))
		w.TechContact = defaultContact
	}

	ver, verErr, tempCon = client.GetVerifiedContact(billingContactID, startTime)
	if len(verErr) != 0 {
		errs = append(errs, verErr...)
		return
	}
	if ver {
		w.BillingContact = tempCon
	} else {
		errs = append(errs, fmt.Errorf("Unable to find verified Billing Contact with ID %d, using default", billingContactID))
		w.BillingContact = defaultContact
	}

	hosts := w.Domain.CurrentRevision.Hostnames
	for _, host := range hosts {
		ver, verErr, hos := client.GetVerifiedHost(host.ID, startTime)
		if len(verErr) != 0 {
			errs = append(errs, verErr...)
			return
		}
		if ver {
			w.Hosts = append(w.Hosts, hos)
		} else {
			errs = append(errs, fmt.Errorf("Unable to find verified Billing Contact with ID %d", billingContactID))
			return
		}
	}

	return
}

// func displayErrors(errs []error) bool {
// 	retval := false
// 	for _, err := range errs {
// 		log.Errorf("\t\t%s", err)
// 		retval = true
// 	}
// 	return retval
// }

// ExpandPath will take a relative file path and expand the file path
// to be an absolute path (like shell expansion)
func ExpandPath(filePath string) (string, error) {
	if len(filePath) > 0 {
		if string(filePath[0]) == "/" {
			return filePath, nil
		}
		wd, wdErr := os.Getwd()
		return path.Join(wd, filePath), wdErr
	}
	return "", errors.New("No Path Set")
}

// GetCAFile will return the path to the ca cert file if it is set at
// the command line or in the config file (in that order). If the ca
// cert file is not set in either location, an error is returned
func GetCAFile() (string, error) {
	if *caFile != "" {
		return *caFile, nil
	}
	if conf.Certs.CACertPath != "" {
		return ExpandPath(conf.Certs.CACertPath)
	}
	return "", errors.New("No CA File Path set")
}

// GetKeyFile will return the path to the key file if it is set at the
// command line or in the config file (in that order). If the key file
// is not set in either location, an error is returned
func GetKeyFile() (string, error) {
	if *keyFile != "" {
		return *keyFile, nil
	}
	if conf.Certs.KeyPath != "" {
		return ExpandPath(conf.Certs.KeyPath)
	}
	return "", errors.New("No Key File Path set")
}

// GetCertFile will return the path to the cert file if it is set at the
// command line or in the config file (in that order). If the cert file
// is not set in either location, an error is returned
func GetCertFile() (string, error) {
	if *certFile != "" {
		return *certFile, nil
	}
	if conf.Certs.CertPath != "" {
		return ExpandPath(conf.Certs.CertPath)
	}
	return "", errors.New("No Cert File Path set")
}

// GetAppURL will return the URL for the application server if it is set
// at the command line or in the config file (in that order). If the app
// server url is not set in either location, an error will be returned.
func GetAppURL() (string, error) {
	if *appServer != "" {
		return *appServer, nil
	}
	if conf.Defaults.AppServer != "" {
		return conf.Defaults.AppServer, nil
	}
	return "", errors.New("No App server URL was found")
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

func dictify(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

var whoisTemplate = `{{define "domain"}}domain: {{.DomainLower}}
DomainName: {{.DomainName}}
Registry Domain ID: {{.DoaminRegistryID}}
Registrar WHOIS Server: {{.RegInfo.WHOISServer}}
Registrar URL: {{.RegInfo.URL}}
Updated Date: {{.UpdateDate}}
Creation Date: {{.CreateDate}}
Registrar Registration Expiration Date: {{.ExpireDate}}
Sponsoring Registrar: {{.RegInfo.Name}}
Sponsoring Registrar IANA ID: {{.RegInfo.IANAID}}
Registrar Abuse Contact Email: {{.RegInfo.AbuseContactEmail}}
Registrar Abuse Contact Phone: {{.RegInfo.AbuseContactPhone}}
{{range $status := .DomainStatuses}}Domain Status: {{$status}}
{{end}}{{template "contact" dict "Name" "Registrant" "Cont" .Registrant}}
{{template "contact" dict "Name" "Admin" "Cont" .AdminContact}}
{{template "contact" dict "Name" "Tech" "Cont" .AdminContact}}
{{template "contact" dict "Name" "Billing" "Cont" .BillingContact}}
{{range $host := .Hosts}}Name Server: {{$host.HostName}}
{{end}}DNSSEC: {{.DNSSECSigned}}
URL of the ICANN WHOIS Data Problem Reporting System: http://wdprs.internic.net/
>>> Last update of WHOIS database: {{.Timestamp}} <<<
{{.RegInfo.NoticeText}}
{{end}}
{{define "contact"}}{{.Name}} Name: {{.Cont.CurrentRevision.Name}}
{{.Name}} Organization: {{.Cont.CurrentRevision.Org}}
{{.Name}} Street: {{.Cont.CurrentRevision.AddressStreet1}}
{{$a2len := len .Cont.CurrentRevision.AddressStreet2}}{{if gt $a2len 0}}{{.Name}} Street: {{.Cont.CurrentRevision.AddressStreet2}}
{{end}}{{$a3len := len .Cont.CurrentRevision.AddressStreet3}}{{if gt $a3len 0}}{{.Name}} Street: {{.Cont.CurrentRevision.AddressStreet3}}
{{end}}{{.Name}} City: {{.Cont.CurrentRevision.AddressCity}}
{{.Name}} State/Province: {{.Cont.CurrentRevision.AddressState}}
{{.Name}} Postal Code: {{.Cont.CurrentRevision.AddressPostalCode}}
{{.Name}} Country: {{.Cont.CurrentRevision.AddressCountry}}
{{.Name}} Phone: {{.Cont.CurrentRevision.VoicePhoneNumber}}
{{.Name}} Phone Ext: {{.Cont.CurrentRevision.VoicePhoneExtension}}
{{.Name}} Fax: {{.Cont.CurrentRevision.FaxPhoneNumber}}
{{.Name}} Fax Ext: {{.Cont.CurrentRevision.FaxPhoneExtension}}
{{.Name}} Email: {{.Cont.CurrentRevision.EmailAddress}}{{end}}`
