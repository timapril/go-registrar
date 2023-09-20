package epp

import (
	"encoding/xml"
	"strings"
)

// Renew is used to construct a renew message.
type Renew struct {
	XMLName xml.Name `xml:"renew" json:"-"`

	DomainRenewObj  *DomainRenew  `xml:",omitempty" json:"domain"`
	GenericRenewObj *GenericRenew `xml:",omitempty" json:"renew"`
}

// GetEPPRenew is used to generate a renew message envelope to later be
// populated with one of the renew objects (Domain).
func GetEPPRenew(TransactionID string) Epp {
	epp := GetEPPCommand(TransactionID)
	epp.CommandObject.RenewObject = &Renew{}

	return epp
}

// GenericRenew is used to receive a renew object from a client.
type GenericRenew struct {
	XMLName              xml.Name `xml:"renew" json:"-"`
	XMLNSDomain          string   `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSxsi             string   `xml:"xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`

	DomainName     string       `xml:"name" json:"name"`
	CurrentExpDate string       `xml:"curExpDate" json:"currentExpDate"`
	RenewPeriod    DomainPeriod `xml:"period" json:"period"`
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (r Renew) TypedMessage() Renew {
	out := Renew{}

	if r.GenericRenewObj.XMLNSDomain != "" {
		out.DomainRenewObj = &DomainRenew{}
		out.DomainRenewObj.XMLNSDomain = r.GenericRenewObj.XMLNSDomain
		out.DomainRenewObj.XMLNSxsi = r.GenericRenewObj.XMLNSxsi
		out.DomainRenewObj.XMLxsiSchemaLocation = r.GenericRenewObj.XMLxsiSchemaLocation
		out.DomainRenewObj.DomainName = r.GenericRenewObj.DomainName
		out.DomainRenewObj.CurrentExpDate = r.GenericRenewObj.CurrentExpDate
		out.DomainRenewObj.RenewPeriod = r.GenericRenewObj.RenewPeriod
	}

	return out
}

// DomainRenew is used to generate renew message for a domain.
type DomainRenew struct {
	XMLName              xml.Name `xml:"domain:renew" json:"-"`
	XMLNSDomain          string   `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`

	DomainName     string       `xml:"domain:name" json:"domain.name"`
	CurrentExpDate string       `xml:"domain:curExpDate" json:"domain.curExpDate"`
	RenewPeriod    DomainPeriod `xml:"domain:period" json:"domain.period"`
}

const (
	// CommandRenewType represents a renew command.
	CommandRenewType string = "epp.command.renew"

	// CommandRenewDomainType represents a domain renew request.
	CommandRenewDomainType string = "epp.command.renew.domain"
)

// GetEPPDomainRenew returnes a populated EPP Domain renew object.
func GetEPPDomainRenew(DomainName string, CurrentExpDate string, RenewPeriod DomainPeriod, TransactionID string) Epp {
	domainNameUpper := strings.ToUpper(DomainName)

	epp := GetEPPRenew(TransactionID)

	domainRenew := &DomainRenew{}
	domainRenew.XMLNSDomain = DomainXMLNS
	domainRenew.XMLNSxsi = W3XMLNSxsi
	domainRenew.XMLxsiSchemaLocation = DomainSchema

	domainRenew.DomainName = domainNameUpper
	domainRenew.CurrentExpDate = CurrentExpDate
	domainRenew.RenewPeriod = RenewPeriod

	if strings.HasSuffix(domainNameUpper, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(domainNameUpper, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	epp.CommandObject.RenewObject.DomainRenewObj = domainRenew

	return epp
}

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (r Renew) MessageType() string {
	if r.DomainRenewObj != nil {
		return CommandRenewDomainType
	}

	return CommandRenewType
}
