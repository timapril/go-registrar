package epp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// DomainXMLNS represents the namespace used for XML Domain objects.
	DomainXMLNS string = "urn:ietf:params:xml:ns:domain-1.0"

	// DomainSchema represents the schama loccation for XML Domain objects.
	DomainSchema string = "urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd"

	// HostXMLNS represents the namespace used for XML Host objects.
	HostXMLNS string = "urn:ietf:params:xml:ns:host-1.0"

	// HostSchema represents the schama loccation for XML Host objects.
	HostSchema string = "urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd"

	// ContactXMLNS represents the namespace used for XML Contact objects.
	ContactXMLNS string = "urn:ietf:params:xml:ns:contact-1.0"

	// ContactSchema represents the schama loccation for XML Contact objects.
	ContactSchema string = "urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd"

	// W3XMLNSxsi is the string that is presnet in the XMLNSxsi field for
	// EPP objects.
	W3XMLNSxsi string = "http://www.w3.org/2001/XMLSchema-instance"
)

const (
	// StatusServerHold is a constant to represent the state where an
	// object is in a server hold.
	StatusServerHold string = "serverHold"

	// StatusClientHold is a constant to represents the state where an
	// object is in a client hold.
	StatusClientHold string = "clientHold"

	// StatusClientUpdateProhibited is a constant to represent the state
	// where object is update locked on the registrar side.
	StatusClientUpdateProhibited string = "clientUpdateProhibited"

	// StatusClientDeleteProhibited is a constant to represent the state
	// where object is delete locked on the registrar side.
	StatusClientDeleteProhibited string = "clientDeleteProhibited"

	// StatusClientRenewProhibited is a constant to represent the state
	// where object is renew locked on the registrar side.
	StatusClientRenewProhibited string = "clientRenewProhibited"

	// StatusClientTransferProhibited is a constant to represent the state
	// where object is transfer locked on the registrar side.
	StatusClientTransferProhibited string = "clientTransferProhibited"

	// StatusServerUpdateProhibited is a constant to represent the state
	// where object is update locked on the registry side.
	StatusServerUpdateProhibited string = "serverUpdateProhibited"

	// StatusServerDeleteProhibited is a constant to represent the state
	// where object is delete locked on the registry side.
	StatusServerDeleteProhibited string = "serverDeleteProhibited"

	// StatusServerRenewProhibited is a constant to represent the state
	// where object is renew locked on the registry side.
	StatusServerRenewProhibited string = "serverRenewProhibited"

	// StatusServerTransferProhibited is a constant to represent the state
	// where object is transfer locked on the registry side.
	StatusServerTransferProhibited string = "serverTransferProhibited"

	// StatusOK is a constant to represent the OK status for an object
	// when no other statuses are present.
	StatusOK string = "ok"

	// StatusLinked is a constant to represent if one object is linked to
	// some other object, often used as an indication that the object is
	// in use, an example would be a Host is linked to a Domain.
	StatusLinked string = "linked"

	// StatusPendingCreate is a constant to represent an object that is
	// pending creation.
	StatusPendingCreate string = "pendingCreate"

	// StatusPendingDelete is a constant to represent an object that is
	// pending deletion.
	StatusPendingDelete string = "pendingDelete"

	// StatusPendingTransfer is a constant to represent an object that is
	// pending transfer.
	StatusPendingTransfer string = "pendingTransfer"

	// StatusPendingUpdate is a constant to represent that an object has a
	// pending update.
	StatusPendingUpdate string = "pendingUpdate"

	// StatusPendingRenew is a constant to represent that an object has a
	// pending renewal.
	StatusPendingRenew string = "pendingRenew"
)

var (
	// ErrNoTransctionFound indicates that no transaction was found in the EPP
	// request.
	ErrNoTransctionFound = errors.New("no transaction found")

	// ErrServerTransactionIDNotFound indicates that no server transaction id
	// was found in the response message.
	ErrServerTransactionIDNotFound = errors.New("no server transcation id found")
)

// Epp is used to construct and receive <epp> messages.
type Epp struct {
	XMLName xml.Name `xml:"epp" json:"-"`
	// XMLNS                  string    `xml:"xmlns,omitempty,attr"`
	XMLNSxsi               string    `xml:"xmlns:xsi,attr,omitempty" json:"xmlns.xsi"`
	XMLNSxsiIn             string    `xml:"xsi,attr,omitempty" json:"xmlns.xsiin"`
	XMLxsiSchemaLocation   string    `xml:"xsi:schemaLocation,attr,omitempty" json:"xmlns.schemaLocation"`
	XMLxsiSchemaLocationIn string    `xml:"schemaLocation,attr,omitempty" json:"xmlns.schemaLocationIn"`
	CommandObject          *Command  `xml:",omitempty" json:"command"`
	GreetingObject         *Greeting `xml:",omitempty" json:"greeting"`
	HelloObject            *Hello    `xml:",omitempty" json:"hello"`
	ResponseObject         *Response `xml:",omitempty" json:"response"`
}

