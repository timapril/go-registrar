package epp

import (
	"encoding/xml"
	"strings"
)

// Info is used to construct and receive <info> messages.
type Info struct {
	XMLName        xml.Name     `xml:"info" json:"-"`
	DomainInfoObj  *DomainInfo  `xml:",omitempty" json:"domain"`
	HostInfoObj    *HostInfo    `xml:",omitempty" json:"host"`
	ContactInfoObj *ContactInfo `xml:",omitempty" json:"contact"`

	GenericInfoObj *GenericInfo `xml:",omitempty" json:"info"`
}

// GetEPPInfo is used to generate an info message envelope to later be
// populated with one of the info objects (Host, Domain or Contact).
func GetEPPInfo(TransactionID string) Epp {
	epp := GetEPPCommand(TransactionID)
	epp.CommandObject.InfoObject = &Info{}

	return epp
}

// GenericInfo is used to receive a Info message from a client.
type GenericInfo struct {
	XMLName             xml.Name `xml:"info" json:"-"`
	XMLNSDomain         string   `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSHost           string   `xml:"host,attr" json:"xmlns.host"`
	XMLNSContact        string   `xml:"contact,attr" json:"xmlns.contact"`
	XMLNsXSI            string   `xml:"xsi,attr" json:"xmlns.xsi"`
	XMLNsSchemaLocation string   `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`

	ID              string           `xml:"id" json:"id"`
	Name            GenericName      `xml:"name" json:"name"`
	GenericAuthInfo *GenericAuthInfo `xml:"authInfo" jason:"authInfo"`
}

// GenericName is used to receive a Name block for an info object from
// a client.
type GenericName struct {
	XMLName xml.Name        `xml:"name" json:"-"`
	Hosts   DomainInfoHosts `xml:"hosts,attr,omitempty" json:"hosts"`
	Name    string          `xml:",chardata" json:"name"`
}

// GenericAuthInfo is used to receive a AuthInfo block from a client.
type GenericAuthInfo struct {
	XMLName  xml.Name `xml:"authInfo" json:"authInfo"`
	Password string   `xml:"pw" json:"pw"`
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (i Info) TypedMessage() Info {
	out := Info{}

	if i.GenericInfoObj.XMLNSContact != "" {
		out.ContactInfoObj = &ContactInfo{}
		out.ContactInfoObj.XMLNS = i.GenericInfoObj.XMLNSContact
		out.ContactInfoObj.XMLNSSchemaLocation = i.GenericInfoObj.XMLNsSchemaLocation
		out.ContactInfoObj.XMLNSxsi = i.GenericInfoObj.XMLNsXSI
		out.ContactInfoObj.ID = i.GenericInfoObj.ID

		if i.GenericInfoObj.GenericAuthInfo != nil {
			out.ContactInfoObj.ContactAuthObj = &ContactAuth{}
			out.ContactInfoObj.ContactAuthObj.Password = i.GenericInfoObj.GenericAuthInfo.Password
		}
	}

	if i.GenericInfoObj.XMLNSDomain != "" {
		out.DomainInfoObj = &DomainInfo{}
		out.DomainInfoObj.XMLNS = i.GenericInfoObj.XMLNSDomain
		out.DomainInfoObj.XMLNSSchemaLocation = i.GenericInfoObj.XMLNsSchemaLocation
		out.DomainInfoObj.XMLNSxsi = i.GenericInfoObj.XMLNsXSI

		out.DomainInfoObj.Domain.Name = i.GenericInfoObj.Name.Name
		out.DomainInfoObj.Domain.Hosts = i.GenericInfoObj.Name.Hosts

		if i.GenericInfoObj.GenericAuthInfo != nil {
			out.DomainInfoObj.DomainAuthObj = &DomainAuth{}
			out.DomainInfoObj.DomainAuthObj.Password = i.GenericInfoObj.GenericAuthInfo.Password
		}
	}

	if i.GenericInfoObj.XMLNSHost != "" {
		out.HostInfoObj = &HostInfo{}
		out.HostInfoObj.XMLNS = i.GenericInfoObj.XMLNSHost
		out.HostInfoObj.XMLNSSchemaLocation = i.GenericInfoObj.XMLNsSchemaLocation
		out.HostInfoObj.XMLNSxsi = i.GenericInfoObj.XMLNsXSI

		out.HostInfoObj.Name = i.GenericInfoObj.Name.Name

		if i.GenericInfoObj.GenericAuthInfo != nil {
			out.HostInfoObj.HostAuthObj = &HostAuth{}
			out.HostInfoObj.HostAuthObj.Password = i.GenericInfoObj.GenericAuthInfo.Password
		}
	}

	return out
}

// DomainInfo is used to generate an info message for a domain.
type DomainInfo struct {
	XMLName             xml.Name       `xml:"domain:info" json:"-"`
	XMLNS               string         `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNSxsi            string         `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLNSSchemaLocation string         `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Domain              DomainNameInfo `xml:"domain:name" json:"domain.name"`
	DomainAuthObj       *DomainAuth    `xml:"domain:authInfo,omitempty" json:"domain.authInfo"`
}

// DomainNameInfo is used to generate a NameInfo chunk for a domain
// with an indicator of all or no hosts.
type DomainNameInfo struct {
	XMLName xml.Name        `xml:"domain:name" json:"domain.name"`
	Hosts   DomainInfoHosts `xml:"hosts,attr" json:"hosts"`
	Name    string          `xml:",chardata" json:"name"`
}

