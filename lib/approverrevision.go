// Package lib provides the objects required to operate registrar
package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
)

// More information about the ApproverRevision Object and its States
// can be found in /doc/approverrevision.md.

// ApproverRevision represents individual versions of an Approver object.
type ApproverRevision struct {
	Model
	ApproverID    int64  `json:"ApproverID"`
	RevisionState string `json:"RevisionState"`

	DesiredState string `json:"DesiredState"`

	Name         string `json:"Name"`
	EmailAddress string `json:"EmailAddress"`
	Role         string `json:"Role"`
	Username     string `json:"Username"`
	EmployeeID   int64  `json:"EmployeeID"`
	Department   string `json:"Department"`
	IsAdmin      bool   `json:"IsAdmin"`
	SavedNotes   string `json:"SavedNotes"   sql:"size:16384"`

	RequiredApproverSets []ApproverSet `gorm:"many2many:required_approverset_to_approverrevision" json:"RequiredApproverSets"`
	InformedApproverSets []ApproverSet `gorm:"many2many:informed_approverset_to_approverrevision" json:"InformedApproverSets"`

	Fingerprint string `json:"Fingerprint"`
	PublicKey   string `json:"PublicKey"   sql:"size:16384"`

	CR   ChangeRequest `json:"CR"`
	CRID sql.NullInt64 `json:"CRID"`

	IssueCR string `json:"IssueCR" sql:"size:256"`
	Notes   string `json:"Notes"   sql:"size:2048"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`

	ApprovalStartTime  *time.Time `json:"ApprovalStartTime"`
	ApprovalStartBy    string     `json:"ApprovalStartBy"`
	PromotedTime       *time.Time `json:"PromotedTime"`
	SupersededTime     *time.Time `json:"SuspendedTime"`
	ApprovalFailedTime *time.Time `json:"ApprovalFailedTime"`
}

