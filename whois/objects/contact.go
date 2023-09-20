package objects

import (
	"errors"

	"github.com/timapril/go-registrar/lib"
)

// Contact holds the data required to generate a WHOIS response.
type Contact struct {
	ID int64

	Name string
	Org  string

	AddressStreet1    string
	AddressStreet2    string
	AddressStreet3    string
	AddressCity       string
	AddressState      string
	AddressPostalCode string
	AddressCountry    string

	VoicePhoneNumber    string
	VoicePhoneExtension string
	FaxPhoneNumber      string
	FaxPhoneExtension   string

	EmailAddress string
}

// WHOISContactFromExport takes a lib.ContactExport object and extracts
// the current values that are required to display the valid WHOIS
// information.
func WHOISContactFromExport(libcontact *lib.ContactExport) (Contact, error) {
	c := Contact{}

	c.ID = libcontact.ID

	currentRevision := libcontact.CurrentRevision
	if currentRevision.ID == 0 {
		return c, errors.New("Unable to find current revision")
	}
	// Basic Org Info
	c.Name = currentRevision.Name
	c.Org = currentRevision.Org

	// Address
	c.AddressStreet1 = currentRevision.AddressStreet1
	c.AddressStreet2 = currentRevision.AddressStreet2
	c.AddressStreet3 = currentRevision.AddressStreet3
	c.AddressCity = currentRevision.AddressCity
	c.AddressState = currentRevision.AddressState
	c.AddressPostalCode = currentRevision.AddressPostalCode
	c.AddressCountry = currentRevision.AddressCountry

	// Phone Number
	c.VoicePhoneNumber = currentRevision.VoicePhoneNumber
	c.VoicePhoneExtension = currentRevision.VoicePhoneExtension

	// Fax Number
	c.FaxPhoneNumber = currentRevision.FaxPhoneNumber
	c.FaxPhoneExtension = currentRevision.FaxPhoneExtension

	// Email
	c.EmailAddress = currentRevision.EmailAddress

	return c, nil
}
