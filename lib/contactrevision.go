package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// ContactRevision represents individual versions of a Contact Object.
type ContactRevision struct {
	Model
	ContactID int64

	RevisionState string
	DesiredState  string

	ContactStatus                  string
	ClientDeleteProhibitedStatus   bool
	ServerDeleteProhibitedStatus   bool
	ClientTransferProhibitedStatus bool
	ServerTransferProhibitedStatus bool
	ClientUpdateProhibitedStatus   bool
	ServerUpdateProhibitedStatus   bool

	Name string `sql:"size:128"`
	Org  string `sql:"size:128"`

	AddressStreet1    string `sql:"size:128"`
	AddressStreet2    string `sql:"size:128"`
	AddressStreet3    string `sql:"size:128"`
	AddressCity       string `sql:"size:64"`
	AddressState      string `sql:"size:32"`
	AddressPostalCode string `sql:"size:32"`
	AddressCountry    string `sql:"size:3"`

	VoicePhoneNumber    string `sql:"size:32"`
	VoicePhoneExtension string `sql:"size:32"`
	FaxPhoneNumber      string `sql:"size:32"`
	FaxPhoneExtension   string `sql:"size:32"`

	EmailAddress string `sql:"size:128"`

	SavedNotes string `sql:"size:16384"`

	RequiredApproverSets []ApproverSet `gorm:"many2many:required_approverset_to_contactrevision"`
	InformedApproverSets []ApproverSet `gorm:"many2many:informed_approverset_to_contactrevision"`

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

// ContactRevisionExport is an object that is used to export the
// current version of a Contact Revision.
type ContactRevisionExport struct {
	ID        int64 `json:"ID"`
	ContactID int64 `json:"ContactID"`

	RevisionState string `json:"RevisionState"`
	DesiredState  string `json:"DesiredState"`

	ClientDeleteProhibitedStatus   bool `json:"ClientDeleteProhibitedStatus"`
	ServerDeleteProhibitedStatus   bool `json:"ServerDeleteProhibitedStatus"`
	ClientTransferProhibitedStatus bool `json:"ClientTransferProhibitedStatus"`
	ServerTransferProhibitedStatus bool `json:"ServerTransferProhibitedStatus"`
	ClientUpdateProhibitedStatus   bool `json:"ClientUpdateProhibitedStatus"`
	ServerUpdateProhibitedStatus   bool `json:"ServerUpdateProhibitedStatus"`

	Name string `json:"Name"`
	Org  string `json:"Org"`

	AddressStreet1    string `json:"AddressStreet1"`
	AddressStreet2    string `json:"AddressStreet2"`
	AddressStreet3    string `json:"AddressStreet3"`
	AddressCity       string `json:"AddressCity"`
	AddressState      string `json:"AddressState"`
	AddressPostalCode string `json:"AddressPostalCode"`
	AddressCountry    string `json:"AddressCountry"`

	VoicePhoneNumber    string `json:"VoicePhoneNumber"`
	VoicePhoneExtension string `json:"VoicePhoneExtension"`
	FaxPhoneNumber      string `json:"FaxPhoneNumber"`
	FaxPhoneExtension   string `json:"FaxPhoneExtension"`

	EmailAddress string `json:"EmailAddress"`

	SavedNotes string `json:"SaveNotes"`

	ChangeRequestID int64 `json:"ChangeRequestID"`

	IssueCR string `json:"IssuerCR"`
	Notes   string `json:"Notes"`

	RequiredApproverSets []ApproverSetExportShort `json:"RequiredApproverSets"`
	InformedApproverSets []ApproverSetExportShort `json:"InformedApproverSets"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// Compare is used to compare an export version of an object to the
// full revision to verify that all of the values are the same.
func (cre ContactRevisionExport) Compare(contactRevision ContactRevision) (pass bool, errs []error) {
	pass = true

	if cre.ID != contactRevision.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if cre.ContactID != contactRevision.ContactID {
		errs = append(errs, fmt.Errorf("the ContactID fields did not match"))
		pass = false
	}

	if cre.DesiredState != contactRevision.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if cre.Name != contactRevision.Name {
		errs = append(errs, fmt.Errorf("the Name fields did not match"))
		pass = false
	}

	if cre.Org != contactRevision.Org {
		errs = append(errs, fmt.Errorf("the Org fields did not match"))
		pass = false
	}

	if cre.EmailAddress != contactRevision.EmailAddress {
		errs = append(errs, fmt.Errorf("the EmailAddress fields did not match"))
		pass = false
	}

	if cre.AddressStreet1 != contactRevision.AddressStreet1 {
		errs = append(errs, fmt.Errorf("the AddressStreet1 fields did not match"))
		pass = false
	}

	if cre.AddressStreet2 != contactRevision.AddressStreet2 {
		errs = append(errs, fmt.Errorf("the AddressStreet2 fields did not match"))
		pass = false
	}

	if cre.AddressStreet3 != contactRevision.AddressStreet3 {
		errs = append(errs, fmt.Errorf("the AddressStreet3 fields did not match"))
		pass = false
	}

	if cre.AddressCity != contactRevision.AddressCity {
		errs = append(errs, fmt.Errorf("the AddressCity fields did not match"))
		pass = false
	}

	if cre.AddressState != contactRevision.AddressState {
		errs = append(errs, fmt.Errorf("the AddressState fields did not match"))
		pass = false
	}

	if cre.AddressPostalCode != contactRevision.AddressPostalCode {
		errs = append(errs, fmt.Errorf("the AddressPostalCode fields did not match"))
		pass = false
	}

	if cre.AddressCountry != contactRevision.AddressCountry {
		errs = append(errs, fmt.Errorf("the AddressCountry fields did not match"))
		pass = false
	}

	if cre.VoicePhoneNumber != contactRevision.VoicePhoneNumber {
		errs = append(errs, fmt.Errorf("the VoicePhoneNumber fields did not match"))
		pass = false
	}

	if cre.VoicePhoneExtension != contactRevision.VoicePhoneExtension {
		errs = append(errs, fmt.Errorf("the VoicePhoneExtension fields did not match"))
		pass = false
	}

	if cre.FaxPhoneNumber != contactRevision.FaxPhoneNumber {
		errs = append(errs, fmt.Errorf("the FaxPhoneNumber fields did not match"))
		pass = false
	}

	if cre.FaxPhoneExtension != contactRevision.FaxPhoneExtension {
		errs = append(errs, fmt.Errorf("the FaxPhoneExtension fields did not match"))
		pass = false
	}

	if cre.SavedNotes != contactRevision.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if cre.ClientDeleteProhibitedStatus != contactRevision.ClientDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ServerDeleteProhibitedStatus != contactRevision.ServerDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ClientTransferProhibitedStatus != contactRevision.ClientTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ServerTransferProhibitedStatus != contactRevision.ServerTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ClientUpdateProhibitedStatus != contactRevision.ClientUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ServerUpdateProhibitedStatus != contactRevision.ServerUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.IssueCR != contactRevision.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if cre.Notes != contactRevision.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetListToExportShort(contactRevision.RequiredApproverSets, cre.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetListToExportShort(contactRevision.InformedApproverSets, cre.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// CompareExport is used to compare an export version of an object to
// another export revision to verify that all of the values are the
// same.
func (cre ContactRevisionExport) CompareExport(contactRevision ContactRevisionExport) (pass bool, errs []error) {
	pass = true

	if cre.ID != contactRevision.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if cre.ContactID != contactRevision.ContactID {
		errs = append(errs, fmt.Errorf("the ContactID fields did not match"))
		pass = false
	}

	if cre.DesiredState != contactRevision.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if cre.Name != contactRevision.Name {
		errs = append(errs, fmt.Errorf("the Name fields did not match"))
		pass = false
	}

	if cre.Org != contactRevision.Org {
		errs = append(errs, fmt.Errorf("the Org fields did not match"))
		pass = false
	}

	if cre.EmailAddress != contactRevision.EmailAddress {
		errs = append(errs, fmt.Errorf("the EmailAddress fields did not match"))
		pass = false
	}

	if cre.AddressStreet1 != contactRevision.AddressStreet1 {
		errs = append(errs, fmt.Errorf("the AddressStreet1 fields did not match"))
		pass = false
	}

	if cre.AddressStreet2 != contactRevision.AddressStreet2 {
		errs = append(errs, fmt.Errorf("the AddressStreet2 fields did not match"))
		pass = false
	}

	if cre.AddressStreet3 != contactRevision.AddressStreet3 {
		errs = append(errs, fmt.Errorf("the AddressStreet3 fields did not match"))
		pass = false
	}

	if cre.AddressCity != contactRevision.AddressCity {
		errs = append(errs, fmt.Errorf("the AddressCity fields did not match"))
		pass = false
	}

	if cre.AddressState != contactRevision.AddressState {
		errs = append(errs, fmt.Errorf("the AddressState fields did not match"))
		pass = false
	}

	if cre.AddressPostalCode != contactRevision.AddressPostalCode {
		errs = append(errs, fmt.Errorf("the AddressPostalCode fields did not match"))
		pass = false
	}

	if cre.AddressCountry != contactRevision.AddressCountry {
		errs = append(errs, fmt.Errorf("the AddressCountry fields did not match"))
		pass = false
	}

	if cre.VoicePhoneNumber != contactRevision.VoicePhoneNumber {
		errs = append(errs, fmt.Errorf("the VoicePhoneNumber fields did not match"))
		pass = false
	}

	if cre.VoicePhoneExtension != contactRevision.VoicePhoneExtension {
		errs = append(errs, fmt.Errorf("the VoicePhoneExtension fields did not match"))
		pass = false
	}

	if cre.FaxPhoneNumber != contactRevision.FaxPhoneNumber {
		errs = append(errs, fmt.Errorf("the FaxPhoneNumber fields did not match"))
		pass = false
	}

	if cre.FaxPhoneExtension != contactRevision.FaxPhoneExtension {
		errs = append(errs, fmt.Errorf("the FaxPhoneExtension fields did not match"))
		pass = false
	}

	if cre.SavedNotes != contactRevision.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if cre.ClientDeleteProhibitedStatus != contactRevision.ClientDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ServerDeleteProhibitedStatus != contactRevision.ServerDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ClientTransferProhibitedStatus != contactRevision.ClientTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ServerTransferProhibitedStatus != contactRevision.ServerTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ClientUpdateProhibitedStatus != contactRevision.ClientUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.ServerUpdateProhibitedStatus != contactRevision.ServerUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if cre.IssueCR != contactRevision.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if cre.Notes != contactRevision.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetExportShortLists(contactRevision.RequiredApproverSets, cre.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetExportShortLists(contactRevision.InformedApproverSets, cre.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// VoiceNumber will return the phone number and extension in one string.
func (cre ContactRevisionExport) VoiceNumber() string {
	return formatNumberForDisplay(cre.VoicePhoneNumber, cre.VoicePhoneExtension)
}

// FaxNumber will return the fax number and extension in one string.
func (cre ContactRevisionExport) FaxNumber() string {
	return formatNumberForDisplay(cre.FaxPhoneNumber, cre.FaxPhoneExtension)
}

// formatNumberForDisplay will format a phone number into the number and
// extension for display.
func formatNumberForDisplay(number, extension string) string {
	if len(extension) == 0 {
		return number
	}

	return fmt.Sprintf("%sx%s", number, extension)
}

// EscrowAddress is used to format the mailing address into a single line that
// can be used for the RDE escrow.
func (cre ContactRevisionExport) EscrowAddress() string {
	addr := ""
	if len(cre.AddressStreet1) > 0 {
		addr = fmt.Sprintf("%s%s, ", addr, cre.AddressStreet1)
	}

	if len(cre.AddressStreet2) > 0 {
		addr = fmt.Sprintf("%s%s, ", addr, cre.AddressStreet2)
	}

	if len(cre.AddressStreet3) > 0 {
		addr = fmt.Sprintf("%s%s, ", addr, cre.AddressStreet3)
	}

	if len(cre.AddressCity) > 0 || len(cre.AddressState) > 0 || len(cre.AddressPostalCode) > 0 {
		var line string

		if len(cre.AddressCity) > 0 && len(cre.AddressState) > 0 {
			line = fmt.Sprintf("%s, %s", cre.AddressCity, cre.AddressState)
		} else {
			line = fmt.Sprintf("%s%s", cre.AddressCity, cre.AddressState)
		}

		if len(cre.AddressCity) > 0 || len(cre.AddressState) > 0 {
			line = fmt.Sprintf("%s %s", line, cre.AddressPostalCode)
		} else {
			line = fmt.Sprint(cre.AddressPostalCode)
		}

		addr = fmt.Sprintf("%s%s, ", addr, line)
	}

	if len(cre.AddressCountry) > 0 {
		addr = fmt.Sprintf("%s%s, ", addr, cre.AddressCountry)
	}

	if len(addr) > 0 {
		return strings.TrimSpace(strings.Trim(strings.TrimSpace(addr), ","))
	}

	return addr
}

// ContactRevisionPage are used to hold all the information required to
// render the ContactRevision HTML template.
type ContactRevisionPage struct {
	IsEditable                 bool
	IsNew                      bool
	Revision                   ContactRevision
	PendingActions             map[string]string
	ValidApproverSets          map[int64]string
	ParentContact              *Contact
	SuggestedRequiredApprovers map[int64]ApproverSetDisplayObject
	SuggestedInformedApprovers map[int64]ApproverSetDisplayObject

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (c *ContactRevisionPage) GetCSRFToken() string {
	return c.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (c *ContactRevisionPage) SetCSRFToken(newToken string) {
	c.CSRFToken = newToken
}

// The ContactRevisionsPage type is used to render the html template
// which lists all of the ContactRevisions currently in the
// registrar system
//
// TODO: Add paging support.
type ContactRevisionsPage struct {
	ContactRevisions []ContactRevision

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (c *ContactRevisionsPage) GetCSRFToken() string {
	return c.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (c *ContactRevisionsPage) SetCSRFToken(newToken string) {
	c.CSRFToken = newToken
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (cre ContactRevisionExport) ToJSON() (string, error) {
	if cre.ID <= 0 {
		return "", errors.New("invalid revision ID")
	}

	byteArr, jsonErr := json.MarshalIndent(cre, "", "  ")

	return string(byteArr), jsonErr
}

// GetDiff will return an empty string and an error for a revision. A
// Diff is not available for revision objects.
func (cre ContactRevisionExport) GetDiff() (string, error) {
	return "", errors.New("unable to diff a single revision")
}

// GetExportVersion returns a export version of the ContactRevision
// Object.
func (c *ContactRevision) GetExportVersion() RegistrarObjectExport {
	export := ContactRevisionExport{
		ID:                             c.ID,
		ContactID:                      c.ContactID,
		DesiredState:                   c.DesiredState,
		RevisionState:                  c.RevisionState,
		ClientDeleteProhibitedStatus:   c.ClientDeleteProhibitedStatus,
		ServerDeleteProhibitedStatus:   c.ServerDeleteProhibitedStatus,
		ClientTransferProhibitedStatus: c.ClientTransferProhibitedStatus,
		ServerTransferProhibitedStatus: c.ServerTransferProhibitedStatus,
		ClientUpdateProhibitedStatus:   c.ClientUpdateProhibitedStatus,
		ServerUpdateProhibitedStatus:   c.ServerUpdateProhibitedStatus,
		Name:                           c.Name,
		Org:                            c.Org,
		AddressStreet1:                 c.AddressStreet1,
		AddressStreet2:                 c.AddressStreet2,
		AddressStreet3:                 c.AddressStreet3,
		AddressCity:                    c.AddressCity,
		AddressState:                   c.AddressState,
		AddressPostalCode:              c.AddressPostalCode,
		AddressCountry:                 c.AddressCountry,
		VoicePhoneNumber:               c.VoicePhoneNumber,
		VoicePhoneExtension:            c.VoicePhoneExtension,
		FaxPhoneNumber:                 c.FaxPhoneNumber,
		FaxPhoneExtension:              c.FaxPhoneExtension,
		EmailAddress:                   c.EmailAddress,
		SavedNotes:                     c.SavedNotes,
		IssueCR:                        c.IssueCR,
		Notes:                          c.Notes,
		CreatedAt:                      c.CreatedAt,
		CreatedBy:                      c.CreatedBy,
	}
	export.RequiredApproverSets = GetApproverSetExportArr(c.RequiredApproverSets)
	export.InformedApproverSets = GetApproverSetExportArr(c.InformedApproverSets)

	if c.CRID.Valid {
		export.ChangeRequestID = c.CRID.Int64
	} else {
		export.ChangeRequestID = -1
	}

	return export
}

// GetExportVersionAt returns an export version of the Contact Revision
// Object at the timestamp provided if possible otherwise an error is
// returned. If a pending version existed at the time it will be
// excluded from the object.
func (c *ContactRevision) GetExportVersionAt(_ *DBCache, _ int64) (obj RegistrarObjectExport, err error) {
	return obj, errors.New("GetExportVersionAt is not usable for revisions")
}

// IsDesiredState will return true iff the state passed in matches the
// desired state of the revision.
func (c ContactRevision) IsDesiredState(state string) bool {
	return c.DesiredState == state
}

// HasHappened is a helper function that will return true if the value
// of the timestamp who's name is passes has happened after the revision
// was created. This function is intended to be used with templates.
func (c ContactRevision) HasHappened(actionType string) bool {
	switch actionType {
	case EventUpdated:
		return !c.UpdatedAt.Before(c.CreatedAt)
	case EventApprovalStarted:
		if c.ApprovalStartTime != nil {
			return !c.ApprovalStartTime.Before(c.CreatedAt)
		}

		return false
	case EventApprovalFailed:
		if c.ApprovalFailedTime != nil {
			return !c.ApprovalFailedTime.Before(c.CreatedAt)
		}

		return false
	case EventPromoted:
		if c.PromotedTime != nil {
			return !c.PromotedTime.Before(c.CreatedAt)
		}

		return false
	case EventSuperseded:
		if c.SupersededTime != nil {
			return !c.SupersededTime.Before(c.CreatedAt)
		}

		return false
	default:
		logger.Errorf("Unknown actiontype: %s", actionType)

		return false
	}
}

// GetPage will return an object that can be used to render the HTML
// template for the ContactRevision.
func (c *ContactRevision) GetPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &ContactRevisionPage{IsNew: true}
	ret.IsEditable = c.IsEditable()
	ret.PendingActions = c.GetActions(true)

	if c.ID != 0 {
		ret.IsNew = false
	}

	ret.Revision = *c
	ret.ParentContact = &Contact{}

	if err = dbCache.FindByID(ret.ParentContact, c.ContactID); err != nil {
		return
	}

	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	return ret, err
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the db query into the dbcache object.
func (c *ContactRevision) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, c, func() (err error) {
		if err = dbCache.DB.Model(c).Related(&c.RequiredApproverSets, "RequiredApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return
			}
		}
		if err = dbCache.DB.Model(c).Related(&c.InformedApproverSets, "InformedApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return
			}
		}
		for idx := range c.RequiredApproverSets {
			if err = c.RequiredApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return
			}
		}
		for idx := range c.InformedApproverSets {
			if err = c.InformedApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return
			}
		}

		return
	})
}

// PrepareShallow populate all of the fields for a given object and
// not any of the linked object.
func (c *ContactRevision) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, c, FuncNOPErrFunc)
}

// ContactRevisionActionGOTOChangeRequest is the name of the action that
// will trigger a redirect to the current CR of an object.
const ContactRevisionActionGOTOChangeRequest string = "gotochangerequest"

// GetActions will return a list of possible actions that can be taken
// while in the current state
//
// TODO: handle all states.
func (c *ContactRevision) GetActions(isSelf bool) map[string]string {
	ret := make(map[string]string)

	if c.RevisionState == StateNew {
		ret["Start Approval Process"] = fmt.Sprintf("/action/%s/%d/startapproval", ContactRevisionType, c.ID)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", ContactRevisionType, c.ID)

		if isSelf {
			ret["View Parent Contact"] = fmt.Sprintf("/view/%s/%d", ContactType, c.ContactID)
		} else {
			ret["View/Edit Contact Revision"] = fmt.Sprintf("/view/%s/%d", ContactRevisionType, c.ID)
		}
	}

	if c.RevisionState == StatePendingApproval {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", ContactRevisionType, c.ID, ContactRevisionActionGOTOChangeRequest)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", ContactRevisionType, c.ID)

		if isSelf {
			ret["View Parent Contact"] = fmt.Sprintf("/view/%s/%d", ContactType, c.ContactID)
		}
	}

	if c.RevisionState == StateCancelled {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", ContactRevisionType, c.ID, ContactRevisionActionGOTOChangeRequest)

		if isSelf {
			ret["View Parent Contact"] = fmt.Sprintf("/view/%s/%d", ContactType, c.ContactID)
		}
	}

	return ret
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (c *ContactRevision) GetType() string {
	return ContactRevisionType
}

// IsCancelled returns true iff the object has been canclled.
func (c *ContactRevision) IsCancelled() bool {
	return c.RevisionState == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (c *ContactRevision) IsEditable() bool {
	return c.RevisionState == StateNew
}

// GetPreviewName will generate and return the preview name text for the
// associated with this contact revision.
func (c *ContactRevision) GetPreviewName() string {
	return c.Name + "\n" + c.Org
}

// GetPreviewAddress will generate and return the preview address text for the
// associated with this contact revision.
func (c *ContactRevision) GetPreviewAddress() string {
	return c.FullAddress()
}

// GetPreviewPhone will generate and return the preview phone number text for
// the associated with this contact revision.
func (c *ContactRevision) GetPreviewPhone() string {
	return fmt.Sprintf("Phone: %s %s\nFax: %s %s", c.VoicePhoneNumber, c.VoicePhoneExtension, c.FaxPhoneNumber, c.FaxPhoneExtension)
}

// GetPreviewEmail will generate and return the preview email address text for
// the associated with this contact revision.
func (c *ContactRevision) GetPreviewEmail() string {
	return c.EmailAddress
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple Contact Revisions.
//
// TODO: Add paging support
// TODO: Add filtering.
func (c *ContactRevision) GetAllPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &ContactRevisionsPage{}

	err = dbCache.FindAll(&ret.ContactRevisions)

	return ret, err
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
//
// TODO: add status flags
// TODO: Handle domain current status (new, transfer in).
func (c *ContactRevision) ParseFromForm(request *http.Request, dbCache *DBCache) error {
	var err2, err3 error

	conID, err1 := strconv.ParseInt(request.FormValue("revision_contact_id"), 10, 64)

	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	c.CreatedBy = runame
	c.UpdatedBy = runame

	c.ContactID = conID
	c.RevisionState = StateNew

	c.Name = request.FormValue("revision_name")
	c.Org = request.FormValue("revision_org")
	c.AddressStreet1 = request.FormValue("revision_addess_street_1")
	c.AddressStreet2 = request.FormValue("revision_addess_street_2")
	c.AddressStreet3 = request.FormValue("revision_addess_street_3")
	c.AddressCity = request.FormValue("revision_address_city")
	c.AddressState = request.FormValue("revision_address_state")
	c.AddressPostalCode = request.FormValue("revision_address_postal_code")
	c.AddressCountry = request.FormValue("revision_address_country")

	c.VoicePhoneNumber = request.FormValue("revision_voice_phone_number")
	c.VoicePhoneExtension = request.FormValue("revision_voice_phone_extension")
	c.FaxPhoneNumber = request.FormValue("revision_fax_phone_number")
	c.FaxPhoneExtension = request.FormValue("revision_fax_phone_extension")

	c.EmailAddress = request.FormValue("revision_email")
	c.SavedNotes = request.FormValue("revision_saved_notes")

	c.IssueCR = request.FormValue("revision_issue_cr")
	c.Notes = request.FormValue("revision_notes")

	c.RequiredApproverSets, err2 = ParseApproverSets(request, dbCache, "approver_set_required_id", true)
	c.InformedApproverSets, err3 = ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

	c.DesiredState = GetActiveInactive(request.FormValue("revision_desiredstate"))

	//
	// DomainStatus string
	//
	c.ClientDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_delete"))
	c.ServerDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_delete"))
	c.ClientTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_transfer"))
	c.ServerTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_transfer"))
	c.ClientUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_update"))
	c.ServerUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_update"))

	if err1 != nil {
		return fmt.Errorf("unable to parse revision contact id: %w", err1)
	}

	if err2 != nil {
		return err2
	}

	if err3 != nil {
		return err3
	}

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse ContactRevision object with the changes that
// were made. An error object is always the second return value which
// is nil when no errors have occurred during parsing otherwise an error
// is returned.
//
// TODO: verify public key
// TODO: verify fingerprint.
func (c *ContactRevision) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, _ Config) error {
	if c.ID > 0 {
		runame, err := GetRemoteUser(request)
		if err != nil {
			return err
		}

		if c.RevisionState == StateNew {
			c.UpdatedBy = runame

			c.RevisionState = StateNew

			c.Name = request.FormValue("revision_name")
			c.Org = request.FormValue("revision_org")
			c.AddressStreet1 = request.FormValue("revision_addess_street_1")
			c.AddressStreet2 = request.FormValue("revision_addess_street_2")
			c.AddressStreet3 = request.FormValue("revision_addess_street_3")
			c.AddressCity = request.FormValue("revision_address_city")
			c.AddressState = request.FormValue("revision_address_state")
			c.AddressPostalCode = request.FormValue("revision_address_postal_code")
			c.AddressCountry = request.FormValue("revision_address_country")

			c.VoicePhoneNumber = request.FormValue("revision_voice_phone_number")
			c.VoicePhoneExtension = request.FormValue("revision_voice_phone_extension")
			c.FaxPhoneNumber = request.FormValue("revision_fax_phone_number")
			c.FaxPhoneExtension = request.FormValue("revision_fax_phone_extension")

			c.EmailAddress = request.FormValue("revision_email")
			c.SavedNotes = request.FormValue("revision_saved_notes")

			c.IssueCR = request.FormValue("revision_issue_cr")
			c.Notes = request.FormValue("revision_notes")

			RequiredApproverSets, err1 := ParseApproverSets(request, dbCache, "approver_set_required_id", true)
			InformedApproverSets, err2 := ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

			c.DesiredState = GetActiveInactive(request.FormValue("revision_desiredstate"))

			c.ClientDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_delete"))
			c.ServerDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_delete"))
			c.ClientTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_transfer"))
			c.ServerTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_transfer"))
			c.ClientUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_update"))
			c.ServerUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_update"))

			if err1 != nil {
				return err1
			}

			if err2 != nil {
				return err2
			}

			updateAppSetErr := UpdateApproverSets(c, dbCache, "RequiredApproverSets", RequiredApproverSets)
			if updateAppSetErr != nil {
				return err
			}

			c.RequiredApproverSets = RequiredApproverSets

			updateAppSetErr = UpdateApproverSets(c, dbCache, "InformedApproverSets", InformedApproverSets)
			if updateAppSetErr != nil {
				return err
			}

			c.InformedApproverSets = InformedApproverSets
		} else {
			return fmt.Errorf("cannot update an object not in the %s state", c.RevisionState)
		}

		return nil
	}

	return errors.New("to update an contact revision the ID must be greater than 0")
}

// Cancel will change the State of a revision from either "new" or
// "pendingapproval" to "cancelled"
//
// TODO: If in pending approval, cancel the change request and all
// approval objects
// TODO: Consider moving the db query into the dbcache object.
func (c *ContactRevision) Cancel(dbCache *DBCache, conf Config) (errs []error) {
	if c.RevisionState == StateNew || c.RevisionState == StatePendingApproval {
		c.RevisionState = StateCancelled

		if err := dbCache.DB.Model(c).UpdateColumns(ContactRevision{RevisionState: StateCancelled}).Error; err != nil {
			errs = append(errs, err)

			return errs
		}

		err := dbCache.Purge(c)
		if err != nil {
			errs = append(errs, err)

			return errs
		}

		parent := Contact{}

		if err := dbCache.FindByID(&parent, c.ContactID); err != nil {
			errs = append(errs, err)

			return errs
		}

		_, parentErrs := parent.UpdateState(dbCache, conf)
		errs = append(errs, parentErrs...)

		if c.CRID.Valid {
			changeRequest := ChangeRequest{}
			if err := dbCache.FindByID(&changeRequest, c.CRID.Int64); err != nil {
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

// StartApprovalProcess creates a change request to start the process of
// approvnig a new Change Request. If the Change Request was created
// no error is returned, otherwise an error will be returned.
//
// TODO: Check if a CR already exists for this object
// TODO: Ensure that if an error occures no changes are made.
func (c *ContactRevision) StartApprovalProcess(request *http.Request, dbCache *DBCache, conf Config) (err error) {
	runame, ruerr := GetRemoteUser(request)
	if ruerr != nil {
		return errors.New("no username set")
	}

	logger.Infof("starting approval for ID: %d\n", c.ID)

	if c.RevisionState != StateNew {
		return fmt.Errorf("cannot start approval for %s %d, state is '%s' not 'new'", ContactRevisionType, c.ID, c.RevisionState)
	}

	if err = c.Prepare(dbCache); err != nil {
		return err
	}

	contact := Contact{}

	if err = dbCache.FindByID(&contact, c.ContactID); err != nil {
		return err
	}

	currev := contact.CurrentRevision

	logger.Debugf("Parent Contact ID: %d", contact.GetID())

	export := contact.GetExportVersion()

	changeRequestJSON, err1 := export.ToJSON()
	diff, err2 := export.GetDiff()

	if contact.PendingRevision.ID == contact.CurrentRevisionID.Int64 || err1 != nil || err2 != nil {
		return errors.New("unable to create diff for approval")
	}

	logger.Debugf("Diff Len: %d", len(diff))
	logger.Debugf("JSON Len: %d", len(changeRequestJSON))

	changeRequest := ChangeRequest{
		RegistrarObjectType: ContactType,
		RegistrarObjectID:   c.ContactID,
		ProposedRevisionID:  c.ID,
		State:               StateNew,
		InitialRevisionID:   contact.CurrentRevisionID,
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
			State:           StateNew, CreatedBy: runame, UpdatedBy: runame,
			UpdatedAt: TimeNow(), CreatedAt: TimeNow(),
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

	c.CR = changeRequest

	if err = c.CRID.Scan(changeRequest.ID); err != nil {
		return fmt.Errorf("unable to scan change request ID: %w", err)
	}

	c.RevisionState = StatePendingApproval
	c.UpdatedBy = runame
	c.UpdatedAt = TimeNow()

	if c.ApprovalStartTime == nil {
		c.ApprovalStartTime = &time.Time{}
	}

	*c.ApprovalStartTime = TimeNow()
	c.ApprovalStartBy = runame

	if err = dbCache.Save(c); err != nil {
		return err
	}

	contact = Contact{}

	if err = dbCache.FindByID(&contact, c.ContactID); err != nil {
		return err
	}

	if contact.CurrentRevisionID.Valid {
		if contact.CurrentRevision.DesiredState != StateBootstrap {
			if contact.CurrentRevision.DesiredState == StateActive {
				contact.State = StateActivePendingApproval
			} else {
				contact.State = StateInactivePendingApproval
			}
		}
	} else {
		contact.State = StatePendingNew
		sendErr := c.NewContactEmail(c.Name, conf)
		if sendErr != nil {
			logger.Error(sendErr.Error())
		}
	}

	contact.UpdatedBy = runame
	contact.UpdatedAt = TimeNow()

	if err = dbCache.Save(&contact); err != nil {
		return err
	}

	_, errs := changeRequest.UpdateState(dbCache, conf)
	if len(errs) != 0 {
		return errs[0]
	}

	return nil
}

// NewContactEmail will generate and send an email upon the creation of
// a new contact
// TODO: make this email a template with the registrar name a variable.
func (c *ContactRevision) NewContactEmail(contactName string, conf Config) error {
	subject := fmt.Sprintf("Registrar: New Contact Created - %s", contactName)
	message := fmt.Sprintf(`Hello,

This message is to inform you that a new contact has been created in the
registrar system. You can view the contact information at the link
below. If you feel that you should be part of one of the approver sets
associated with this contact please send a message to
%s or respond to this thread.

%s/view/contactrevision/%d

Thank you,
The registrar Admins
`, conf.Email.FromEmail, conf.Server.AppURL, c.ID)

	return conf.SendAllEmail(subject, message, []string{conf.Email.Announce})
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (c *ContactRevision) TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, validCSRF bool, authMethod AuthType, conf Config) (errs []error) {
	switch actionName {
	case ActionCancel:
		if validCSRF {
			// TODO: Figure out who can cancel
			cancelErrs1 := c.Cancel(dbCache, conf)
			if cancelErrs1 != nil {
				errs = append(errs, cancelErrs1...)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ContactRevisionType, c.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ActionStartApproval:
		if validCSRF {
			appErr := c.StartApprovalProcess(request, dbCache, conf)
			if appErr != nil {
				errs = append(errs, appErr)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ContactRevisionType, c.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ContactRevisionActionGOTOChangeRequest:
		if c.CRID.Valid {
			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ChangeRequestType, c.CRID.Int64), http.StatusFound)

			return errs
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(c.GetExportVersion()))

			return errs
		}
	default:
		errs = append(errs, fmt.Errorf("unknown action %s for %s", actionName, ContactRevisionType))

		return errs
	}

	errs = append(errs, errors.New("unable to take action"))

	return errs
}

// Promote will mark an ContactRevision as the current revision for an
// Contact if it has not been cancelled or failed approval.
func (c *ContactRevision) Promote(ddbCache *DBCache) (err error) {
	if err = c.Prepare(ddbCache); err != nil {
		return err
	}

	if c.RevisionState == StateCancelled || c.RevisionState == StateApprovalFailed {
		return errors.New("cannot promote revision in cancelled or approvalfailed state")
	}

	if c.CRID.Valid {
		changeRequest := ChangeRequest{}
		if err = ddbCache.FindByID(&changeRequest, c.CRID.Int64); err != nil {
			return err
		}

		if changeRequest.State == StateApproved {
			c.RevisionState = c.DesiredState

			if c.PromotedTime == nil {
				c.PromotedTime = &time.Time{}
			}

			*c.PromotedTime = TimeNow()

			if err = ddbCache.Save(c); err != nil {
				return err
			}
		} else {
			return errors.New("cannot promote revision which has not been approved")
		}
	} else {
		return errors.New("no Change Request has been created for this approver revision")
	}

	return err
}

// Supersed will mark an ContactRevision as a superseded revision for
// an Contact.
func (c *ContactRevision) Supersed(dbCache *DBCache) (err error) {
	if err = c.Prepare(dbCache); err != nil {
		return err
	}

	if c.RevisionState != StateActive && c.RevisionState != StateInactive && c.RevisionState != StateBootstrap {
		return fmt.Errorf("cannot supersed a revision not in active, inactive or bootstrap state (in %s)", c.RevisionState)
	}

	c.RevisionState = StateSuperseded

	if c.SupersededTime == nil {
		c.SupersededTime = &time.Time{}
	}

	*c.SupersededTime = TimeNow()

	return dbCache.Save(c)
}

// Decline will mark an ContactRevision as decline for an Contact.
func (c *ContactRevision) Decline(dbCache *DBCache) (err error) {
	if err = c.Prepare(dbCache); err != nil {
		return err
	}

	if c.RevisionState != StatePendingApproval {
		return errors.New("only revisions in pendingapproval may be declined")
	}

	c.RevisionState = StateApprovalFailed

	if c.ApprovalFailedTime == nil {
		c.ApprovalFailedTime = &time.Time{}
	}

	*c.ApprovalFailedTime = TimeNow()

	return dbCache.Save(c)
}

// FullAddress generates a string of the full address and returns it.
func (c *ContactRevision) FullAddress() string {
	var addr string

	if len(c.AddressStreet1) > 0 {
		addr = fmt.Sprintf("%s%s\n", addr, c.AddressStreet1)
	}

	if len(c.AddressStreet2) > 0 {
		addr = fmt.Sprintf("%s%s\n", addr, c.AddressStreet2)
	}

	if len(c.AddressStreet3) > 0 {
		addr = fmt.Sprintf("%s%s\n", addr, c.AddressStreet3)
	}

	if len(c.AddressCity) > 0 || len(c.AddressState) > 0 || len(c.AddressPostalCode) > 0 {
		var line string

		if len(c.AddressCity) > 0 && len(c.AddressState) > 0 {
			line = fmt.Sprintf("%s, %s", c.AddressCity, c.AddressState)
		} else {
			line = fmt.Sprintf("%s%s", c.AddressCity, c.AddressState)
		}

		if len(c.AddressCity) > 0 || len(c.AddressState) > 0 {
			line = fmt.Sprintf("%s %s", line, c.AddressPostalCode)
		} else {
			line = fmt.Sprint(c.AddressPostalCode)
		}

		addr = fmt.Sprintf("%s%s\n", addr, line)
	}

	if len(c.AddressCountry) > 0 {
		addr = fmt.Sprintf("%s%s\n", addr, c.AddressCountry)
	}

	if len(addr) > 0 {
		return strings.Trim(addr, "\n")
	}

	return addr
}

// IsActive returns true if RevisionState is StateActive or StateBootstrap.
func (c *ContactRevision) IsActive() bool {
	return c.RevisionState == StateActive || c.RevisionState == StateBootstrap
}

// GetRequiredApproverSets prepares object and returns the ApproverSets.
func (c *ContactRevision) GetRequiredApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = c.Prepare(dbCache); err != nil {
		return approverSets, err
	}

	approverSets = c.RequiredApproverSets

	return approverSets, err
}

// GetInformedApproverSets prepares object and returns the ApproverSets.
func (c *ContactRevision) GetInformedApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = c.Prepare(dbCache); err != nil {
		return approverSets, err
	}

	approverSets = c.InformedApproverSets

	return approverSets, err
}

// EscrowAddress is used to format the mailing address into a single line that
// can be used for the RDE escrow.
func (c *ContactRevision) EscrowAddress() string {
	var addr string

	if len(c.AddressStreet1) > 0 {
		addr = fmt.Sprintf("%s%s, ", addr, c.AddressStreet1)
	}

	if len(c.AddressStreet2) > 0 {
		addr = fmt.Sprintf("%s%s, ", addr, c.AddressStreet2)
	}

	if len(c.AddressStreet3) > 0 {
		addr = fmt.Sprintf("%s%s, ", addr, c.AddressStreet3)
	}

	if len(c.AddressCity) > 0 || len(c.AddressState) > 0 || len(c.AddressPostalCode) > 0 {
		var line string

		if len(c.AddressCity) > 0 && len(c.AddressState) > 0 {
			line = fmt.Sprintf("%s, %s", c.AddressCity, c.AddressState)
		} else {
			line = fmt.Sprintf("%s%s", c.AddressCity, c.AddressState)
		}

		if len(c.AddressCity) > 0 || len(c.AddressState) > 0 {
			line = fmt.Sprintf("%s %s", line, c.AddressPostalCode)
		} else {
			line = fmt.Sprint(c.AddressPostalCode)
		}

		addr = fmt.Sprintf("%s%s, ", addr, line)
	}

	if len(c.AddressCountry) > 0 {
		addr = fmt.Sprintf("%s%s, ", addr, c.AddressCountry)
	}

	if len(addr) > 0 {
		return strings.TrimSpace(strings.Trim(strings.TrimSpace(addr), ","))
	}

	return addr
}

// VoiceNumber will return the phone number and extension in one string.
func (c *ContactRevision) VoiceNumber() string {
	return formatNumberForDisplay(c.VoicePhoneNumber, c.VoicePhoneExtension)
}

// FaxNumber will return the fax number and extension in one string.
func (c *ContactRevision) FaxNumber() string {
	return formatNumberForDisplay(c.FaxPhoneNumber, c.FaxPhoneExtension)
}

// MigrateDBContactRevision will run the automigrate function for the
// ContactRevision object.
func MigrateDBContactRevision(dbCache *DBCache) {
	dbCache.AutoMigrate(&ContactRevision{})
}
