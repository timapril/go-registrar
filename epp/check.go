package epp

import (
	"encoding/xml"
	"strings"
)

// Check is used to construct and receive <check> messages.
type Check struct {
	XMLName xml.Name `xml:"check" json:"-"`

	DomainChecks  []DomainCheck  `xml:"domain:check" json:"domain.check"`
	HostChecks    []HostCheck    `xml:"host:check" json:"host.check"`
	ContactChecks []ContactCheck `xml:"contact:check" json:"contact.check"`

	GenericChecks []GenericCheck `xml:"check" json:"check"`
}

// GetEPPCheck Returns an uninitialized EPP Check object.
func GetEPPCheck(TransactionID string) Epp {
	epp := GetEPPCommand(TransactionID)
	epp.CommandObject.CheckObject = &Check{}

	return epp
}

// GenericCheck is used to receive a Check mesage from a client.
type GenericCheck struct {
	XMLNSDomain         string   `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSHost           string   `xml:"host,attr" json:"xmlns.host"`
	XMLNSContact        string   `xml:"contact,attr" json:"xmlns.contact"`
	XMLNsSchemaLocation string   `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	IDs                 []string `xml:"id" json:"id"`
	Names               []string `xml:"name" json:"name"`
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (c Check) TypedMessage() Check {
	out := Check{}

	for _, gen := range c.GenericChecks {
		if gen.XMLNSContact != "" {
			cc := ContactCheck{}
			cc.XMLNSContact = ContactXMLNS
			cc.XMLNSxsi = W3XMLNSxsi
			cc.XMLxsiSchemaLocation = ContactSchema
			cc.ContactIDs = append(cc.ContactIDs, gen.IDs...)
			out.ContactChecks = append(out.ContactChecks, cc)
		}

		if gen.XMLNSDomain != "" {
			dc := DomainCheck{}
			dc.XMLNSDomain = DomainXMLNS
			dc.XMLNSxsi = W3XMLNSxsi
			dc.XMLxsiSchemaLocation = DomainSchema
			dc.DomainNames = append(dc.DomainNames, gen.Names...)
			out.DomainChecks = append(out.DomainChecks, dc)
		}

		if gen.XMLNSHost != "" {
			hc := HostCheck{}
			hc.XMLNSHost = HostXMLNS
			hc.XMLNSxsi = W3XMLNSxsi
			hc.XMLxsiSchemaLocation = HostSchema
			hc.HostNames = append(hc.HostNames, gen.Names...)
			out.HostChecks = append(out.HostChecks, hc)
		}
	}

	return out
}

// DomainCheck is used to construct and receive <domain:check> messages.
type DomainCheck struct {
	XMLName              xml.Name `xml:"domain:check" json:"-"`
	XMLNSDomain          string   `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"  json:"xmlns.schemaLocation"`
	DomainNames          []string `xml:"domain:name" json:"domain.name"`
}

// GetEPPDomainCheck Returns an uninitialized EPP Domain Check object.
func GetEPPDomainCheck(DomainName string, TransactionID string) Epp {
	domainName := strings.ToUpper(DomainName)
	epp := GetEPPCheck(TransactionID)
	domainCheck := DomainCheck{}

	domainCheck.XMLNSDomain = DomainXMLNS
	domainCheck.XMLNSxsi = W3XMLNSxsi
	domainCheck.XMLxsiSchemaLocation = DomainSchema
	domainCheck.DomainNames = append(domainCheck.DomainNames, domainName)

	epp.CommandObject.CheckObject.DomainChecks = append(epp.CommandObject.CheckObject.DomainChecks, domainCheck)

	if strings.HasSuffix(domainName, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(domainName, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	return epp
}

// HostCheck is used to construct and receive <host:check> messages.
type HostCheck struct {
	XMLName              xml.Name `xml:"host:check" json:"host.check"`
	XMLNSHost            string   `xml:"xmlns:host,attr" json:"xmlns.host"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"  json:"xmlns.schemaLocation"`
	HostNames            []string `xml:"host:name" json:"host.name"`
}

// GetEPPHostCheck is used to generate a host check message.
func GetEPPHostCheck(HostName string, TransactionID string) Epp {
	hostname := strings.ToUpper(HostName)
	epp := GetEPPCheck(TransactionID)
	hostcheck := HostCheck{}

	hostcheck.XMLNSHost = HostXMLNS
	hostcheck.XMLNSxsi = W3XMLNSxsi
	hostcheck.XMLxsiSchemaLocation = HostSchema

	hostcheck.HostNames = append(hostcheck.HostNames, hostname)
	epp.CommandObject.CheckObject.HostChecks = append(epp.CommandObject.CheckObject.HostChecks, hostcheck)

	if strings.HasSuffix(hostname, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(hostname, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	return epp
}

// ContactCheck is used to construct and receive <contact:check> messages.
type ContactCheck struct {
	XMLName              xml.Name `xml:"contact:check" json:"-"`
	XMLNSContact         string   `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"  json:"xmlns.schemaLocation"`
	ContactIDs           []string `xml:"contact:id" json:"contact.id"`
}

// GetEPPContactCheck is used to generate a contact check message.
func GetEPPContactCheck(ContactID string, TransactionID string) Epp {
	epp := GetEPPCheck(TransactionID)
	contactcheck := ContactCheck{}
	contactcheck.XMLNSContact = ContactXMLNS
	contactcheck.XMLNSxsi = W3XMLNSxsi
	contactcheck.XMLxsiSchemaLocation = ContactSchema

	contactcheck.ContactIDs = append(contactcheck.ContactIDs, ContactID)

	epp.CommandObject.CheckObject.ContactChecks = append(epp.CommandObject.CheckObject.ContactChecks, contactcheck)

	return epp
}

// CheckMessageType is used to indicate that a message is a check
// message.
const CheckMessageType = "check"

const (
	// CommandCheckType represents a check command.
	CommandCheckType string = "epp.command.check"

	// CommandCheckContactType represents a contact check command.
	CommandCheckContactType string = "epp.command.check.contact"

	// CommandCheckDomainType represents a domain check command.
	CommandCheckDomainType string = "epp.command.check.domain"

	// CommandCheckHostType represents a host check command.
	CommandCheckHostType string = "epp.command.check.host"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (c *Check) MessageType() string {
	numCont := len(c.ContactChecks)
	numHost := len(c.HostChecks)
	numDom := len(c.DomainChecks)

	if numCont != 0 && numHost == 0 && numDom == 0 {
		return CommandCheckContactType
	}

	if numCont == 0 && numHost != 0 && numDom == 0 {
		return CommandCheckHostType
	}

	if numCont == 0 && numHost == 0 && numDom != 0 {
		return CommandCheckDomainType
	}

	return CommandCheckType
}
