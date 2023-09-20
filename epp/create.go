package epp

import (
	"encoding/xml"
	"errors"
	"strings"
)

// ErrUnknownContactType indicates that the contact type was not
// supported by the package.
var ErrUnknownContactType = errors.New("unknown Contact Type")

// Create is used to hold one of the subordinate create objects for
// either a Host, Domain or Contact.
type Create struct {
	XMLName xml.Name `xml:"create"  json:"-"`

	DomainCreateObj  *DomainCreate  `xml:",omitempty" json:"DomainCreate,omitempty"`
	HostCreateObj    *HostCreate    `xml:",omitempty" json:"HostCreate,omitempty"`
	ContactCreateObj *ContactCreate `xml:",omitempty" json:"ContactCreate,omitempty"`

	GenericCreateObj *GenericCreate `xml:",omitempty" json:"GenericaCreate,omitempty"`
}

// GetEPPCreate is used to generate a create message envelope to later
// be populated with one of the create objects (Host, Domain or Contact).
func GetEPPCreate(TransactionID string) Epp {
	epp := GetEPPCommand(TransactionID)
	epp.CommandObject.CreateObject = &Create{}

	return epp
}

// GenericCreate is used to receive a Create message from a client.
type GenericCreate struct {
	XMLName              xml.Name `xml:"create"  json:"-"`
	XMLNSDomain          string   `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSHost            string   `xml:"host,attr" json:"xmlns.host"`
	XMLNSContact         string   `xml:"contact,attr" json:"xmlns.contact"`
	XMLNSxsi             string   `xml:"xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`

	ID             string              `xml:"id" json:"id"`
	Name           string              `xml:"name" json:"name"`
	Period         DomainPeriod        `xml:"period" json:"period"`
	Registrant     string              `xml:"registrant" json:"registrant"`
	Contacts       []GenericContact    `xml:"contact" json:"contact"`
	PostalInfoList []GenericPostalInfo `xml:"postalInfo" json:"postalInfo"`
	VoiceNumber    GenericPhone        `xml:"voice" json:"voice"`
	FaxNumber      GenericPhone        `xml:"fax" json:"fax"`
	Email          string              `xml:"email" json:"email"`
	Hosts          []string            `xml:"ns>hostObj" json:"ns.hostObj"`
	Addresses      []GenericHostAddr   `xml:"addr" json:"addresses"`

	AuthObj *GenericAuthInfo `xml:"authInfo"`
}

// DomainCreate is used to generate a create message for a domain.
type DomainCreate struct {
	XMLName              xml.Name `xml:"domain:create"  json:"-"`
	XMLNSDomain          string   `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"  json:"xmlns.schemaLocation"`

	DomainName string          `xml:"domain:name" json:"domain.name"`
	Period     DomainPeriod    `xml:"domain:period" json:"domain.period"`
	Hosts      *DomainHostList `xml:",omitempty" json:"hosts"`
	Registrant string          `xml:"domain:registrant,omitempty" json:"domain.registrant"`
	Contacts   []DomainContact `xml:"domain:contact" json:"domain.contact"`

	DomainAuthObj *DomainAuth `xml:"domain:authInfo" json:"domain.authInfo"`
}

