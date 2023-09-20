package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/timapril/go-registrar/client"
	"github.com/timapril/go-registrar/keychain"
	"github.com/timapril/go-registrar/lib"

	logging "github.com/op/go-logging"
	"gopkg.in/gcfg.v1"
)

var (
	getApproval = flag.Int("get_approval", -1, "The approval number to download to approve")
	getDecline  = flag.Int("get_decline", -1, "The approval number to download to decline")
	getSig      = flag.Int("get_sig", -1, "Download the signature of an approval")
	pushSig     = flag.Int("push_sig", -1, "The approval number to push a signature for")
	checkWork   = flag.Bool("check_work", false, "Check Work will review the data in the server to see what work needs to be done")

	getType = flag.String("type", "", "The type of object to download")
	getID   = flag.Int("id", -1, "The ID of the object to download")

	approverID = flag.Int("approver_id", -1, "The ID of the approver to use for a query")
	certFile   = flag.String("cert", "", "A PEM eoncoded certificate file.")
	keyFile    = flag.String("key", "", "A PEM encoded private key file.")
	caFile     = flag.String("CA", "", "A PEM eoncoded CA's certificate file.")
	configPath = flag.String("conf", "~/.reg", "A configuration file to provide default values for the Registrar client application")

	keychainEnabled = flag.Bool("keychain.enabled", false, "If keychain should be used for auth or not")
	keychainName    = flag.String("keychain.name", "", "The name of the keychain entry holding the key passphrase")
	keychainAccount = flag.String("keychain.account", "", "The account name for the keychain entry holding the key passphrase")

	appServer = flag.String("server", "", "The application server to connect to")

	outFilePath = flag.String("out", "", "The filename that the output will be written to")
	inFilePath  = flag.String("in", "", "The file that should be used as data into the program")

	getWhois       = flag.Bool("whois", false, "Should the WHOIS block be output")
	defaultContact = flag.Int64("default_contact", 1, "The ID of the default contact that should be set")

	spoofCert   = flag.String("spoofcert", "", "Set if a spoof cert should be used to connect")
	spoofHeader = flag.String("spoofheader", "client-cert", "The header to spoof for client cert communication")

	verbose = flag.Bool("v", false, "Verbose logging")
)

var log = logging.MustGetLogger("registrar")
var format = logging.MustStringFormatter(
	"%{color}%{level:.4s}  ▶ %{time:15:04:05.000000} %{shortfile} § %{longfunc} %{id:03x}%{color:reset} %{message}",
)

var byteNewline = []byte("\n")

// Config is an object that holds the configuration for the client
// application. gcfg parses the configuarion file into this format of
// object
type Config struct {
	Certs struct {
		CACertPath string
		CertPath   string
		KeyPath    string
	}
	Defaults struct {
		ApproverID int64
		AppServer  string
	}
	Mac struct {
		KeyKeyChainEnabled bool
		KeyKeychainName    string
		KeyKeychainAccount string
	}
}

var conf Config

