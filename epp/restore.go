package epp

import (
	"encoding/xml"
	"time"
)

// RestoreExtension is used to generate a Restore extension.
type RestoreExtension struct {
	XMLName              xml.Name         `xml:"rgp:update" json:"-"`
	XMLNSRgp             string           `xml:"xmlns:rgp,attr" json:"xmlns.rgp"`
	XMLNSxsi             string           `xml:"xmlns:xsi,attr" json:"xmlns.xsi"`
	XMLxsiSchemaLocation string           `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Operation            RestoreOperation `xml:"rgp:restore" json:"rgp.restore"`
}

// RestoreOperation is used to indicate which operation should be taken
// by the server and can pass the report to the registry.
type RestoreOperation struct {
	XMLName   xml.Name             `xml:"rgp:restore" json:"-"`
	Operation RestoreOperationType `xml:"op,attr" json:"operation"`
	Report    *RestoreReport       `xml:",omitempty" json:"report"`
}

// RestoreReport is used to pass required data back to the registry when
// reporting a domain restoration.
type RestoreReport struct {
	XMLName            xml.Name `xml:"rgp:report" json:"-"`
	PreWHOISData       string   `xml:"rgp:preData" json:"rgp.preData"`
	PostWHOISData      string   `xml:"rgp:postData" json:"rgp.postData"`
	DomainDeletedTime  string   `xml:"rgp:delTime" json:"rgp.delTime"`
	DomainRestoredTime string   `xml:"rgp:resTime" json:"rgp.resTime"`
	Reason             string   `xml:"rgp:resReason" json:"rgp.resReason"`
	Statements         []string `xml:"rgp:statement" json:"rgp.statement"`
	Other              string   `xml:"rgp:other" json:"rgp.other"`
}

// SetDeleteTime is used to set the timestamp that the Domain was
// deleted in the required format.
func (r *RestoreReport) SetDeleteTime(t time.Time) {
	r.DomainDeletedTime = t.Format(EPPTimeFormat)
}

// SetRestoreTime is used to set the timestamp that the Domain was
// restored in the required format.
func (r *RestoreReport) SetRestoreTime(t time.Time) {
	r.DomainRestoredTime = t.Format(EPPTimeFormat)
}

// RestoreOperationType is used to indicate which type of restore
// operation should be taken by the server.
type RestoreOperationType string

const (
	// RestoreOperationRequest is used to indicate that a restore request
	// is being sent to the server.
	RestoreOperationRequest RestoreOperationType = "request"

	// RestoreOperationReport is used to indicate that a restore report
	// is being sent to the server.
	RestoreOperationReport RestoreOperationType = "report"
)

// GetEPPDomainRestoreRequest constructs a Restore Report message for
// the domain passed.
func GetEPPDomainRestoreRequest(DomainName string, TransactionID string) Epp {
	epp := GetEPPDomainUpdate(DomainName, &DomainUpdateAddRemove{}, &DomainUpdateAddRemove{}, &DomainUpdateChange{}, TransactionID)
	rre := &RestoreExtension{}
	rre.XMLNSRgp = "urn:ietf:params:xml:ns:rgp-1.0"
	rre.XMLNSxsi = W3XMLNSxsi
	rre.XMLxsiSchemaLocation = "urn:ietf:params:xml:ns:rgp-1.0 rgp-1.0.xsd"
	rre.Operation.Operation = RestoreOperationRequest

	epp.CommandObject.ExtensionObject.RestoreRequest = rre

	return epp
}

// GetEPPDomainRestoreReport constructs a Restore Report message for the
// domain passed with the required report data.
func GetEPPDomainRestoreReport(DomainName string, Report *RestoreReport, TransactionID string) Epp {
	epp := GetEPPDomainUpdate(DomainName, &DomainUpdateAddRemove{}, &DomainUpdateAddRemove{}, &DomainUpdateChange{}, TransactionID)
	rre := &RestoreExtension{}
	rre.XMLNSRgp = "urn:ietf:params:xml:ns:rgp-1.0"
	rre.XMLNSxsi = W3XMLNSxsi
	rre.XMLxsiSchemaLocation = "urn:ietf:params:xml:ns:rgp-1.0 rgp-1.0.xsd"
	rre.Operation.Operation = RestoreOperationReport
	rre.Operation.Report = Report
	epp.CommandObject.ExtensionObject.RestoreRequest = rre

	return epp
}
