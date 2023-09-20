package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// DomainRevision represents individual versions of a Domain Object.
type DomainRevision struct {
	Model
	DomainID int64

	RevisionState string
	DesiredState  string

	DomainStatus string

	Owners string
	Class  string

	ClientDeleteProhibitedStatus   bool
	ServerDeleteProhibitedStatus   bool
	ClientHoldStatus               bool
	ServerHoldStatus               bool
	ClientRenewProhibitedStatus    bool
	ServerRenewProhibitedStatus    bool
	ClientTransferProhibitedStatus bool
	ServerTransferProhibitedStatus bool
	ClientUpdateProhibitedStatus   bool
	ServerUpdateProhibitedStatus   bool

	DomainRegistrant       Contact
	DomainRegistrantID     int64
	DomainAdminContact     Contact
	DomainAdminContactID   int64
	DomainTechContact      Contact
	DomainTechContactID    int64
	DomainBillingContact   Contact
	DomainBillingContactID int64
	Hostnames              []Host `gorm:"many2many:host_to_domainrevision"`

	DSDataEntries []DSDataEntry

	SavedNotes string `sql:"size:16384"`

	RequiredApproverSets []ApproverSet `gorm:"many2many:required_approverset_to_domainrevision"`
	InformedApproverSets []ApproverSet `gorm:"many2many:informed_approverset_to_domainrevision"`

	CR   ChangeRequest
	CRID sql.NullInt64

	IssueCR string `sql:"size:256"`
	Notes   string `sql:"size:2048"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`

	ApprovalStartTime  *time.Time
	ApprovalStartBy    string
	PromotedTime       *time.Time
	SupersededTime     *time.Time
	ApprovalFailedTime *time.Time
}

