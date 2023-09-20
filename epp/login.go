package epp

import (
	"encoding/xml"
)

var (
	// EPPLoginVersion is the current version of the login object for EPP sessions.
	EPPLoginVersion = "1.0"

	// EPPLoginLanguage indicates the language of the EPP session, which defaults to
	// english.
	EPPLoginLanguage = "en"
)

// GetEPPLogin will create an EPP <login> message.
func GetEPPLogin(username string, password string, transactionID string, svcMenu ServiceMenu) Epp {
	login := Login{}

	login.ClientID = username
	login.Password = password

	login.LoginOptions.Version = EPPLoginVersion
	login.LoginOptions.Language = EPPLoginLanguage

	login.Services.URIs = svcMenu.URIs
	login.Services.ServiceExtensionsURIs = svcMenu.ServiceExtensionsURIs

	epp := GetEPPCommand(transactionID)
	epp.CommandObject.LoginObject = &login

	return epp
}

func GetEPPLoginPasswordChange(username string, password string, newPassword string, transactionID string, svcMenu ServiceMenu) Epp {
	login := Login{}

	login.ClientID = username
	login.Password = password
	login.NewPassword = newPassword

	login.LoginOptions.Version = EPPLoginVersion
	login.LoginOptions.Language = EPPLoginLanguage

	login.Services.URIs = svcMenu.URIs
	login.Services.ServiceExtensionsURIs = svcMenu.ServiceExtensionsURIs

	epp := GetEPPCommand(transactionID)
	epp.CommandObject.LoginObject = &login

	return epp
}

// Login is used to construct and receive <login> messages.
type Login struct {
	XMLName      xml.Name `xml:"login" json:"-"`
	ClientID     string   `xml:"clID" json:"clID"`
	Password     string   `xml:"pw" json:"pw"`
	NewPassword  string   `xml:"newPW,omitempty" json:"newPW"`
	LoginOptions struct {
		XMLName  xml.Name `xml:"options" json:"-"`
		Version  string   `xml:"version" json:"version"` // default 1.0
		Language string   `xml:"lang" json:"lang"`       // default en
	} `xml:"options" json:"options"`
	Services struct {
		XMLName               xml.Name `xml:"svcs" json:"-"`
		URIs                  []string `xml:"objURI" json:"objURI"`
		ServiceExtensionsURIs []string `xml:"svcExtension>extURI" json:"svcExtensions.extURI"`
	} `xml:"svcs" json:"svcs"`
}

const (
	// CommandLoginType represents a login command.
	CommandLoginType string = "epp.command.login"

	// CommandLogoutType represents a logout command.
	CommandLogoutType string = "epp.command.logout"
)

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (l Login) TypedMessage() Login {
	return l
}

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (l Login) MessageType() string {
	return CommandLoginType
}

// GetEPPLogout will create an EPP <logout> message.
func GetEPPLogout(transactionID string) Epp {
	logout := Logout{}

	epp := GetEPPCommand(transactionID)
	epp.CommandObject.LogoutObject = &logout

	return epp
}

// Logout is used to construct and receive <logout> messages.
type Logout struct {
	XMLName xml.Name `xml:"logout" json:"logout"`
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (l Logout) TypedMessage() Logout {
	return l
}

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (l Logout) MessageType() string {
	return CommandLogoutType
}
