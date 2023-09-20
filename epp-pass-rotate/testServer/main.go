package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/epp"
)

var log = logging.MustGetLogger("registrar")
var format = logging.MustStringFormatter(
	"%{color}%{level:.4s} %{time:15:04:05.000000} %{shortfile} %{callpath} %{longfunc} %{id:03x}%{color:reset} %{message}",
)

var (
	verbose     = flag.Bool("v", false, "Verbose logging")
	veryverbose = flag.Bool("vv", false, "Very Verbose logging")
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
	flag.Parse()

	ll := logging.ERROR
	if *verbose {
		ll = logging.INFO
	}
	if *veryverbose {
		ll = logging.DEBUG
	}

	prepareLogging(ll)

	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide a port number!")
		return
	}

	PORT := ":" + arguments[1]
	l, err := net.Listen("tcp4", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {

	scanner := bufio.NewScanner(c)
	scanner.Split(WireSplit)

	msg1 := `<?xml version="1.0" encoding="UTF-8" standalone="no"?><epp xmlns="urn:ietf:params:xml:ns:epp-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd"><greeting><svID>EPP Server Stub</svID><svDate>2015-09-22T23:53:39.854Z</svDate><svcMenu><version>1.0</version><lang>en</lang><objURI>http://www.verisign.com/epp/lowbalance-poll-1.0</objURI><objURI>urn:ietf:params:xml:ns:contact-1.0</objURI><objURI>http://www.verisign.com/epp/balance-1.0</objURI><objURI>urn:ietf:params:xml:ns:domain-1.0</objURI><objURI>http://www.verisign.com/epp/registry-1.0</objURI><objURI>http://www.nic.name/epp/nameWatch-1.0</objURI><objURI>http://www.verisign-grs.com/epp/suggestion-1.1</objURI><objURI>http://www.verisign.com/epp/rgp-poll-1.0</objURI><objURI>http://www.nic.name/epp/defReg-1.0</objURI><objURI>urn:ietf:params:xml:ns:host-1.0</objURI><objURI>http://www.nic.name/epp/emailFwd-1.0</objURI><objURI>http://www.verisign.com/epp/whowas-1.0</objURI><svcExtension><extURI>urn:ietf:params:xml:ns:secDNS-1.1</extURI><extURI>http://www.verisign.com/epp/whoisInf-1.0</extURI><extURI>urn:ietf:params:xml:ns:secDNS-1.0</extURI><extURI>http://www.verisign.com/epp/idnLang-1.0</extURI><extURI>http://www.nic.name/epp/persReg-1.0</extURI><extURI>http://www.verisign.com/epp/jobsContact-1.0</extURI><extURI>urn:ietf:params:xml:ns:coa-1.0</extURI><extURI>http://www.verisign-grs.com/epp/namestoreExt-1.1</extURI><extURI>http://www.verisign.com/epp/sync-1.0</extURI><extURI>http://www.verisign.com/epp/premiumdomain-1.0</extURI><extURI>http://www.verisign.com/epp/relatedDomain-1.0</extURI><extURI>urn:ietf:params:xml:ns:launch-1.0</extURI><extURI>urn:ietf:params:xml:ns:rgp-1.0</extURI></svcExtension></svcMenu><dcp><access><all></all></access><statement><purpose><admin></admin><prov></prov></purpose><recipient><ours></ours><public></public></recipient><retention><stated></stated></retention></statement></dcp></greeting></epp>`

	msgSize := uint32(len(msg1) + 4)

	err := binary.Write(c, binary.BigEndian, &msgSize)
	if err != nil {
		log.Error(err)
		return
	}
	fmt.Println(msgSize)

	_, err = c.Write([]byte(msg1))
	if err != nil {
		log.Error(err)
		return
	}
	fmt.Println("Wrote Greeting")

	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for scanner.Scan() {
		text := scanner.Text()
		fmt.Println(text)

		res := epp.GetEPPResponseResult("1", "1", 1000, epp.ResponseCode1000)
		b, _ := res.EncodeEPP()
		_, err = c.Write(b)
		if err != nil {
			log.Error(err)
			return
		}

		// c.Write([]byte(string("foo")))
	}
	c.Close()
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
