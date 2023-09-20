package epp

import (
	"encoding/xml"
)

// Command is used to construct and receive <command> messages.
type Command struct {
	XMLName xml.Name `xml:"command" json:"-"`
	XMLNS   string   `xml:"xmlns,attr,omitempty" json:"xmlns"`

	LoginObject     *Login     `xml:"login" json:"login"`
	LogoutObject    *Logout    `xml:"logout" json:"logout"`
	CheckObject     *Check     `xml:"check" json:"check"`
	CreateObject    *Create    `xml:"create" json:"create"`
	InfoObject      *Info      `xml:"info" json:"info"`
	DeleteObject    *Delete    `xml:"delete" json:"delete"`
	UpdateObject    *Update    `xml:"update" json:"update"`
	TransferObject  *Transfer  `xml:"transfer" json:"transfer"`
	RenewObject     *Renew     `xml:"renew" json:"renew"`
	PollObject      *Poll      `xml:"poll" json:"poll"`
	ExtensionObject *Extension `xml:"extension" json:"extension"`

	TransactionID string `xml:"clTRID" json:"clTRID"`
}

// GetEPPCommand Returns an uninitialized EPP Command object.
func GetEPPCommand(TransactionID string) Epp {
	epp := GetEPP()
	epp.CommandObject = new(Command)
	epp.CommandObject.XMLNS = "urn:ietf:params:xml:ns:epp-1.0"
	epp.CommandObject.TransactionID = TransactionID

	return epp
}

const (
	// CommandType represents a <command> message.
	CommandType string = "epp.command"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (c *Command) MessageType() string {
	objType := CommandType
	count := 0

	if c.LoginObject != nil {
		count++

		objType = c.LoginObject.MessageType()
	}

	if c.CheckObject != nil {
		count++

		objType = c.CheckObject.MessageType()
	}

	if c.CreateObject != nil {
		count++

		objType = c.CreateObject.MessageType()
	}

	if c.InfoObject != nil {
		count++

		objType = c.InfoObject.MessageType()
	}

	if c.DeleteObject != nil {
		count++

		objType = c.DeleteObject.MessageType()
	}

	if c.UpdateObject != nil {
		count++

		objType = c.UpdateObject.MessageType()
	}

	if c.TransferObject != nil {
		count++

		objType = c.TransferObject.MessageType()
	}

	if c.RenewObject != nil {
		count++

		objType = c.RenewObject.MessageType()
	}

	if c.LogoutObject != nil {
		count++

		objType = c.LogoutObject.MessageType()
	}

	if c.PollObject != nil {
		count++

		objType = c.PollObject.MessageType()
	}

	if count > 1 {
		return CommandType
	}

	return objType
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (c Command) TypedMessage() Command {
	out := Command{}

	out.XMLName = c.XMLName
	out.XMLNS = c.XMLNS

	if c.LoginObject != nil {
		login := c.LoginObject.TypedMessage()
		out.LoginObject = &login
	}

	if c.LogoutObject != nil {
		logout := c.LogoutObject.TypedMessage()
		out.LogoutObject = &logout
	}

	if c.CheckObject != nil {
		check := c.CheckObject.TypedMessage()
		out.CheckObject = &check
	}

	if c.CreateObject != nil {
		create := c.CreateObject.TypedMessage()
		out.CreateObject = &create
	}

	if c.InfoObject != nil {
		info := c.InfoObject.TypedMessage()
		out.InfoObject = &info
	}

	if c.DeleteObject != nil {
		del := c.DeleteObject.TypedMessage()
		out.DeleteObject = &del
	}

	if c.UpdateObject != nil {
		update := c.UpdateObject.TypedMessage()
		out.UpdateObject = &update
	}

	if c.TransferObject != nil {
		transfer := c.TransferObject.TypedMessage()
		out.TransferObject = &transfer
	}

	if c.RenewObject != nil {
		renew := c.RenewObject.TypedMessage()
		out.RenewObject = &renew
	}

	// TODO
	if c.ExtensionObject != nil {
		extension := c.ExtensionObject.TypedMessage()
		out.ExtensionObject = &extension
	}

	out.TransactionID = c.TransactionID

	return out
}
