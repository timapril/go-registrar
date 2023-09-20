package epp

import (
	"encoding/xml"
	"strings"
)

// Update is used to hold one of the subordinate create objects for
// either a Host, Domain or Contact.
type Update struct {
	XMLName xml.Name `xml:"update" json:"-"`

	DomainUpdateObj  *DomainUpdate  `xml:",omitempty" json:"domain"`
	HostUpdateObj    *HostUpdate    `xml:",omitempty" json:"host"`
	ContactUpdateObj *ContactUpdate `xml:",omitempty" json:"contact"`

	GenericUpdateObj *GenericUpdate `xml:",omitempty" json:"update"`
}

// GetEPPUpdate is used to generate an update message envelope to later
// be populated with one of the create objects (Host, Domain or Contact).
func GetEPPUpdate(TransactionID string) Epp {
	epp := GetEPPCommand(TransactionID)
	epp.CommandObject.UpdateObject = &Update{}

	return epp
}

// GenericUpdate is used to receive a Update request that has not been
// typed yet and can be used later to create a typed version.
type GenericUpdate struct {
	XMLName              xml.Name `xml:"update" json:"-"`
	XMLNSDomain          string   `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSHost            string   `xml:"host,attr" json:"xmlns.host"`
	XMLNSContact         string   `xml:"contact,attr" json:"xmlns.contact"`
	XMLNSxsi             string   `xml:"xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`

	Name string `xml:"name" json:"name"`
	ID   string `xml:"id" json:"id"`

	AddObject    GenericUpdateAddRemove `xml:"add" json:"add"`
	RemoveObject GenericUpdateAddRemove `xml:"rem" json:"rem"`
	ChangeObject GenereicUpdateChange   `xml:"chg" json:"chg"`
}

// GenereicUpdateChange is used to receive an UpdateChange request that
// has not been typed yet and can be used to generate the typed version
// of the object later.
type GenereicUpdateChange struct {
	XMLName     xml.Name           `xml:"chg" json:"-"`
	Registrant  string             `xml:"registrant,omitempty" json:"registrant"`
	Name        string             `xml:"name,omitempty" json:"name"`
	Postal      *GenericPostalInfo `xml:"postalInfo,omitempty" json:"postalInfo"`
	VoiceNumber *GenericPhone      `xml:"voice,omitempty" json:"voice"`
	FaxNumber   *GenericPhone      `xml:"fax" json:"fax"`
	Email       string             `xml:"email,omitempty" json:"email"`
	AuthInfo    *GenericAuthInfo   `xml:"authInfo,omitempty" json:"authInfo"`
}

// GenericUpdateAddRemove is used to receive an AddRemove request that
// hast not been typed yet and can be used later to getnerate a typed
// version.
type GenericUpdateAddRemove struct {
	Hosts     []string          `xml:"ns>hostObj" json:"ns.hostObj"`
	Contacts  []GenericContact  `xml:"contact" json:"contact"`
	Statuses  []GenericStatus   `xml:"status" json:"status"`
	Addresses []GenericHostAddr `xml:"addr" json:"addr"`
}

// DomainUpdate is used to generate an update message for a domain.
type DomainUpdate struct {
	XMLName              xml.Name `xml:"domain:update" json:"-"`
	XMLNSDomain          string   `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`

	DomainName string `xml:"domain:name" json:"domain.name"`

	AddObject    DomainUpdateAddRemove `xml:"domain:add" json:"domain.add"`
	RemoveObject DomainUpdateAddRemove `xml:"domain:rem" json:"domain.rem"`
	ChangeObject DomainUpdateChange    `xml:"domain:chg" json:"domain.chg"`
}

// DomainUpdateChange is used to generate the change portion of a domain
// update object.
type DomainUpdateChange struct {
	XMLName    xml.Name    `xml:"domain:chg" json:"-"`
	Registrant string      `xml:"domain:registrant,omitempty" json:"domain.registrant"`
	AuthInfo   *DomainAuth `xml:"domain:authInfo,omitempty" json:"domain.authInfo"`
}

