package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"strings"
	"time"

	wordwrap "github.com/mitchellh/go-wordwrap"
	"github.com/op/go-logging"

	whois "github.com/timapril/go-registrar/whois-parse"
)

var verbose = flag.Bool("v", false, "Verbose logging")

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

var (
	infile = flag.String("in", "", "the file to get the domain names from")
)

// Registrar holds all of the targets for a given registrar
type Registrar struct {
	Targets map[string][]string
}

// NewRegiatrar will create and return an initializied registrar
func NewRegiatrar() *Registrar {
	r := &Registrar{}
	r.Targets = make(map[string][]string)
	return r
}

func (r *Registrar) addTarget(email, domain string) {
	if domains, ok := r.Targets[email]; ok {
		r.Targets[email] = append(domains, domain)
	} else {
		r.Targets[email] = []string{domain}
	}
}

func getInfo(domain string) (registrar string, email string, err error) {

	fmt.Printf("Get whois info for: %s\n", domain)
	resp, errs := whois.Query(domain)
	if len(errs) != 0 {
		fmt.Println(errs)
		return
	}

	for _, rawline := range strings.Split(resp.FullLookup, "\n") {
		line := strings.TrimSpace(rawline)
		prefix := "Registrar: "
		if strings.HasPrefix(line, prefix) {
			registrar = strings.TrimSpace(line[len(prefix):])
		}
	}

	for _, rawline := range strings.Split(resp.FullLookup, "\n") {
		line := strings.TrimSpace(rawline)
		prefix := "Registrant Email: "
		if strings.HasPrefix(line, prefix) {
			email = strings.TrimSpace(line[len(prefix):])
		}
	}

	if len(strings.TrimSpace(registrar)) == 0 || len(strings.TrimSpace(email)) == 0 {
		fmt.Println(resp.FullLookup)
		return registrar, email, errors.New("could not find required field")
	}
	return

}

func main() {

	flag.Parse()

	ll := logging.ERROR
	if *verbose {
		ll = logging.DEBUG
	}

	prepareLogging(ll)

	tmpl, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		fmt.Println(err)
		return
	}

	if *infile == "" {
		fmt.Println("No file name passed")
	}

	domainData, err := os.ReadFile(*infile)
	if err != nil {
		log.Error(err)
		return
	}

	registrars := make(map[string]*Registrar)

	for _, row := range strings.Split(string(domainData), "\n") {
		domain := strings.TrimSpace(row)
		if len(domain) == 0 {
			continue
		}

		registrar, email, err := getInfo(domain)
		if err != nil {
			log.Error(err)
			time.Sleep(1 * time.Second)
			registrar, email, err = getInfo(domain)
			if err != nil {
				log.Error(err)
				time.Sleep(5 * time.Second)
				registrar, email, err = getInfo(domain)
				if err != nil {
					log.Error(err)
					time.Sleep(10 * time.Second)
					registrar, email, err = getInfo(domain)
					if err != nil {
						log.Error(err)
						return
					}
				}
			}
		}
		if _, ok := registrars[registrar]; !ok {
			registrars[registrar] = NewRegiatrar()
		}
		registrars[registrar].addTarget(email, domain)

	}

	for registrar, data := range registrars {

		for email, domains := range data.Targets {
			ed := emailData{}
			ed.Email = email
			ed.Domains = strings.Join(domains, ", ")
			ed.CurrentRegistrar = registrar
			ed.CurrentDate = time.Now().Format("Monday, January 02, 2006")
			ed.ExpireDate = time.Now().Add(time.Hour * 24 * 5).Format("Monday, January 02, 2006")
			buf := bytes.Buffer{}
			err := tmpl.Execute(&buf, ed)
			if err != nil {
				log.Error(err)
				return
			}
			wrapped := wordwrap.WrapString(buf.String(), 72)
			fmt.Println("")
			fmt.Println(wrapped)
			sendEmail, promptErr := PromptToConfirm("Send e-mail?")
			if promptErr != nil {
				log.Error(promptErr)
				return
			}
			if sendEmail {
				// TODO: make config option
				c, err := smtp.Dial("smtp.example.com:25")
				if err != nil {
					log.Fatal(err)
				}
				defer c.Close()

				err = c.Mail(email)
				if err != nil {
					log.Error(err)
					return
				}
				// TODO: make config option
				err = c.Rcpt("registrar@example.com")
				if err != nil {
					log.Error(err)
					return
				}
				wc, err := c.Data()
				if err != nil {
					fmt.Println(err)
					return
				}
				defer wc.Close()
				buf2 := bytes.NewBufferString(wrapped)
				if _, err = buf2.WriteTo(wc); err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}

}

type emailData struct {
	Email            string
	Domains          string
	CurrentRegistrar string
	CurrentDate      string
	ExpireDate       string
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
		resp := strings.ToLower(strings.TrimSpace(text))

		if resp == "yes" {
			return true, nil
		}
		if resp == "no" {
			return false, nil
		}

		fmt.Printf("Error: Invalid input your input must be \"Yes\" or \"No\"\n")
	}
}

var emailTemplate = `Subject: Domain Transfer Request
From: Registrar <registrar@example.com>
To: {{.Email}}
Cc: registrar@example.com

Attention: {{.Email}}

Re: Transfer of {{.Domains}}

The current registrar of record for this domain name is {{.CurrentRegistrar}}.

Example, LLC. (IANAID: 6500) has received a request from <who> on {{.CurrentDate}} for us to become the new registrar of record.

You have received this message because you are listed as the Registered
Name Holder or Administrative contact for this domain name in the WHOIS
database.

Please read the following important information about transferring your
domain name(s):

* You must agree to enter into a new Registration Agreement with us.
  You can review the full terms and conditions of the Agreement by
  contacting Example, LLC. for more information.
* Once you have entered into the Agreement, the transfer will take
  place within five (5) calendar days unless the current registrar of
  record denies the request.
* Once a transfer takes place, you will not be able to transfer to
  another registrar for 60 days, apart from a transfer back to the
  original registrar, in cases where both registrars so agree or where
  a decision in the dispute resolution process so directs.

If you WISH TO PROCEED with the transfer, you must respond to this message via email (note if you do not respond by {{.ExpireDate}}, {{.Domains}} will not be transferred to us.).

Please email us with the following message:

"I confirm that I have read the Domain Name Transfer - Request for Confirmation Message.

I confirm that I wish to proceed with the transfer of {{.Domains}} from {{.CurrentRegistrar}} to Example, LLC."

If you DO NOT WANT the transfer to proceed, then don't respond to this message.

If you have any questions about this process, please contact registrar@example.com.`
