package epp

import (
	"encoding/xml"
)

// Hello is used to construct and receive <hello> messages.
type Hello struct {
	XMLName xml.Name `xml:"hello" json:"-"`
	XMLNS   string   `xml:"xmlns,attr,omitempty" json:"xmlns"`
}

// GetEPPHello will create an EPP <hello> message.
func GetEPPHello() Epp {
	epp := GetEPP()

	hello := Hello{}
	hello.XMLNS = "urn:ietf:params:xml:ns:epp-1.0"

	epp.HelloObject = &hello

	return epp
}

const (
	// HelloType represents a <hello> message.
	HelloType string = "epp.hello"
)

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (h Hello) TypedMessage() Hello {
	return h
}

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (h Hello) MessageType() string {
	return HelloType
}
