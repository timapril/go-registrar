package lib

import (
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
)

// APIUserRevision represents individual versions of an APIUser object.
type APIUserRevision struct {
	Model
	APIUserID int64 `json:"APIUserID"`

	RevisionState string `json:"RevisionState"`
	DesiredState  string `json:"DesiredState"`

	Name        string `json:"Name"`
	Description string `json:"Description" sql:"size:16384"`
	Serial      string `json:"Serial"      sql:"size:256"`
	Certificate string `json:"Certificate" sql:"size:16384"`
	SavedNotes  string `json:"SavedNotes"  sql:"size:16384"`

	IsAdmin     bool `json:"IsAdmin"`
	IsEPPClient bool `json:"IsEPPClient"`

	RequiredApproverSets []ApproverSet `gorm:"many2many:required_approverset_to_apiuserrevision" json:"RequiredApproverSets"`
	InformedApproverSets []ApproverSet `gorm:"many2many:informed_approverset_to_apiuserrevision" json:"InformedApproverSets"`

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
	SupersededTime     *time.Time `json:"SupersededTime"`
	ApprovalFailedTime *time.Time `json:"ApprovalFailedTime"`
}

// APIUserRevisionExport is an object that is used to export the current
// version of an APIUser Revision.
type APIUserRevisionExport struct {
	ID        int64 `json:"ID"`
	APIUserID int64 `json:"APIUserID"`

	DesiredState string `json:"DesiredState"`

	Name        string `json:"Name"`
	Description string `json:"Description"`
	Serial      string `json:"Serial"`
	Certificate string `json:"Certificate"`
	SavedNotes  string `json:"SavedNotes"`

	IsAdmin     bool `json:"IsAdmin"`
	IsEPPClient bool `json:"IsEPPClient"`

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
func (are APIUserRevisionExport) Compare(apiUserRevision APIUserRevision) (pass bool, errs []error) {
	pass = true

	if are.ID != apiUserRevision.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if are.APIUserID != apiUserRevision.APIUserID {
		errs = append(errs, fmt.Errorf("the ApproverID fields did not match"))
		pass = false
	}

	if are.DesiredState != apiUserRevision.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if are.Name != apiUserRevision.Name {
		errs = append(errs, fmt.Errorf("the Name fields did not match"))
		pass = false
	}

	if are.Description != apiUserRevision.Description {
		errs = append(errs, fmt.Errorf("the EmailAddress fields did not match"))
		pass = false
	}

	if are.Serial != apiUserRevision.Serial {
		errs = append(errs, fmt.Errorf("the Serial fields did not match"))
		pass = false
	}

	if are.Certificate != apiUserRevision.Certificate {
		errs = append(errs, fmt.Errorf("the Role fields did not match"))
		pass = false
	}

	if are.SavedNotes != apiUserRevision.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if are.IsAdmin != apiUserRevision.IsAdmin {
		errs = append(errs, fmt.Errorf("the IsAdmin fields did not match"))
		pass = false
	}

	if are.IsEPPClient != apiUserRevision.IsEPPClient {
		errs = append(errs, fmt.Errorf("the IsEPPClient fields did not match"))
		pass = false
	}

	if are.IssueCR != apiUserRevision.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if are.Notes != apiUserRevision.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetListToExportShort(apiUserRevision.RequiredApproverSets, are.RequiredApproverSets)

	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetListToExportShort(apiUserRevision.InformedApproverSets, are.InformedApproverSets)

	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// CompareExport is used to compare an export version of an object to
// another export revision to verify that all of the values are the
// same.
func (are APIUserRevisionExport) CompareExport(apiUserRevisionExport APIUserRevisionExport) (pass bool, errs []error) {
	pass = true

	if are.ID != apiUserRevisionExport.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if are.APIUserID != apiUserRevisionExport.APIUserID {
		errs = append(errs, fmt.Errorf("the ApproverID fields did not match"))
		pass = false
	}

	if are.DesiredState != apiUserRevisionExport.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if are.Name != apiUserRevisionExport.Name {
		errs = append(errs, fmt.Errorf("the Name fields did not match"))
		pass = false
	}

	if are.Description != apiUserRevisionExport.Description {
		errs = append(errs, fmt.Errorf("the EmailAddress fields did not match"))
		pass = false
	}

	if are.Serial != apiUserRevisionExport.Serial {
		errs = append(errs, fmt.Errorf("the Serial fields did not match"))
		pass = false
	}

	if are.Certificate != apiUserRevisionExport.Certificate {
		errs = append(errs, fmt.Errorf("the Role fields did not match"))
		pass = false
	}

	if are.SavedNotes != apiUserRevisionExport.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if are.IsAdmin != apiUserRevisionExport.IsAdmin {
		errs = append(errs, fmt.Errorf("the IsAdmin fields did not match"))
		pass = false
	}

	if are.IsEPPClient != apiUserRevisionExport.IsEPPClient {
		errs = append(errs, fmt.Errorf("the IsEPPClient fields did not match"))
		pass = false
	}

	if are.IssueCR != apiUserRevisionExport.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if are.Notes != apiUserRevisionExport.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetExportShortLists(apiUserRevisionExport.RequiredApproverSets, are.RequiredApproverSets)

	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetExportShortLists(apiUserRevisionExport.InformedApproverSets, are.InformedApproverSets)

	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// ToJSON will return a string containing a JSON representation of the
// object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (are APIUserRevisionExport) ToJSON() (string, error) {
	if are.ID <= 0 {
		return "", errors.New("invalid revision ID")
	}

	byteArr, jsonErr := json.MarshalIndent(are, "", "  ")

	return string(byteArr), jsonErr
}

// GetDiff will return an empty string and an error for a revision. A
// Diff is not available for revision objects.
func (are APIUserRevisionExport) GetDiff() (string, error) {
	return "", errors.New("unable to diff a single revision")
}

// APIUserRevisionPage are used to hold all the information required to
// render the APIUserRevision HTML template.
type APIUserRevisionPage struct {
	IsEditable                 bool
	IsNew                      bool
	Revision                   APIUserRevision
	PendingActions             map[string]string
	ValidApproverSets          map[int64]string
	ParentAPIUser              *APIUser
	SuggestedRequiredApprovers map[int64]ApproverSetDisplayObject
	SuggestedInformedApprovers map[int64]ApproverSetDisplayObject

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *APIUserRevisionPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *APIUserRevisionPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// The APIUserRevisionsPage type is used to render the html template
// which lists all of the APIUserRevisions currently in the
// registrar system.
//
// TODO: Add paging support.
type APIUserRevisionsPage struct {
	APIUserRevisions []APIUserRevision

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *APIUserRevisionsPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *APIUserRevisionsPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// GetExportVersion returns a export version of the APIRevisionRevision
// Object.
func (a *APIUserRevision) GetExportVersion() RegistrarObjectExport {
	export := APIUserRevisionExport{
		ID:           a.ID,
		APIUserID:    a.APIUserID,
		DesiredState: a.DesiredState,
		Name:         a.Name,
		Description:  a.Description,
		Certificate:  a.Certificate,
		SavedNotes:   a.SavedNotes,
		Serial:       a.Serial,
		IsAdmin:      a.IsAdmin,
		IsEPPClient:  a.IsEPPClient,
		IssueCR:      a.IssueCR,
		Notes:        a.Notes,
		CreatedAt:    a.CreatedAt,
		CreatedBy:    a.CreatedBy,
	}

	if a.CRID.Valid {
		export.ChangeRequestID = a.CRID.Int64
	} else {
		export.ChangeRequestID = -1
	}

	export.RequiredApproverSets = GetApproverSetExportArr(a.RequiredApproverSets)
	export.InformedApproverSets = GetApproverSetExportArr(a.InformedApproverSets)

	return export
}

// GetExportVersionAt returns an export version of the APIUser Revision
// Object at the timestamp provided if possible otherwise an error is
// returned. If a pending version existed at the time it will be
// excluded from the object.
func (a *APIUserRevision) GetExportVersionAt(_ *DBCache, _ int64) (obj RegistrarObjectExport, err error) {
	return obj, errors.New("GetExportVersionAt is not usable for revisions")
}

// StartApprovalProcess creates a change request to start the process of
// approvnig a new Change Request. If the Change Request was created
// no error is returned, otherwise an error will be returned.
//
// TODO: Check if a CR already exists for this object.
// TODO: Ensure that if an error occures no changes are made.
func (a *APIUserRevision) StartApprovalProcess(request *http.Request, dbCache *DBCache, conf Config) (err error) {
	runame, ruerr := GetRemoteUser(request)

	if ruerr != nil {
		return errors.New("no username set")
	}

	logger.Infof("starting approval for ID: %d", a.ID)

	if a.RevisionState != StateNew {
		return fmt.Errorf("cannot start approval for %s %d, state is '%s' not 'new'", APIUserRevisionType, a.ID, a.RevisionState)
	}

	if err = a.Prepare(dbCache); err != nil {
		return err
	}

	apiuser := APIUser{}

	if err = dbCache.FindByID(&apiuser, a.APIUserID); err != nil {
		return err
	}

	curRev := apiuser.CurrentRevision
	logger.Debugf("Parent API User ID: %d", apiuser.GetID())

	export := apiuser.GetExportVersion()

	changeRequestJSON, err1 := export.ToJSON()
	diff, err2 := export.GetDiff()

	if apiuser.PendingRevision.ID == apiuser.CurrentRevisionID.Int64 || err1 != nil || err2 != nil {
		return errors.New("unable to create diff for approval")
	}

	logger.Debugf("Diff Len: %d", len(diff))
	logger.Debugf("JSON Len: %d", len(changeRequestJSON))

	changeRequest := ChangeRequest{
		RegistrarObjectType: APIUserType,
		RegistrarObjectID:   a.APIUserID,
		ProposedRevisionID:  a.ID,
		State:               StateNew,
		InitialRevisionID:   apiuser.CurrentRevisionID,
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

	if len(curRev.RequiredApproverSets) == 0 {
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

	logger.Debugf("%d Required Approver Sets", len(curRev.RequiredApproverSets))

	for _, approverSet := range curRev.RequiredApproverSets {
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
		return fmt.Errorf("unable to assign CRID: %w", err)
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

	apiuser = APIUser{}

	if err = dbCache.FindByID(&apiuser, a.APIUserID); err != nil {
		return err
	}

	if apiuser.CurrentRevisionID.Valid {
		if apiuser.CurrentRevision.DesiredState != StateBootstrap {
			if apiuser.CurrentRevision.DesiredState == StateActive {
				apiuser.State = StateActivePendingApproval
			} else {
				apiuser.State = StateInactivePendingApproval
			}
		} else {
			apiuser.State = StatePendingBootstrap
		}
	} else {
		apiuser.State = StatePendingNew
	}

	apiuser.UpdatedBy = runame
	apiuser.UpdatedAt = TimeNow()

	if err = dbCache.Save(&apiuser); err != nil {
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
func (a *APIUserRevision) GetState(cleartextState string) string {
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
func (a *APIUserRevision) ParseFromForm(request *http.Request, dbCache *DBCache) error {
	var err2, err3 error

	apiuserID, err1 := strconv.ParseInt(request.FormValue("revision_apiuser_id"), 10, 64)
	if err1 != nil {
		return fmt.Errorf("unable to parse revision_apiuser_id: %w", err1)
	}

	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	a.CreatedBy = runame
	a.UpdatedBy = runame

	a.APIUserID = apiuserID
	a.RevisionState = StateNew

	a.DesiredState = a.GetState(request.FormValue("revision_desiredstate"))

	a.Name = request.FormValue("revision_name")
	a.Description = request.FormValue("revision_description")
	a.Certificate = request.FormValue("revision_certificate")
	a.SavedNotes = request.FormValue("revision_saved_notes")

	a.IsAdmin = GetCheckboxState(request.FormValue("revision_is_admin"))
	a.IsEPPClient = GetCheckboxState(request.FormValue("revision_is_epp_client"))

	a.IssueCR = request.FormValue("revision_issue_cr")
	a.Notes = request.FormValue("revision_notes")

	serial, serialErr := GetCertificateSerial([]byte(a.Certificate))
	a.Serial = serial

	a.RequiredApproverSets, err2 = ParseApproverSets(request, dbCache, "approver_set_required_id", true)
	a.InformedApproverSets, err3 = ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

	if err2 != nil {
		return err2
	}

	if err3 != nil {
		return err3
	}

	if serialErr != nil {
		return serialErr
	}

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse APIUserRevision object with the changes that
// were made. An error object is always the second return value which
// is nil when no errors have occurred during parsing otherwise an error
// is returned.
//
// TODO: verify public key.
// TODO: verify fingerprint.
func (a *APIUserRevision) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, _ Config) error {
	if a.ID > 0 {
		runame, err := GetRemoteUser(request)
		if err != nil {
			return err
		}

		if a.RevisionState == StateNew {
			a.UpdatedBy = runame

			a.DesiredState = a.GetState(request.FormValue("revision_desiredstate"))

			a.Name = request.FormValue("revision_name")
			a.Description = request.FormValue("revision_description")

			a.Certificate = request.FormValue("revision_certificate")
			serial, serialErr := GetCertificateSerial([]byte(a.Certificate))
			logger.Debugf("Serial of new cert: %s", serial)
			a.Serial = serial
			a.SavedNotes = request.FormValue("revision_saved_notes")

			a.IsAdmin = GetCheckboxState(request.FormValue("revision_is_admin"))
			a.IsEPPClient = GetCheckboxState(request.FormValue("revision_is_epp_client"))

			a.IssueCR = request.FormValue("revision_issue_cr")
			a.Notes = request.FormValue("revision_notes")

			RequiredApproverSets, err2 := ParseApproverSets(request, dbCache, "approver_set_required_id", true)
			InformedApproverSets, err3 := ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

			if err2 != nil {
				return err2
			}

			if err3 != nil {
				return err3
			}

			if serialErr != nil {
				return serialErr
			}

			appSetUpdateErr := UpdateApproverSets(a, dbCache, "RequiredApproverSets", RequiredApproverSets)

			if err != nil {
				return appSetUpdateErr
			}

			a.RequiredApproverSets = RequiredApproverSets

			appSetUpdateErr = UpdateApproverSets(a, dbCache, "InformedApproverSets", InformedApproverSets)

			if err != nil {
				return appSetUpdateErr
			}

			a.InformedApproverSets = InformedApproverSets
		} else {
			return fmt.Errorf("cannot update an object not in the %s state", a.RevisionState)
		}

		return nil
	}

	return errors.New("to update an APIUserRevision the ID must be greater than 0")
}

// IsActive returns true if RevisionState is StateActive or StateBootstrap.
func (a *APIUserRevision) IsActive() bool {
	return a.RevisionState == StateActive || a.RevisionState == StateBootstrap
}

// GetRequiredApproverSets prepares object and returns the ApproverSets.
func (a *APIUserRevision) GetRequiredApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = a.Prepare(dbCache); err != nil {
		return
	}

	approverSets = a.RequiredApproverSets

	return
}

// GetInformedApproverSets prepares object and returns the ApproverSets.
func (a *APIUserRevision) GetInformedApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = a.Prepare(dbCache); err != nil {
		return
	}

	approverSets = a.InformedApproverSets

	return
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (a *APIUserRevision) GetType() string {
	return APIUserRevisionType
}

// GetPage will return an object that can be used to render the HTML
// template for the APIUserRevision.
func (a *APIUserRevision) GetPage(dbCache *DBCache, _ string, _ string) (arop RegistrarObjectPage, err error) {
	ret := &APIUserRevisionPage{IsNew: true}
	ret.IsEditable = a.IsEditable()
	ret.PendingActions = a.GetActions(true)

	if a.ID != 0 {
		ret.IsNew = false
	}

	ret.Revision = *a
	ret.ParentAPIUser = &APIUser{}

	if err = dbCache.FindByID(ret.ParentAPIUser, a.APIUserID); err != nil {
		return
	}

	err = ret.ParentAPIUser.Prepare(dbCache)

	if err != nil {
		return
	}

	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	return ret, err
}

// HasHappened is a helper function that will return true if the value
// of the timestamp who's name is passes has happened after the revision
// was created. This function is intended to be used with templates.
func (a APIUserRevision) HasHappened(actionType string) bool {
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
func (a *APIUserRevision) GetAllPage(dbCache *DBCache, _ string, _ string) (arop RegistrarObjectPage, err error) {
	ret := &APIUserRevisionsPage{}

	err = dbCache.FindAll(&ret.APIUserRevisions)

	return ret, err
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the db query into the dbcache object.
func (a *APIUserRevision) Prepare(dbCache *DBCache) (err error) {
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
func (a *APIUserRevision) Cancel(dbCache *DBCache, conf Config) (errs []error) {
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

		parent := APIUser{}

		if err := dbCache.FindByID(&parent, a.APIUserID); err != nil {
			errs = append(errs, err)
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
func (a *APIUserRevision) IsCancelled() bool {
	return a.RevisionState == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (a *APIUserRevision) IsEditable() bool {
	return a.RevisionState == StateNew
}

// IsDesiredState will return true iff the state passed in matches the
// desired state of the revision.
func (a APIUserRevision) IsDesiredState(state string) bool {
	return a.DesiredState == state
}

// GetActions will return a list of possible actions that can be taken
// while in the current state.
//
// TODO: handle all states.
func (a *APIUserRevision) GetActions(isSelf bool) map[string]string {
	ret := make(map[string]string)

	if a.RevisionState == StateNew {
		ret["Start Approval Process"] = fmt.Sprintf("/action/%s/%d/startapproval", APIUserRevisionType, a.ID)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", APIUserRevisionType, a.ID)

		if isSelf {
			ret["View Parent API User"] = fmt.Sprintf("/view/%s/%d", APIUserType, a.APIUserID)
		} else {
			ret["View/Edit API User Revision"] = fmt.Sprintf("/view/%s/%d", APIUserRevisionType, a.ID)
		}
	}

	if a.RevisionState == StatePendingApproval {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", APIUserRevisionType, a.ID, APIUserRevisionActionGOTOChangeRequest)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", APIUserRevisionType, a.ID)

		if isSelf {
			ret["View Parent API User"] = fmt.Sprintf("/view/%s/%d", APIUserType, a.APIUserID)
		}
	}

	if a.RevisionState == StateCancelled {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", APIUserRevisionType, a.ID, APIUserRevisionActionGOTOChangeRequest)

		if isSelf {
			ret["View Parent API User"] = fmt.Sprintf("/view/%s/%d", APIUserType, a.APIUserID)
		}
	}

	return ret
}

// APIUserRevisionActionGOTOChangeRequest is the name of the action that
// will trigger a redirect to the current CR of an object.
const APIUserRevisionActionGOTOChangeRequest string = "gotochangerequest"

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (a *APIUserRevision) TakeAction(responseWriter http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, validCSRF bool, authMethod AuthType, conf Config) (errs []error) {
	switch actionName {
	case ActionCancel:
		// TODO: Figure out who can cancel
		if validCSRF {
			cancelErrs1 := a.Cancel(dbCache, conf)
			if cancelErrs1 != nil {
				errs = append(errs, cancelErrs1...)

				return errs
			}

			http.Redirect(responseWriter, request, fmt.Sprintf("/view/%s/%d", APIUserRevisionType, a.ID), http.StatusFound)

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

			http.Redirect(responseWriter, request, fmt.Sprintf("/view/%s/%d", APIUserRevisionType, a.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ApproverRevisionActionGOTOChangeRequest:
		if a.CRID.Valid {
			http.Redirect(responseWriter, request, fmt.Sprintf("/view/%s/%d", ChangeRequestType, a.CRID.Int64), http.StatusFound)

			return errs
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(responseWriter, GenerateObjectResponse(a.GetExportVersion()))

			return errs
		}
	default:
		errs = append(errs, fmt.Errorf("unknown action %s for %s", actionName, APIUserRevisionType))

		return errs
	}

	errs = append(errs, errors.New("unable to take action"))

	return errs
}

// Promote will mark an ApproverRevision as the current revision for an
// Approver if it has not been cancelled or failed approval.
func (a *APIUserRevision) Promote(dbCache *DBCache) (err error) {
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
func (a *APIUserRevision) Supersed(dbCache *DBCache) (err error) {
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

	err = dbCache.Save(a)

	return err
}

// Decline will mark an ApproverRevision as decline for an Approver.
func (a *APIUserRevision) Decline(dbCache *DBCache) (err error) {
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
	if err = dbCache.Save(a); err != nil {
		return err
	}

	return
}

// GetCertificate takes a byte string containing a PEM encoded
// certificate and returns returnes an x509.certificate object. If there
// is a problem parsing the certificate then an error is returned.
func GetCertificate(pemEncoded []byte) (cert *x509.Certificate, err error) {
	block, _ := pem.Decode(pemEncoded)

	if block == nil {
		err = errors.New("failed to parse certificate PEM")

		return cert, err
	}

	cert, err = x509.ParseCertificate(block.Bytes)

	if err != nil {
		return cert, fmt.Errorf("error parsing certificate: %w", err)
	}

	return cert, nil
}

// GetCertificate returnes an x509.certificate object for a API User
// Revision if it is set. If there is a problem parsing the certificate,
// no certificate is set or if the APIUserRevision object is not set
// then an error is returned.
func (a *APIUserRevision) GetCertificate() (cert *x509.Certificate, err error) {
	if a.prepared {
		cert, err = GetCertificate([]byte(a.Certificate))
	} else {
		err = errors.New("APIUserRevision object was not prepared")
	}

	return cert, err
}

// GetCertificateSerial takes a byte string containing a PEM encoded
// certificate and returns the serial number of the certificate if
// it can read the certificate. If there is an error reading the
// certificate then an error is returned.
func GetCertificateSerial(pemEncoded []byte) (serial string, err error) {
	var cert *x509.Certificate

	if cert, err = GetCertificate(pemEncoded); err != nil {
		return
	}

	serial = fmt.Sprintf("%d", cert.SerialNumber)

	return
}

// MigrateDBAPIUserRevision will run the automigrate function for the
// APIUserRevision object.
func MigrateDBAPIUserRevision(dbCache *DBCache) {
	dbCache.AutoMigrate(&APIUserRevision{})
}
