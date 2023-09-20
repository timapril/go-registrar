package epp

import (
	"encoding/xml"
	"strings"
)

// Transfer is used to hold one of the subordinate transfer objects for
// either a Domain or a Contact.
type Transfer struct {
	XMLName xml.Name `xml:"transfer" json:"-"`

	Operation TransferOperationType `xml:"op,attr" json:"op"`

	DomainTransferObj  *DomainTransfer  `xml:",omitempty" json:"domain"`
	ContactTransferObj *ContactTransfer `xml:",omitempty" json:"contact"`

	GenericTransferObj *GenericTransfer `xml:",omitempty" json:"transfer"`
}

// TransferOperationType is a type that represents the possible transfer
// operations allowed in EPP.
type TransferOperationType string

const (

	// TransferRequest represents an EPP transfer request message.
	TransferRequest TransferOperationType = "request"

	// TransferQuery represents an EPP transfer query message.
	TransferQuery TransferOperationType = "query"

	// TransferApprove represents an EPP transfer approve message.
	TransferApprove TransferOperationType = "approve"

	// TransferReject represents an EPP transfer reject message.
	TransferReject TransferOperationType = "reject"

	// TransferCancel represents an EPP transfer cancel message.
	TransferCancel TransferOperationType = "cancel"
)

// GetEPPTransfer is used to generate a transfer message envelope to
// later be populated with one of the transfer objects (Domain or
// Contact).
func GetEPPTransfer(Op TransferOperationType, TransactionID string) Epp {
	epp := GetEPPCommand(TransactionID)
	epp.CommandObject.TransferObject = &Transfer{}
	epp.CommandObject.TransferObject.Operation = Op

	return epp
}