// GetEPPDomainUpdateChange constructs a DomainUpdateChange object used
// for changing the registrant ID and the auth password for a domain.
func GetEPPDomainUpdateChange(RegistrantID *string, AuthPassword *string) DomainUpdateChange {
	duc := DomainUpdateChange{}

	if RegistrantID != nil {
		duc.Registrant = *RegistrantID
	}

	if AuthPassword != nil {
		duc.AuthInfo = &DomainAuth{}
		duc.AuthInfo.Password = *AuthPassword
	}

	return duc
}

// DomainUpdateAddRemove is used to generate the add or rem portion of
// a domain update object.
type DomainUpdateAddRemove struct {
	Hosts    *DomainHostList `xml:"domain:ns" json:"domain.ns"`
	Contacts []DomainContact `xml:"domain:contact" json:"domain.contact"`
	Statuses []DomainStatus  `xml:"domain:status" json:"domain.status"`
}

// GetEPPDomainUpdateAddRemove constructs a DomainUpdateAddRemove object
// used for adding or removing properties from a domain.
func GetEPPDomainUpdateAddRemove(Hosts []string, Contacts []DomainContact, Statuses []string) DomainUpdateAddRemove {
	duar := DomainUpdateAddRemove{}

	if len(Hosts) != 0 {
		duar.Hosts = &DomainHostList{}

		for _, host := range Hosts {
			newHost := DomainHost{Value: host}
			duar.Hosts.Hosts = append(duar.Hosts.Hosts, newHost)
		}
	}

	duar.Contacts = Contacts

	for _, status := range Statuses {
		ds := DomainStatus{StatusFlag: status}
		duar.Statuses = append(duar.Statuses, ds)
	}

	return duar
}

// DomainStatus is used to represent a status of a domain.
type DomainStatus struct {
	XMLName    xml.Name `xml:"domain:status" json:"-"`
	StatusFlag string   `xml:"s,attr" json:"status"`
}