// GetEPP Returns an uninitialized EPP object.
func GetEPP() Epp {
	return Epp{
		// XMLNS:                "urn:ietf:params:xml:ns:epp-1.0",
		XMLNSxsi:             "http://www.w3.org/2001/XMLSchema-instance",
		XMLxsiSchemaLocation: "urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd",
	}
}

// ToString turns the EPP message into an XML string so it can be
// printed.
func (e Epp) ToString() (string, error) {
	message, err := xml.MarshalIndent(e, "", "  ")

	return string(message), err
}

// ToStringCS is used to convert an epp message into a multi-lined
// string of text representing the XML version of the epp object passed
// prefixed with the value provided as an argument
//
//	eg. a hello prefixed with "S:" would look something like this:
//
// C: <?xml version="1.0" encoding="UTF-8" standalone="no"?>
// C: <epp xmlns="urn:ietf:params:xml:ns:epp-1.0"
//
//	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
//	xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
//
// C: <hello xmlns="urn:ietf:params:xml:ns:epp-1.0"/>
// C: </epp>.
func (e Epp) ToStringCS(prefix string) (string, error) {
	messageString, err := e.ToString()

	var buffer bytes.Buffer
	for _, line := range strings.Split(messageString, "\n") {
		buffer.WriteString(prefix)
		buffer.WriteString(line)
		buffer.WriteString("\n")
	}

	return buffer.String(), err
}

const (
	// GenericEPPType represents a <epp> message that is unknown.
	GenericEPPType string = "epp.generic"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (e Epp) MessageType() string {
	if e.GreetingObject != nil && e.CommandObject == nil && e.HelloObject == nil && e.ResponseObject == nil {
		return e.GreetingObject.MessageType()
	}

	if e.CommandObject != nil && e.GreetingObject == nil && e.HelloObject == nil && e.ResponseObject == nil {
		return e.CommandObject.MessageType()
	}

	if e.HelloObject != nil && e.GreetingObject == nil && e.CommandObject == nil && e.ResponseObject == nil {
		return e.HelloObject.MessageType()
	}

	if e.ResponseObject != nil && e.GreetingObject == nil && e.CommandObject == nil && e.HelloObject == nil {
		return e.ResponseObject.MessageType()
	}

	return GenericEPPType
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (e Epp) TypedMessage() Epp {
	out := Epp{}

	out.XMLName = e.XMLName
	if e.XMLNSxsi != "" {
		out.XMLNSxsi = e.XMLNSxsi
	} else {
		out.XMLNSxsi = e.XMLNSxsiIn
	}

	if e.XMLxsiSchemaLocation != "" {
		out.XMLxsiSchemaLocation = e.XMLxsiSchemaLocation
	} else {
		out.XMLxsiSchemaLocation = e.XMLxsiSchemaLocationIn
	}

	if e.CommandObject != nil {
		command := e.CommandObject.TypedMessage()
		out.CommandObject = &command
	}

	if e.GreetingObject != nil {
		greeting := e.GreetingObject.TypedMessage()
		out.GreetingObject = &greeting
	}

	if e.HelloObject != nil {
		hello := e.HelloObject.TypedMessage()
		out.HelloObject = &hello
	}

	if e.ResponseObject != nil {
		response := e.ResponseObject.TypedMessage()
		out.ResponseObject = &response
	}

	return out
}

// GetTransactionID will return the transaction ID associated with the
// epp command object if it is set otherwise an error is returned.
func (e Epp) GetTransactionID() (string, error) {
	if e.CommandObject != nil {
		return e.CommandObject.TransactionID, nil
	}

	return "", ErrNoTransctionFound
}

// GetServerTransactionID will return the transaction ID from the server
// associated with the epp response object if it is set, otherwise an error is
// returned.
func (e Epp) GetServerTransactionID() (string, error) {
	if e.ResponseObject != nil {
		return e.ResponseObject.TransactionID.ServerTransactionID, nil
	}

	return "", ErrServerTransactionIDNotFound
}

// ContactAuth is used to hold the auth password for a contact.
type ContactAuth struct {
	Password string `xml:"contact:pw,omitempty" json:"contact.pw"`
}

// DomainAuth is used to hold the auth password for a domain.
type DomainAuth struct {
	Password string `xml:"domain:pw,omitempty" json:"domain.pw"`
}

// HostAuth is used to hold the auth password for a host.
type HostAuth struct {
	Password string `xml:"host:pw,omitempty" json:"host.pw"`
}

// EPPTimeFormat is used when formating a time for EPP message.
const EPPTimeFormat = "2006-01-02T15:04:05.0000Z"

// EPPTimeFormat2 is used when formating a time for EPP message.
const EPPTimeFormat2 = "2006-01-02T15:04:05Z"

// EPPDateForamt is used when formating a date for an EPP message.
const EPPDateForamt = "2006-01-02"

// DateTimeToDate is used to convert a date time to just a date.
func DateTimeToDate(dateTime string) (string, error) {
	time, err := time.Parse(EPPTimeFormat2, dateTime)
	if err != nil {
		return "", fmt.Errorf("error parsing date: %w", err)
	}

	return time.Format(EPPDateForamt), nil
}

// DefaultEPPPort is the default port that servers will accept
// connections on.
const DefaultEPPPort int = 1700
