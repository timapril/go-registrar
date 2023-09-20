package epp

import (
	"encoding/xml"
)

// Poll is used to construct and receive <poll> messages.
type Poll struct {
	XMLName   xml.Name      `xml:"poll" json:"-"`
	MessageID string        `xml:"msgID,attr,omitempty" json:"msgID"`
	Operation PollOperation `xml:"op,attr" json:"op"`
}

// PollOperation is a type that represents the valid poll operations
// that may be requested.
type PollOperation string

const (
	// PollOperationRequest represents a poll request operation.
	PollOperationRequest PollOperation = "req"

	// PollOperationAcknowledge represents a poll acknowledge operation.
	PollOperationAcknowledge PollOperation = "ack"
)

// GetEPPPollRequest will create an epp <poll> command message with the
// operation type of request.
func GetEPPPollRequest(transactionID string) Epp {
	eppCmd := GetEPPCommand(transactionID)
	eppCmd.CommandObject.PollObject = &Poll{}
	eppCmd.CommandObject.PollObject.Operation = PollOperationRequest

	return eppCmd
}

// GetEPPPollAcknowledge will create an epp <poll> command message with
// the operation type of acknowledge.
func GetEPPPollAcknowledge(messageID string, transactionID string) Epp {
	eppCmd := GetEPPCommand(transactionID)
	eppCmd.CommandObject.PollObject = &Poll{}
	eppCmd.CommandObject.PollObject.Operation = PollOperationAcknowledge
	eppCmd.CommandObject.PollObject.MessageID = messageID

	return eppCmd
}

const (
	// PollType represents a <poll> command message.
	PollType string = "epp.command.poll"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (h Poll) MessageType() string {
	return PollType
}
