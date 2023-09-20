package objects

import (
	"errors"

	"github.com/timapril/go-registrar/lib"
)

// Domain holds the data required to generate a WHOIS response and
// the IDs of the linked objects.
type Domain struct {
	ID int64

	DomainName string
	DomainROID string

	DomainCreationDate string
	DomainUpdateDate   string

	DomainStatuses []string

	RegistrantContactID int64
	AdminContactID      int64
	TechContactID       int64

	HostIDs []int64
}

// WHOISDomainFromExport takes a lib.DomainExport object and extracts
// the current values that are required to display the valid WHOIS
// information.
func WHOISDomainFromExport(libdomain *lib.DomainExport) (Domain, error) {
	d := Domain{}

	d.ID = libdomain.ID
	d.DomainName = libdomain.DomainName
	d.DomainROID = libdomain.DomainROID

	currentRevision := libdomain.CurrentRevision

	// if the current revision is 0 there is something wrong. No data
	// should be set
	if currentRevision.ID == 0 {
		return d, errors.New("Unable to find current revision")
	}
	// Domain Contacts
	d.RegistrantContactID = currentRevision.DomainRegistrant.ID
	d.AdminContactID = currentRevision.DomainAdminContact.ID
	d.TechContactID = currentRevision.DomainTechContact.ID

	// Host Objects
	for _, host := range currentRevision.Hostnames {
		d.HostIDs = append(d.HostIDs, host.ID)
	}

	// Status Flags
	if currentRevision.ClientDeleteProhibitedStatus {
		d.DomainStatuses = append(d.DomainStatuses, "clientDeleteProhibited")
	}

	if currentRevision.ServerDeleteProhibitedStatus {
		d.DomainStatuses = append(d.DomainStatuses, "serverDeleteProhibited")
	}

	if currentRevision.ClientUpdateProhibitedStatus {
		d.DomainStatuses = append(d.DomainStatuses, "clientUpdateProhibited")
	}

	if currentRevision.ServerUpdateProhibitedStatus {
		d.DomainStatuses = append(d.DomainStatuses, "serverUpdateProhibited")
	}

	if currentRevision.ClientTransferProhibitedStatus {
		d.DomainStatuses = append(d.DomainStatuses, "clientTransferProhibited")
	}

	if currentRevision.ServerTransferProhibitedStatus {
		d.DomainStatuses = append(d.DomainStatuses, "serverTransferProhibited")
	}

	if currentRevision.ClientRenewProhibitedStatus {
		d.DomainStatuses = append(d.DomainStatuses, "clientRenewProhibited")
	}

	if currentRevision.ServerRenewProhibitedStatus {
		d.DomainStatuses = append(d.DomainStatuses, "serverRenewProhibited")
	}

	if currentRevision.ClientHoldStatus {
		d.DomainStatuses = append(d.DomainStatuses, "clientHold")
	}

	if currentRevision.ServerHoldStatus {
		d.DomainStatuses = append(d.DomainStatuses, "serverHold")
	}

	// If no statuses are set the domain status is OK
	if len(d.DomainStatuses) == 0 {
		d.DomainStatuses = append(d.DomainStatuses, "ok")
	}

	return d, nil
}