// DomainRevisionExport is an object that is used to export the
// current version of a Domain Revision.
type DomainRevisionExport struct {
	ID       int64 `json:"ID"`
	DomainID int64 `json:"DomainID"`

	RevisionState string `json:"RevisionState"`
	DesiredState  string `json:"DesiredState"`

	Owners string `json:"Owner"`
	Class  string `json:"Class"`

	ClientDeleteProhibitedStatus   bool `json:"ClientDeleteProhibitedStatus"`
	ServerDeleteProhibitedStatus   bool `json:"ServerDeleteProhibitedStatus"`
	ClientHoldStatus               bool `json:"ClientHoldStatus"`
	ServerHoldStatus               bool `json:"ServerHoldStatus"`
	ClientRenewProhibitedStatus    bool `json:"ClientRenewProhibitedStatus"`
	ServerRenewProhibitedStatus    bool `json:"ServerRenewProhibitedStatus"`
	ClientTransferProhibitedStatus bool `json:"ClientTransferProhibitedStatus"`
	ServerTransferProhibitedStatus bool `json:"ServerTransferProhibitedStatus"`
	ClientUpdateProhibitedStatus   bool `json:"ClientUpdateProhibitedStatus"`
	ServerUpdateProhibitedStatus   bool `json:"ServerUpdateProhibitedStatus"`

	DomainRegistrant     ContactExportShort `json:"DomainRegistrant"`
	DomainAdminContact   ContactExportShort `json:"DomainAdminContact"`
	DomainTechContact    ContactExportShort `json:"DomainTechContact"`
	DomainBillingContact ContactExportShort `json:"DomainBillingContact"`
	Hostnames            []HostExportShort  `json:"Hostnames"`

	DSDataEntries []DSDataEntry `json:"DSDataEntries"`

	SavedNotes string `json:"SavedNotes"`

	ChangeRequestID int64 `json:"ChangeRequestID"`

	IssueCR string `json:"IssueCR"`
	Notes   string `json:"Notes"`

	RequiredApproverSets []ApproverSetExportShort `json:"RequiredApproverSets"`
	InformedApproverSets []ApproverSetExportShort `json:"InformedApproverSets"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// Compare is used to compare an export version of an object to the
// full revision to verify that all of the values are the same.
func (dre DomainRevisionExport) Compare(domainRevision DomainRevision) (pass bool, errs []error) {
	pass = true

	if dre.ID != domainRevision.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if dre.DomainID != domainRevision.DomainID {
		errs = append(errs, fmt.Errorf("the ContactID fields did not match"))
		pass = false
	}

	if dre.DesiredState != domainRevision.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if dre.Owners != domainRevision.Owners {
		errs = append(errs, fmt.Errorf("the Owners fields did not match"))
		pass = false
	}

	if dre.Class != domainRevision.Class {
		errs = append(errs, fmt.Errorf("the Class fields did not match"))
	}

	if dre.DomainRegistrant.ID != domainRevision.DomainRegistrantID {
		errs = append(errs, fmt.Errorf("the DomainRegistrantID fields did not match"))
		pass = false
	}

	if dre.DomainAdminContact.ID != domainRevision.DomainAdminContactID {
		errs = append(errs, fmt.Errorf("the DomainAdminContactID fields did not match"))
		pass = false
	}

	if dre.DomainTechContact.ID != domainRevision.DomainTechContactID {
		errs = append(errs, fmt.Errorf("the DomainTechContactID fields did not match"))
		pass = false
	}

	if dre.DomainBillingContact.ID != domainRevision.DomainBillingContactID {
		errs = append(errs, fmt.Errorf("the DomainBillingContactID fields did not match"))
		pass = false
	}

	if dre.ClientDeleteProhibitedStatus != domainRevision.ClientDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ServerDeleteProhibitedStatus != domainRevision.ServerDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ClientTransferProhibitedStatus != domainRevision.ClientTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ServerTransferProhibitedStatus != domainRevision.ServerTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ClientUpdateProhibitedStatus != domainRevision.ClientUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ServerUpdateProhibitedStatus != domainRevision.ServerUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ClientRenewProhibitedStatus != domainRevision.ClientRenewProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientRenewProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ServerRenewProhibitedStatus != domainRevision.ServerRenewProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerRenewProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ClientHoldStatus != domainRevision.ClientHoldStatus {
		errs = append(errs, fmt.Errorf("the ClientHoldStatus fields did not match"))
		pass = false
	}

	if dre.ServerHoldStatus != domainRevision.ServerHoldStatus {
		errs = append(errs, fmt.Errorf("the ServerHoldStatus fields did not match"))
		pass = false
	}

	if dre.SavedNotes != domainRevision.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if dre.IssueCR != domainRevision.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if dre.Notes != domainRevision.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	hostnameCheck := CompareToHostListExportShortList(domainRevision.Hostnames, dre.Hostnames)
	if !hostnameCheck {
		errs = append(errs, fmt.Errorf("the hostnames did not match"))
		pass = false
	}

	dsDataEntriesCheck := CompareDSDataEntries(domainRevision.DSDataEntries, dre.DSDataEntries)
	if !dsDataEntriesCheck {
		errs = append(errs, fmt.Errorf("the DS Data Entries did not match"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetListToExportShort(domainRevision.RequiredApproverSets, dre.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetListToExportShort(domainRevision.InformedApproverSets, dre.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// CompareExport is used to compare an export version of an object to
// another export revision to verify that all of the values are the
// same.
func (dre DomainRevisionExport) CompareExport(domainRevision DomainRevisionExport) (pass bool, errs []error) {
	pass = true

	if dre.ID != domainRevision.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if dre.DomainID != domainRevision.DomainID {
		errs = append(errs, fmt.Errorf("the ContactID fields did not match"))
		pass = false
	}

	if dre.DesiredState != domainRevision.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if dre.Owners != domainRevision.Owners {
		errs = append(errs, fmt.Errorf("the Owners fields did not match"))
		pass = false
	}

	if dre.Class != domainRevision.Class {
		errs = append(errs, fmt.Errorf("the Class fields did not match"))
	}

	if dre.DomainRegistrant.ID != domainRevision.DomainRegistrant.ID {
		errs = append(errs, fmt.Errorf("the DomainRegistrantID fields did not match"))
		pass = false
	}

	if dre.DomainAdminContact.ID != domainRevision.DomainAdminContact.ID {
		errs = append(errs, fmt.Errorf("the DomainAdminContactID fields did not match"))
		pass = false
	}

	if dre.DomainTechContact.ID != domainRevision.DomainTechContact.ID {
		errs = append(errs, fmt.Errorf("the DomainTechContactID fields did not match"))
		pass = false
	}

	if dre.DomainBillingContact.ID != domainRevision.DomainBillingContact.ID {
		errs = append(errs, fmt.Errorf("the DomainBillingContactID fields did not match"))
		pass = false
	}

	if dre.ClientDeleteProhibitedStatus != domainRevision.ClientDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ServerDeleteProhibitedStatus != domainRevision.ServerDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ClientTransferProhibitedStatus != domainRevision.ClientTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ServerTransferProhibitedStatus != domainRevision.ServerTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ClientUpdateProhibitedStatus != domainRevision.ClientUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ServerUpdateProhibitedStatus != domainRevision.ServerUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ClientRenewProhibitedStatus != domainRevision.ClientRenewProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientRenewProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ServerRenewProhibitedStatus != domainRevision.ServerRenewProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerRenewProhibitedStatus fields did not match"))
		pass = false
	}

	if dre.ClientHoldStatus != domainRevision.ClientHoldStatus {
		errs = append(errs, fmt.Errorf("the ClientHoldStatus fields did not match"))
		pass = false
	}

	if dre.ServerHoldStatus != domainRevision.ServerHoldStatus {
		errs = append(errs, fmt.Errorf("the ServerHoldStatus fields did not match"))
		pass = false
	}

	if dre.SavedNotes != domainRevision.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if dre.IssueCR != domainRevision.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if dre.Notes != domainRevision.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	hostnameCheck := CompareToHostExportShortLists(domainRevision.Hostnames, dre.Hostnames)
	if !hostnameCheck {
		errs = append(errs, fmt.Errorf("the hostnames did not match"))
		pass = false
	}

	dsDataEntriesCheck := CompareDSDataEntries(domainRevision.DSDataEntries, dre.DSDataEntries)
	if !dsDataEntriesCheck {
		errs = append(errs, fmt.Errorf("the DS Data Entries did not match"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetExportShortLists(domainRevision.RequiredApproverSets, dre.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetExportShortLists(domainRevision.InformedApproverSets, dre.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// DomainRevisionPage are used to hold all the information required to
// render the DomainRevision HTML template.
type DomainRevisionPage struct {
	IsEditable                 bool
	IsNew                      bool
	Revision                   DomainRevision
	PendingActions             map[string]string
	ValidApproverSets          map[int64]string
	ValidHosts                 map[int64]string
	ValidContacts              map[int64]string
	ParentDomain               *Domain
	SuggestedRequiredApprovers map[int64]ApproverSetDisplayObject
	SuggestedInformedApprovers map[int64]ApproverSetDisplayObject
	SuggestedHostnames         []Host
	SuggestedDSData            []DSDataEntry
	DNSSECAlgorithms           map[int64]string
	DNSSECDigestTypes          map[int64]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (d *DomainRevisionPage) GetCSRFToken() string {
	return d.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (d *DomainRevisionPage) SetCSRFToken(newToken string) {
	d.CSRFToken = newToken
}

// The DomainRevisionsPage type is used to render the html template
// which lists all of the DomainRevisions currently in the
// registrar system
//
// TODO: Add paging support.
type DomainRevisionsPage struct {
	DomainRevisions []DomainRevision

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (d *DomainRevisionsPage) GetCSRFToken() string {
	return d.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (d *DomainRevisionsPage) SetCSRFToken(newToken string) {
	d.CSRFToken = newToken
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (dre DomainRevisionExport) ToJSON() (string, error) {
	if dre.ID <= 0 {
		return "", errors.New("invalid revision ID")
	}

	byteArr, jsonErr := json.MarshalIndent(dre, "", "  ")

	return string(byteArr), jsonErr
}

// GetDiff will return an empty string and an error for a revision. A
// Diff is not available for revision objects.
func (dre DomainRevisionExport) GetDiff() (string, error) {
	return "", errors.New("unable to diff a single revision")
}

// GetExportVersion returns a export version of the DomainRevision
// Object.
func (d *DomainRevision) GetExportVersion() RegistrarObjectExport {
	export := DomainRevisionExport{
		ID:                             d.ID,
		DomainID:                       d.DomainID,
		DesiredState:                   d.DesiredState,
		RevisionState:                  d.RevisionState,
		Owners:                         d.Owners,
		Class:                          d.Class,
		ClientDeleteProhibitedStatus:   d.ClientDeleteProhibitedStatus,
		ServerDeleteProhibitedStatus:   d.ServerDeleteProhibitedStatus,
		ClientHoldStatus:               d.ClientHoldStatus,
		ServerHoldStatus:               d.ServerHoldStatus,
		ClientRenewProhibitedStatus:    d.ClientRenewProhibitedStatus,
		ServerRenewProhibitedStatus:    d.ServerRenewProhibitedStatus,
		ClientTransferProhibitedStatus: d.ClientTransferProhibitedStatus,
		ServerTransferProhibitedStatus: d.ServerTransferProhibitedStatus,
		ClientUpdateProhibitedStatus:   d.ClientUpdateProhibitedStatus,
		ServerUpdateProhibitedStatus:   d.ServerUpdateProhibitedStatus,
		DSDataEntries:                  d.DSDataEntries,
		SavedNotes:                     d.SavedNotes,
		IssueCR:                        d.IssueCR,
		Notes:                          d.Notes,
		DomainRegistrant:               d.DomainRegistrant.GetExportVersionShort(),
		DomainAdminContact:             d.DomainAdminContact.GetExportVersionShort(),
		DomainTechContact:              d.DomainTechContact.GetExportVersionShort(),
		DomainBillingContact:           d.DomainBillingContact.GetExportVersionShort(),
		CreatedAt:                      d.CreatedAt,
		CreatedBy:                      d.CreatedBy,
	}

	for idx := range d.Hostnames {
		export.Hostnames = append(export.Hostnames, d.Hostnames[idx].GetExportShortVersion())
	}

	if d.CRID.Valid {
		export.ChangeRequestID = d.CRID.Int64
	} else {
		export.ChangeRequestID = -1
	}

	export.RequiredApproverSets = GetApproverSetExportArr(d.RequiredApproverSets)
	export.InformedApproverSets = GetApproverSetExportArr(d.InformedApproverSets)

	return export
}

// GetExportVersionAt returns an export version of the DomainRevision
// Object at the timestamp provided if possible otherwise an error is
// returned. If a pending version existed at the time it will be
// excluded from the object.
func (d *DomainRevision) GetExportVersionAt(_ *DBCache, _ int64) (obj RegistrarObjectExport, err error) {
	return obj, errors.New("GetExportVersionAt is not usable for revisions")
}

// IsDesiredState will return true iff the state passed in matches the
// desired state of the revision.
func (d DomainRevision) IsDesiredState(state string) bool {
	return d.DesiredState == state
}

// HasHappened is a helper function that will return true if the value
// of the timestamp who's name is passes has happened after the revision
// was created. This function is intended to be used with templates.
func (d DomainRevision) HasHappened(actionType string) bool {
	switch actionType {
	case EventUpdated:
		return !d.UpdatedAt.Before(d.CreatedAt)
	case EventApprovalStarted:
		if d.ApprovalStartTime != nil {
			return !d.ApprovalStartTime.Before(d.CreatedAt)
		}

		return false
	case EventApprovalFailed:
		if d.ApprovalFailedTime != nil {
			return !d.ApprovalFailedTime.Before(d.CreatedAt)
		}

		return false
	case EventPromoted:
		if d.PromotedTime != nil {
			return !d.PromotedTime.Before(d.CreatedAt)
		}

		return false
	case EventSuperseded:
		if d.SupersededTime != nil {
			return !d.SupersededTime.Before(d.CreatedAt)
		}

		return false
	default:
		logger.Errorf("Unknown actiontype: %s", actionType)

		return false
	}
}

// GetPage will return an object that can be used to render the HTML
// template for the DomainRevision.
func (d *DomainRevision) GetPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &DomainRevisionPage{IsNew: true}
	ret.IsEditable = d.IsEditable()
	ret.PendingActions = d.GetActions(true)

	if d.ID != 0 {
		ret.IsNew = false
	}

	ret.Revision = *d
	ret.ParentDomain = &Domain{}

	if err = dbCache.FindByID(ret.ParentDomain, d.DomainID); err != nil {
		return rop, err
	}

	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)
	if err != nil {
		return rop, err
	}

	hostMap, hostMapErr := GetValidHostMap(dbCache)
	if hostMapErr != nil {
		err = hostMapErr

		return rop, err
	}

	contactMap, contactMapErr := GetValidContactMap(dbCache)
	if contactMapErr != nil {
		err = contactMapErr

		return rop, err
	}

	ret.ValidHosts = hostMap
	ret.ValidContacts = contactMap
	ret.DNSSECAlgorithms = DNSSECAlgorithms
	ret.DNSSECDigestTypes = DNSSECDigestTypes

	return ret, nil
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the query into dbcache.
func (d *DomainRevision) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, d, func() (err error) {
		if err = dbCache.DB.Model(d).Related(&d.RequiredApproverSets, "RequiredApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return ErrExcessAPIKeysLocated
			}
		}

		if err = dbCache.DB.Model(d).Related(&d.InformedApproverSets, "InformedApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return err
			}
		}

		if err = dbCache.DB.Model(d).Related(&d.Hostnames, "Hostnames").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return err
			}
		}

		for idx := range d.RequiredApproverSets {
			if err = d.RequiredApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return err
			}
		}

		for idx := range d.InformedApproverSets {
			if err = d.InformedApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return err
			}
		}

		for idx := range d.Hostnames {
			if err = d.Hostnames[idx].PrepareDisplayShallow(dbCache); err != nil {
				return err
			}
		}

		d.DomainRegistrant = Contact{}
		if err = dbCache.FindByID(&d.DomainRegistrant, d.DomainRegistrantID); err != nil {
			return err
		}

		d.DomainAdminContact = Contact{}
		if err = dbCache.FindByID(&d.DomainAdminContact, d.DomainAdminContactID); err != nil {
			return err
		}

		d.DomainTechContact = Contact{}
		if err = dbCache.FindByID(&d.DomainTechContact, d.DomainTechContactID); err != nil {
			return err
		}

		d.DomainBillingContact = Contact{}
		if err = dbCache.FindByID(&d.DomainBillingContact, d.DomainBillingContactID); err != nil {
			return err
		}

		return dbCache.DB.Where("domain_revision_id = ?", d.ID).Find(&d.DSDataEntries).Error
	})
}

// DomainRevisionActionGOTOChangeRequest is the name of the action that
// will trigger a redirect to the current CR of an object.
const DomainRevisionActionGOTOChangeRequest string = "gotochangerequest"

// GetActions will return a list of possible actions that can be taken
// while in the current state
//
// TODO: handle all states.
func (d *DomainRevision) GetActions(isSelf bool) map[string]string {
	ret := make(map[string]string)

	if d.RevisionState == StateNew {
		ret["Start Approval Process"] = fmt.Sprintf("/action/%s/%d/startapproval", DomainRevisionType, d.ID)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", DomainRevisionType, d.ID)

		if isSelf {
			ret["View Parent Domain"] = fmt.Sprintf("/view/%s/%d", DomainType, d.DomainID)
		} else {
			ret["View/Edit Domain Revision"] = fmt.Sprintf("/view/%s/%d", DomainRevisionType, d.ID)
		}
	}

	if d.RevisionState == StatePendingApproval {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", DomainRevisionType, d.ID, DomainRevisionActionGOTOChangeRequest)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", DomainRevisionType, d.ID)

		if isSelf {
			ret["View Parent Domain"] = fmt.Sprintf("/view/%s/%d", DomainType, d.DomainID)
		}
	}

	if d.RevisionState == StateCancelled {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", DomainRevisionType, d.ID, DomainRevisionActionGOTOChangeRequest)

		if isSelf {
			ret["View Parent Domain"] = fmt.Sprintf("/view/%s/%d", DomainType, d.DomainID)
		}
	}

	return ret
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (d *DomainRevision) GetType() string {
	return DomainRevisionType
}

// IsCancelled returns true iff the object has been canclled.
func (d *DomainRevision) IsCancelled() bool {
	return d.RevisionState == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (d *DomainRevision) IsEditable() bool {
	return d.RevisionState == StateNew
}

// GetPreviewHostnames will generate and return the preview hostnames text for
// the associated with this domain revision.
func (d *DomainRevision) GetPreviewHostnames() string {
	ret := ""

	for _, hos := range d.Hostnames {
		ret += hos.HostName + "\n"
	}

	return ret
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple Domain Revisions.
//
// TODO: Add paging support
// TODO: Add filtering.
func (d *DomainRevision) GetAllPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &DomainRevisionsPage{}

	err = dbCache.FindAll(&ret.DomainRevisions)

	return ret, err
}

// Cancel will change the State of a revision from either "new" or
// "pendingapproval" to "cancelled"
//
// TODO: If in pending approval, cancel the change request and all
// approval objects
// TODO: Consider moving the query into dbcache.
func (d *DomainRevision) Cancel(dbCache *DBCache, conf Config) (errs []error) {
	if d.RevisionState == StateNew || d.RevisionState == StatePendingApproval {
		d.RevisionState = StateCancelled

		if err := dbCache.DB.Model(d).UpdateColumns(DomainRevision{RevisionState: StateCancelled}).Error; err != nil {
			errs = append(errs, err)

			return errs
		}

		err := dbCache.Purge(d)
		if err != nil {
			errs = append(errs, err)

			return errs
		}

		parent := Domain{}

		if err := dbCache.FindByID(&parent, d.DomainID); err != nil {
			errs = append(errs, err)

			return errs
		}

		_, parentErrs := parent.UpdateState(dbCache, conf)
		errs = append(errs, parentErrs...)

		if d.CRID.Valid {
			changeRequest := ChangeRequest{}

			if err := dbCache.FindByID(&changeRequest, d.CRID.Int64); err != nil {
				errs = append(errs, err)
			}

			_, crErrs := changeRequest.UpdateState(dbCache, conf)
			errs = append(errs, crErrs...)
		}

		return errs
	}

	errs = append(errs, errors.New("unable to cancel object not in 'new' or 'pending approval' state"))

	return errs
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (d *DomainRevision) TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, validCSRF bool, authMethod AuthType, conf Config) (errs []error) {
	switch actionName {
	case ActionCancel:
		if validCSRF {
			// TODO: Figure out who can cancel
			cancelErrs1 := d.Cancel(dbCache, conf)

			if cancelErrs1 != nil {
				errs = append(errs, cancelErrs1...)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", DomainRevisionType, d.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ActionStartApproval:
		if validCSRF {
			appErr := d.StartApprovalProcess(request, dbCache, conf)

			if appErr != nil {
				errs = append(errs, appErr)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", DomainRevisionType, d.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case DomainRevisionActionGOTOChangeRequest:
		if d.CRID.Valid {
			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ChangeRequestType, d.CRID.Int64), http.StatusFound)

			return errs
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(d.GetExportVersion()))

			return errs
		}
	default:
		errs = append(errs, fmt.Errorf("unknown action %s for %s", actionName, DomainRevisionType))

		return errs
	}

	errs = append(errs, errors.New("unable to take action"))

	return errs
}

// Promote will mark an DomainRevision as the current revision for an
// Domain if it has not been cancelled or failed approval.
func (d *DomainRevision) Promote(dbCache *DBCache) (err error) {
	if err = d.Prepare(dbCache); err != nil {
		return err
	}

	if d.RevisionState == StateCancelled || d.RevisionState == StateApprovalFailed {
		return errors.New("cannot promote revision in cancelled or approvalfailed state")
	}

	if d.CRID.Valid {
		changeRequest := ChangeRequest{}

		if err = dbCache.FindByID(&changeRequest, d.CRID.Int64); err != nil {
			return err
		}

		if changeRequest.State == StateApproved {
			d.RevisionState = d.DesiredState

			if d.PromotedTime == nil {
				d.PromotedTime = &time.Time{}
			}

			*d.PromotedTime = TimeNow()

			if err = dbCache.Save(d); err != nil {
				return err
			}
		} else {
			return errors.New("cannot promote revision which has not been approved")
		}
	} else {
		return errors.New("no Change Request has been created for this domain revision")
	}

	return err
}

// Supersed will mark an DomainRevision as a superseded revision for
// an Domain.
func (d *DomainRevision) Supersed(dbCache *DBCache) (err error) {
	if err = d.Prepare(dbCache); err != nil {
		return err
	}

	if d.RevisionState != StateActive && d.RevisionState != StateInactive && d.RevisionState != StateExternal && d.RevisionState != StateBootstrap {
		return fmt.Errorf("cannot supersed a revision not in active, inactive, external or bootstrap state (in %s)", d.RevisionState)
	}

	d.RevisionState = StateSuperseded
	if d.SupersededTime == nil {
		d.SupersededTime = &time.Time{}
	}

	*d.SupersededTime = TimeNow()

	return dbCache.Save(d)
}

// Decline will mark an DomainRevision as decline for an Domain.
func (d *DomainRevision) Decline(dbCache *DBCache) (err error) {
	if err = d.Prepare(dbCache); err != nil {
		return
	}

	if d.RevisionState != StatePendingApproval {
		return errors.New("only revisions in pendingapproval may be declined")
	}

	d.RevisionState = StateApprovalFailed

	if d.ApprovalFailedTime == nil {
		d.ApprovalFailedTime = &time.Time{}
	}

	*d.ApprovalFailedTime = TimeNow()

	return dbCache.Save(d)
}

// StartApprovalProcess creates a change request to start the process of
// approvnig a new Change Request. If the Change Request was created
// no error is returned, otherwise an error will be returned.
//
// TODO: Check if a CR already exists for this object
// TODO: Ensure that if an error occures no changes are made.
func (d *DomainRevision) StartApprovalProcess(request *http.Request, dbCache *DBCache, conf Config) (err error) {
	runame, ruerr := GetRemoteUser(request)
	if ruerr != nil {
		return errors.New("no username set")
	}

	logger.Infof("starting approval for ID: %d\n", d.ID)

	if d.RevisionState != StateNew {
		return fmt.Errorf("cannot start approval for %s %d, state is '%s' not 'new'", DomainRevisionType, d.ID, d.RevisionState)
	}

	if err = d.Prepare(dbCache); err != nil {
		return err
	}

	domain := Domain{}

	if err = dbCache.FindByID(&domain, d.DomainID); err != nil {
		return err
	}

	currev := domain.CurrentRevision
	logger.Debugf("Parent Domain ID: %d", domain.GetID())

	export := domain.GetExportVersion()

	changeRequestJSON, err1 := export.ToJSON()
	diff, err2 := export.GetDiff()

	if domain.PendingRevision.ID == domain.CurrentRevisionID.Int64 || err1 != nil || err2 != nil {
		return errors.New("unable to create diff for approval")
	}

	logger.Debugf("Diff Len: %d", len(diff))
	logger.Debugf("JSON Len: %d", len(changeRequestJSON))

	changeRequest := ChangeRequest{
		RegistrarObjectType: DomainType,
		RegistrarObjectID:   d.DomainID,
		ProposedRevisionID:  d.ID,
		State:               StateNew,
		InitialRevisionID:   domain.CurrentRevisionID,
		ChangeJSON:          changeRequestJSON,
		ChangeDiff:          diff,
		CreatedBy:           runame,
		UpdatedBy:           runame,
		UpdatedAt:           TimeNow(),
		CreatedAt:           TimeNow(),
	}

	if err = dbCache.Save(&changeRequest); err != nil {
		return err
	}

	logger.Debugf("Change Request ID: %d", changeRequest.GetID())

	if len(currev.RequiredApproverSets) == 0 {
		logger.Debug("No approver sets requied, defaulting to Approver Set 1")

		app := Approval{
			ChangeRequestID: changeRequest.ID,
			ApproverSetID:   1,
			State:           StateNew,
			CreatedBy:       runame,
			UpdatedBy:       runame,
			UpdatedAt:       TimeNow(),
			CreatedAt:       TimeNow(),
		}

		changeRequest.Approvals = append(changeRequest.Approvals, app)
	}

	logger.Debugf("%d Required Approver Sets", len(currev.RequiredApproverSets))

	for _, approverSet := range currev.RequiredApproverSets {
		app := Approval{
			ChangeRequestID: changeRequest.ID,
			ApproverSetID:   approverSet.ID,
			State:           StateNew,
			CreatedBy:       runame,
			UpdatedBy:       runame,
			UpdatedAt:       TimeNow(),
			CreatedAt:       TimeNow(),
		}

		changeRequest.Approvals = append(changeRequest.Approvals, app)
	}

	if err = dbCache.Save(&changeRequest); err != nil {
		return err
	}

	for _, approval := range changeRequest.Approvals {
		logger.Debugf("Approval %d created for Approver Set %d", approval.GetID(), approval.ApproverSetID)
	}

	d.CR = changeRequest

	if err = d.CRID.Scan(changeRequest.ID); err != nil {
		return fmt.Errorf("error scanning change request ID: %w", err)
	}

	d.RevisionState = StatePendingApproval
	d.UpdatedBy = runame
	d.UpdatedAt = TimeNow()

	if d.ApprovalStartTime == nil {
		d.ApprovalStartTime = &time.Time{}
	}

	*d.ApprovalStartTime = TimeNow()
	d.ApprovalStartBy = runame

	if err = dbCache.Save(d); err != nil {
		return err
	}

	domain = Domain{}

	if err = dbCache.FindByID(&domain, d.DomainID); err != nil {
		return err
	}

	if domain.CurrentRevisionID.Valid {
		if domain.CurrentRevision.DesiredState != StateBootstrap {
			if domain.CurrentRevision.DesiredState == StateActive {
				domain.State = StateActivePendingApproval
			} else if domain.CurrentRevision.DesiredState == StateInactive {
				domain.State = StateInactivePendingApproval
			} else if domain.CurrentRevision.DesiredState == StateExternal {
				domain.State = StateExternalPendingApproval
			}
		}
	} else {
		if domain.PendingRevision.DesiredState == StateActive || domain.PendingRevision.DesiredState == StateInactive {
			domain.State = StatePendingNew
			sendErr := d.NewDomainEmail(domain.DomainName, conf)

			if sendErr != nil {
				logger.Error(sendErr.Error())
			}
		} else if domain.PendingRevision.DesiredState == StateExternal {
			domain.State = StatePendingNewExternal
			sendErr := d.NewDomainEmail(domain.DomainName, conf)

			if sendErr != nil {
				logger.Error(sendErr.Error())
			}
		}
	}

	domain.UpdatedBy = runame
	domain.UpdatedAt = TimeNow()

	if err = dbCache.Save(&domain); err != nil {
		return err
	}

	_, errs := changeRequest.UpdateState(dbCache, conf)
	if len(errs) != 0 {
		return errs[0]
	}

	return nil
}

// NewDomainEmail will generate and send an email upon the creation of
// a new domain
// TODO: Move this email to a template and make the registrar name a variable.
func (d *DomainRevision) NewDomainEmail(domainName string, conf Config) error {
	subject := fmt.Sprintf("Registrar: New Domain Created - %s", domainName)
	message := fmt.Sprintf(`Hello,

This message is to inform you that a new domain has been created in the
registrar system. You can view the domain information at the link
below. If you feel that you should be part of one of the approver sets
associated with this domain please send a message to
%s or respond to this thread.

%s/view/domainrevision/%d

Thank you,
The registrar Admins
`, conf.Email.FromEmail, conf.Server.AppURL, d.ID)

	return conf.SendAllEmail(subject, message, []string{conf.Email.Announce})
}

// ErrDomainContactNotSet is returned when a domain contact is not set
// when parsing a domain revision update.
var ErrDomainContactNotSet = errors.New("all domain contacts must be set in each revision")

const (
	// DomainClassHighValue is used to represent domains that are
	// high value domains for a customer. These doamis are often registry
	// locked if supported.
	DomainClassHighValue string = "high-value"

	// DomainClassInUse is used to reprsent domains that are in use but not to
	// the level that require registry locks.
	DomainClassInUse string = "in-use"

	// DomainClassParked is used to represent the domain class for domains
	// that are owned but and parked.
	DomainClassParked string = "parked"

	// DomainClassOther is a pseudo class for domains that have classes
	// that are not defined above.
	DomainClassOther string = "other"
)

// IsSelectedClass determines if the selected class matches the argument
// passed. In the case of an "other" passed, the function will check to
// make sure that the class does not match any of the defined classes.
func (d DomainRevision) IsSelectedClass(class string) bool {
	if class == DomainClassOther {
		return d.Class != DomainClassHighValue &&
			d.Class != DomainClassInUse &&
			d.Class != DomainClassParked
	}

	return d.Class == class
}

// getSelectedDomainClass is used to pull the domain_class from a
// request object while checking that the class is of an acceptable
// value.
func getSelectedDomainClass(request *http.Request) (string, error) {
	selectedClass := request.FormValue("domain_class")
	otherClass := request.FormValue("domain_class_other")

	if len(otherClass) != 0 {
		if selectedClass != "other" {
			return "", fmt.Errorf("entering a domain class and selecting a domain class is not supported")
		}

		if otherClass == "other" {
			return "", fmt.Errorf("a domain class of %s is not supported", otherClass)
		}

		return otherClass, nil
	}

	switch selectedClass {
	case DomainClassHighValue:
		return DomainClassHighValue, nil
	case DomainClassInUse:
		return DomainClassInUse, nil
	case DomainClassParked:
		return DomainClassParked, nil
	case DomainClassOther:
		return "", errors.New("a domain class must be provided")
	default:
		return "", fmt.Errorf("a domain class of %s is not supported", selectedClass)
	}
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
//
// TODO: add status flags
// TODO: Handle domain current status (new, transfer in).
func (d *DomainRevision) ParseFromForm(request *http.Request, dbCache *DBCache) error {
	var err2, err3, err4 error

	domainID, err1 := strconv.ParseInt(request.FormValue("revision_domain_id"), 10, 64)

	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	d.CreatedBy = runame
	d.UpdatedBy = runame

	d.DomainID = domainID
	d.RevisionState = StateNew

	owners := request.FormValue("revision_owners")
	if len(owners) == 0 {
		return errors.New("an owner must be set in order to create a domain")
	}

	d.Owners = owners

	var classErr error

	d.Class, classErr = getSelectedDomainClass(request)
	if classErr != nil {
		return classErr
	}

	d.RequiredApproverSets, err2 = ParseApproverSets(request, dbCache, "approver_set_required_id", true)
	d.InformedApproverSets, err3 = ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

	d.DesiredState = GetActiveInactiveExternal(request.FormValue("revision_desiredstate"))

	d.Hostnames, err4 = ParseHostList(request, dbCache, "hostname")

	d.SavedNotes = request.FormValue("revision_saved_notes")

	d.IssueCR = request.FormValue("revision_issue_cr")
	d.Notes = request.FormValue("revision_notes")

	domainRegistrantID, err5 := strconv.ParseInt(request.FormValue("revision_registrant_contact"), 10, 64)
	domainAdminContactID, err6 := strconv.ParseInt(request.FormValue("revision_admin_contact"), 10, 64)
	domainTechContactID, err7 := strconv.ParseInt(request.FormValue("revision_tech_contact"), 10, 64)
	domainBillingContactID, err8 := strconv.ParseInt(request.FormValue("revision_billing_contact"), 10, 64)

	// GOREG-1: Adding check to make sure that the Domain Contacts is
	// set before continuing to populate the object
	if domainRegistrantID == 0 || domainAdminContactID == 0 ||
		domainTechContactID == 0 || domainBillingContactID == 0 {
		return ErrDomainContactNotSet
	}

	d.DomainRegistrantID = domainRegistrantID
	d.DomainAdminContactID = domainAdminContactID
	d.DomainTechContactID = domainTechContactID
	d.DomainBillingContactID = domainBillingContactID

	//
	// DomainStatus string
	//
	d.ClientDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_delete"))
	d.ServerDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_delete"))
	d.ClientHoldStatus = GetCheckboxState(request.FormValue("revision_client_hold"))
	d.ServerHoldStatus = GetCheckboxState(request.FormValue("revision_server_hold"))
	d.ClientRenewProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_renew"))
	d.ServerRenewProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_renew"))
	d.ClientTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_transfer"))
	d.ServerTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_transfer"))
	d.ClientUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_update"))
	d.ServerUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_update"))

	dsDataEntries, dsDataErrs := ParseDSDataEntries(request, dbCache, "ds_entry")

	if err1 != nil {
		return fmt.Errorf("unable to parse revision domain id: %w", err1)
	}

	if err2 != nil {
		return err2
	}

	if err3 != nil {
		return err3
	}

	if err4 != nil {
		return err4
	}

	if err5 != nil {
		return fmt.Errorf("unable to parse revision registrant contact id: %w", err5)
	}

	if err6 != nil {
		return fmt.Errorf("unable to parse revision admin contact id: %w", err6)
	}

	if err7 != nil {
		return fmt.Errorf("unable to parse revision tech contact id: %w", err7)
	}

	if err8 != nil {
		return fmt.Errorf("unable to parse revision billing contact id: %w", err8)
	}

	if len(dsDataErrs) != 0 {
		var errStrs []string

		for _, err := range dsDataErrs {
			errStrs = append(errStrs, err.Error())
		}

		return errors.New(strings.Join(errStrs, "\n"))
	}

	d.DSDataEntries = dsDataEntries

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse DomainRevision object with the changes that
// were made. An error object is always the second return value which
// is nil when no errors have occurred during parsing otherwise an error
// is returned.
//
// TODO: verify public key
// TODO: verify fingerprint
// TODO: Consider moving the query into dbcache.
func (d *DomainRevision) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, _ Config) (err error) {
	if d.ID > 0 {
		runame, err := GetRemoteUser(request)
		if err != nil {
			return err
		}

		if d.RevisionState == StateNew {
			d.UpdatedBy = runame

			d.RevisionState = StateNew

			RequiredApproverSets, err1 := ParseApproverSets(request, dbCache, "approver_set_required_id", true)
			InformedApproverSets, err2 := ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

			d.DesiredState = GetActiveInactiveExternal(request.FormValue("revision_desiredstate"))

			d.SavedNotes = request.FormValue("revision_saved_notes")

			owners := request.FormValue("revision_owners")
			if len(owners) == 0 {
				return errors.New("an owner must be set in order to create a domain")
			}

			d.Owners = owners

			var classErr error

			d.Class, classErr = getSelectedDomainClass(request)

			if classErr != nil {
				return classErr
			}

			Hostnames, err3 := ParseHostList(request, dbCache, "hostname")

			domainRegistrantID, err5 := strconv.ParseInt(request.FormValue("revision_registrant_contact"), 10, 64)
			domainAdminContactID, err6 := strconv.ParseInt(request.FormValue("revision_admin_contact"), 10, 64)
			domainTechContactID, err7 := strconv.ParseInt(request.FormValue("revision_tech_contact"), 10, 64)
			domainBillingContactID, err8 := strconv.ParseInt(request.FormValue("revision_billing_contact"), 10, 64)

			// GOREG-1: Adding check to make sure that the Domain Contacts is
			// set before continuing to populate the object
			if domainRegistrantID == 0 || domainAdminContactID == 0 ||
				domainTechContactID == 0 || domainBillingContactID == 0 {
				return ErrDomainContactNotSet
			}

			d.IssueCR = request.FormValue("revision_issue_cr")
			d.Notes = request.FormValue("revision_notes")

			d.DomainRegistrantID = domainRegistrantID
			d.DomainAdminContactID = domainAdminContactID
			d.DomainTechContactID = domainTechContactID
			d.DomainBillingContactID = domainBillingContactID

			d.ClientDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_delete"))
			d.ServerDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_delete"))
			d.ClientHoldStatus = GetCheckboxState(request.FormValue("revision_client_hold"))
			d.ServerHoldStatus = GetCheckboxState(request.FormValue("revision_server_hold"))
			d.ClientRenewProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_renew"))
			d.ServerRenewProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_renew"))
			d.ClientTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_transfer"))
			d.ServerTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_transfer"))
			d.ClientUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_update"))
			d.ServerUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_update"))

			dsDataEntries, dsDataErrs := ParseDSDataEntries(request, dbCache, "ds_entry")

			if err1 != nil {
				return err1
			}

			if err2 != nil {
				return err2
			}

			if err3 != nil {
				return err3
			}

			if err5 != nil {
				return fmt.Errorf("unable to parse revision registrant contact id: %w", err5)
			}

			if err6 != nil {
				return fmt.Errorf("unable to parse revision admin contact id: %w", err6)
			}

			if err7 != nil {
				return fmt.Errorf("unable to parse revision tech contact id: %w", err7)
			}

			if err8 != nil {
				return fmt.Errorf("unable to parse revision billing contact id: %w", err8)
			}

			if len(dsDataErrs) != 0 {
				var errStrs []string

				for _, err := range dsDataErrs {
					errStrs = append(errStrs, err.Error())
				}

				return errors.New(strings.Join(errStrs, "\n"))
			}

			d.DSDataEntries = dsDataEntries
			if err = dbCache.DB.Where("domain_revision_id = ?", d.ID).Delete(DSDataEntry{}).Error; err != nil {
				return err
			}

			updateAppSetErr := UpdateApproverSets(d, dbCache, "RequiredApproverSets", RequiredApproverSets)
			if updateAppSetErr != nil {
				return err
			}

			d.RequiredApproverSets = RequiredApproverSets

			updateAppSetErr = UpdateApproverSets(d, dbCache, "InformedApproverSets", InformedApproverSets)
			if updateAppSetErr != nil {
				return err
			}

			d.InformedApproverSets = InformedApproverSets

			updateHostsErr := UpdateHosts(d, dbCache, "Hostnames", Hostnames)
			if updateHostsErr != nil {
				return err
			}

			d.Hostnames = Hostnames
		} else {
			return fmt.Errorf("cannot update an object not in the %s state", d.RevisionState)
		}

		return nil
	}

	return errors.New("to update an domain revision the ID must be greater than 0")
}

// DSDataEntry is an object that will hold a single DS Data entry used
// to indicate how a domain is signed with DNSSEC.
type DSDataEntry struct {
	ID               int64 `gorm:"primary_key:yes"`
	DomainRevisionID int64
	KeyTag           int64
	Algorithm        int64
	DigestType       int64
	Digest           string `sql:"size:256"`
}

// ErrUnableToParseDSDataEntry defines the error returned when a DS Data
// entry is not able to be parsed.
var ErrUnableToParseDSDataEntry = errors.New("nnable to parse DS Data Entry")

const dsDataTokenLength = 4

// ParseFromFormValue parses a value from a HTML form into a DSDataEntry
// taking into account the encoding used by the web UI.
func (ds *DSDataEntry) ParseFromFormValue(input string) error {
	tokens := strings.Split(input, ":")

	if len(tokens) == dsDataTokenLength {
		var err error
		ds.KeyTag, err = strconv.ParseInt(tokens[0], 10, 64)

		// Make sure the KeyTag is in the correct range
		if err != nil {
			logger.Errorf("Key Tag Parse Error: %w", err)

			return ErrUnableToParseDSDataEntry
		}

		if ds.KeyTag <= 0 || ds.KeyTag > 65535 {
			logger.Error("Key Tag out of range")

			return ErrUnableToParseDSDataEntry
		}

		// Make sure that the Algorithm is valid
		ds.Algorithm, err = strconv.ParseInt(tokens[1], 10, 64)
		if err != nil {
			logger.Errorf("Algorithm Parse Error: %w", err)

			return ErrUnableToParseDSDataEntry
		}

		_, algoOK := DNSSECAlgorithms[ds.Algorithm]

		if !algoOK {
			logger.Error("Unknown DNSSEC Algorithm")

			return ErrUnableToParseDSDataEntry
		}

		// Make sure that the Digest Type is valid
		ds.DigestType, err = strconv.ParseInt(tokens[2], 10, 64)
		if err != nil {
			logger.Errorf("Digest Type Parse Error: %w", err)

			return ErrUnableToParseDSDataEntry
		}

		_, algoOK = DNSSECDigestTypes[ds.DigestType]
		if !algoOK {
			logger.Error("Unknown DNSSEC Digest Type")

			return ErrUnableToParseDSDataEntry
		}

		if len(tokens[3]) == 0 {
			logger.Error("No Digest Provided")

			return ErrUnableToParseDSDataEntry
		}

		re := regexp.MustCompile("^[a-zA-Z0-9]*$")

		if re.MatchString(tokens[3]) {
			ds.Digest = strings.ToUpper(tokens[3])
		} else {
			logger.Error("Invalid Digest")

			return ErrUnableToParseDSDataEntry
		}
	} else {
		logger.Error("Token Length Error")

		return ErrUnableToParseDSDataEntry
	}

	return nil
}

// DisplayName formates a DSData Entry to be displayed as part of a
// HTML form.
func (ds DSDataEntry) DisplayName() string {
	return ds.FormValue()
}

// FormValue will format the DSData Entry so it can be used as the
// value for a html form item.
func (ds DSDataEntry) FormValue() string {
	return fmt.Sprintf("%d:%d:%d:%s", ds.KeyTag, ds.Algorithm, ds.DigestType, ds.Digest)
}

// FormDivName creates a name that can be used as the ID for a div tag
// in the domain selection forms.
func (ds DSDataEntry) FormDivName() string {
	return fmt.Sprintf("%d-%d-%d-%s", ds.KeyTag, ds.Algorithm, ds.DigestType, ds.Digest)
}

// CompareDSDataEntries compares a list of DSDataEntries to another
// set of DSDataEntries that were from an export version of an object.
// If the counts match and the IDs for the DSDataEntries match true is
// returned otherwise, false is returned.
func CompareDSDataEntries(dse []DSDataEntry, dseo []DSDataEntry) bool {
	exportShortCount := len(dseo)
	fullCount := len(dse)

	if fullCount != exportShortCount {
		return false
	}

	for _, full := range dse {
		found := false

		for _, export := range dseo {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	for _, export := range dseo {
		found := false

		for _, full := range dse {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	return true
}

// ParseDSDataEntries takes a http Request, a database connection and
// the html ID of the ds data entry list to parse and will return an
// array of DSDataEntries that are represented in the http request. If
// an error occurs parsing any of the DS Data Entry a list of errors
// (one for each problem parsing) will be returned and the address will
// be excluded from the returned list.
func ParseDSDataEntries(request *http.Request, _ *DBCache, htmlID string) ([]DSDataEntry, []error) {
	var (
		dsDatas []DSDataEntry
		errs    []error
	)

	for _, dsDataEntry := range request.Form[htmlID] {
		dsData := &DSDataEntry{}
		dsDataEntryError := dsData.ParseFromFormValue(dsDataEntry)

		if dsDataEntryError != nil {
			errs = append(errs, fmt.Errorf("unable to parse ds data entry %w", dsDataEntryError))
		} else {
			dsDatas = append(dsDatas, *dsData)
		}
	}

	return dsDatas, errs
}

// UpdateDSDataEntries will update a set of DS Data Entries for a domain
// that is passed in to make the list reflect the list passed as the
// third parameter.
//
// TODO: Consider moving the query into dbcache.
func UpdateDSDataEntries(domainRevision *DomainRevision, dbCache *DBCache, association string, dsDataEntries []DSDataEntry) error {
	err := dbCache.DB.Model(domainRevision).Association(association).Clear().Error
	if err != nil {
		return err
	}

	for _, dsDataEntry := range dsDataEntries {
		err = dbCache.DB.Model(domainRevision).Association(association).Append(dsDataEntry).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// DNSSECAlgorithms is a map of DNSSEC algorithm type IDs to the
// algorithm name as defined by IANA
// Source: http://www.iana.org/assignments/dns-sec-alg-numbers/dns-sec-alg-numbers.xhtml
// IANA Page Last Updated: 2014-03-31.
var DNSSECAlgorithms = map[int64]string{
	// 0, "RESERVED_0", // RFC4034, RFC4398
	1: "RSAMD5", // RFC3110, RFC4034
	2: "DH",     // RFC2539
	3: "DSA",    // RFC3755
	// 4:"RESERVED_4", // RFC6725
	5: "RSASHA1",            // RFC3110, RFC4034
	6: "DSA-NSEC3-SHA1",     // RFC5155
	7: "RSASHA1-NSEC3-SHA1", // RFC5155
	8: "RSASHA256",          // RFC5155
	// 9:"RESERVED_9", // RFC6725
	10: "RSASHA512", // RFC5702
	// 11:"RESERVED_11", // RFC6725
	12: "ECC-GOST",        // RFC5933
	13: "ECDSAP256SHA256", // RFC6605
	14: "ECDSAP384SHA384", // RFC6605
	// 15-122: "UNASSIGNED_15-122"
	// 123-251: "RESERVED_123-251" // RFC4034 RFC6014
	252: "INDIRECT",   // RFC4034, RFC6014
	253: "PRIVATEDNS", // RFC4034
	254: "PRIVATEOID", // RFC4034
	// "RESERVED_255", 255 // RFC4034
}

// DNSSECDigestTypes is a map of DNSSEC digest types to the digest name
// as defined by IANA
// Source: http://www.iana.org/assignments/ds-rr-types/ds-rr-types.xhtml
// IANA Page Last Updated: 2012-04-13.
var DNSSECDigestTypes = map[int64]string{
	// 0: "RESERVED_0"    // RFC3658
	1: "SHA-1",           // RFC3658
	2: "SHA-256",         // RFC4509
	3: "GOST R 34.11-94", // RFC5933
	4: "SHA-384",         // RFC6605
	// 5-255: "UNASSIGNED_5-255"
}

// IsActive returns true if RevisionState is StateActive or StateBootstrap.
func (d *DomainRevision) IsActive() bool {
	return d.RevisionState == StateActive || d.RevisionState == StateBootstrap
}

// GetRequiredApproverSets prepares object and returns the ApproverSets.
func (d *DomainRevision) GetRequiredApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = d.Prepare(dbCache); err != nil {
		return
	}

	approverSets = d.RequiredApproverSets

	return
}

// GetInformedApproverSets prepares object and returns the ApproverSets.
func (d *DomainRevision) GetInformedApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = d.Prepare(dbCache); err != nil {
		return
	}

	approverSets = d.InformedApproverSets

	return
}

// MigrateDBDomainRevision will run the automigrate function for the
// DomainRevision object.
func MigrateDBDomainRevision(dbCache *DBCache) {
	dbCache.AutoMigrate(&DomainRevision{})
	dbCache.AutoMigrate(&DSDataEntry{})
	dbCache.DB.Model(&DSDataEntry{}).AddForeignKey("domain_revision_id", "domain_revisions(id)", "CASCADE", "RESTRICT")
}
