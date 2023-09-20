package objects

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/timapril/go-registrar/lib"
)

// WHOIS objects are used to hold a list of Domains, Hosts and Contacts
// available for the WHOIS server to serve.
type WHOIS struct {
	Domains  map[string]Domain
	Hosts    map[string]Host
	Contacts map[string]Contact

	domainsLookup map[string]Domain

	DefaultContact Contact

	LastUpdate time.Time
}

// NewWHOIS will create a new WHOIS object and prepare it to be
// populated.
func NewWHOIS() WHOIS {
	w := WHOIS{}
	w.Domains = make(map[string]Domain)
	w.Hosts = make(map[string]Host)
	w.Contacts = make(map[string]Contact)

	return w
}

// AddHost takes a lib.HostExport object and adds it to the WHOIS
// object's list of hosts.
func (w *WHOIS) AddHost(host *lib.HostExport) error {
	hos, err := WHOISHostFromExport(host)
	if err != nil {
		w.Hosts[strconv.Itoa(int(host.ID))] = hos
	}

	return nil
}

// AddContact takes a lib.ContactExport object and adds it to the WHOIS
// object's list of contacts.
func (w *WHOIS) AddContact(contact *lib.ContactExport) error {
	con, err := WHOISContactFromExport(contact)
	if err != nil {
		w.Contacts[strconv.Itoa(int(contact.ID))] = con
	}

	return nil
}

// SetDefaultContact takes a lib.ContactExport and sets it as the
// default contact to use in the event that the expected contact is
// not available.
func (w *WHOIS) SetDefaultContact(contact *lib.ContactExport) error {
	con, err := WHOISContactFromExport(contact)
	if err != nil {
		w.DefaultContact = con
	}

	return nil
}

// AddDomain takes a lib.DomainExport object and adds it to the WHOIS
// object's list of domains.
func (w *WHOIS) AddDomain(domain *lib.DomainExport) error {
	dom, err := WHOISDomainFromExport(domain)
	if err == nil {
		w.Domains[strconv.Itoa(int(domain.ID))] = dom
	}

	return err
}

// ToJSON attempts to turn the object into a JSON version that
// can be used to store the data or transmit it.
func (w *WHOIS) ToJSON() (string, error) {
	data, err := json.MarshalIndent(w, "", "  ")

	return string(data), err
}

// Index takes the WHOIS object and makes the data accessible in a
// way that can be used to search quickly when the server is operating.
func (w *WHOIS) Index() {
	w.domainsLookup = make(map[string]Domain)
	for _, dom := range w.Domains {
		w.domainsLookup[strings.ToUpper(dom.DomainName)] = dom
	}
}

// GetContact will lookup the contact requested by its ID and if the
// object cannot be found the default contact is returned.
func (w *WHOIS) GetContact(contactID int) Contact {
	if con, ok := w.Contacts[strconv.Itoa(contactID)]; ok {
		return con
	}

	return w.DefaultContact
}

// GetHost will attempt to lookup a host based on its ID and if
// no matching hosts are found an empty host is returned.
func (w *WHOIS) GetHost(hostID int) Host {
	if hos, ok := w.Hosts[strconv.Itoa(hostID)]; ok {
		return hos
	}

	return Host{}
}

// Find takes a search string and tries to find a domain that has
// a matching hostname. If no domain is found an error is returned.
func (w *WHOIS) Find(domain string) (Domain, error) {
	if dom, ok := w.domainsLookup[strings.ToUpper(domain)]; ok {
		return dom, nil
	}
	return Domain{}, errors.New("Unable to find domain")
}
