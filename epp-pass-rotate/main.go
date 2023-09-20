package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	// "gopkg.in/gcfg.v1"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/client"
	"github.com/timapril/go-registrar/epp"
	eppclient "github.com/timapril/go-registrar/epp/client"
	"github.com/timapril/go-registrar/keychain"
)

var (
	verbose     = flag.Bool("v", false, "Verbose logging")
	veryverbose = flag.Bool("vv", false, "Very Verbose logging")

	runID = flag.Int("id", 0, "The run ID to use")

	eppServerString = flag.String("epp", "localhost:1700", "the host to connect to")
	eppUser         = flag.String("user", "registrar", "the username to connect with")
)

var RecvChannel chan epp.Epp

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

var log = logging.MustGetLogger("registrar")
var format = logging.MustStringFormatter(
	"%{color}%{level:.4s} %{time:15:04:05.000000} %{shortfile} %{callpath} %{longfunc} %{id:03x}%{color:reset} %{message}",
)

// prepareLogging sets up logging so the application can have
// configurable logging
func prepareLogging(level logging.Level) {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLevel := logging.AddModuleLevel(backendFormatter)
	backendLevel.SetLevel(level, "")
	logging.SetBackend(backendLevel)
}

func main() {
	RecvChannel = make(chan epp.Epp, 100)

	flag.Parse()

	if *runID == 0 {
		fmt.Println("No -id set")
		return
	}

	conf := Config{}

	ll := logging.ERROR
	if *verbose {
		ll = logging.INFO
	}
	if *veryverbose {
		ll = logging.DEBUG
	}

	prepareLogging(ll)

	var passphrase string
	var newPassphrase string

	lineNum := 0
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch lineNum {
		case 0:
			passphrase = line
		case 1:
			newPassphrase = line
		}
		lineNum++
	}

	if lineNum < 2 {
		fmt.Println("expected 2 lines in stdin, first is the current passphrase, second is the new passphrase")
		return
	}

	fmt.Println(passphrase)
	fmt.Println(newPassphrase)

	// cli, cliErr := conf.GetRegistrarClient()
	// if cliErr != nil {
	// 	log.Error(cliErr)
	// 	return
	// }

	if len(passphrase) == 0 {
		log.Error("No passphrase found")
		return
	}

	// runID, errs := cli.RequestNewEPPRunID()
	// if len(errs) != 0 {
	// 	for _, err := range errs {
	// 		log.Error(err)
	// 	}
	// 	return
	// }

	conf.VerisignEPP.Password = passphrase
	conf.VerisignEPP.TransactionPrefix = fmt.Sprintf("%s-%d-%s-", conf.VerisignEPP.TransactionPrefix, runID, time.Now().Format("020106"))
	// rr.TransactionIDPrefix = conf.VerisignEPP.TransactionPrefix

	// epp.GetEPPLoginPasswordChange()

	conn, connectionErr := net.Dial("tcp", *eppServerString)
	if connectionErr != nil {
		log.Critical(fmt.Sprintf("An error occured opening the connection: %s", connectionErr.Error()))
		return
	}

	// connWriter := bufio.NewWriter(conn)
	go EPPListener(conn)

	msg := <-RecvChannel

	fmt.Println(*msg.GreetingObject)

	loginMessage := epp.GetEPPLoginPasswordChange(*eppUser, passphrase, newPassphrase, "REG-9999-01", (*msg.GreetingObject).SvcMenu)
	fmt.Println(loginMessage)
	messageBytes, err := loginMessage.EncodeEPP()
	if err != nil {
		log.Error(err)
		return
	}
	fmt.Println(string(messageBytes))
	_, err = conn.Write(messageBytes)
	if err != nil {
		log.Error(err)
		return
	}

	msg = <-RecvChannel
	fmt.Println(msg)

	logout := epp.GetEPPLogout("REG-9999-02")
	messageBytes, err = logout.EncodeEPP()
	if err != nil {
		log.Error(err)
		return
	}
	fmt.Println(string(messageBytes))
	_, err = conn.Write(messageBytes)
	if err != nil {
		log.Error(err)
		return
	}

	msg = <-RecvChannel
	fmt.Println(msg)

}

// WireSplit is used by the bufio.Scanner to split epp messages apart
// before they are unmarshalled
func WireSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	avilDataLen := uint32(len(data))
	if avilDataLen >= 4 {
		var msgSize uint32
		lenBuf := bytes.NewBuffer(data[:4])

		err = binary.Read(lenBuf, binary.BigEndian, &msgSize)
		if avilDataLen >= msgSize && err == nil {
			advance = int(msgSize)
			token = data[4:msgSize]
			err = nil
			return
		}
		return 0, nil, nil
	}
	return 0, nil, nil
}

// EPPListener starts to listen on the connection from the server and
// processes the messages that it gets back
func EPPListener(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Split(WireSplit)

	log.Info("Starting to listen for incoming messages")

	for scanner.Scan() {
		text := scanner.Text()
		log.Debug(text)
		outObj, unmarshallErr := epp.UnmarshalMessage([]byte(text))
		if unmarshallErr != nil {
			log.Error(unmarshallErr.Error())
			return
		}
		typed := outObj.TypedMessage()

		RecvChannel <- typed
		log.Debug("Listen: Message sent to the channel")
	}
}
