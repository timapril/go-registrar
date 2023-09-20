package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/gcfg.v1"

	"github.com/op/go-logging"
	"github.com/timapril/go-registrar/client"
	"github.com/timapril/go-registrar/keychain"
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

	Mac keychain.Conf

	Testing struct {
		SpoofCert  string
		CertHeader string
	}
}

var log = logging.MustGetLogger("registrar")
var format = logging.MustStringFormatter(
	"%{color}%{level:.4s} %{time:15:04:05.000000} %{shortfile} %{callpath} %{longfunc} %{id:03x}%{color:reset} %{message}",
)

var (
	verbose     = flag.Bool("v", false, "Verbose logging")
	veryverbose = flag.Bool("vv", false, "Very Verbose logging")

	confpath = flag.String("conf", "./conf", "The path to the configuration file")

	username = flag.String("username", "", "The username for the passphrase")
	infile   = flag.String("in", "", "The path to the encrypted passphrase")

	pull = flag.Bool("test", false, "Download the saved encrypted passphrase to test if the upload worked")
)

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

	cache := client.DiskCacheConfig{}
	cache.Enabled = false
	cache.CacheDirectory = "./cache"

	if c.Testing.SpoofCert != "" {
		log.Debug("Using the testing credentials rather than production")
		cli.Prepare(c.GetConnectionURL(), log, cache)
		spoofCert, readErr := os.ReadFile(c.Testing.SpoofCert)
		if readErr != nil {
			err = readErr
			return
		}
		cli.SpoofCertificateForTesting(string(spoofCert), string(c.Testing.CertHeader))
	} else {
		cli.PrepareSSL(c.GetConnectionURL(), c.Certs.CertPath, c.Certs.KeyPath, c.Certs.CACertPath, c.Mac, log, cache)
	}

	return cli, nil
}

func main() {
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

	rawData, readErr := os.ReadFile(*infile)
	if readErr != nil {
		log.Error(readErr)
		return
	}

	if *username == "" {
		log.Error("The username may not be empty")
		return
	}

	confErr := gcfg.ReadFileInto(&conf, *confpath)
	if confErr != nil {
		log.Error(confErr)
		return
	}

	cli, cliErr := conf.GetRegistrarClient()
	if cliErr != nil {
		log.Error(cliErr)
		return
	}

	if !*pull {
		log.Debugf("Set the encrypted passphrase for %d", *username)
		setErr := cli.SetEncryptedPassphrase(*username, string(rawData))
		if setErr != nil {
			log.Error(setErr)
		}
	} else {
		log.Debugf("Get the encrypted passphrase for %d", *username)
		passphrase, getErr := cli.GetEncryptedPassphrase(*username)
		if getErr != nil {
			log.Error(getErr)
			return
		}
		if strings.TrimSpace(passphrase) != strings.TrimSpace(string(rawData)) {
			log.Error("Passphrases do not match")
		} else {
			log.Info("Passphrases match")
		}
	}
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