// DomainInfoHosts is a type used to check that the provided option is
// a valid DomainInfoHost option.
type DomainInfoHosts string

const (
	// DomainInfoHostsAll is used to indicate that the domain info
	// request should include all host information.
	DomainInfoHostsAll DomainInfoHosts = "all"

	// DomainInfoHostsDelegated is used to indicate that the domain info
	// request should include delegated host information only.
	DomainInfoHostsDelegated DomainInfoHosts = "del"

	// DomainInfoHostsSubordinate is used to indicate that the domain info
	// request should include subordinate host information only.
	DomainInfoHostsSubordinate DomainInfoHosts = "sub"

	// DomainInfoHostsNone is used to indicate that the domain info
	// request should include no host information.
	DomainInfoHostsNone DomainInfoHosts = "none"
)

// GetEPPDomainInfo returns an populated EPP Domain Info object.
func GetEPPDomainInfo(DomainName string, TransactionID string, AuthPassword string, Hosts DomainInfoHosts) Epp {
	epp := GetEPPInfo(TransactionID)
	domainInfo := &DomainInfo{}
	domainInfo.XMLNS = "urn:ietf:params:xml:ns:domain-1.0"
	domainInfo.XMLNSxsi = W3XMLNSxsi
	domainInfo.XMLNSSchemaLocation = "urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd"

	domainNameUpper := strings.ToUpper(DomainName)
	domainInfo.Domain.Hosts = Hosts
	domainInfo.Domain.Name = domainNameUpper

	if AuthPassword != "" {
		domainInfo.DomainAuthObj = new(DomainAuth)
		domainInfo.DomainAuthObj.Password = AuthPassword
	}

	epp.CommandObject.InfoObject.DomainInfoObj = domainInfo

	if strings.HasSuffix(domainNameUpper, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(domainNameUpper, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	return epp
}

// HostInfo is used to generate an info message for a host.
type HostInfo struct {
	XMLName             xml.Name  `xml:"host:info" json:"-"`
	XMLNS               string    `xml:"xmlns:host,attr" json:"xmlns.hosts"`
	XMLNSxsi            string    `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLNSSchemaLocation string    `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string    `xml:"host:name" json:"host.name"`
	HostAuthObj         *HostAuth `xml:"host:authInfo,omitempty" json:"host.authInfo"`
}

// GetEPPHostInfo returns a populated EPP Host Info object.
func GetEPPHostInfo(HostName string, TransactionID string) Epp {
	hostNameUpper := strings.ToUpper(HostName)

	epp := GetEPPInfo(TransactionID)
	hostInfo := &HostInfo{}
	hostInfo.XMLNS = "urn:ietf:params:xml:ns:host-1.0"
	hostInfo.XMLNSxsi = W3XMLNSxsi
	hostInfo.XMLNSSchemaLocation = "urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd"
	hostInfo.Name = hostNameUpper

	epp.CommandObject.InfoObject.HostInfoObj = hostInfo

	if strings.HasSuffix(hostNameUpper, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(hostNameUpper, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	return epp
}

// ContactInfo is used to generate an info message for a contact.
type ContactInfo struct {
	XMLName             xml.Name     `xml:"contact:info" json:"-"`
	XMLNS               string       `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNSxsi            string       `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLNSSchemaLocation string       `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	ID                  string       `xml:"contact:id" json:"contact.id"`
	ContactAuthObj      *ContactAuth `xml:"contact:authInfo,omitempty" json:"contact.authInfo"`
}

// GetEPPContactInfo returns a populated EPP Contact Info object.
func GetEPPContactInfo(ContactID string, TransactionID string, AuthPassword string) Epp {
	epp := GetEPPInfo(TransactionID)
	contactInfo := &ContactInfo{}
	contactInfo.XMLNS = "urn:ietf:params:xml:ns:contact-1.0"
	contactInfo.XMLNSxsi = W3XMLNSxsi
	contactInfo.XMLNSSchemaLocation = "urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd"
	contactInfo.ID = ContactID

	if AuthPassword != "" {
		contactInfo.ContactAuthObj = new(ContactAuth)
		contactInfo.ContactAuthObj.Password = AuthPassword
	}

	epp.CommandObject.InfoObject.ContactInfoObj = contactInfo

	return epp
}

// InfoMessageType is used to indicate that a message is a info
// message.
const InfoMessageType = "info"

const (
	// CommandInfoType represents a info command.
	CommandInfoType string = "epp.command.info"

	// CommandInfoContactType represents a contact info command.
	CommandInfoContactType string = "epp.command.info.contact"

	// CommandInfoDomainType represents a domain info command.
	CommandInfoDomainType string = "epp.command.info.domain"

	// CommandInfoHostType represents a host info command.
	CommandInfoHostType string = "epp.command.info.host"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (i *Info) MessageType() string {
	if i.ContactInfoObj != nil && i.HostInfoObj == nil && i.DomainInfoObj == nil {
		return CommandInfoContactType
	}

	if i.ContactInfoObj == nil && i.HostInfoObj != nil && i.DomainInfoObj == nil {
		return CommandInfoHostType
	}

	if i.ContactInfoObj == nil && i.HostInfoObj == nil && i.DomainInfoObj != nil {
		return CommandInfoDomainType
	}

	return CommandInfoType
}
