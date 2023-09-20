package lib

import (
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aryann/difflib"
	"github.com/jinzhu/gorm"
)

var (
	// ErrIDNotSet indicates that an object ID was not set when it should be.
	ErrIDNotSet = errors.New("ID not set")

	// ErrAPIUserMayNotBeDirectlyUpdated indicates that the APIUser object may not
	// be directly updated.
	ErrAPIUserMayNotBeDirectlyUpdated = errors.New("APIUser may not be directly updated")
)

// APIUser is an object that represents a API account that may be used
// to query the registrar system without being an authenticated
// employee.
type APIUser struct {
	Model
	State string `json:"state"`

	CurrentRevision   APIUserRevision   `json:"currentRevision"`
	CurrentRevisionID sql.NullInt64     `json:"currentRevisionID"`
	Revisions         []APIUserRevision `json:"revision"`
	PendingRevision   APIUserRevision   `json:"pendingRevision"   sql:"-"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

// APIUserExportFull is an object that is used to export the current
// state of an API User object. The full version of the export object
// also contains the current and pending revision (if either exists).
type APIUserExportFull struct {
	ID    int64  `json:"ID"`
	State string `json:"state"`

	CurrentRevision APIUserRevisionExport `json:"currentRevision"`
	PendingRevision APIUserRevisionExport `json:"pendingRevision"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
}

