package epp

import (
	"encoding/xml"
	"fmt"
	"time"
)

// SyncUpdateExtension is used to hold an Expire Month and Day value
// used for passing a SyncUpdate extension to the registry.
type SyncUpdateExtension struct {
	XMLName              xml.Name `xml:"sync:update" json:"-"`
	XMLNSSync            string   `xml:"xmlns:sync,attr" json:"xmlns.sync"`
	XMLNSxsi             string   `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	ExpMonthDay          string   `xml:"sync:expMonthDay" json:"sync.expMonthDay"`
}

// GetEPPDomainSyncUpdate constructs a SyncUpdate message for the domain
// passed attempting to set the expiration date to the month and Day
// passed.
func GetEPPDomainSyncUpdate(DomainName string, ExpMonth time.Month, ExpDay int, TransactionID string) Epp {
	epp := GetEPPDomainUpdate(DomainName, &DomainUpdateAddRemove{}, &DomainUpdateAddRemove{}, &DomainUpdateChange{}, TransactionID)
	su := &SyncUpdateExtension{}
	su.XMLNSSync = "http://www.verisign.com/epp/sync-1.0"
	su.XMLNSxsi = W3XMLNSxsi
	su.XMLxsiSchemaLocation = "http://www.verisign.com/epp/sync-1.0 sync-1.0.xsd"
	su.ExpMonthDay = fmt.Sprintf("--%02d-%02d", ExpMonth, ExpDay)
	epp.CommandObject.ExtensionObject.SyncUpdateObject = su

	return epp
}
