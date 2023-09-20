package epp

import (
	"encoding/xml"
)

// SecDNSUpdate is used to represent an update for the DSData for a domain.
type SecDNSUpdate struct {
	XMLName              xml.Name  `xml:"secDNS:update" json:"-"`
	XMLNSSecDNS          string    `xml:"xmlns:secDNS,attr" json:"xmlns.secDNS"`
	XMLNSxsi             string    `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string    `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	RemDS                *[]DSData `xml:"secDNS:rem>secDNS:dsData,omitempty" json:"secDNS.rem.secDNS.dsData"`
	AddDS                *[]DSData `xml:"secDNS:add>secDNS:dsData,omitempty" json:"secDNS.add.secDNS.dsData"`
}

// GetEPPDomainSecDNSUpdate constructs a SecDNS update message for the domain
// passed attempting to add or remove DS records for the domain.
func GetEPPDomainSecDNSUpdate(DomainName string, DStoAdd, DStoRemove []DSData, TransactionID string) Epp {
	epp := GetEPPDomainUpdate(DomainName, &DomainUpdateAddRemove{}, &DomainUpdateAddRemove{}, &DomainUpdateChange{}, TransactionID)
	dsUpdate := &SecDNSUpdate{}
	dsUpdate.XMLNSSecDNS = "urn:ietf:params:xml:ns:secDNS-1.1"
	dsUpdate.XMLNSxsi = "http://www.w3.org/2001/XMLSchema-instance"
	dsUpdate.XMLxsiSchemaLocation = "urn:ietf:params:xml:ns:secDNS-1.1 secDNS-1.1.xsd"

	if len(DStoAdd) != 0 {
		dsUpdate.AddDS = &DStoAdd
	}

	if len(DStoRemove) != 0 {
		dsUpdate.RemDS = &DStoRemove
	}

	epp.CommandObject.ExtensionObject.SecDNSUpdate = dsUpdate

	return epp
}