// GenericTransfer is used to receive a Transfer message from a client.
type GenericTransfer struct {
	XMLName             xml.Name `xml:"transfer" json:"-"`
	XMLNSDomain         string   `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSContact        string   `xml:"contact,attr" json:"xmlns.contact"`
	XMLNsXSI            string   `xml:"xsi,attr" json:"xmlns.xsi"`
	XMLNsSchemaLocation string   `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`

	DomainName string           `xml:"name" json:"name"`
	Period     *DomainPeriod    `xml:"period,omitempty" json:"period"`
	ContactID  string           `xml:"id" json:"id"`
	AuthInfo   *GenericAuthInfo `xml:"authInfo,omitempty" json:"authInfo"`
}

// DomainTransfer is used to generate a transfer message for a domain.
type DomainTransfer struct {
	XMLName              xml.Name `xml:"domain:transfer" json:"-"`
	XMLNSDomain          string   `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`

	DomainName string        `xml:"domain:name" json:"domain.name"`
	Period     *DomainPeriod `xml:"domain:period,omitempty" json:"domain.period"`
	AuthInfo   *DomainAuth   `xml:"domain:authInfo,omitempty" json:"domain.authInfo"`
}

// GetEPPDomainTransfer is used to generate a domain transfer message
// with the provided domain name, operation type and transaction id.
func GetEPPDomainTransfer(DomainName string, Op TransferOperationType, TransactionID string) Epp {
	domainNameUpper := strings.ToUpper(DomainName)
	epp := GetEPPTransfer(Op, TransactionID)

	domainTransfer := &DomainTransfer{}
	domainTransfer.XMLNSDomain = DomainXMLNS
	domainTransfer.XMLNSxsi = W3XMLNSxsi
	domainTransfer.XMLxsiSchemaLocation = DomainSchema

	domainTransfer.DomainName = domainNameUpper

	epp.CommandObject.TransferObject.DomainTransferObj = domainTransfer

	if strings.HasSuffix(domainTransfer.DomainName, ".COM") {
		exten := GetCOMNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	if strings.HasSuffix(domainTransfer.DomainName, ".NET") {
		exten := GetNETNamestoreExtension()
		epp.CommandObject.ExtensionObject = &exten
	}

	return epp
}

// GetEPPDomainTransferRequest generates a Domain Transfer request
// message using the domain name, period and passowrd passed.
func GetEPPDomainTransferRequest(DomainName string, Period DomainPeriod, Password string, TransactionID string) Epp {
	epp := GetEPPDomainTransfer(DomainName, TransferRequest, TransactionID)
	epp.CommandObject.TransferObject.DomainTransferObj.Period = &Period
	auth := &DomainAuth{}
	auth.Password = Password
	epp.CommandObject.TransferObject.DomainTransferObj.AuthInfo = auth

	return epp
}

// GetEPPDomainTransferQuery generates a Domain Transfer query message
// using the domain name passed.
func GetEPPDomainTransferQuery(DomainName string, TransactionID string) Epp {
	epp := GetEPPDomainTransfer(DomainName, TransferQuery, TransactionID)

	return epp
}

// GetEPPDomainTransferApprove generates a Domain Transfer approve
// message using the domain name passed.
func GetEPPDomainTransferApprove(DomainName string, TransactionID string) Epp {
	epp := GetEPPDomainTransfer(DomainName, TransferApprove, TransactionID)

	return epp
}

// GetEPPDomainTransferReject generates a Domain Transfer reject
// message using the domain name passed.
func GetEPPDomainTransferReject(DomainName string, TransactionID string) Epp {
	epp := GetEPPDomainTransfer(DomainName, TransferReject, TransactionID)

	return epp
}

// GetEPPDomainTransferCancel generates a Domain Transfer cancel
// message using the domain name passed.
func GetEPPDomainTransferCancel(DomainName string, TransactionID string) Epp {
	epp := GetEPPDomainTransfer(DomainName, TransferCancel, TransactionID)

	return epp
}

// ContactTransfer is used to generate a transfer message for a contact.
type ContactTransfer struct {
	XMLName              xml.Name `xml:"contact:transfer" json:"-"`
	XMLNSContact         string   `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`

	ContactID string       `xml:"contact:id" json:"contact.id"`
	AuthInfo  *ContactAuth `xml:"contact:authInfo,omitempty" json:"contact.authInfo"`
}

// GetEPPContactTransfer is used to generate a contact transfer message
// with the provided contact id, operation type and transaction id.
func GetEPPContactTransfer(ContactID string, Op TransferOperationType, TransactionID string) Epp {
	epp := GetEPPTransfer(Op, TransactionID)

	contactTransfer := &ContactTransfer{}
	contactTransfer.XMLNSContact = ContactXMLNS
	contactTransfer.XMLNSxsi = W3XMLNSxsi
	contactTransfer.XMLxsiSchemaLocation = ContactSchema

	contactTransfer.ContactID = ContactID

	epp.CommandObject.TransferObject.ContactTransferObj = contactTransfer

	return epp
}

// GetEPPContactTransferRequest generates a Contact Transfer request
// message using the contact id and password passed.
func GetEPPContactTransferRequest(ContactID string, Password string, TransactionID string) Epp {
	epp := GetEPPContactTransfer(ContactID, TransferRequest, TransactionID)
	auth := &ContactAuth{}
	auth.Password = Password
	epp.CommandObject.TransferObject.ContactTransferObj.AuthInfo = auth

	return epp
}

// GetEPPContactTransferQuery generates a Contact Transfer query
// message using the contact id passed.
func GetEPPContactTransferQuery(ContactID string, TransactionID string) Epp {
	epp := GetEPPContactTransfer(ContactID, TransferQuery, TransactionID)

	return epp
}

// GetEPPContactTransferApprove generates a Contact Transfer approve
// message using the contact id passed.
func GetEPPContactTransferApprove(ContactID string, TransactionID string) Epp {
	epp := GetEPPContactTransfer(ContactID, TransferApprove, TransactionID)

	return epp
}

// GetEPPContactTransferReject generates a Contact Transfer reject
// message using the contact id passed.
func GetEPPContactTransferReject(ContactID string, TransactionID string) Epp {
	epp := GetEPPContactTransfer(ContactID, TransferReject, TransactionID)

	return epp
}

// GetEPPContactTransferCancel generates a Contact Transfer cancel
// message using the contact id passed.
func GetEPPContactTransferCancel(ContactID string, TransactionID string) Epp {
	epp := GetEPPContactTransfer(ContactID, TransferCancel, TransactionID)

	return epp
}

const (
	// CommandTransferType represents a transfer command.
	CommandTransferType string = "epp.command.transfer"

	// CommandTransferContactRequestType represnets a contact transfer request.
	CommandTransferContactRequestType string = "epp.command.transfer.contact.request"

	// CommandTransferContactQueryType represnets a contact transfer query.
	CommandTransferContactQueryType string = "epp.command.transfer.contact.query"

	// CommandTransferContactApproveType represnets a contact transfer approve.
	CommandTransferContactApproveType string = "epp.command.transfer.contact.approve"

	// CommandTransferContactRejectType represnets a contact transfer reject.
	CommandTransferContactRejectType string = "epp.command.transfer.contact.reject"

	// CommandTransferContactCancelType represnets a contact transfer cancel.
	CommandTransferContactCancelType string = "epp.command.transfer.contact.cancel"

	// CommandTransferDomainRequestType represnets a domain transfer request.
	CommandTransferDomainRequestType string = "epp.command.transfer.domain.request"

	// CommandTransferDomainQueryType represnets a domain transfer query.
	CommandTransferDomainQueryType string = "epp.command.transfer.domain.query"

	// CommandTransferDomainApproveType represnets a domain transfer approve.
	CommandTransferDomainApproveType string = "epp.command.transfer.domain.approve"

	// CommandTransferDomainRejectType represnets a domain transfer reject.
	CommandTransferDomainRejectType string = "epp.command.transfer.domain.reject"

	// CommandTransferDomainCancelType represnets a domain transfer cancel.
	CommandTransferDomainCancelType string = "epp.command.transfer.domain.cancel"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (t *Transfer) MessageType() string {
	if t.ContactTransferObj != nil && t.DomainTransferObj == nil {
		switch t.Operation {
		case TransferRequest:
			return CommandTransferContactRequestType
		case TransferQuery:
			return CommandTransferContactQueryType
		case TransferApprove:
			return CommandTransferContactApproveType
		case TransferReject:
			return CommandTransferContactRejectType
		case TransferCancel:
			return CommandTransferContactCancelType
		}
	}

	if t.ContactTransferObj == nil && t.DomainTransferObj != nil {
		switch t.Operation {
		case TransferRequest:
			return CommandTransferDomainRequestType
		case TransferQuery:
			return CommandTransferDomainQueryType
		case TransferApprove:
			return CommandTransferDomainApproveType
		case TransferReject:
			return CommandTransferDomainRejectType
		case TransferCancel:
			return CommandTransferDomainCancelType
		}
	}

	return CommandTransferType
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (t Transfer) TypedMessage() Transfer {
	out := Transfer{}
	out.Operation = t.Operation

	if t.GenericTransferObj.XMLNSContact != "" {
		out.ContactTransferObj = &ContactTransfer{}
		out.ContactTransferObj.XMLNSContact = t.GenericTransferObj.XMLNSContact
		out.ContactTransferObj.XMLNSxsi = t.GenericTransferObj.XMLNsXSI
		out.ContactTransferObj.XMLxsiSchemaLocation = t.GenericTransferObj.XMLNsSchemaLocation
		out.ContactTransferObj.ContactID = t.GenericTransferObj.ContactID

		if t.GenericTransferObj.AuthInfo != nil {
			out.ContactTransferObj.AuthInfo = &ContactAuth{}
			out.ContactTransferObj.AuthInfo.Password = t.GenericTransferObj.AuthInfo.Password
		}
	}

	if t.GenericTransferObj.XMLNSDomain != "" {
		out.DomainTransferObj = &DomainTransfer{}
		out.DomainTransferObj.XMLNSDomain = t.GenericTransferObj.XMLNSDomain
		out.DomainTransferObj.XMLNSxsi = t.GenericTransferObj.XMLNsXSI
		out.DomainTransferObj.XMLxsiSchemaLocation = t.GenericTransferObj.XMLNsSchemaLocation
		out.DomainTransferObj.DomainName = t.GenericTransferObj.DomainName

		if t.GenericTransferObj.Period != nil {
			out.DomainTransferObj.Period = &DomainPeriod{}
			out.DomainTransferObj.Period.Unit = t.GenericTransferObj.Period.Unit
			out.DomainTransferObj.Period.Value = t.GenericTransferObj.Period.Value
		}

		if t.GenericTransferObj.AuthInfo != nil {
			out.DomainTransferObj.AuthInfo = &DomainAuth{}
			out.DomainTransferObj.AuthInfo.Password = t.GenericTransferObj.AuthInfo.Password
		}
	}

	return out
}