// https://gist.github.com/michaljemala/d6f4e01c4834bf47a9c4
func main() {
	flag.Parse()

	ll := logging.ERROR
	if *verbose {
		ll = logging.DEBUG
	}

	prepareLogging(ll)

	confPath, pathErr := ExpandPath(*configPath)
	if pathErr != nil {
		log.Fatal(pathErr.Error())
	}
	confErr := gcfg.ReadFileInto(&conf, confPath)
	if confErr != nil {
		log.Fatal(confErr.Error())
	}

	certFileLoc, certFileLocErr := GetCertFile()
	if certFileLocErr != nil {
		log.Fatal(certFileLocErr.Error())
	}
	keyFileLoc, keyFileLocErr := GetKeyFile()
	if keyFileLocErr != nil {
		log.Fatal(keyFileLocErr.Error())
	}
	caFileLoc, caFileLocErr := GetCAFile()
	if caFileLocErr != nil {
		log.Fatal(caFileLocErr.Error())
	}

	appURL, appURLErr := GetAppURL()
	if appURLErr != nil {
		log.Fatal(appURLErr.Error())
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

	kcc := getKeyChainConf()

	cache := client.DiskCacheConfig{}
	cache.Enabled = false
	cache.CacheDirectory = "./cache"

	cli := client.Client{}

	if *spoofCert == "" {
		cli.PrepareSSL(appURL, certFileLoc, keyFileLoc, caFileLoc, kcc, log, cache)
	} else {
		cli.Prepare(appURL, log, cache)
		ss, readErr := ioutil.ReadFile(*spoofCert)
		if readErr != nil {
			fmt.Println(readErr)
			return
		}
		cli.SpoofCertificateForTesting(string(ss), *spoofHeader)
	}

	if *getApproval != -1 || *getDecline != -1 {
		action := ""
		var approvalID int64

		if *getApproval != -1 {
			action = lib.ActionApproved
			approvalID = int64(*getApproval)
		} else if *getDecline != -1 {
			action = lib.ActionDeclined
			approvalID = int64(*getDecline)
		} else {
			log.Fatal("Unable to determine weather to get an approval or a denial")
		}

		log.Infof("Getting approval ID: %d", *getApproval)
		approverID, approverIDErr := GetApproverID()
		if approverIDErr != nil {
			log.Fatal(approverIDErr.Error())
		}
		log.Infof("ApproverID: %d", approverID)
		approvalString, approvalErrs := cli.GetApproval(approvalID, approverID, action)
		displayErrors(approvalErrs)
		_, err := outFile.Write(approvalString)
		if err != nil {
			log.Fatal(err)
		}
		_, err = outFile.Write(byteNewline)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *getSig != -1 {
		log.Infof("Getting signed approval for Approval %d", *getSig)
		sigString, sigErrs := cli.GetSig(int64(*getSig))
		displayErrors(sigErrs)
		_, err := outFile.Write(sigString)
		if err != nil {
			log.Fatal(err)
		}
		_, err = outFile.Write(byteNewline)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *pushSig != -1 {
		if *inFilePath != "" {
			log.Infof("Pushing signed approval for Approval %d", *pushSig)
			data, fileErr := ioutil.ReadFile(*inFilePath)
			if fileErr != nil {
				log.Fatalf("Error encounted when trying to read -in file: %s", fileErr.Error())
			}

			token, tokenErrs := cli.GetToken()
			if displayErrors(tokenErrs) {
				return
			}
			log.Debugf("Got a token: %s", token)

			displayErrors(cli.PushSig(int64(*pushSig), data, token))
		} else {
			log.Fatal("-in required for -push_sig but not set")
		}
	}

	if *checkWork {
		workHosts, _, hostsErrs := cli.GetHostsWork()
		displayErrors(hostsErrs)
		for _, workHost := range workHosts {
			fmt.Println(workHost)
		}
		workContacts, _, contactsErrs := cli.GetContactsWork()
		displayErrors(contactsErrs)
		for _, workContact := range workContacts {
			fmt.Println(workContact)
		}
		workDomains, _, domainsErrs := cli.GetDomainsWork()
		displayErrors(domainsErrs)
		for _, workDomain := range workDomains {
			fmt.Println(workDomain)
		}
	}

	if *getType != "" && *getID > 0 {
		fmt.Printf("Get object of type %s with the ID %d\n", *getType, *getID)
		idInt64 := int64(*getID)
		switch *getType {
		case "domain":
			obj, errs := cli.GetDomain(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "domainrevision":
			obj, errs := cli.GetDomainRevision(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "contact":
			obj, errs := cli.GetContact(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "contactrevision":
			obj, errs := cli.GetContactRevision(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "host":
			obj, errs := cli.GetHost(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "hostrevision":
			obj, errs := cli.GetHostRevision(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "approver":
			obj, errs := cli.GetApprover(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "approverrevision":
			obj, errs := cli.GetApproverRevision(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "approverset":
			obj, errs := cli.GetApproverSet(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "approversetrevision":
			obj, errs := cli.GetApproverSetRevision(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "changerequest":
			obj, errs := cli.GetChangeRequest(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "approval":
			obj, errs := cli.GetApprovalObject(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "apiuser":
			obj, errs := cli.GetAPIUser(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		case "apiuserrevision":
			obj, errs := cli.GetAPIUserRevision(idInt64)
			if len(errs) != 0 {
				for _, err := range errs {
					log.Error(err.Error())
				}
				log.Fatal("Unexpected error trying to retrieve object")
			}
			fmt.Println(obj)
		default:
			log.Fatal(fmt.Sprintf("Unknown object type %s", *getType))
		}
	}

	if *getWhois {
		WHOISObject, errs := cli.GetWHOIS(*defaultContact)
		displayErrors(errs)

		data, err := WHOISObject.ToJSON()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(data)
	}
}

func displayErrors(errs []error) bool {
	retval := false
	for _, err := range errs {
		log.Error(fmt.Sprintf("\t\t%s", err))
		retval = true
	}
	return retval
}

// getKeyChainConf parses the keychain name and account values from the
// command line if they are set and if they are not set the values from
// the config file are loaded.
func getKeyChainConf() (kcc keychain.Conf) {
	if *keychainName != "" {
		kcc.MacKeychainName = *keychainName
	} else {
		kcc.MacKeychainName = conf.Mac.KeyKeychainName
	}

	if *keychainAccount != "" {
		kcc.MacKeychainAccount = *keychainAccount
	} else {
		kcc.MacKeychainAccount = conf.Mac.KeyKeychainAccount
	}

	kcc.MacKeychainEnabled = conf.Mac.KeyKeyChainEnabled || *keychainEnabled

	return
}

// ExpandPath will take a relative file path and expand the file path
// to be an absolute path (like shell expansion)
func ExpandPath(filePath string) (string, error) {
	if len(filePath) > 0 {
		if string(filePath[0:2]) == "~/" {
			usr, err := user.Current()
			if err != nil {
				return "", err
			}
			return path.Join(usr.HomeDir, filePath[2:]), nil
		}
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

// GetApproverID will return the ID of the approver that has been set
// or return an error if non has been set
func GetApproverID() (int64, error) {
	if *approverID != -1 {
		return int64(*approverID), nil
	}
	if conf.Defaults.ApproverID > 0 {
		return conf.Defaults.ApproverID, nil
	}
	return -1, errors.New("No approver ID set")
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