// GetEPPDomainCreate is used to generate a domain create message.
func GetEPPDomainCreate(DomainName string, Period DomainPeriod,
	Hosts []DomainHost, RegistrantID *string, adminID *string,
	techID *string, billingID *string, Password string,
	TransactionID string,
) Epp {
	domainName := strings.ToUpper(DomainName)

	epp := GetEPPCreate(TransactionID)

	domainCreate := &DomainCreate{}
	domainCreate.XMLNSDomain = DomainXMLNS
	domainCreate.XMLNSxsi = W3XMLNSxsi
	domainCreate.XMLxsiSchemaLocation = DomainSchema

	domainCreate.DomainName = domainName
	domainCreate.Period = Period

	if len(Hosts) != 0 {
		domainCreate.Hosts = &DomainHostList{}
		domainCreate.Hosts.Hosts = Hosts
	}

	if RegistrantID != nil {
		domainCreate.Registrant = *RegistrantID
	}

	if adminID != nil {
		domainCreate.Contacts = append(domainCreate.Contacts, GetEPPDomainContact("admin", *adminID))
	}

	if techID != nil {
		domainCreate.Contacts = append(domainCreate.Contacts, GetEPPDomainContact("tech", *techID))
	}

	if billingID != nil {
		domainCreate.Contacts = append(domainCreate.Contacts, GetEPPDomainContact("billing", *billingID))
	}

	if len(Password) != 0 {
		domainCreate.DomainAuthObj = &DomainAuth{}
		domainCreate.DomainAuthObj.Password = Password
	}

	if strings.HasSuffix(domainName, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(domainName, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	epp.CommandObject.CreateObject.DomainCreateObj = domainCreate

	return epp
}

// DomainContact constructs a contact object used for mapping from
// domains to contacts while having the type of contact included in
// the object.
type DomainContact struct {
	XMLName xml.Name    `xml:"domain:contact"  json:"-"`
	Type    ContactType `xml:"type,attr" json:"type"`
	Value   string      `xml:",chardata" json:"value"`
}

// ContactType is a type to indicate which type a contact is.
type ContactType string

const (
	// Tech is used to indicate a technical contact.
	Tech ContactType = "tech"

	// Admin is used to indicate an administrative contact.
	Admin ContactType = "admin"

	// Billing is used to indicate an billing contact.
	Billing ContactType = "billing"
)

// ContactTypeFromString takes a string describing a contact type and
// converts it into a ContactType object or returns an error of no
// matching type is found.
func ContactTypeFromString(ct string) (c ContactType, err error) {
	switch strings.ToLower(ct) {
	case "tech":
		c = Tech
	case "admin":
		c = Admin
	case "billing":
		c = Billing
	default:
		err = ErrUnknownContactType
	}

	return
}

// GetEPPDomainContact creates and returns a domain contact map object
// that has the value and type set.
func GetEPPDomainContact(ContactType ContactType, ContactID string) DomainContact {
	dc := DomainContact{}
	dc.Type = ContactType
	dc.Value = ContactID

	return dc
}

// DomainPeriod is a period of time to be conveyed to the EPP server
// consisting of a unit (Years or Months) and a number of time units.
type DomainPeriod struct {
	Unit  DomainPeriodUnit `xml:"unit,attr" json:"unit"`
	Value int              `xml:",chardata" json:"value"`
}

// DomainPeriodUnit represents the unit of time for EPP.
type DomainPeriodUnit string

const (
	// DomainPeriodYear is used to represent a unit of time corresponding
	// to a Year for EPP.
	DomainPeriodYear DomainPeriodUnit = "y"

	// DomainPeriodMonth is used to represent a unit of time corresponding
	// to a month for EPP.
	DomainPeriodMonth DomainPeriodUnit = "m"
)

// GetEPPDomainPeriod is used to create an DomainPeriod.
func GetEPPDomainPeriod(Unit DomainPeriodUnit, Value int) DomainPeriod {
	dp := DomainPeriod{}
	dp.Unit = Unit
	dp.Value = Value

	return dp
}

// DomainHost is used to represent a host for a domain object.
type DomainHost struct {
	XMLName xml.Name `xml:"domain:hostObj" json:"-"`
	Value   string   `xml:",chardata" json:"value"`
}

// DomainHostList is used to represent a list of host objects under a
// domain.
type DomainHostList struct {
	XMLName xml.Name     `xml:"domain:ns,omitempty" json:"-"`
	Hosts   []DomainHost `xml:"domain:hostObj" json:"hosts"`
}

// HostCreate is used to generate a create message for a host.
type HostCreate struct {
	XMLName              xml.Name      `xml:"host:create"  json:"-"`
	XMLNSHost            string        `xml:"xmlns:host,attr" json:"xmlns.host"`
	XMLNSxsi             string        `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string        `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	HostName             string        `xml:"host:name" json:"hostname"`
	Addresses            []HostAddress `xml:"host:addr" json:"address"`
}

// HostAddress is used to hold the required information for a
// HostAddress in a host create object. The values are Address, to
// contain the IP address and the IP version which is allowed to be
// "v4" and "v6" depending on the IP type.
type HostAddress struct {
	XMLName   xml.Name      `xml:"host:addr"  json:"-"`
	IPVersion IPVersionName `xml:"ip,attr" json:"ip"`
	Address   string        `xml:",chardata" json:"address"`
}

// IPVersionName is a type to indicate which IP version an Address is.
type IPVersionName string

const (
	// IPv4 is used to indicate an IPv4 address.
	IPv4 IPVersionName = "v4"

	// IPv6 is used to indicate an IPv6 address.
	IPv6 IPVersionName = "v6"
)

// GetEPPHostCreate is used to generate a host create message by
// providing a hostname, IPv4 and IPv6 addresses and a transaction ID.
func GetEPPHostCreate(hostname string, ipv4Addresses []string, ipv6Addresses []string, TransactionID string) Epp {
	epp := GetEPPCreate(TransactionID)
	hostCreate := &HostCreate{}
	hostCreate.XMLNSHost = HostXMLNS
	hostCreate.XMLNSxsi = W3XMLNSxsi
	hostCreate.XMLxsiSchemaLocation = HostSchema
	hostCreate.HostName = strings.ToUpper(hostname)

	for _, ip := range ipv4Addresses {
		ha := HostAddress{IPVersion: IPv4, Address: ip}
		hostCreate.Addresses = append(hostCreate.Addresses, ha)
	}

	for _, ip := range ipv6Addresses {
		ha := HostAddress{IPVersion: IPv6, Address: ip}
		hostCreate.Addresses = append(hostCreate.Addresses, ha)
	}

	epp.CommandObject.CreateObject.HostCreateObj = hostCreate

	if strings.HasSuffix(hostCreate.HostName, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(hostCreate.HostName, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	return epp
}

// ContactCreate is used to generate a create message for a contact.
type ContactCreate struct {
	XMLName              xml.Name `xml:"contact:create"  json:"-"`
	XMLNSContact         string   `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`

	ID             string       `xml:"contact:id" json:"contact.id"`
	PostalInfoList []PostalInfo `xml:"contact:postalInfo" json:"contact.postalInfo"`
	VoiceNumber    PhoneNumber  `xml:"contact:voice" json:"voiceNumber"`
	FaxNumber      PhoneNumber  `xml:"contact:fax" json:"faxNumber"`
	Email          string       `xml:"contact:email" json:"contact.email"`
	ContactAuthObj *ContactAuth `xml:"contact:authInfo,omitempty" json:"contact.authInfo"`
}

// GetEPPContactCreate is used to generate a contact create message.
func GetEPPContactCreate(contactID string, postalInfo PostalInfo, Email string,
	Voice PhoneNumber, Fax PhoneNumber, Password string,
	TransactionID string,
) Epp {
	epp := GetEPPCreate(TransactionID)

	contactCreate := &ContactCreate{}
	contactCreate.XMLNSContact = ContactXMLNS
	contactCreate.XMLNSxsi = W3XMLNSxsi
	contactCreate.XMLxsiSchemaLocation = ContactSchema

	contactCreate.ID = contactID
	contactCreate.PostalInfoList = append(contactCreate.PostalInfoList, postalInfo)
	contactCreate.Email = Email
	contactCreate.VoiceNumber = Voice
	contactCreate.FaxNumber = Fax

	if len(Password) != 0 {
		contactCreate.ContactAuthObj = &ContactAuth{}
		contactCreate.ContactAuthObj.Password = Password
	}

	epp.CommandObject.CreateObject.ContactCreateObj = contactCreate

	return epp
}

// PostalInfo represents a Postal Info block of a create message for
// a contact.
type PostalInfo struct {
	XMLName        xml.Name          `xml:"contact:postalInfo" json:"-"`
	PostalInfoType string            `xml:"type,attr" json:"postalInfoType"`
	Name           string            `xml:"contact:name" json:"name"`
	Org            string            `xml:"contact:org" json:"org"`
	Address        PostalInfoAddress `xml:"contact:addr" json:"address"`
}

// PostalInfoAddress represents the address portion of a postal info
// object.
type PostalInfoAddress struct {
	XMLName xml.Name `xml:"contact:addr" json:"-"`
	Street  []string `xml:"contact:street" json:"street"`
	City    string   `xml:"contact:city" json:"city"`
	Sp      string   `xml:"contact:sp" json:"sp"`
	Pc      string   `xml:"contact:pc" json:"pc"`
	Cc      string   `xml:"contact:cc" json:"cc"`
}

// GetEPPPostalInfo uses the parameters passed to generate a postal
// info object that can be used in a contact create message. There are
// three street values allowed but if a value is left blank it will
// be excluded from the list of entries created.
func GetEPPPostalInfo(PostalType string, Name string, Org string,
	Street1 string, Street2 string, Street3 string, City string,
	StateProv string, PostalCode string, Country string,
) PostalInfo {
	postalInfo := PostalInfo{}
	postalInfo.PostalInfoType = PostalType
	postalInfo.Name = Name
	postalInfo.Org = Org

	if len(Street1) != 0 {
		postalInfo.Address.Street = append(postalInfo.Address.Street, Street1)
	}

	if len(Street2) != 0 {
		postalInfo.Address.Street = append(postalInfo.Address.Street, Street2)
	}

	if len(Street3) != 0 {
		postalInfo.Address.Street = append(postalInfo.Address.Street, Street3)
	}

	postalInfo.Address.City = City
	postalInfo.Address.Sp = StateProv
	postalInfo.Address.Pc = PostalCode
	postalInfo.Address.Cc = Country

	return postalInfo
}

// PhoneNumber is used to hold the information related to a Phone Number
// as EPP needs is represnted.
type PhoneNumber struct {
	Number    string `xml:",chardata" json:"number"`
	Extension string `xml:"x,attr" json:"extension"`
}

// GetEPPPhoneNumber is used to generate a PhoneNumber object from a
// phone number and extension (if valid).
func GetEPPPhoneNumber(Number string, Extension string) PhoneNumber {
	pn := PhoneNumber{}
	pn.Number = Number
	pn.Extension = Extension

	return pn
}

const (
	// CommandCreateType represents a create command.
	CommandCreateType string = "epp.command.create"

	// CommandCreateContactType represents a contact creation request.
	CommandCreateContactType string = "epp.command.create.contact"

	// CommandCreateDomainType represents a domain creation request.
	CommandCreateDomainType string = "epp.command.create.domain"

	// CommandCreateHostType represents a host creation request.
	CommandCreateHostType string = "epp.command.create.host"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (c *Create) MessageType() string {
	if c.ContactCreateObj != nil && c.DomainCreateObj == nil &&
		c.HostCreateObj == nil {
		return CommandCreateContactType
	}

	if c.ContactCreateObj == nil && c.DomainCreateObj != nil &&
		c.HostCreateObj == nil {
		return CommandCreateDomainType
	}

	if c.ContactCreateObj == nil && c.DomainCreateObj == nil &&
		c.HostCreateObj != nil {
		return CommandCreateHostType
	}

	return CommandCreateType
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (c Create) TypedMessage() Create {
	out := Create{}
	if c.GenericCreateObj.XMLNSContact != "" {
		out.ContactCreateObj = &ContactCreate{}
		out.ContactCreateObj.XMLNSContact = c.GenericCreateObj.XMLNSContact
		out.ContactCreateObj.XMLNSxsi = c.GenericCreateObj.XMLNSxsi
		out.ContactCreateObj.XMLxsiSchemaLocation = c.GenericCreateObj.XMLxsiSchemaLocation

		out.ContactCreateObj.ID = c.GenericCreateObj.ID
		out.ContactCreateObj.VoiceNumber.Extension = c.GenericCreateObj.VoiceNumber.Extension
		out.ContactCreateObj.VoiceNumber.Number = c.GenericCreateObj.VoiceNumber.Number
		out.ContactCreateObj.FaxNumber.Extension = c.GenericCreateObj.FaxNumber.Extension
		out.ContactCreateObj.FaxNumber.Number = c.GenericCreateObj.FaxNumber.Number
		out.ContactCreateObj.Email = c.GenericCreateObj.Email

		for _, postalInfoItem := range c.GenericCreateObj.PostalInfoList {
			postalInfo := PostalInfo{}
			postalInfo.PostalInfoType = postalInfoItem.Type
			postalInfo.Name = postalInfoItem.Name
			postalInfo.Org = postalInfoItem.Org

			postalInfo.Address.Street = postalInfoItem.Addr.Streets
			postalInfo.Address.City = postalInfoItem.Addr.City
			postalInfo.Address.Sp = postalInfoItem.Addr.StateProv
			postalInfo.Address.Pc = postalInfoItem.Addr.PostalCode
			postalInfo.Address.Cc = postalInfoItem.Addr.Country
			out.ContactCreateObj.PostalInfoList = append(out.ContactCreateObj.PostalInfoList, postalInfo)
		}

		if c.GenericCreateObj.AuthObj != nil {
			out.ContactCreateObj.ContactAuthObj = &ContactAuth{}
			out.ContactCreateObj.ContactAuthObj.Password = c.GenericCreateObj.AuthObj.Password
		}
	}

	if c.GenericCreateObj.XMLNSDomain != "" {
		out.DomainCreateObj = &DomainCreate{}
		out.DomainCreateObj.XMLNSDomain = c.GenericCreateObj.XMLNSDomain
		out.DomainCreateObj.XMLNSxsi = c.GenericCreateObj.XMLNSxsi
		out.DomainCreateObj.XMLxsiSchemaLocation = c.GenericCreateObj.XMLxsiSchemaLocation

		out.DomainCreateObj.DomainName = c.GenericCreateObj.Name
		out.DomainCreateObj.Period.Unit = c.GenericCreateObj.Period.Unit
		out.DomainCreateObj.Period.Value = c.GenericCreateObj.Period.Value
		out.DomainCreateObj.Registrant = c.GenericCreateObj.Registrant

		if len(c.GenericCreateObj.Hosts) > 0 {
			out.DomainCreateObj.Hosts = &DomainHostList{}

			for _, ns := range c.GenericCreateObj.Hosts {
				host := DomainHost{}
				host.Value = ns
				out.DomainCreateObj.Hosts.Hosts = append(out.DomainCreateObj.Hosts.Hosts, host)
			}
		}

		for _, inCont := range c.GenericCreateObj.Contacts {
			cont := DomainContact{}
			ct, err := ContactTypeFromString(inCont.Type)
			cont.Type = ct
			cont.Value = inCont.Value

			if err == nil {
				out.DomainCreateObj.Contacts = append(out.DomainCreateObj.Contacts, cont)
			}
		}

		if c.GenericCreateObj.AuthObj != nil {
			out.DomainCreateObj.DomainAuthObj = &DomainAuth{}
			out.DomainCreateObj.DomainAuthObj.Password = c.GenericCreateObj.AuthObj.Password
		}
	}

	if c.GenericCreateObj.XMLNSHost != "" {
		out.HostCreateObj = &HostCreate{}
		out.HostCreateObj.XMLNSHost = c.GenericCreateObj.XMLNSHost
		out.HostCreateObj.XMLNSxsi = c.GenericCreateObj.XMLNSxsi
		out.HostCreateObj.XMLxsiSchemaLocation = c.GenericCreateObj.XMLxsiSchemaLocation

		out.HostCreateObj.HostName = c.GenericCreateObj.Name

		for _, hos := range c.GenericCreateObj.Addresses {
			ha := HostAddress{}
			ha.Address = hos.Address
			ha.IPVersion = hos.IPVersion
			out.HostCreateObj.Addresses = append(out.HostCreateObj.Addresses, ha)
		}
	}

	return out
}