// ApproverRevisionExport is an object that is used to export the
// current version of an Approver Revision.
type ApproverRevisionExport struct {
	ID         int64 `json:"ID"`
	ApproverID int64 `json:"ApproverID"`

	DesiredState string `json:"DesiredState"`

	Name         string `json:"Name"`
	EmailAddress string `json:"EmailAddress"`
	Role         string `json:"Role"`
	Username     string `json:"Username"`
	EmployeeID   int64  `json:"EmployeeID"`
	Department   string `json:"Department"`
	IsAdmin      bool   `json:"IsAdmin"`
	SavedNotes   string `json:"SavedNotes"`

	RequiredApproverSets []ApproverSetExportShort `json:"RequiredApproverSets"`
	InformedApproverSets []ApproverSetExportShort `json:"InformedApproverSets"`

	Fingerprint string `json:"Fingerprint"`
	PublicKey   string `json:"PublicKey"`

	ChangeRequestID int64 `json:"ChangeRequestID"`

	IssueCR string `json:"IssueCR"`
	Notes   string `json:"Notes"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// Compare is used to compare an export version of an object to the
// full revision to verify that all of the values are the same.
func (are ApproverRevisionExport) Compare(approverRevision ApproverRevision) (pass bool, errs []error) {
	pass = true

	if are.ID != approverRevision.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if are.ApproverID != approverRevision.ApproverID {
		errs = append(errs, fmt.Errorf("the ApproverID fields did not match"))
		pass = false
	}

	if are.DesiredState != approverRevision.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if are.Name != approverRevision.Name {
		errs = append(errs, fmt.Errorf("the Name fields did not match"))
		pass = false
	}

	if are.EmailAddress != approverRevision.EmailAddress {
		errs = append(errs, fmt.Errorf("the EmailAddress fields did not match"))
		pass = false
	}

	if are.Role != approverRevision.Role {
		errs = append(errs, fmt.Errorf("the Role fields did not match"))
		pass = false
	}

	if are.Username != approverRevision.Username {
		errs = append(errs, fmt.Errorf("the Username fields did not match"))
		pass = false
	}

	if are.EmployeeID != approverRevision.EmployeeID {
		errs = append(errs, fmt.Errorf("the EmployeeID fields did not match"))
		pass = false
	}

	if are.Department != approverRevision.Department {
		errs = append(errs, fmt.Errorf("the Department fields did not match"))
		pass = false
	}

	if are.IsAdmin != approverRevision.IsAdmin {
		errs = append(errs, fmt.Errorf("the Is Admin fields did not match"))
		pass = false
	}

	if are.SavedNotes != approverRevision.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if are.Fingerprint != approverRevision.Fingerprint {
		errs = append(errs, fmt.Errorf("the Fingerprint fields did not match"))
		pass = false
	}

	if are.PublicKey != approverRevision.PublicKey {
		errs = append(errs, fmt.Errorf("the PublicKey fields did not match"))
		pass = false
	}

	if are.IssueCR != approverRevision.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if are.Notes != approverRevision.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetListToExportShort(approverRevision.RequiredApproverSets, are.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetListToExportShort(approverRevision.InformedApproverSets, are.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// CompareExport is used to compare an export version of an object to
// another export revision to verify that all of the values are the
// same.
func (are ApproverRevisionExport) CompareExport(approverRevisionExport ApproverRevisionExport) (pass bool, errs []error) {
	pass = true

	if are.ID != approverRevisionExport.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if are.ApproverID != approverRevisionExport.ApproverID {
		errs = append(errs, fmt.Errorf("the ApproverID fields did not match"))
		pass = false
	}

	if are.DesiredState != approverRevisionExport.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if are.Name != approverRevisionExport.Name {
		errs = append(errs, fmt.Errorf("the Name fields did not match"))
		pass = false
	}

	if are.EmailAddress != approverRevisionExport.EmailAddress {
		errs = append(errs, fmt.Errorf("the EmailAddress fields did not match"))
		pass = false
	}

	if are.Role != approverRevisionExport.Role {
		errs = append(errs, fmt.Errorf("the Role fields did not match"))
		pass = false
	}

	if are.Username != approverRevisionExport.Username {
		errs = append(errs, fmt.Errorf("the Username fields did not match"))
		pass = false
	}

	if are.EmployeeID != approverRevisionExport.EmployeeID {
		errs = append(errs, fmt.Errorf("the EmployeeID fields did not match"))
		pass = false
	}

	if are.Department != approverRevisionExport.Department {
		errs = append(errs, fmt.Errorf("the Department fields did not match"))
		pass = false
	}

	if are.IsAdmin != approverRevisionExport.IsAdmin {
		errs = append(errs, fmt.Errorf("the Is Admin fields did not match"))
		pass = false
	}

	if are.SavedNotes != approverRevisionExport.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if are.Fingerprint != approverRevisionExport.Fingerprint {
		errs = append(errs, fmt.Errorf("the Fingerprint fields did not match"))
		pass = false
	}

	if are.PublicKey != approverRevisionExport.PublicKey {
		errs = append(errs, fmt.Errorf("the PublicKey fields did not match"))
		pass = false
	}

	if are.IssueCR != approverRevisionExport.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if are.Notes != approverRevisionExport.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetExportShortLists(approverRevisionExport.RequiredApproverSets, are.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetExportShortLists(approverRevisionExport.InformedApproverSets, are.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (are ApproverRevisionExport) ToJSON() (string, error) {
	if are.ID <= 0 {
		return "", errors.New("invalid revision ID")
	}

	byteArr, jsonErr := json.MarshalIndent(are, "", "  ")

	return string(byteArr), jsonErr
}

// GetDiff will return an empty string and an error for a revision. A
// Diff is not available for revision objects.
func (are ApproverRevisionExport) GetDiff() (string, error) {
	return "", errors.New("unable to diff a single revision")
}

// ApproverRevisionPage are used to hold all the information required to
// render the AppproverRevision HTML template.
type ApproverRevisionPage struct {
	IsEditable                 bool
	IsNew                      bool
	Revision                   ApproverRevision
	PendingActions             map[string]string
	ValidApproverSets          map[int64]string
	ParentApprover             *Approver
	SuggestedRequiredApprovers map[int64]ApproverSetDisplayObject
	SuggestedInformedApprovers map[int64]ApproverSetDisplayObject

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApproverRevisionPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApproverRevisionPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// The ApproverRevisionsPage type is used to render the html template
// which lists all of the AppproverRevisions currently in the
// registrar system.
//
// TODO: Add paging support.
type ApproverRevisionsPage struct {
	ApproverRevisions []ApproverRevision

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApproverRevisionsPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApproverRevisionsPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// GetExportVersion returns a export version of the ApproverRevision
// Object.
func (a *ApproverRevision) GetExportVersion() RegistrarObjectExport {
	export := ApproverRevisionExport{
		ID:           a.ID,
		ApproverID:   a.ApproverID,
		DesiredState: a.DesiredState,
		Name:         a.Name,
		EmailAddress: a.EmailAddress,
		Role:         a.Role,
		Username:     a.Username,
		EmployeeID:   a.EmployeeID,
		Department:   a.Department,
		IsAdmin:      a.IsAdmin,
		SavedNotes:   a.SavedNotes,
		Fingerprint:  a.Fingerprint,
		PublicKey:    a.PublicKey,
		IssueCR:      a.IssueCR,
		Notes:        a.Notes,
		CreatedAt:    a.CreatedAt,
		CreatedBy:    a.CreatedBy,
	}

	export.RequiredApproverSets = GetApproverSetExportArr(a.RequiredApproverSets)
	export.InformedApproverSets = GetApproverSetExportArr(a.InformedApproverSets)

	if a.CRID.Valid {
		export.ChangeRequestID = a.CRID.Int64
	} else {
		export.ChangeRequestID = -1
	}

	return export
}

// GetExportVersionAt returns an export version of the Approver Revision
// Object at the timestamp provided if possible otherwise an error is
// returned. If a pending version existed at the time it will be
// excluded from the object.
func (a *ApproverRevision) GetExportVersionAt(_ *DBCache, _ int64) (obj RegistrarObjectExport, err error) {
	return obj, errors.New("GetExportVersionAt is not usable for revisions")
}

// StartApprovalProcess creates a change request to start the process of
// approvnig a new Change Request. If the Change Request was created
// no error is returned, otherwise an error will be returned.
//
// TODO: Check if a CR already exists for this object.
// TODO: Ensure that if an error occures no changes are made.
func (a *ApproverRevision) StartApprovalProcess(request *http.Request, dbCache *DBCache, conf Config) (err error) {
	runame, ruerr := GetRemoteUser(request)
	if ruerr != nil {
		return errors.New("no username set")
	}

	logger.Infof("starting approval for ID: %d\n", a.ID)

	if a.RevisionState != StateNew {
		return fmt.Errorf("cannot start approval for %s %d, state is '%s' not 'new'", ApproverRevisionType, a.ID, a.RevisionState)
	}

	if err = a.Prepare(dbCache); err != nil {
		return err
	}

	approver := Approver{}

	if err = dbCache.FindByID(&approver, a.ApproverID); err != nil {
		return err
	}

	currentRev := approver.CurrentRevision

	logger.Debugf("Parent Approver ID: %d", approver.GetID())

	export := approver.GetExportVersion()

	changeRequestJSON, err1 := export.ToJSON()
	diff, err2 := export.GetDiff()

	if approver.PendingRevision.ID == approver.CurrentRevisionID.Int64 || err1 != nil || err2 != nil {
		return errors.New("unable to create diff for approval")
	}

	logger.Debugf("Diff Len: %d", len(diff))
	logger.Debugf("JSON Len: %d", len(changeRequestJSON))

	changeRequest := ChangeRequest{
		RegistrarObjectType: ApproverType,
		RegistrarObjectID:   a.ApproverID,
		ProposedRevisionID:  a.ID,
		State:               StateNew,
		InitialRevisionID:   approver.CurrentRevisionID,
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

	if len(currentRev.RequiredApproverSets) == 0 {
		logger.Debug("No approver sets requied, defaulting to Approver Set 1")

		app := Approval{
			ChangeRequestID: changeRequest.ID,
			ApproverSetID:   1,
			State:           StateNew, CreatedBy: runame, UpdatedBy: runame,
			UpdatedAt: TimeNow(), CreatedAt: TimeNow(),
		}
		changeRequest.Approvals = append(changeRequest.Approvals, app)
	}

	logger.Debugf("%d Required Approver Sets", len(currentRev.RequiredApproverSets))

	for _, approverSet := range currentRev.RequiredApproverSets {
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

	a.CR = changeRequest

	if err = a.CRID.Scan(changeRequest.ID); err != nil {
		return fmt.Errorf("error getting crid: %w", err)
	}

	a.RevisionState = StatePendingApproval
	a.UpdatedBy = runame
	a.UpdatedAt = TimeNow()

	if a.ApprovalStartTime == nil {
		a.ApprovalStartTime = &time.Time{}
	}

	*a.ApprovalStartTime = TimeNow()
	a.ApprovalStartBy = runame

	if err = dbCache.Save(a); err != nil {
		return err
	}

	approver = Approver{}

	if err = dbCache.FindByID(&approver, a.ApproverID); err != nil {
		return err
	}

	if approver.CurrentRevisionID.Valid {
		if approver.CurrentRevision.DesiredState != StateBootstrap {
			if approver.CurrentRevision.DesiredState == StateActive {
				approver.State = StateActivePendingApproval
			} else {
				approver.State = StateInactivePendingApproval
			}
		} else {
			approver.State = StatePendingBootstrap
		}
	} else {
		approver.State = StatePendingNew
	}

	approver.UpdatedBy = runame
	approver.UpdatedAt = TimeNow()

	if err = dbCache.Save(&approver); err != nil {
		return err
	}

	_, errs := changeRequest.UpdateState(dbCache, conf)
	if len(errs) != 0 {
		return errs[0]
	}

	return nil
}

// GetState is used to verify that a state submitted in an HTTP request
// is either "active" or "inactive". If the submitted state does not
// match either of the options "active" is returned. "bootstrap" is not
// an allowed state via HTTP.
func (a *ApproverRevision) GetState(cleartextState string) string {
	if cleartextState == StateInactive {
		return StateInactive
	}

	return StateActive
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
//
// TODO: verify public key.
// TODO: verify fingerprint.
func (a *ApproverRevision) ParseFromForm(request *http.Request, dbCache *DBCache) error {
	var err3, err4 error

	appID, err1 := strconv.ParseInt(request.FormValue("revision_approver_id"), 10, 64)
	empID, err2 := strconv.ParseInt(request.FormValue("revision_empid"), 10, 64)

	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	a.CreatedBy = runame
	a.UpdatedBy = runame

	a.ApproverID = appID
	a.RevisionState = StateNew

	a.DesiredState = a.GetState(request.FormValue("revision_desiredstate"))

	a.Name = request.FormValue("revision_name")
	a.EmailAddress = request.FormValue("revision_email")
	a.Role = request.FormValue("revision_role")
	a.Username = request.FormValue("revision_username")
	a.EmployeeID = empID
	a.Department = request.FormValue("revision_dept")
	a.SavedNotes = request.FormValue("revision_saved_notes")

	a.IsAdmin = GetCheckboxState(request.FormValue("is_admin"))

	a.PublicKey = request.FormValue("revision_pubkey")
	a.Fingerprint, _ = keyToFingerprint(a.PublicKey)

	a.IssueCR = request.FormValue("revision_issue_cr")
	a.Notes = request.FormValue("revision_notes")

	a.RequiredApproverSets, err3 = ParseApproverSets(request, dbCache, "approver_set_required_id", true)
	a.InformedApproverSets, err4 = ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

	if err1 != nil {
		return fmt.Errorf("error parsing revision approver id: %w", err1)
	}

	if err2 != nil {
		return fmt.Errorf("error parsing revision employee id: %w", err2)
	}

	if err3 != nil {
		return err3
	}

	if err4 != nil {
		return err4
	}

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse ApproverRevision object with the changes that
// were made. An error object is always the second return value which
// is nil when no errors have occurred during parsing otherwise an error
// is returned.
//
// TODO: verify public key.
// TODO: verify fingerprint.
func (a *ApproverRevision) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, _ Config) error {
	if a.ID > 0 {
		runame, err := GetRemoteUser(request)
		if err != nil {
			return err
		}

		if a.RevisionState == StateNew {
			empID, err2 := strconv.ParseInt(request.FormValue("revision_empid"), 10, 64)
			RequiredApproverSets, err3 := ParseApproverSets(request, dbCache, "approver_set_required_id", true)
			InformedApproverSets, err4 := ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

			if err2 != nil {
				return fmt.Errorf("error parsing revision employee id: %w", err2)
			}

			if err3 != nil {
				return err3
			}

			if err4 != nil {
				return err4
			}

			a.UpdatedBy = runame

			a.DesiredState = a.GetState(request.FormValue("revision_desiredstate"))
			a.Name = request.FormValue("revision_name")
			a.EmailAddress = request.FormValue("revision_email")
			a.Role = request.FormValue("revision_role")
			a.Username = request.FormValue("revision_username")
			a.EmployeeID = empID
			a.Department = request.FormValue("revision_dept")
			a.SavedNotes = request.FormValue("revision_saved_notes")

			a.IsAdmin = GetCheckboxState(request.FormValue("is_admin"))

			a.PublicKey = request.FormValue("revision_pubkey")
			a.Fingerprint, _ = keyToFingerprint(a.PublicKey)

			a.IssueCR = request.FormValue("revision_issue_cr")
			a.Notes = request.FormValue("revision_notes")

			updateAppSetErr := UpdateApproverSets(a, dbCache, "RequiredApproverSets", RequiredApproverSets)

			if updateAppSetErr != nil {
				return err
			}

			a.RequiredApproverSets = RequiredApproverSets

			updateAppSetErr = UpdateApproverSets(a, dbCache, "InformedApproverSets", InformedApproverSets)

			if updateAppSetErr != nil {
				return err
			}

			a.InformedApproverSets = InformedApproverSets
		} else {
			return fmt.Errorf("cannot update an object not in the %s state", a.RevisionState)
		}

		return nil
	}

	return errors.New("to update an approver revision the ID must be greater than 0")
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (a *ApproverRevision) GetType() string {
	return ApproverRevisionType
}

// GetPage will return an object that can be used to render the HTML
// template for the ApproverRevision.
func (a *ApproverRevision) GetPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &ApproverRevisionPage{IsNew: true}
	ret.IsEditable = a.IsEditable()
	ret.PendingActions = a.GetActions(true)

	if a.ID != 0 {
		ret.IsNew = false
	}

	ret.Revision = *a
	ret.ParentApprover = &Approver{}

	if err = dbCache.FindByID(ret.ParentApprover, a.ApproverID); err != nil {
		return ret, err
	}

	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	return ret, err
}

// HasHappened is a helper function that will return true if the value
// of the timestamp who's name is passes has happened after the revision
// was created. This function is intended to be used with templates.
func (a ApproverRevision) HasHappened(actionType string) bool {
	switch actionType {
	case EventUpdated:
		return !a.UpdatedAt.Before(a.CreatedAt)
	case EventApprovalStarted:
		if a.ApprovalStartTime != nil {
			return !a.ApprovalStartTime.Before(a.CreatedAt)
		}

		return false
	case EventApprovalFailed:
		if a.ApprovalFailedTime != nil {
			return !a.ApprovalFailedTime.Before(a.CreatedAt)
		}

		return false
	case EventPromoted:
		if a.PromotedTime != nil {
			return !a.PromotedTime.Before(a.CreatedAt)
		}

		return false
	case EventSuperseded:
		if a.SupersededTime != nil {
			return !a.SupersededTime.Before(a.CreatedAt)
		}

		return false
	}

	return false
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple Approver Revisions.
//
// TODO: Add paging support.
// TODO: Add filtering.
func (a *ApproverRevision) GetAllPage(dbCache *DBCache, _ string, _ string) (RegistrarObjectPage, error) {
	ret := &ApproverRevisionsPage{}

	err := dbCache.FindAll(&ret.ApproverRevisions)

	return ret, err
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the db query into the dbcache object.
func (a *ApproverRevision) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, a, func() (err error) {
		if err = dbCache.DB.Model(a).Related(&a.RequiredApproverSets, "RequiredApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return
			}
		}

		if err = dbCache.DB.Model(a).Related(&a.InformedApproverSets, "InformedApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return
			}
		}

		for idx := range a.RequiredApproverSets {
			if err = a.RequiredApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return
			}
		}

		for idx := range a.InformedApproverSets {
			if err = a.InformedApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return
			}
		}

		return
	})
}

// Cancel will change the State of a revision from either "new" or
// "pendingapproval" to "cancelled".
//
// TODO: If in pending approval, cancel the change request and all
// approval objects.
// TODO: Consider moving the db query into the dbcache object.
func (a *ApproverRevision) Cancel(dbCache *DBCache, conf Config) (errs []error) {
	if a.RevisionState == StateNew || a.RevisionState == StatePendingApproval {
		a.RevisionState = StateCancelled

		if err := dbCache.DB.Model(a).UpdateColumns(ApproverRevision{RevisionState: StateCancelled}).Error; err != nil {
			errs = append(errs, err)

			return errs
		}

		err := dbCache.Purge(a)
		if err != nil {
			errs = append(errs, err)

			return errs
		}

		parent := Approver{}

		if err := dbCache.FindByID(&parent, a.ApproverID); err != nil {
			errs = append(errs, err)

			return errs
		}

		_, parentErrs := parent.UpdateState(dbCache, conf)
		errs = append(errs, parentErrs...)

		if a.CRID.Valid {
			changeRequest := ChangeRequest{}

			if err := dbCache.FindByID(&changeRequest, a.CRID.Int64); err != nil {
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

// IsCancelled returns true iff the object has been canclled.
func (a *ApproverRevision) IsCancelled() bool {
	return a.RevisionState == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (a *ApproverRevision) IsEditable() bool {
	return a.RevisionState == StateNew
}

// IsDesiredState will return true iff the state passed in matches the
// desired state of the revision.
func (a ApproverRevision) IsDesiredState(state string) bool {
	return a.DesiredState == state
}

// GetActions will return a list of possible actions that can be taken
// while in the current state.
//
// TODO: handle all states.
func (a *ApproverRevision) GetActions(isSelf bool) map[string]string {
	ret := make(map[string]string)

	if a.RevisionState == StateNew {
		ret["Start Approval Process"] = fmt.Sprintf("/action/%s/%d/startapproval", ApproverRevisionType, a.ID)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", ApproverRevisionType, a.ID)

		if isSelf {
			ret["View Parent Approver"] = fmt.Sprintf("/view/%s/%d", ApproverType, a.ApproverID)
		} else {
			ret["View/Edit Approver Revision"] = fmt.Sprintf("/view/%s/%d", ApproverRevisionType, a.ID)
		}
	}

	if a.RevisionState == StatePendingApproval {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", ApproverRevisionType, a.ID, ApproverRevisionActionGOTOChangeRequest)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", ApproverRevisionType, a.ID)

		if isSelf {
			ret["View Parent Approver"] = fmt.Sprintf("/view/%s/%d", ApproverType, a.ApproverID)
		}
	}

	if a.RevisionState == StateCancelled {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", ApproverRevisionType, a.ID, ApproverRevisionActionGOTOChangeRequest)

		if isSelf {
			ret["View Parent Approver"] = fmt.Sprintf("/view/%s/%d", ApproverType, a.ApproverID)
		}
	}

	return ret
}

// ApproverRevisionActionGOTOChangeRequest is the name of the action
// that will trigger a redirect to the current CR of an object.
const ApproverRevisionActionGOTOChangeRequest string = "gotochangerequest"

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (a *ApproverRevision) TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, validCSRF bool, authMethod AuthType, conf Config) (errs []error) {
	switch actionName {
	case ActionCancel:
		// TODO: Figure out who can cancel
		if validCSRF {
			cancelErrs1 := a.Cancel(dbCache, conf)
			if cancelErrs1 != nil {
				errs = append(errs, cancelErrs1...)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ApproverRevisionType, a.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ActionStartApproval:
		if validCSRF {
			appErr := a.StartApprovalProcess(request, dbCache, conf)
			if appErr != nil {
				errs = append(errs, appErr)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ApproverRevisionType, a.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ApproverRevisionActionGOTOChangeRequest:
		if a.CRID.Valid {
			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ChangeRequestType, a.CRID.Int64), http.StatusFound)

			return errs
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(a.GetExportVersion()))

			return errs
		}
	default:
		errs = append(errs, fmt.Errorf("unknown action %s for %s", actionName, ApproverRevisionType))

		return errs
	}

	errs = append(errs, errors.New("unable to take action"))

	return errs
}

// Promote will mark an ApproverRevision as the current revision for an
// Approver if it has not been cancelled or failed approval.
func (a *ApproverRevision) Promote(dbCache *DBCache) (err error) {
	if err = a.Prepare(dbCache); err != nil {
		return err
	}

	if a.RevisionState == StateCancelled || a.RevisionState == StateApprovalFailed {
		return errors.New("cannot promote revision in cancelled or approvalfailed state")
	}

	if a.CRID.Valid {
		changeRequest := ChangeRequest{}

		if err = dbCache.FindByID(&changeRequest, a.CRID.Int64); err != nil {
			return err
		}

		if changeRequest.State == StateApproved {
			a.RevisionState = a.DesiredState

			if a.PromotedTime == nil {
				a.PromotedTime = &time.Time{}
			}

			*a.PromotedTime = TimeNow()

			if err = dbCache.Save(a); err != nil {
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

// Supersed will mark an ApproverRevision as a superseded revision for
// an Approver.
func (a *ApproverRevision) Supersed(dbCache *DBCache) (err error) {
	if err = a.Prepare(dbCache); err != nil {
		return err
	}

	if a.RevisionState != StateActive && a.RevisionState != StateInactive && a.RevisionState != StateBootstrap {
		return fmt.Errorf("cannot supersed a revision not in active, inactive or bootstrap state (in %s)", a.RevisionState)
	}

	a.RevisionState = StateSuperseded

	if a.SupersededTime == nil {
		a.SupersededTime = &time.Time{}
	}

	*a.SupersededTime = TimeNow()

	return dbCache.Save(a)
}

// Decline will mark an ApproverRevision as decline for an Approver.
func (a *ApproverRevision) Decline(dbCache *DBCache) (err error) {
	if err = a.Prepare(dbCache); err != nil {
		return err
	}

	if a.RevisionState != StatePendingApproval {
		return errors.New("only revisions in pendingapproval may be declined")
	}

	a.RevisionState = StateApprovalFailed

	if a.ApprovalFailedTime == nil {
		a.ApprovalFailedTime = &time.Time{}
	}

	*a.ApprovalFailedTime = TimeNow()

	return dbCache.Save(a)
}

// IsActive returns true if RevisionState is StateActive or StateBootstrap.
func (a *ApproverRevision) IsActive() bool {
	return a.RevisionState == StateActive || a.RevisionState == StateBootstrap
}

// GetRequiredApproverSets prepares object and returns the ApproverSets.
func (a *ApproverRevision) GetRequiredApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = a.Prepare(dbCache); err != nil {
		return approverSets, err
	}

	approverSets = a.RequiredApproverSets

	return approverSets, err
}

// GetInformedApproverSets prepares object and returns the ApproverSets.
func (a *ApproverRevision) GetInformedApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = a.Prepare(dbCache); err != nil {
		return approverSets, err
	}

	approverSets = a.InformedApproverSets

	return approverSets, err
}

// MigrateDBApproverRevision will run the automigrate function for the
// ApproverRevision object.
func MigrateDBApproverRevision(dbCache *DBCache) {
	dbCache.AutoMigrate(&ApproverRevision{})
}
