package epp

import (
	"encoding/xml"
	"strings"
)

// Delete is used to hold one of the subordinate delete objects for
// either a Host, Domain or Contact.
type Delete struct {
	XMLName xml.Name `xml:"delete" json:"-"`

	DomainDeleteObj  *DomainDelete  `xml:",omitempty" json:"domain.delete"`
	HostDeleteObj    *HostDelete    `xml:",omitempty" json:"host.delete"`
	ContactDeleteObj *ContactDelete `xml:",omitempty" json:"contact.delete"`

	GenericDeleteObj *GenericDelete `xml:",omitempty" json:"delete"`
}

// GetEPPDelete is used to generate a delete message envelope to later
// be populated with one of the delte objects (Host, Domain or Contact).
func GetEPPDelete(TransactionID string) Epp {
	epp := GetEPPCommand(TransactionID)
	epp.CommandObject.DeleteObject = &Delete{}

	return epp
}

// GenericDelete is used to receive a delete request that have not been
// typed yet and can be typed at a later time.
type GenericDelete struct {
	XMLName              xml.Name `xml:"delete" json:"-"`
	XMLNSDomain          string   `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSHost            string   `xml:"host,attr" json:"xmlns.host"`
	XMLNSContact         string   `xml:"contact,attr" json:"xmlns.contact"`
	XMLNSxsi             string   `xml:"xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"schemaLocation,attr"  json:"xmlns.schemaLocation"`

	Name string `xml:"name" json:"name"`
	ID   string `xml:"id" json:"id"`
}

// DomainDelete is used to generate a delete message for a domain.
type DomainDelete struct {
	XMLName              xml.Name `xml:"domain:delete" json:"-"`
	XMLNSDomain          string   `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"  json:"xmlns.schemaLocation"`

	DomainName string `xml:"domain:name" json:"domain.name"`
}

// GetEPPDomainDelete is used to generate a domain delete message.
func GetEPPDomainDelete(DomainName string, TransactionID string) Epp {
	domainName := strings.ToUpper(DomainName)

	epp := GetEPPDelete(TransactionID)
	domainDelete := &DomainDelete{}
	domainDelete.XMLNSDomain = DomainXMLNS
	domainDelete.XMLNSxsi = W3XMLNSxsi
	domainDelete.XMLxsiSchemaLocation = DomainSchema

	domainDelete.DomainName = domainName

	if strings.HasSuffix(domainName, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(domainName, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	epp.CommandObject.DeleteObject.DomainDeleteObj = domainDelete

	return epp
}

// HostDelete is used to generate a delete message for a host.
type HostDelete struct {
	XMLName              xml.Name `xml:"host:delete" json:"-"`
	XMLNSHost            string   `xml:"xmlns:host,attr" json:"xmlns.host"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"  json:"xmlns.schemaLocation"`

	HostName string `xml:"host:name" json:"host.name"`
}

// GetEPPHostDelete is used to generate a host delete message.
func GetEPPHostDelete(HostName string, TransactionID string) Epp {
	hostName := strings.ToUpper(HostName)

	epp := GetEPPDelete(TransactionID)
	hostDelete := &HostDelete{}
	hostDelete.XMLNSHost = HostXMLNS
	hostDelete.XMLNSxsi = W3XMLNSxsi
	hostDelete.XMLxsiSchemaLocation = HostSchema

	hostDelete.HostName = hostName

	if strings.HasSuffix(hostName, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(hostName, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	epp.CommandObject.DeleteObject.HostDeleteObj = hostDelete

	return epp
}

// ContactDelete is used to generate a delete message for a contact.
type ContactDelete struct {
	XMLName              xml.Name `xml:"contact:delete" json:"-"`
	XMLNSContact         string   `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"  json:"xmlns.schemaLocation"`

	ContactID string `xml:"contact:id" json:"contact.id"`
}

// GetEPPContactDelete is used to generate a contact delete message.
func GetEPPContactDelete(ContactID string, TransactionID string) Epp {
	epp := GetEPPDelete(TransactionID)
	contactDelete := &ContactDelete{}
	contactDelete.XMLNSContact = ContactXMLNS
	contactDelete.XMLNSxsi = W3XMLNSxsi
	contactDelete.XMLxsiSchemaLocation = ContactSchema

	contactDelete.ContactID = ContactID

	epp.CommandObject.DeleteObject.ContactDeleteObj = contactDelete

	return epp
}

const (
	// CommandDeleteType represents a delete command.
	CommandDeleteType string = "epp.command.delete"

	// CommandDeleteContactType represent a contact delete request.
	CommandDeleteContactType string = "epp.command.delete.contact"

	// CommandDeleteDomainType represent a domain delete request.
	CommandDeleteDomainType string = "epp.command.delete.domain"

	// CommandDeleteHostType represent a host delete request.
	CommandDeleteHostType string = "epp.command.delete.host"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (d *Delete) MessageType() string {
	if d.ContactDeleteObj != nil && d.DomainDeleteObj == nil && d.HostDeleteObj == nil {
		return CommandDeleteContactType
	}

	if d.ContactDeleteObj == nil && d.DomainDeleteObj != nil && d.HostDeleteObj == nil {
		return CommandDeleteDomainType
	}

	if d.ContactDeleteObj == nil && d.DomainDeleteObj == nil && d.HostDeleteObj != nil {
		return CommandDeleteHostType
	}

	return CommandDeleteType
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (d Delete) TypedMessage() Delete {
	out := Delete{}

	if d.GenericDeleteObj.XMLNSContact != "" {
		out.ContactDeleteObj = &ContactDelete{}
		out.ContactDeleteObj.XMLNSContact = d.GenericDeleteObj.XMLNSContact
		out.ContactDeleteObj.XMLNSxsi = d.GenericDeleteObj.XMLNSxsi
		out.ContactDeleteObj.XMLxsiSchemaLocation = d.GenericDeleteObj.XMLxsiSchemaLocation
		out.ContactDeleteObj.ContactID = d.GenericDeleteObj.ID
	}

	if d.GenericDeleteObj.XMLNSDomain != "" {
		out.DomainDeleteObj = &DomainDelete{}
		out.DomainDeleteObj.XMLNSDomain = d.GenericDeleteObj.XMLNSDomain
		out.DomainDeleteObj.XMLNSxsi = d.GenericDeleteObj.XMLNSxsi
		out.DomainDeleteObj.XMLxsiSchemaLocation = d.GenericDeleteObj.XMLxsiSchemaLocation
		out.DomainDeleteObj.DomainName = d.GenericDeleteObj.Name
	}

	if d.GenericDeleteObj.XMLNSHost != "" {
		out.HostDeleteObj = &HostDelete{}
		out.HostDeleteObj.XMLNSHost = d.GenericDeleteObj.XMLNSHost
		out.HostDeleteObj.XMLNSxsi = d.GenericDeleteObj.XMLNSxsi
		out.HostDeleteObj.XMLxsiSchemaLocation = d.GenericDeleteObj.XMLxsiSchemaLocation
		out.HostDeleteObj.HostName = d.GenericDeleteObj.Name
	}

	return out
}