// GetDiff will return a string containing a formatted diff of the
// current and pending revisions for the APIUser object. An empty
// string and an error are returned if an error occures during the
// processing.
//
// TODO: Handle diff for objects that do not have a pending revision.
// TODO: Handle diff for objects that do not have a current revision.
func (a APIUserExportFull) GetDiff() (string, error) {
	current, _ := a.CurrentRevision.ToJSON()

	pending, err2 := a.PendingRevision.ToJSON()
	if err2 != nil {
		return "", err2
	}

	output := difflib.Diff(strings.Split(current, "\n"), strings.Split(pending, "\n"))

	outputText := ""

	for _, line := range output {
		outputText = fmt.Sprintf("%s\n%s", outputText, line.String())
	}

	return outputText, nil
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (a APIUserExportFull) ToJSON() (string, error) {
	if a.ID <= 0 {
		return "", ErrIDNotSet
	}

	byteArr, jsonErr := json.MarshalIndent(a, "", "  ")

	return string(byteArr), jsonErr
}

// APIUserExportShort is an object that is used to export the current
// state of an APIUser object. The short version of the export object
// does not contain the current or pending revision.
type APIUserExportShort struct {
	ID    int64  `json:"ID"`
	State string `json:"state"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (a APIUserExportShort) ToJSON() (string, error) {
	byteArr, err := json.MarshalIndent(a, "", "  ")

	return string(byteArr), err
}

// APIUserPage is used to hold all the information required to render
// the APIUser HTML template.
type APIUserPage struct {
	Editable            bool
	IsNew               bool
	App                 APIUser
	CurrentRevisionPage *APIUserRevisionPage
	PendingRevisionPage *APIUserRevisionPage
	PendingActions      map[string]string
	ValidApproverSets   map[int64]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *APIUserPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *APIUserPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
	a.PendingRevisionPage.CSRFToken = newToken
}

// APIUsersPage is used to render the html template which lists all of
// the APIUsers currently in the registrar system.
//
// TODO: Add paging support.
// TODO: Add filtering support.
type APIUsersPage struct {
	APIUsers []APIUser

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *APIUsersPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *APIUsersPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// GetExportVersion returns an export version of the APIUser Object.
func (a *APIUser) GetExportVersion() RegistrarObjectExport {
	export := APIUserExportFull{
		ID:              a.ID,
		State:           a.State,
		PendingRevision: (a.PendingRevision.GetExportVersion()).(APIUserRevisionExport),
		CurrentRevision: (a.CurrentRevision.GetExportVersion()).(APIUserRevisionExport),
		CreatedAt:       a.CreatedAt,
		CreatedBy:       a.CreatedBy,
	}

	return export
}

// GetExportVersionAt returns an export version of the APIUser Object
// at the timestamp provided if possible otherwise an error is returned.
// If a pending version existed at the time it will be excluded from
// the object.
func (a *APIUser) GetExportVersionAt(dbCache *DBCache, timestamp int64) (obj RegistrarObjectExport, err error) {
	aur := APIUserRevision{}

	// Grab the most recent promoted object before the time provided
	// where the promoted time is after the time stated above
	// If no objects were found meeting the criteria, return an error
	if err = dbCache.GetRevisionAtTime(&aur, a.ID, timestamp); err != nil {
		return
	}

	// Otherwise, prepare the new object, fix the currnet object a little
	// and remove the pending revision if any
	if err = aur.Prepare(dbCache); err != nil {
		return
	}

	a.CurrentRevision = aur
	a.CurrentRevisionID.Int64 = aur.ID
	a.CurrentRevisionID.Valid = true
	a.PendingRevision = APIUserRevision{}

	// Return the export version and no error
	obj = a.GetExportVersion()

	return
}

// GetExportShortVersion returns an export version of the APIUser
// Object in its short form.
func (a *APIUser) GetExportShortVersion() APIUserExportShort {
	export := APIUserExportShort{
		ID:        a.ID,
		State:     a.State,
		CreatedAt: a.CreatedAt,
		CreatedBy: a.CreatedBy,
	}

	return export
}

// HasRevision returns true iff a current revision exists, otherwise
// false.
//
// TODO: add a check to verify that the current revision has an approved
// change request.
func (a APIUser) HasRevision() bool {
	return a.CurrentRevisionID.Valid
}

// HasPendingRevision returns true iff a pending revision exists for the
// Approver, otherwise false.
func (a APIUser) HasPendingRevision() bool {
	return a.PendingRevision.ID != 0
}

// APIUserName is the name that can be used to reference the Name field
// of the current apiuser revision.
const APIUserName string = "Name"

// APIUserDescription is the name that can be used to reference the
// Description field of the current apiuser revision.
const APIUserDescription string = "Description"

// APIUserCertificate is the name that can be used to reference the
// Certificate field of the current apiuser revision.
const APIUserCertificate string = "Certificate"

// APIUserSerial is the name that can be used to reference the
// Serial field of the current apiuser revision.
const APIUserSerial string = "Serial"

// SuggestedRevisionValue takes a string naming the field that is being
// requested and returns a string containing the suggested value for
// the field in a new pending revision.
//
// TODO: add other fields that have been added.
func (a APIUser) SuggestedRevisionValue(field string) string {
	if a.CurrentRevisionID.Valid {
		switch field {
		case APIUserName:
			return a.CurrentRevision.Name
		case APIUserDescription:
			return a.CurrentRevision.Description
		case APIUserCertificate:
			return a.CurrentRevision.Certificate
		case APIUserSerial:
			return a.CurrentRevision.Serial
		case SavedObjectNote:
			return a.CurrentRevision.SavedNotes
		}
	}

	return ""
}

// SuggestedRevisionBool takes a string naming the flag that is being
// requested and returnes a bool containing the suggested value for the
// field in the new revision.
//
// TODO: add other fields that have been added.
func (a APIUser) SuggestedRevisionBool(field string) bool {
	if a.CurrentRevisionID.Valid {
		switch field {
		case DesiredStateActive:
			return a.CurrentRevision.DesiredState == StateActive
		case DesiredStateInactive:
			return a.CurrentRevision.DesiredState == StateInactive
		case "IsAdmin":
			return a.CurrentRevision.IsAdmin
		case "IsEPPClient":
			return a.CurrentRevision.IsEPPClient
		}
	}

	return false
}

// UnPreparedAPIUserError is the text of an error that is displayed
// when a apiuser has not been prepared before use.
const UnPreparedAPIUserError = "Error: APIUser Not Prepared"

// GetCurrentValue is used to get the current value of a field in a
// revision if a current revision exists, otherwise an empty string is
// returned.
func (a *APIUser) GetCurrentValue(field string) (ret string) {
	if !a.prepared {
		return UnPreparedAPIUserError
	}

	return a.SuggestedRevisionValue(field)
}

// GetDisplayName will return a name for the APIUser that can be
// used to display a shortened version of the invormation to users.
func (a *APIUser) GetDisplayName() string {
	return fmt.Sprintf("%d - %s", a.ID, a.GetCurrentValue(APIUserName))
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
func (a *APIUser) ParseFromForm(request *http.Request, _ *DBCache) error {
	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	a.State = StateNew
	a.CreatedBy = runame
	a.UpdatedBy = runame

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse APIUser object with the changes that were
// made. An error object is always the second return value which is nil
// when no errors have occurred during parsing otherwise an error is
// returned.
func (a *APIUser) ParseFromFormUpdate(_ *http.Request, _ *DBCache, _ Config) error {
	return ErrAPIUserMayNotBeDirectlyUpdated
}

// VerifyCR Checks to make sure that all of the values and approvals
// within a change request match the approver that it is linked to.
func (a *APIUser) VerifyCR(dbCache *DBCache) (checksOut bool, errs []error) {
	return VerifyCR(dbCache, a, nil)
}

// GetCurrentRevisionID will return the id of the current API User
// Revision for the api user object.
func (a *APIUser) GetCurrentRevisionID() sql.NullInt64 {
	return a.CurrentRevisionID
}

// GetPendingRevisionID will return the current pending revision for the
// API User object if it exists. If no pending revision exists a 0 is
// returned.
func (a *APIUser) GetPendingRevisionID() int64 {
	return a.PendingRevision.ID
}

// GetPendingCRID will return the current CR id if it is set, otherwise
// a nil will be returned (in the form of a sql.NullInt64).
func (a *APIUser) GetPendingCRID() sql.NullInt64 {
	return a.PendingRevision.CRID
}

// ComparePendingToCallback will return a function that will compare the
// current revision object to itself after changes have been made.
func (a *APIUser) ComparePendingToCallback(loadFn CompareLoadFn) (retFn CompareReturnFn) {
	exp := APIUserExportFull{}
	loadFn(&exp)

	return func() (pass bool, errs []error) {
		return exp.PendingRevision.Compare(a.PendingRevision)
	}
}

// IsAdmin will return true if the current revision has the user marked as an
// admin and false otherwise. If there is an error retireveing if the user is
// an admin it will be returned.
func (a *APIUser) IsAdmin(dbCache *DBCache) (isAdmin bool, err error) {
	err = a.Prepare(dbCache)
	if err != nil {
		return false, err
	}

	if a.CurrentRevisionID.Valid && a.CurrentRevisionID.Int64 != 0 {
		return a.CurrentRevision.IsAdmin, nil
	}

	return false, ErrNoCurrentRevision
}

// UpdateState can be called at any point to check the state of the
// APIUser and update it if necessary.
//
// TODO: Implement.
// TODO: Make sure callers check errors.
func (a *APIUser) UpdateState(dbCache *DBCache, _ Config) (changesMade bool, errs []error) {
	logger.Infof("UpdateState called on APIUser %d (%s)", a.ID, a.State)

	changesMade = false

	if err := a.Prepare(dbCache); err != nil {
		errs = append(errs, err)

		return changesMade, errs
	}

	switch a.State {
	case StateNew, StateInactive, StateActive:
		logger.Infof("UpdateState for APIUser at state \"%s\", Nothing to do", a.State)
	case StatePendingBootstrap, StatePendingNew, StateActivePendingApproval, StateInactivePendingApproval:
		logger.Infof("UpdateState called on APIUser %d pending PRID %d CRID %+v", a.ID, a.PendingRevision.ID, a.PendingRevision.CRID)

		if a.PendingRevision.ID != 0 && a.PendingRevision.CRID.Valid {
			changeRequest := ChangeRequest{}

			if err := dbCache.FindByID(&changeRequest, a.PendingRevision.CRID.Int64); err != nil {
				errs = append(errs, err)
			}

			crChecksOut, crcheckErrs := VerifyCR(dbCache, a, &changeRequest)

			if len(crcheckErrs) != 0 {
				errs = append(errs, crcheckErrs...)
			}

			if crChecksOut {
				if changeRequest.State == StateApproved {
					// Promote the new revision
					targetState := a.PendingRevision.DesiredState

					if err := a.PendingRevision.Promote(dbCache); err != nil {
						errs = append(errs, fmt.Errorf("error promoting revision: %w", err))

						return changesMade, errs
					}

					if a.CurrentRevisionID.Valid {
						if err := a.CurrentRevision.Supersed(dbCache); err != nil {
							errs = append(errs, fmt.Errorf("error superseding revision: %w", err))

							return changesMade, errs
						}
					}

					newAPI := APIUser{Model: Model{ID: a.ID}}

					if err := newAPI.PrepareShallow(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if targetState == StateActive || targetState == StateInactive {
						newAPI.State = targetState
					} else {
						errs = append(errs, ErrPendingRevisionInInvalidState)
						logger.Errorf("Unexpected target state: %s", targetState)

						return changesMade, errs
					}

					var revision sql.NullInt64
					revision.Int64 = a.PendingRevision.ID
					revision.Valid = true
					newAPI.CurrentRevisionID = revision

					if err := dbCache.Save(&newAPI); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}
				} else if changeRequest.State == StateDeclined {
					logger.Infof("CR %d has been declined", changeRequest.GetID())
					if err := a.PendingRevision.Decline(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					newAPI := APIUser{}
					if err := dbCache.FindByID(&newAPI, a.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if a.CurrentRevisionID.Valid {
						curRev := APIUserRevision{}
						if err := dbCache.FindByID(&curRev, a.CurrentRevisionID.Int64); err != nil {
							errs = append(errs, err)

							return changesMade, errs
						}
						if curRev.RevisionState == StateBootstrap {
							newAPI.State = StateActive
						} else {
							newAPI.State = curRev.RevisionState
						}
					} else {
						newAPI.State = StateNew
					}

					if err := dbCache.Save(&newAPI); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}
				} else {
					errs = append(errs, ErrUnableToHandleState)
					logger.Errorf("unknown state: %s", changeRequest.State)

					return changesMade, errs
				}
			} else {
				return changesMade, errs
			}
		} else {
			// If a pending revision was not found then we have to go back to
			// either active, inactive or bootstrap depending on the state
			if a.State == StatePendingBootstrap {
				a.State = StateBootstrap
				changesMade = true
			} else if a.CurrentRevisionID.Valid {
				a.State = a.CurrentRevision.DesiredState
				changesMade = true
			} else {
				a.State = StateNew
				changesMade = true
			}
		}

	default:
		errs = append(errs, ErrUpdateStateNotImplemented)

		logger.Errorf("UpdateState for %t at state %s not implemented", a, a.State)
	}

	if changesMade {
		if err := dbCache.Save(a); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}
	}

	return changesMade, errs
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (a *APIUser) GetType() string {
	return APIUserType
}

// IsCancelled returns true iff the object has been canclled.
func (a *APIUser) IsCancelled() bool {
	return a.State == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (a *APIUser) IsEditable() bool {
	return a.State == StateNew
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
func (a *APIUser) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, a, func() (err error) {
		// If there is a current revision, load the revision into the
		// CurrentRevision field
		if a.CurrentRevisionID.Valid {
			a.CurrentRevision = APIUserRevision{}
			if err = dbCache.FindByID(&a.CurrentRevision, a.CurrentRevisionID.Int64); err != nil {
				return
			}
		}

		// Grab the pending revison if it exists and prepare the revision
		if err = dbCache.GetNewAndPendingRevisions(a); err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return nil
			}

			return
		}

		err = a.PendingRevision.Prepare(dbCache)

		return
	})
}

// PrepareShallow populates all of the fields for the given object and
// not any of the linked objects.
func (a *APIUser) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, a, FuncNOPErrFunc)
}

// GetPendingRevision implements the RegistrarParent interface and returns
// the pending revision pointer.
func (a *APIUser) GetPendingRevision() RegistrarObject {
	return &a.PendingRevision
}

// GetPage will return an object that can be used to render the HTML
// template for the APIUser.
func (a *APIUser) GetPage(dbCache *DBCache, username string, email string) (rop RegistrarObjectPage, err error) {
	ret := &APIUserPage{Editable: true, IsNew: true}

	if a.ID != 0 {
		ret.Editable = false
		ret.IsNew = false
	}

	ret.App = *a
	ret.PendingActions = make(map[string]string)
	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	if err != nil {
		return ret, err
	}

	if a.PendingRevision.ID != 0 {
		ret.PendingActions = a.PendingRevision.GetActions(false)
	}

	if a.CurrentRevisionID.Valid {
		rawPage, rawPageErr := a.CurrentRevision.GetPage(dbCache, username, email)
		ret.CurrentRevisionPage = rawPage.(*APIUserRevisionPage)

		if rawPageErr != nil {
			return ret, rawPageErr
		}
	}

	if a.HasPendingRevision() {
		rawPage, rawPageErr := a.PendingRevision.GetPage(dbCache, username, email)
		ret.PendingRevisionPage = rawPage.(*APIUserRevisionPage)

		if rawPageErr != nil {
			return ret, rawPageErr
		}
	} else {
		ret.PendingRevisionPage = &APIUserRevisionPage{IsEditable: true, IsNew: true, ValidApproverSets: ret.ValidApproverSets}
	}

	ret.PendingRevisionPage.ParentAPIUser = a
	ret.PendingRevisionPage.SuggestedRequiredApprovers = make(map[int64]ApproverSetDisplayObject)
	ret.PendingRevisionPage.SuggestedInformedApprovers = make(map[int64]ApproverSetDisplayObject)

	if a.CurrentRevision.ID != 0 {
		for _, appSet := range a.CurrentRevision.RequiredApproverSets {
			ret.PendingRevisionPage.SuggestedRequiredApprovers[appSet.ID] = appSet.GetDisplayObject()
		}

		for _, appSet := range a.CurrentRevision.InformedApproverSets {
			ret.PendingRevisionPage.SuggestedInformedApprovers[appSet.ID] = appSet.GetDisplayObject()
		}
	} else {
		appSet, prepErr := GetDefaultApproverSet(dbCache)

		if prepErr == nil {
			ret.PendingRevisionPage.SuggestedRequiredApprovers[1] = appSet.GetDisplayObject()
		} else {
			return ret, ErrNoDefaultApprover
		}
	}

	return ret, nil
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple approvers.
//
// TODO: Add paging support.
// TODO: Add filtering.
func (a *APIUser) GetAllPage(dbCache *DBCache, _ string, _ string) (aop RegistrarObjectPage, err error) {
	ret := &APIUsersPage{}

	err = dbCache.FindAll(&ret.APIUsers)

	if err != nil {
		return
	}

	for idx := range ret.APIUsers {
		err = ret.APIUsers[idx].Prepare(dbCache)
		if err != nil {
			return
		}
	}

	return ret, nil
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (a *APIUser) TakeAction(response http.ResponseWriter, _ *http.Request, _ *DBCache, actionName string, _ bool, authMethod AuthType, _ Config) (errs []error) {
	switch actionName {
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(a.GetExportVersion()))

			return errs
		}
	}

	return errs
}

// GetRequiredApproverSets returns the list of approver sets that are
// required for the APIUser (if a valid approver revision exists). If
// no APIUser revisions are found, a default of the infosec approver
// set will be returned.
func (a *APIUser) GetRequiredApproverSets(dbCache *DBCache) (approvers []ApproverSet, err error) {
	return GetRequiredApproverSets(dbCache, a)
}

// GetInformedApproverSets returns the list of approver sets that are
// informed for the APIUser (if a valid approver revision exists). If
// no approver revisions are found, an empty list will be returned.
func (a *APIUser) GetInformedApproverSets(dbCache *DBCache) (as []ApproverSet, err error) {
	return GetInformedApproverSets(dbCache, a)
}

// GetCertName returnes a string that will identify an API user like a
// regular user.
func (a *APIUser) GetCertName() string {
	version := "nocurrentrevision"

	if a.CurrentRevisionID.Valid {
		version = fmt.Sprintf("%d", a.CurrentRevisionID.Int64)
	}

	return fmt.Sprintf("apiuser%d-%s", a.ID, version)
}

// GetAPIUserFromPEM takes a byte string containing a PEM encoded
// certificate and returns the API User associated with the certificate
// if there is only one API user that matches. If there is more than
// one API User with a matching key or no users, an error is returned
// and a nil object is returend.
//
// TODO: Consider moving the db query into the dbcache object.
func GetAPIUserFromPEM(dbCache *DBCache, pemCert []byte) (user *APIUser, err error) {
	var cert *x509.Certificate

	var revs []APIUserRevision

	var candidateRevs []APIUserRevision

	if cert, err = GetCertificate(pemCert); err != nil {
		return user, err
	}

	serial := fmt.Sprintf("%d", cert.SerialNumber)

	if err = dbCache.DB.Where("serial = ? and revision_state in (?)", serial, StateActive).Find(&revs).Error; err != nil {
		return user, err
	}

	for _, rev := range revs {
		if err = rev.Prepare(dbCache); err != nil {
			return user, err
		}

		revCert, revCertErr := rev.GetCertificate()

		if revCertErr == nil {
			if revCert.Equal(cert) {
				candidateRevs = append(candidateRevs, rev)
			}
		} else {
			logger.Error(revCertErr.Error())
		}
	}

	if len(candidateRevs) > 1 {
		err = ErrExcessAPIKeysLocated
	} else if len(candidateRevs) == 1 {
		user = &APIUser{}

		if err = dbCache.FindByID(user, candidateRevs[0].APIUserID); err != nil {
			return user, err
		}
	} else {
		err = ErrNoAPIUsersForKey
	}

	return user, err
}

// MigrateDBAPIUser will run the automigrate function for the Approver
// object.
func MigrateDBAPIUser(dbCache *DBCache) {
	dbCache.AutoMigrate(&APIUser{})
}
