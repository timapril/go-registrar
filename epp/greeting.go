package epp

import (
	"encoding/xml"
)

// Greeting is used to construct and receive <greeting> messages.
type Greeting struct {
	XMLName    xml.Name    `xml:"greeting" json:"-"`
	ServerID   string      `xml:"svID" json:"svID"`
	ServerDate string      `xml:"svDate" json:"svDate"`
	SvcMenu    ServiceMenu `xml:"svcMenu" json:"svcMenu"`
	DCP        GreetingDCP `xml:"dcp" json:"dcp"`
}

// GreetingDCP is used to for the DCP section of a Greeting message.
type GreetingDCP struct {
	XMLName   xml.Name             `xml:"dcp" json:"-"`
	Access    GreetingDCPAccess    `xml:"access" json:"access"`
	Statement GreetingDCPStateemnt `xml:"statement" json:"statement"`
}

// GreetingDCPAccess represents the access section of a Greeting>DCP
// message.
type GreetingDCPAccess struct {
	XMLName xml.Name `xml:"access" json:"-"`
	All     string   `xml:"all" json:"all"`
}

// GreetingDCPStateemnt represents the statement section of a
// Greeting>DCP message.
type GreetingDCPStateemnt struct {
	XMLName   xml.Name                      `xml:"statement" json:"-"`
	Purpose   GreetingDCPStateemntPurpose   `xml:"purpose" json:"purpose"`
	Recipient GreetingDCPStateemntRecipient `xml:"recipient" json:"recipient"`
	Retention GreetingDCPStateemntRetention `xml:"retention" json:"retention"`
}

// GreetingDCPStateemntPurpose represents the purpose section of a
// Greeting>DCP>Statement message.
type GreetingDCPStateemntPurpose struct {
	XMLName xml.Name `xml:"purpose" json:"-"`
	Admin   string   `xml:"admin" json:"admin"`
	Prov    string   `xml:"prov" json:"prov"`
}

// GreetingDCPStateemntRecipient represents the Recipient section of a
// Greeting>DCP>Statement message.
type GreetingDCPStateemntRecipient struct {
	XMLName xml.Name `xml:"recipient" json:"-"`
	Ours    string   `xml:"ours" json:"ours"`
	Public  string   `xml:"public" json:"public"`
}

// GreetingDCPStateemntRetention represents the Retention section of a
// Greeting>DCP>Statement message.
type GreetingDCPStateemntRetention struct {
	XMLName xml.Name `xml:"retention" json:"-"`
	Stated  string   `xml:"stated" json:"started"`
}

const (
	// GreetingType represents a <greeting> message.
	GreetingType string = "epp.greeting"
)

// GetEPPGreeting will create an EPP <greeting> message.
func GetEPPGreeting(svcMenu ServiceMenu) Epp {
	greeting := Greeting{}

	greeting.SvcMenu = svcMenu

	epp := GetEPP()
	epp.GreetingObject = &greeting

	return epp
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (g Greeting) TypedMessage() Greeting {
	return g
}

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (g Greeting) MessageType() string {
	return GreetingType
}
