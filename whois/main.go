package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/gcfg.v1"

	"github.com/timapril/go-registrar/whois/objects"
)

// Config is used to load the configuration for the WHOIS server.
type Config struct {
	Templates struct {
		Path string
	}
	Server struct {
		Listen       string
		ListenReload string
		DataFile     string
	}
}

func main() {
	conf := Config{}

	err := gcfg.ReadFileInto(&conf, "/etc/whois")
	if err != nil {
		log.Fatal(err)
	}

	templates := template.Must(template.New("").Funcs(template.FuncMap{
		"dict": dictify,
	}).ParseGlob(conf.Templates.Path))

	connChan := make(chan net.Conn, 1000)
	updateChan := make(chan net.Conn)

	go listenReload(conf, updateChan)
	go handleServer(conf, templates, connChan, updateChan)

	whoisPort, err := net.Listen("tcp", conf.Server.Listen)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := whoisPort.Accept()
		if err != nil {
			log.Print("SRVERR ", err)
		} else {
			connChan <- conn
		}
	}
}

// listenReload listens on a socket to accpet connections from
// the defined ip/port to reload the data configuration on the
// server.
func listenReload(conf Config, updateChan chan net.Conn) {
	reloadPort, err := net.Listen("tcp", conf.Server.ListenReload)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := reloadPort.Accept()
		if err != nil {
			log.Print("RELOADERR ", err)
		} else {
			updateChan <- conn
		}
	}
}

// handleServer runs the main loop waiting for update requests and
// user queries and sends requests to be processed with a current
// version of the whois data.
func handleServer(conf Config, templates *template.Template, connChan chan net.Conn, updateChan chan net.Conn) {
	whoisData, loadErr := loadData(conf)
	if loadErr != nil {
		log.Fatal(loadErr)
	}
	log.Print("NOTE ", "config loaded")

	for {
		select {
		case updateConn := <-updateChan:
			newData, loadErr := loadData(conf)
			if loadErr != nil {
				msg := fmt.Sprintf("could not parse new data file %s", loadErr)
				log.Print("ERR ", msg)
				_, err := updateConn.Write([]byte(msg))
				if err != nil {
					log.Fatal(err)
				}
			} else {
				whoisData = newData
				msg := "config loaded"
				log.Print("NOTE ", msg)
				_, err := updateConn.Write([]byte(msg))
				if err != nil {
					log.Fatal(err)
				}
			}
			updateConn.Close()
		case conn := <-connChan:
			go handleConn(conn, templates, whoisData)
		}
	}
}

// loadData will use the configuration file to locate the data file
// and then the data will be parsed into the whoisData object.
func loadData(conf Config) (whoisData objects.WHOIS, err error) {
	data, err := os.ReadFile(conf.Server.DataFile)
	if err != nil {
		return whoisData, err
	}

	err = json.Unmarshal(data, &whoisData)
	if err != nil {
		return whoisData, err
	}

	whoisData.Index()

	return whoisData, nil
}

// errorResponse contains the required elements to form a error response.
type errorResponse struct {
	SearchString string
	LastUpdate   string
}

// writeErrorResponse is used to compile an error response for a unknown
// query.
func writeErrorResponse(writer io.Writer, templates *template.Template, search string, updatetime string) {
	er := errorResponse{
		SearchString: search,
		LastUpdate:   updatetime,
	}
	executeTemplate(writer, templates, "error", er)
}

// okResponse contains the required elementes to form a valid response.
type okResponse struct {
	SearchString      string
	LastUpdate        string
	DomainName        string
	DomainObject      objects.Domain
	RegistrantContact objects.Contact
	AdminContact      objects.Contact
	TechContact       objects.Contact
	Hostnames         []string
}

// writeAcceptedPage is used to compile a valid response for a WHOIS
// query.
func writeAcceptedPage(writer io.Writer, templates *template.Template, search string, updatetime string, dom objects.Domain, whoisData objects.WHOIS) {
	or := okResponse{
		SearchString:      search,
		LastUpdate:        updatetime,
		DomainName:        strings.ToUpper(dom.DomainName),
		DomainObject:      dom,
		RegistrantContact: whoisData.GetContact(int(dom.RegistrantContactID)),
		AdminContact:      whoisData.GetContact(int(dom.AdminContactID)),
		TechContact:       whoisData.GetContact(int(dom.TechContactID)),
	}
	for _, hid := range dom.HostIDs {
		hos := whoisData.GetHost(int(hid))
		if hos.HostName != "" {
			or.Hostnames = append(or.Hostnames, hos.HostName)
		}
	}
	executeTemplate(writer, templates, "response", or)
}

// executeTemplate attempts to execute a template with the provided
// object and will log an error if its not able to write the template.
// If the template fails to render nothing will be written to the
// writer.
func executeTemplate(writer io.Writer, templates *template.Template, templateName string, data interface{}) {
	writeErr := templates.ExecuteTemplate(writer, templateName, data)
	if writeErr != nil {
		log.Print("ERR ", writeErr)
	}
}

// handleConn processes a WHOIS request for a single connection.
func handleConn(conn net.Conn, templates *template.Template, whoisData objects.WHOIS) {
	defer conn.Close()

	err := conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	line, _, err := reader.ReadLine()
	if err != nil {
		log.Fatal(err)
	}

	searchString := strings.ToUpper(strings.TrimSpace(string(line)))

	dom, err := whoisData.Find(searchString)
	if err != nil {
		writeErrorResponse(conn, templates, searchString, whoisData.LastUpdate.Format("Mon, 2 Jan 2006 15:04:05 GMT"))
		log.Printf("ERR %s %s", conn.RemoteAddr().String(), searchString)
	}
	writeAcceptedPage(conn, templates, searchString, whoisData.LastUpdate.Format("Mon, 2 Jan 2006 15:04:05 GMT"), dom, whoisData)
	log.Printf("OK %s %s", conn.RemoteAddr().String(), searchString)
}

// dictify is a helper function for the template engine to allow more
// than one object to be passed between templates.
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
