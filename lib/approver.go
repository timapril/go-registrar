// Package lib provides the objects required to operate registrar
package lib

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/aryann/difflib"
	"github.com/jinzhu/gorm"
)

// More information about the Approver Object and its States can be
// found in /doc/approver.md.

// Approver is an object that represents an individual that may be used
// to approve changes within the registrar system.
type Approver struct {
	Model
	State string `json:"State"`

	CurrentRevision   ApproverRevision   `json:"CurrentRevision"`
	CurrentRevisionID sql.NullInt64      `json:"CurrentRevisionID"`
	Revisions         []ApproverRevision `json:"Revisions"`
	PendingRevision   ApproverRevision   `json:"PendingRevision"   sql:"-"`

	ApproverSetRevisions []ApproverSetRevision `gorm:"many2many:approver_to_revision_set;" json:"ApproverSetRevisions"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`
}

// ApproverExportFull is an object that is used to export the current
// state of an approver object. The full version of the export object
// also contains the current and pending revision (if either exist).
type ApproverExportFull struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	CurrentRevision ApproverRevisionExport `json:"CurrentRevision"`
	PendingRevision ApproverRevisionExport `json:"PendingRevision"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// GetDiff will return a string containing a formatted diff of the
// current and pending revisions for the Approver object. An empty
// string and an error are returned if an error occures during the
// processing.
//
// TODO: Handle diff for objects that do not have a pending revision.
// TODO: Handle diff for objects that do not have a current revision.
func (a ApproverExportFull) GetDiff() (string, error) {
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
func (a ApproverExportFull) ToJSON() (string, error) {
	if a.ID <= 0 {
		return "", errors.New("ID not set")
	}

	byteArr, jsonErr := json.MarshalIndent(a, "", "  ")

	return string(byteArr), jsonErr
}

// ApproverExportShort is an object that is used to export the current
// state of an approver object. The short version of the export object
// does not contain the current or pending revision.
//
// TODO: Candidate for removal.
type ApproverExportShort struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (a ApproverExportShort) ToJSON() (string, error) {
	byteArr, err := json.MarshalIndent(a, "", "  ")

	return string(byteArr), err
}

// ApproverPage is used to hold all the information required to render
// the Appprover HTML template.
type ApproverPage struct {
	Editable            bool
	IsNew               bool
	App                 Approver
	CurrentRevisionPage *ApproverRevisionPage
	PendingRevisionPage *ApproverRevisionPage
	PendingActions      map[string]string
	ValidApproverSets   map[int64]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApproverPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApproverPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
	a.PendingRevisionPage.CSRFToken = newToken
}

// ApproversPage is used to render the html template which lists all of
// the Approvers currently in the registrar system.
//
// TODO: Add paging support.
// TODO: Add filtering support.
type ApproversPage struct {
	Approvers []Approver

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApproversPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApproversPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// GetExportVersion returns an export version of the Approver Object.
func (a *Approver) GetExportVersion() RegistrarObjectExport {
	export := ApproverExportFull{
		ID:              a.ID,
		State:           a.State,
		PendingRevision: (a.PendingRevision.GetExportVersion()).(ApproverRevisionExport),
		CurrentRevision: (a.CurrentRevision.GetExportVersion()).(ApproverRevisionExport),
		CreatedAt:       a.CreatedAt,
		CreatedBy:       a.CreatedBy,
	}

	return export
}

// GetExportVersionAt returns an export version of the Approver Object
// at the timestamp provided if possible otherwise an error is returned.
// If a pending version existed at the time it will be excluded from
// the object.
// TODO: Implement.
func (a *Approver) GetExportVersionAt(dbCache *DBCache, timestamp int64) (obj RegistrarObjectExport, err error) {
	approverRevision := ApproverRevision{}

	// Grab the most recent promoted object before the time provided
	// where the promoted time is after the time stated above
	if err = dbCache.GetRevisionAtTime(&approverRevision, a.ID, timestamp); err != nil {
		return
	}

	// Otherwise, prepare the new object, fix the currnet object a little
	// and remove the pending revision if any
	if err = approverRevision.Prepare(dbCache); err != nil {
		return
	}

	a.CurrentRevision = approverRevision
	a.CurrentRevisionID.Int64 = approverRevision.ID
	a.CurrentRevisionID.Valid = true
	a.PendingRevision = ApproverRevision{}

	// Return the export version and no error
	return a.GetExportVersion(), nil
}

// GetExportShortVersion returns an export version of the Approver
// Object in its short form.
func (a *Approver) GetExportShortVersion() ApproverExportShort {
	export := ApproverExportShort{
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
func (a Approver) HasRevision() bool {
	return a.CurrentRevisionID.Valid
}

// HasPendingRevision returns true iff a pending revision exists for the
// Approver, otherwise false.
func (a Approver) HasPendingRevision() bool {
	return a.PendingRevision.ID != 0
}

// SuggestedRevisionValue takes a string naming the field that is being
// requested and returns a string containing the suggested value for
// the field in a new pending revision.
//
// TODO: add other fields that have been added.
func (a Approver) SuggestedRevisionValue(field string) string {
	if a.CurrentRevisionID.Valid {
		switch field {
		case ApproverFieldName:
			return a.CurrentRevision.Name
		case ApproverFieldUsername:
			return a.CurrentRevision.Username
		case ApproverFieldEmployeeID:
			return fmt.Sprintf("%d", a.CurrentRevision.EmployeeID)
		case ApproverFieldDepartment:
			return a.CurrentRevision.Department
		case ApproverFieldFingerprint:
			return a.CurrentRevision.Fingerprint
		case ApproverFieldPublicKey:
			return a.CurrentRevision.PublicKey
		case ApproverFieldEmailAddres:
			return a.CurrentRevision.EmailAddress
		case ApproverFieldRole:
			return a.CurrentRevision.Role
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
func (a Approver) SuggestedRevisionBool(field string) bool {
	if a.CurrentRevisionID.Valid {
		switch field {
		case DesiredStateActive:
			return a.CurrentRevision.DesiredState == StateActive
		case DesiredStateInactive:
			return a.CurrentRevision.DesiredState == StateInactive
		case "IsAdmin":
			return a.CurrentRevision.IsAdmin
		}
	}

	return false
}

// UnPreparedApproverError is the text of an error that is displayed
// when a approver has not been prepared before use.
const UnPreparedApproverError = "Error: Approver Not Prepared"

// GetCurrentValue is used to get the current value of a field in a
// revision if a current revision exists, otherwise an empty string is
// returned.
func (a *Approver) GetCurrentValue(field string) (ret string) {
	if !a.prepared {
		return UnPreparedApproverError
	}

	return a.SuggestedRevisionValue(field)
}

// GetDisplayName will return a name for the Approver that can be
// used to display a shortened version of the invormation to users.
func (a *Approver) GetDisplayName() string {
	return fmt.Sprintf("%d - %s (%s)", a.ID, a.GetCurrentValue(ApproverFieldEmailAddres), a.GetCurrentValue(ApproverFieldRole))
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
func (a *Approver) ParseFromForm(request *http.Request, _ *DBCache) error {
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
// and returns a sparse Approver object with the changes that were
// made. An error object is always the second return value which is nil
// when no errors have occurred during parsing otherwise an error is
// returned.
func (a *Approver) ParseFromFormUpdate(_ *http.Request, _ *DBCache, _ Config) error {
	return errors.New("approvers may not be directly updated")
}

// VerifyCR Checks to make sure that all of the values and approvals
// within a change request match the approver that it is linked to.
//
// TODO: more rigirous check on if the CR approved text matches.
func (a *Approver) VerifyCR(dbCache *DBCache) (checksOut bool, errs []error) {
	return VerifyCR(dbCache, a, nil)
}

// GetCurrentRevisionID will return the id of the current Approver
// Revision for the Approver object.
func (a *Approver) GetCurrentRevisionID() sql.NullInt64 {
	return a.CurrentRevisionID
}

// GetPendingRevisionID will return the current pending revision for the
// Approver object if it exists. If no pending revision exists a 0 is
// returned.
func (a *Approver) GetPendingRevisionID() int64 {
	return a.PendingRevision.ID
}

// GetPendingCRID will return the current CR id if it is set, otherwise
// a nil will be returned (in the form of a sql.NullInt64).
func (a *Approver) GetPendingCRID() sql.NullInt64 {
	return a.PendingRevision.CRID
}

// ComparePendingToCallback will return a function that will compare the
// current revision object to itself after changes have been made.
func (a *Approver) ComparePendingToCallback(loadFn CompareLoadFn) (retFn CompareReturnFn) {
	exp := ApproverExportFull{}

	loadFn(&exp)

	return func() (pass bool, errs []error) {
		return exp.PendingRevision.Compare(a.PendingRevision)
	}
}

// UpdateState can be called at any point to check the state of the
// Approver and update it if necessary.
//
// TODO: Implement.
// TODO: Make sure callers check errors.
func (a *Approver) UpdateState(dbCache *DBCache, _ Config) (changesMade bool, errs []error) {
	logger.Infof("UpdateState called on Approver %d (todo)", a.ID)

	if err := a.Prepare(dbCache); err != nil {
		errs = append(errs, err)

		return changesMade, errs
	}

	changesMade = false

	switch a.State {
	case StateNew, StateInactive, StateActive:
		logger.Infof("UpdateState for Approver at state \"%s\", Nothing to do", a.State)
	case StatePendingBootstrap, StatePendingNew, StateActivePendingApproval, StateInactivePendingApproval:
		if a.PendingRevision.ID != 0 && a.PendingRevision.CRID.Valid {
			crChecksOut, crcheckErrs := a.VerifyCR(dbCache)

			if len(crcheckErrs) != 0 {
				errs = append(errs, crcheckErrs...)
			}

			if crChecksOut {
				changeRequest := ChangeRequest{}
				if err := dbCache.FindByID(&changeRequest, a.PendingRevision.CRID.Int64); err != nil {
					errs = append(errs, err)

					return changesMade, errs
				}

				if changeRequest.State == StateApproved {
					// Promote the new revision
					targetState := a.PendingRevision.DesiredState

					if err := a.PendingRevision.Promote(dbCache); err != nil {
						errs = append(errs, fmt.Errorf("error promoting revision: %s", err.Error()))

						return changesMade, errs
					}

					if a.CurrentRevisionID.Valid {
						if err := a.CurrentRevision.Supersed(dbCache); err != nil {
							errs = append(errs, fmt.Errorf("error superseding revision: %s", err.Error()))

							return changesMade, errs
						}
					}

					newApp := Approver{Model: Model{ID: a.ID}}

					if err := newApp.PrepareShallow(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if targetState == StateActive || targetState == StateInactive {
						newApp.State = targetState
					} else {
						errs = append(errs, ErrPendingRevisionInInvalidState)
						logger.Errorf("Unexpected target state: %s", targetState)

						return changesMade, errs
					}

					var revision sql.NullInt64
					revision.Int64 = a.PendingRevision.ID
					revision.Valid = true
					newApp.CurrentRevisionID = revision

					if err := dbCache.Save(&newApp); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}
				} else if changeRequest.State == StateDeclined {
					logger.Infof("CR %d has been declined", changeRequest.GetID())

					if err := a.PendingRevision.Decline(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					newApp := Approver{}

					if err := dbCache.FindByID(&newApp, a.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if a.CurrentRevisionID.Valid {
						curRev := ApproverRevision{}
						if err := dbCache.FindByID(&curRev, a.CurrentRevisionID.Int64); err != nil {
							errs = append(errs, err)

							return changesMade, errs
						}
						if curRev.RevisionState == StateBootstrap {
							newApp.State = StateActive
						} else {
							newApp.State = curRev.RevisionState
						}
					} else {
						newApp.State = StateNew
					}

					if err := dbCache.Save(&newApp); err != nil {
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
		errs = append(errs, fmt.Errorf("UpdateState for Approver at state \"%s\" not implemented", a.State))
	}

	if changesMade {
		if err := dbCache.Save(a); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}
	}

	return changesMade, errs
}

// AfterSave is used to hook specific functions that are required for
// propogating the approval to the change request and attached objects
// after an approval has changed.
func (a *Approver) AfterSave(dbin *gorm.DB) error {
	dbCache := NewDBCache(dbin)

	return a.PostUpdate(&dbCache, GetConfig())
}

// PostUpdate is called on an object following a change to the object.
func (a *Approver) PostUpdate(dbCache *DBCache, conf Config) error {
	appSets, _ := a.GetActiveApproverSets(dbCache)

	for _, appSet := range appSets {
		_, errs := appSet.UpdateState(dbCache, conf)

		if len(errs) != 0 {
			return errs[0]
		}
	}

	return nil
}

// GetActiveApproverSets is used to get a list of ApproverSets that the
// calling approver is a member of which are in an active or inactive
// state.
//
// TODO: Consider moving the query into dbcache.
func (a *Approver) GetActiveApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	err = dbCache.DB.Where(`
	id IN (SELECT approver_set_id
			FROM approver_set_revisions r
			JOIN approver_to_revision_set jt ON (r.id = jt.approver_set_revision_id)
			WHERE r.revision_state IN ('active' , 'inactive') AND jt.approver_id = ?)
	`, a.ID).Find(&approverSets).Error
	if err != nil {
		return
	}

	for _, appset := range approverSets {
		if err = appset.Prepare(dbCache); err != nil {
			return
		}
	}

	return
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (a *Approver) GetType() string {
	return ApproverType
}

// IsCancelled returns true iff the object has been canclled.
func (a *Approver) IsCancelled() bool {
	return a.State == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (a *Approver) IsEditable() bool {
	return a.State == StateNew
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
func (a *Approver) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, a, func() (err error) {
		// If there is a current revision, load the revision into the
		// CurrentRevision field

		if a.CurrentRevisionID.Valid {
			a.CurrentRevision = ApproverRevision{}
			if err = dbCache.FindByID(&a.CurrentRevision, a.CurrentRevisionID.Int64); err != nil {
				// if errors.Is(err, gorm.RecordNotFound) {
				// 	logger.Infof("CurrentRevision %d not found: %s\n", a.CurrentRevisionID.Int64, err.Error())
				// } else {
				// 	logger.Errorf("CurrentRevision %d error: %s\n", a.CurrentRevisionID.Int64, err.Error())
				// 	return
				// }
				return err
			}
			// if err = db.Related(a, &a.CurrentRevision); err != nil {
			// 	return
			// }
			// if err = a.CurrentRevision.Prepare(db); err != nil {
			// 	return
			// }
		}

		// Grab the pending revison if it exists and prepare the revision
		if err = dbCache.GetNewAndPendingRevisions(a); err == nil {
			if err = a.PendingRevision.Prepare(dbCache); err != nil {
				logger.Errorf("PendingRevision.Prepare error: %s\n", err.Error())

				return err
			}
		} else if errors.Is(err, gorm.RecordNotFound) {
			err = nil
		}

		return err
	})
}

// PrepareShallow populates all of the fields for the given object and
// not any of the linked objects.
func (a *Approver) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, a, FuncNOPErrFunc)
}

// GetPendingRevision implements the RegistrarParent interface and returns
// the pending revision pointer.
func (a *Approver) GetPendingRevision() RegistrarObject {
	return &a.PendingRevision
}

// GetPage will return an object that can be used to render the HTML
// template for the Approver.
func (a *Approver) GetPage(dbCache *DBCache, username string, email string) (rop RegistrarObjectPage, err error) {
	ret := &ApproverPage{Editable: true, IsNew: true}

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

		if rawPageErr != nil {
			err = rawPageErr

			return ret, err
		}

		ret.CurrentRevisionPage = rawPage.(*ApproverRevisionPage)
	}

	if a.HasPendingRevision() {
		rawPage, rawPageErr := a.PendingRevision.GetPage(dbCache, username, email)

		if rawPageErr != nil {
			err = rawPageErr

			return ret, err
		}

		ret.PendingRevisionPage = rawPage.(*ApproverRevisionPage)
	} else {
		ret.PendingRevisionPage = &ApproverRevisionPage{IsEditable: true, IsNew: true, ValidApproverSets: ret.ValidApproverSets}
	}

	ret.PendingRevisionPage.ParentApprover = a
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

		if prepErr != nil {
			return ret, errors.New("unable to find approver 1 - database probably not bootstrapped")
		}

		ret.PendingRevisionPage.SuggestedRequiredApprovers[1] = appSet.GetDisplayObject()
	}

	return ret, nil
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple approvers.
//
// TODO: Add paging support.
// TODO: Add filtering.
func (a *Approver) GetAllPage(dbCache *DBCache, _ string, _ string) (RegistrarObjectPage, error) {
	ret := &ApproversPage{}

	err := dbCache.FindAll(&ret.Approvers)
	if err != nil {
		return ret, err
	}

	for idx := range ret.Approvers {
		err = ret.Approvers[idx].Prepare(dbCache)

		if err != nil {
			return ret, err
		}
	}

	return ret, nil
}

// keyToFingerprint takes a GPG key as a string and tries to form a
// hex fingerprint from it similar to the format that the gpg command
// line tool will provide.
func keyToFingerprint(key string) (fingerprint string, err error) {
	decbuf := bytes.NewBufferString(key + "\n")
	block, err1 := armor.Decode(decbuf)

	if err1 != nil {
		return "", fmt.Errorf("error decoding fingerprint: %w", err1)
	}

	packetReader := packet.NewReader(block.Body)
	entity, err2 := openpgp.ReadEntity(packetReader)

	if err2 != nil {
		return "", fmt.Errorf("error reading entity: %w", err2)
	}

	fp := entity.PrimaryKey.Fingerprint
	fingerprint = fmt.Sprintf("%04X %04X %04X %04X %04X  %04X %04X %04X %04X %04X", fp[0:2], fp[2:4], fp[4:6], fp[6:8], fp[8:10], fp[10:12], fp[12:14], fp[14:16], fp[16:18], fp[18:20])

	return fingerprint, nil
}

// GetGPGKeyBlock will return a openpgp.Entity object if there is a key
// that exists in the current revision and the key is able to be parsed.
// If there is an error parsing the key, the error return value will be
// set to a non-nil value.
func (a *Approver) GetGPGKeyBlock() (*openpgp.Entity, error) {
	logger.Debugf("Getting Public Key for Approver %d", a.ID)

	if a.CurrentRevisionID.Valid && (a.CurrentRevision.RevisionState == StateBootstrap || a.CurrentRevision.RevisionState == StatePendingBootstrap || a.CurrentRevision.RevisionState == StateActive) {
		decbuf := bytes.NewBufferString(a.CurrentRevision.PublicKey + "\n")
		block, err1 := armor.Decode(decbuf)

		if err1 != nil {
			logger.Errorf("Error %s", err1.Error())

			return &openpgp.Entity{}, fmt.Errorf("error decoding gpg block: %w", err1)
		}

		packetReader := packet.NewReader(block.Body)
		entity, err2 := openpgp.ReadEntity(packetReader)

		if err2 != nil {
			logger.Errorf("Error %s", err2.Error())

			return &openpgp.Entity{}, fmt.Errorf("error decoding entity: %w", err2)
		}

		return entity, nil
	}

	return &openpgp.Entity{}, errors.New("unable to find active key")
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (a *Approver) TakeAction(response http.ResponseWriter, _ *http.Request, _ *DBCache, actionName string, _ bool, authMethod AuthType, _ Config) (errs []error) {
	switch actionName {
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(a.GetExportVersion()))

			return
		}
	}

	return
}

// GetRequiredApproverSets returns the list of approver sets that are
// required for the Approver (if a valid approver revision exists). If
// no approver revisions are found, a default of the infosec approver
// set will be returned.
func (a *Approver) GetRequiredApproverSets(dbCache *DBCache) (approvers []ApproverSet, err error) {
	return GetRequiredApproverSets(dbCache, a)
}

// GetInformedApproverSets returns the list of approver sets that are
// informed for the Approver (if a valid approver revision exists). If
// no approver revisions are found, an empty list will be returned.
func (a *Approver) GetInformedApproverSets(dbCache *DBCache) ([]ApproverSet, error) {
	return GetInformedApproverSets(dbCache, a)
}

// GetValidApproverMap will return a map containing the Approvers email
// addresses and roles indexed by their Approver ID. Only Approvers
// with the state "active" are returned.
//
// TODO: Consider moving the db query into the dbcache object.
func GetValidApproverMap(dbCache *DBCache) (map[int64]string, error) {
	var apps []Approver

	ret := make(map[int64]string)

	err := dbCache.DB.Where("state = ?", StateActive).Find(&apps).Error
	if err != nil {
		return ret, err
	}

	for _, approver := range apps {
		err = approver.Prepare(dbCache)

		if err != nil {
			return ret, err
		}

		ret[approver.ID] = approver.GetDisplayName()
	}

	return ret, nil
}

// ParseApprovers takes the http Request, a database connection and the
// html ID of the approver list to parse and will return an array of
// Approvers that correspond to each of the id's from the http request's
// html element. If there are unparsable IDs in the list (could be
// strings or empty fields) and error is returned. Any valid approvers
// that were found will be returned in the array even if an error
// occures.
//
// TODO: test for approver IDs that do not exist.
func ParseApprovers(request *http.Request, dbCache *DBCache, htmlID string) ([]Approver, error) {
	unparsableIDs := ""
	failedParsing := false

	var approvers []Approver

	var outErr error

	for _, appidChunk := range request.Form[htmlID] {
		for _, appidRaw := range strings.Split(appidChunk, " ") {
			appid, err := strconv.ParseInt(appidRaw, 10, 64)

			if err == nil {
				tmpApp := Approver{}

				if err := dbCache.FindByID(&tmpApp, appid); err != nil {
					failedParsing = true
					unparsableIDs = fmt.Sprintf("%s %s", unparsableIDs, appidRaw)
				} else {
					approvers = append(approvers, tmpApp)
				}
			} else {
				failedParsing = true
				unparsableIDs = fmt.Sprintf("%s %s", unparsableIDs, appidRaw)
			}
		}
	}

	outErr = nil

	if failedParsing {
		outErr = fmt.Errorf("an err occurred parsing the following approvers in the %s form field: %s", htmlID, unparsableIDs)
	}

	return approvers, outErr
}

// CompareToApproverListExportShortList compares a list of approver sets
// to a set of approver sets that were from an export version of an
// object. If the counts match and the IDs for the aprover sets match
// true is returned otherwise, false is returned.
func CompareToApproverListExportShortList(apps []Approver, appse []ApproverExportShort) bool {
	exportShortCount := len(appse)
	fullCount := len(apps)

	if fullCount != exportShortCount {
		return false
	}

	for _, full := range apps {
		found := false

		for _, export := range appse {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	for _, export := range appse {
		found := false

		for _, full := range apps {
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

// CompareToApproverExportShortLists compares a list of approver export
// objects to another set of approver exports. If the counts match and
// the IDs for the aprover sets match true is returned otherwise, false
// is returned.
func CompareToApproverExportShortLists(apps []ApproverExportShort, appse []ApproverExportShort) bool {
	exportShortCount := len(appse)
	fullCount := len(apps)

	if fullCount != exportShortCount {
		return false
	}

	for _, full := range apps {
		found := false

		for _, export := range appse {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	for _, export := range appse {
		found := false

		for _, full := range apps {
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

// IsAdminUser is used to test if a user is an admin user. Iff the user
// is an admin, true will be returned. If an error occurs during the
// check, it will be returned, otherwise nil is returned
//
// TODO: Consider moving the db query into the dbcache object.
func IsAdminUser(username string, dbCache *DBCache) (bool, error) {
	appRevs := []ApproverRevision{}
	dbCache.DB.Where("username = ?", username).Where("is_admin = ?", true).Find(&appRevs)

	for _, ar := range appRevs {
		if ar.PromotedTime != nil && ar.SupersededTime == nil {
			return true, nil
		}
	}

	return false, nil
}

// MigrateDBApprover will run the automigrate function for the Approver
// object.
func MigrateDBApprover(dbCache *DBCache) {
	dbCache.AutoMigrate(&Approver{})
}