// GetEPPDomainUpdate is used to generate a doamin update message.
func GetEPPDomainUpdate(DomainName string, DomainAdd *DomainUpdateAddRemove, DomainRemove *DomainUpdateAddRemove, DomainChange *DomainUpdateChange, TransactionID string) Epp {
	domainNameUpper := strings.ToUpper(DomainName)

	epp := GetEPPUpdate(TransactionID)

	domainUpdate := &DomainUpdate{}
	domainUpdate.XMLNSDomain = DomainXMLNS
	domainUpdate.XMLNSxsi = W3XMLNSxsi
	domainUpdate.XMLxsiSchemaLocation = DomainSchema

	domainUpdate.DomainName = domainNameUpper

	if DomainChange != nil {
		domainUpdate.ChangeObject = *DomainChange
	}

	if DomainAdd != nil {
		domainUpdate.AddObject = *DomainAdd
	}

	if DomainRemove != nil {
		domainUpdate.RemoveObject = *DomainRemove
	}

	if strings.HasSuffix(domainNameUpper, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(domainNameUpper, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	epp.CommandObject.UpdateObject.DomainUpdateObj = domainUpdate

	return epp
}

// HostUpdate is used to generate an update message for a host.
type HostUpdate struct {
	XMLName              xml.Name `xml:"host:update" json:"-"`
	XMLNSHost            string   `xml:"xmlns:host,attr" json:"xmlns.host"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`

	HostName string `xml:"host:name" json:"host.name"`

	AddObject    HostUpdateAddRemove `xml:"host:add" json:"host.add"`
	RemoveObject HostUpdateAddRemove `xml:"host:rem" json:"host.rem"`
	ChangeObject *HostUpdateChange   `xml:"host:chg,omitempty" json:"host.chg"`
}

// HostUpdateChange is used to generate the change portion of the host
// update object.
type HostUpdateChange struct {
	XMLName  xml.Name `xml:"host:chg" json:"-"`
	HostName string   `xml:"host:name,omitempty" json:"host.name"`
}

// GetEPPHostUpdateChange constructs a HostUpdateChange object used
// for changing the Hostname of a host object.
func GetEPPHostUpdateChange(NewHostName string) HostUpdateChange {
	return HostUpdateChange{HostName: strings.ToUpper(NewHostName)}
}

// HostUpdateAddRemove is used to generate the add or rem portion of a
// host update object.
type HostUpdateAddRemove struct {
	Addresses []HostAddress `xml:"host:addr" json:"host.addr"`
	Statuses  []HostStatus  `xml:"host:status" json:"host.status"`
}

// HostStatus is used to represent a status of a host.
type HostStatus struct {
	XMLName    xml.Name `xml:"host:status" json:"-"`
	StatusFlag string   `xml:"s,attr" json:"status"`
}

// GetEPPHostUpdateAddRemove constructs the HostUpdateAddRemove object
// used for adding or removing host addresses or statuses from a host.
func GetEPPHostUpdateAddRemove(Addresses []HostAddress, Statuses []string) HostUpdateAddRemove {
	huar := HostUpdateAddRemove{}
	huar.Addresses = Addresses

	for _, status := range Statuses {
		hs := HostStatus{StatusFlag: status}
		huar.Statuses = append(huar.Statuses, hs)
	}

	return huar
}

// GetEPPHostUpdate  is used to generate a host update message.
func GetEPPHostUpdate(HostName string, HostAdd *HostUpdateAddRemove, HostRemove *HostUpdateAddRemove, HostChange *HostUpdateChange, TransactionID string) Epp {
	hostNameUpper := strings.ToUpper(HostName)

	epp := GetEPPUpdate(TransactionID)

	hostUpdate := &HostUpdate{}
	hostUpdate.XMLNSHost = HostXMLNS
	hostUpdate.XMLNSxsi = W3XMLNSxsi
	hostUpdate.XMLxsiSchemaLocation = HostSchema
	hostUpdate.HostName = hostNameUpper

	if HostAdd != nil {
		hostUpdate.AddObject = *HostAdd
	}

	if HostRemove != nil {
		hostUpdate.RemoveObject = *HostRemove
	}

	if HostChange != nil {
		hostUpdate.ChangeObject = HostChange
	}

	epp.CommandObject.UpdateObject.HostUpdateObj = hostUpdate

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

// ContactUpdate is used to generate an update message for a contact.
type ContactUpdate struct {
	XMLName              xml.Name `xml:"contact:update" json:"-"`
	XMLNSContact         string   `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`

	ContactID    string                 `xml:"contact:id" json:"contact.id"`
	AddObject    ContactUpdateAddRemove `xml:"contact:add" json:"contact.add"`
	RemoveObject ContactUpdateAddRemove `xml:"contact:rem" json:"contact.rem"`
	ChangeObject ContactUpdateChange    `xml:"contact:chg" json:"contact.chg"`
}

// GetEPPContactUpdate is used to generate a contact update message.
func GetEPPContactUpdate(ContactID string, Add ContactUpdateAddRemove, Rem ContactUpdateAddRemove, Chg ContactUpdateChange, TransactionID string) Epp {
	epp := GetEPPUpdate(TransactionID)

	contactUpdate := &ContactUpdate{}
	contactUpdate.XMLNSContact = ContactXMLNS
	contactUpdate.XMLNSxsi = W3XMLNSxsi
	contactUpdate.XMLxsiSchemaLocation = ContactSchema

	contactUpdate.ContactID = ContactID
	contactUpdate.AddObject = Add
	contactUpdate.RemoveObject = Rem
	contactUpdate.ChangeObject = Chg

	epp.CommandObject.UpdateObject.ContactUpdateObj = contactUpdate

	return epp
}

// ContactUpdateAddRemove is used to generate the add or rem portion of
// a contact update object.
type ContactUpdateAddRemove struct {
	Statuses []ContactStatus `xml:"contact:status" json:"contact.status"`
}

// GetEPPContactUpdateAddRemove constructs the ContactUpdateAddRemove
// object used for adding or removing statuses from a contact.
func GetEPPContactUpdateAddRemove(Statuses []string) ContactUpdateAddRemove {
	cuar := ContactUpdateAddRemove{}

	for _, status := range Statuses {
		cs := ContactStatus{StatusFlag: status}
		cuar.Statuses = append(cuar.Statuses, cs)
	}

	return cuar
}

// ContactUpdateChange is used to generate the chg portion of
// a contact update object.
type ContactUpdateChange struct {
	Postal      *PostalInfo  `xml:"contact:postalInfo,omitempty" json:"contact.postalInfo"`
	VoiceNumber *PhoneNumber `xml:"contact:voice,omitempty" json:"contact.voice"`
	FaxNumber   *PhoneNumber `xml:"contact:fax" json:"contact.fax"`
	Email       string       `xml:"contact:email,omitempty" json:"contact.email"`
	AuthInfo    *ContactAuth `xml:"contact:authInfo,omitempty" json:"contact.authInfo"`
}

// GetEPPContactUpdateChange constructs the ContactUpdateChange object
// used for changing postalinfo, phone numbers, email addresses or auth
// info for contacts.
func GetEPPContactUpdateChange(Postal *PostalInfo, Voice, Fax *PhoneNumber, EmailAddress, Password string) ContactUpdateChange {
	chg := ContactUpdateChange{}

	chg.Postal = Postal
	chg.VoiceNumber = Voice
	chg.FaxNumber = Fax
	chg.Email = EmailAddress

	if len(Password) != 0 {
		chg.AuthInfo = &ContactAuth{}
		chg.AuthInfo.Password = Password
	}

	return chg
}

// ContactStatus is used to represent a status of a contact.
type ContactStatus struct {
	XMLName    xml.Name `xml:"contact:status" json:"-"`
	StatusFlag string   `xml:"s,attr" json:"status"`
}

const (
	// CommandUpdateType represents a update command.
	CommandUpdateType string = "epp.command.update"

	// CommandUpdateDomainType represent a domain update request.
	CommandUpdateDomainType string = "epp.command.update.domain"

	// CommandUpdateHostType represent a host update request.
	CommandUpdateHostType string = "epp.command.update.host"

	// CommandUpdateContactType represent a contact update request.
	CommandUpdateContactType string = "epp.command.update.contact"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (u *Update) MessageType() string {
	if u.ContactUpdateObj != nil && u.DomainUpdateObj == nil && u.HostUpdateObj == nil {
		return CommandUpdateContactType
	}

	if u.ContactUpdateObj == nil && u.DomainUpdateObj != nil && u.HostUpdateObj == nil {
		return CommandUpdateDomainType
	}

	if u.ContactUpdateObj == nil && u.DomainUpdateObj == nil && u.HostUpdateObj != nil {
		return CommandUpdateHostType
	}

	return CommandUpdateType
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (u Update) TypedMessage() Update {
	out := Update{}
	if u.GenericUpdateObj.XMLNSContact != "" {
		out.ContactUpdateObj = &ContactUpdate{}
		out.ContactUpdateObj.XMLNSContact = u.GenericUpdateObj.XMLNSContact
		out.ContactUpdateObj.XMLNSxsi = u.GenericUpdateObj.XMLNSxsi
		out.ContactUpdateObj.XMLxsiSchemaLocation = u.GenericUpdateObj.XMLxsiSchemaLocation
		out.ContactUpdateObj.ContactID = u.GenericUpdateObj.ID

		for _, status := range u.GenericUpdateObj.AddObject.Statuses {
			out.ContactUpdateObj.AddObject.Statuses = append(out.ContactUpdateObj.AddObject.Statuses, ContactStatus{StatusFlag: status.Value})
		}

		for _, status := range u.GenericUpdateObj.RemoveObject.Statuses {
			out.ContactUpdateObj.RemoveObject.Statuses = append(out.ContactUpdateObj.RemoveObject.Statuses, ContactStatus{StatusFlag: status.Value})
		}

		out.ContactUpdateObj.ChangeObject.Postal = &PostalInfo{}
		out.ContactUpdateObj.ChangeObject.Postal.PostalInfoType = u.GenericUpdateObj.ChangeObject.Postal.Type
		out.ContactUpdateObj.ChangeObject.Postal.Name = u.GenericUpdateObj.ChangeObject.Postal.Name
		out.ContactUpdateObj.ChangeObject.Postal.Org = u.GenericUpdateObj.ChangeObject.Postal.Org
		out.ContactUpdateObj.ChangeObject.Postal.Address.Street = append(out.ContactUpdateObj.ChangeObject.Postal.Address.Street, u.GenericUpdateObj.ChangeObject.Postal.Addr.Streets...)
		out.ContactUpdateObj.ChangeObject.Postal.Address.City = u.GenericUpdateObj.ChangeObject.Postal.Addr.City
		out.ContactUpdateObj.ChangeObject.Postal.Address.Sp = u.GenericUpdateObj.ChangeObject.Postal.Addr.StateProv
		out.ContactUpdateObj.ChangeObject.Postal.Address.Pc = u.GenericUpdateObj.ChangeObject.Postal.Addr.PostalCode
		out.ContactUpdateObj.ChangeObject.Postal.Address.Cc = u.GenericUpdateObj.ChangeObject.Postal.Addr.Country
		out.ContactUpdateObj.ChangeObject.VoiceNumber = &PhoneNumber{}
		out.ContactUpdateObj.ChangeObject.VoiceNumber.Number = u.GenericUpdateObj.ChangeObject.VoiceNumber.Number
		out.ContactUpdateObj.ChangeObject.VoiceNumber.Extension = u.GenericUpdateObj.ChangeObject.VoiceNumber.Extension
		out.ContactUpdateObj.ChangeObject.FaxNumber = &PhoneNumber{}
		out.ContactUpdateObj.ChangeObject.FaxNumber.Number = u.GenericUpdateObj.ChangeObject.FaxNumber.Number
		out.ContactUpdateObj.ChangeObject.FaxNumber.Extension = u.GenericUpdateObj.ChangeObject.FaxNumber.Extension
		out.ContactUpdateObj.ChangeObject.Email = u.GenericUpdateObj.ChangeObject.Email

		if u.GenericUpdateObj.ChangeObject.AuthInfo != nil {
			out.ContactUpdateObj.ChangeObject.AuthInfo = &ContactAuth{}
			out.ContactUpdateObj.ChangeObject.AuthInfo.Password = u.GenericUpdateObj.ChangeObject.AuthInfo.Password
		}
	}

	if u.GenericUpdateObj.XMLNSDomain != "" {
		out.DomainUpdateObj = &DomainUpdate{}
		out.DomainUpdateObj.XMLNSDomain = u.GenericUpdateObj.XMLNSDomain
		out.DomainUpdateObj.XMLNSxsi = u.GenericUpdateObj.XMLNSxsi
		out.DomainUpdateObj.XMLxsiSchemaLocation = u.GenericUpdateObj.XMLxsiSchemaLocation

		out.DomainUpdateObj.DomainName = u.GenericUpdateObj.Name

		for _, status := range u.GenericUpdateObj.AddObject.Statuses {
			out.DomainUpdateObj.AddObject.Statuses = append(out.DomainUpdateObj.AddObject.Statuses, DomainStatus{StatusFlag: status.Value})
		}

		for _, ns := range u.GenericUpdateObj.AddObject.Hosts {
			out.DomainUpdateObj.AddObject.Hosts = &DomainHostList{}
			out.DomainUpdateObj.AddObject.Hosts.Hosts = append(out.DomainUpdateObj.AddObject.Hosts.Hosts, DomainHost{Value: ns})
		}

		for _, cont := range u.GenericUpdateObj.AddObject.Contacts {
			contType, err := ContactTypeFromString(cont.Type)
			if err == nil {
				out.DomainUpdateObj.AddObject.Contacts = append(out.DomainUpdateObj.AddObject.Contacts, DomainContact{Type: contType, Value: cont.Value})
			}
		}

		for _, status := range u.GenericUpdateObj.RemoveObject.Statuses {
			out.DomainUpdateObj.RemoveObject.Statuses = append(out.DomainUpdateObj.RemoveObject.Statuses, DomainStatus{StatusFlag: status.Value})
		}

		for _, ns := range u.GenericUpdateObj.RemoveObject.Hosts {
			out.DomainUpdateObj.RemoveObject.Hosts = &DomainHostList{}
			out.DomainUpdateObj.RemoveObject.Hosts.Hosts = append(out.DomainUpdateObj.RemoveObject.Hosts.Hosts, DomainHost{Value: ns})
		}

		for _, cont := range u.GenericUpdateObj.RemoveObject.Contacts {
			contType, err := ContactTypeFromString(cont.Type)
			if err == nil {
				out.DomainUpdateObj.RemoveObject.Contacts = append(out.DomainUpdateObj.RemoveObject.Contacts, DomainContact{Type: contType, Value: cont.Value})
			}
		}

		out.DomainUpdateObj.ChangeObject.Registrant = u.GenericUpdateObj.ChangeObject.Registrant

		if u.GenericUpdateObj.ChangeObject.AuthInfo != nil {
			out.DomainUpdateObj.ChangeObject.AuthInfo = &DomainAuth{}
			out.DomainUpdateObj.ChangeObject.AuthInfo.Password = u.GenericUpdateObj.ChangeObject.AuthInfo.Password
		}
	}

	if u.GenericUpdateObj.XMLNSHost != "" {
		out.HostUpdateObj = &HostUpdate{}
		out.HostUpdateObj.XMLNSHost = u.GenericUpdateObj.XMLNSHost
		out.HostUpdateObj.XMLNSxsi = u.GenericUpdateObj.XMLNSxsi
		out.HostUpdateObj.XMLxsiSchemaLocation = u.GenericUpdateObj.XMLxsiSchemaLocation

		out.HostUpdateObj.HostName = u.GenericUpdateObj.Name

		for _, status := range u.GenericUpdateObj.AddObject.Statuses {
			out.HostUpdateObj.AddObject.Statuses = append(out.HostUpdateObj.AddObject.Statuses, HostStatus{StatusFlag: status.Value})
		}

		for _, addr := range u.GenericUpdateObj.AddObject.Addresses {
			out.HostUpdateObj.AddObject.Addresses = append(out.HostUpdateObj.AddObject.Addresses, HostAddress{Address: addr.Address, IPVersion: addr.IPVersion})
		}

		for _, status := range u.GenericUpdateObj.RemoveObject.Statuses {
			out.HostUpdateObj.RemoveObject.Statuses = append(out.HostUpdateObj.RemoveObject.Statuses, HostStatus{StatusFlag: status.Value})
		}

		for _, addr := range u.GenericUpdateObj.RemoveObject.Addresses {
			out.HostUpdateObj.RemoveObject.Addresses = append(out.HostUpdateObj.RemoveObject.Addresses, HostAddress{Address: addr.Address, IPVersion: addr.IPVersion})
		}

		if u.GenericUpdateObj.ChangeObject.Name != "" {
			out.HostUpdateObj.ChangeObject = &HostUpdateChange{}

			out.HostUpdateObj.ChangeObject.HostName = u.GenericUpdateObj.ChangeObject.Name
		}
	}

	return out
}
