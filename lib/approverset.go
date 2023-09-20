// Package lib provides the objects required to operate registrar
package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/aryann/difflib"
	"github.com/jinzhu/gorm"
)

// More information about the ApproverSet Object and its States
// can be found in /doc/approverset.md.

// ApproverSet objects are used to group similar approvers together to
// allow equivalence classes of Approvers (increasing availability).
type ApproverSet struct {
	Model
	State string

	CurrentRevision   ApproverSetRevision
	CurrentRevisionID sql.NullInt64
	Revisions         []ApproverSetRevision
	PendingRevision   ApproverSetRevision `sql:"-"`

	KeysSet bool              `sql:"-"`
	Keys    []*openpgp.Entity `sql:"-"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`

	// TA: I think this was added in error, i dont see them used anywhere
	// ApprovalStartTime  *time.Time
	// ApprovalStartBy    string
	// PromotedTime       *time.Time
	// SupersededTime     *time.Time
	// ApprovalFailedTime *time.Time
}

// ApproverSetExportFull is an object that is uesd to export the current
// snapshot of an Approver Set object. The full version of the export
// object contains the current and pending revision objects if they
// exist.
type ApproverSetExportFull struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	CurrentRevision ApproverSetRevisionExport `json:"CurrentRevision"`
	PendingRevision ApproverSetRevisionExport `json:"PendingRevision"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// ApproverSetExportShort is an object that is used to expor the current
// snapshot of an Approver Set object. The short version of the export
// object does not contain any information about the current or pending
// revisions.
type ApproverSetExportShort struct {
	ID    int64
	State string

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// GetDiff will return a string containing a formatted diff of the
// current and pending revisions for the Approver object. An empty
// string and an error are returned if an error occures during the
// processing.
func (a ApproverSetExportFull) GetDiff() (string, error) {
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
func (a ApproverSetExportFull) ToJSON() (string, error) {
	if a.ID <= 0 {
		return "", errors.New("ID not set")
	}

	byteArr, jsonErr := json.MarshalIndent(a, "", "  ")

	return string(byteArr), jsonErr
}

// ApproverSetPage is used to hold all the information required to
// render the Approver Set HTML template.
type ApproverSetPage struct {
	Editable            bool
	IsNew               bool
	AppS                ApproverSet
	CurrentRevisionPage *ApproverSetRevisionPage
	PendingRevisionPage *ApproverSetRevisionPage
	PendingActions      map[string]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApproverSetPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApproverSetPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
	a.PendingRevisionPage.CSRFToken = newToken
}

// ApproverSetsPage is used to render the html template which lists all
// of the Approvers currently in the registrar system.
//
// TODO: Add paging support.
// TODO: Add filtering support.
type ApproverSetsPage struct {
	ApproverSets []ApproverSet

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApproverSetsPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApproverSetsPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// GetExportVersion returns a export version of the Approver Set Object.
//
// TODO add CurrentRevision.
// TODO Add PendingRevision.
func (a *ApproverSet) GetExportVersion() RegistrarObjectExport {
	export := ApproverSetExportFull{
		ID:              a.ID,
		State:           a.State,
		PendingRevision: (a.PendingRevision.GetExportVersion()).(ApproverSetRevisionExport),
		CurrentRevision: (a.CurrentRevision.GetExportVersion()).(ApproverSetRevisionExport),
		CreatedAt:       a.CreatedAt,
		CreatedBy:       a.CreatedBy,
	}

	return export
}

// GetExportVersionAt returns an export version of the Approver Set
// Object at the timestamp provided if possible otherwise an error is
// returned. If a pending version existed at the time it will be
// excluded from the object.
func (a *ApproverSet) GetExportVersionAt(dbCache *DBCache, timestamp int64) (obj RegistrarObjectExport, err error) {
	asr := ApproverSetRevision{}

	// Grab the most recent promoted object before the time provided
	// where the promoted time is after the time stated above
	if err = dbCache.GetRevisionAtTime(&asr, a.ID, timestamp); err != nil {
		return
	}

	// Otherwise, prepare the new object, fix the currnet object a little
	// and remove the pending revision if any
	if err = asr.Prepare(dbCache); err != nil {
		return
	}

	a.CurrentRevision = asr
	a.CurrentRevisionID.Int64 = asr.ID
	a.CurrentRevisionID.Valid = true
	a.PendingRevision = ApproverSetRevision{}

	// Return the export version and no error
	return a.GetExportVersion(), nil
}

// HasRevision returns true iff a current revision exists, otherwise
// false.
//
// TODO: add a check to verify that the current revision has an approved
// change request.
func (a ApproverSet) HasRevision() bool {
	return a.CurrentRevisionID.Valid
}

// HasPendingRevision returns true iff a pending revision exists for the
// Approver Set, otherwise false.
func (a ApproverSet) HasPendingRevision() bool {
	return a.PendingRevision.ID != 0
}

// GetCurrentValue is used to get the current value of a field in a
// revision if a current revision exists, otherwise an empty string is
// returned.
func (a *ApproverSet) GetCurrentValue(field string) (ret string) {
	if !a.prepared {
		return UnPreparedApproverError
	}

	return a.SuggestedRevisionValue(field)
}

// GetDisplayName will return a name for the Approver set that can be
// used to display a shortened version of the invormation to users.
func (a *ApproverSet) GetDisplayName() string {
	return fmt.Sprintf("%d - %s", a.ID, a.GetCurrentValue(ApproverSetFieldTitle))
}

// GetDisplayObject creates a display object for the called approver set.
func (a *ApproverSet) GetDisplayObject() ApproverSetDisplayObject {
	return ApproverSetDisplayObject{
		DisplayName: a.GetDisplayName(),
		ID:          a.ID,
	}
}

// SuggestedRevisionValue takes a string naming the field that is being
// requested and returns a string containing the suggested value for
// the field in a new pending revision.
func (a ApproverSet) SuggestedRevisionValue(field string) string {
	if a.CurrentRevisionID.Valid {
		switch field {
		case ApproverSetFieldTitle:
			return a.CurrentRevision.Title
		case ApproverSetFieldDescription:
			return a.CurrentRevision.Description
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
func (a ApproverSet) SuggestedRevisionBool(field string) bool {
	if a.CurrentRevisionID.Valid {
		switch field {
		case DesiredStateActive:
			return a.CurrentRevision.DesiredState == StateActive
		case DesiredStateInactive:
			return a.CurrentRevision.DesiredState == StateInactive
		}
	}

	return false
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
func (a *ApproverSet) ParseFromForm(request *http.Request, _ *DBCache) error {
	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	a.CreatedBy = runame
	a.UpdatedBy = runame

	a.State = StateNew

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse Approver Set object with the changes that were
// made. An error object is always the second return value which is nil
// when no errors have occurred during parsing otherwise an error is
// returned.
//
// TODO: Implement.
func (a *ApproverSet) ParseFromFormUpdate(_ *http.Request, _ *DBCache, _ Config) error {
	return errors.New("approvers may not be directly updated")
}

// VerifyCR Checks to make sure that all of the values and approvals
// within a change request match the approver set that it is linked to.
//
// TODO: more rigirous check on if the CR approved text matches.
func (a *ApproverSet) VerifyCR(dbCache *DBCache) (checksOut bool, errs []error) {
	return VerifyCR(dbCache, a, nil)
}

// GetCurrentRevisionID will return the id of the current Approver Set
// Revision for the Approver Set object.
func (a *ApproverSet) GetCurrentRevisionID() sql.NullInt64 {
	return a.CurrentRevisionID
}

// GetPendingRevisionID will return the current pending revision for the
// Approver Set object if it exists. If no pending revision exists a 0
// is returned.
func (a *ApproverSet) GetPendingRevisionID() int64 {
	return a.PendingRevision.ID
}

// GetPendingCRID will return the current CR id if it is set, otherwise
// a nil will be returned (in the form of a sql.NullInt64).
func (a *ApproverSet) GetPendingCRID() sql.NullInt64 {
	return a.PendingRevision.CRID
}

// ComparePendingToCallback will return a function that will compare the
// current revision object to itself after changes have been made.
func (a *ApproverSet) ComparePendingToCallback(loadFn CompareLoadFn) (retFn CompareReturnFn) {
	exp := ApproverSetExportFull{}
	loadFn(&exp)

	return func() (pass bool, errs []error) {
		return exp.PendingRevision.Compare(a.PendingRevision)
	}
}

// UpdateState can be called at any point to check the state of the
// Approver Set and update it if necessary.
//
// TODO: Implement.
func (a *ApproverSet) UpdateState(dbCache *DBCache, conf Config) (changesMade bool, errs []error) {
	logger.Infof("UpdateState called on Approver Set %d (todo)", a.ID)

	changeObj := ApproverSet{}
	changeObj.ID = a.ID

	if err := changeObj.PrepareShallow(dbCache); err != nil {
		errs = append(errs, err)

		return changesMade, errs
	}

	if err := a.Prepare(dbCache); err != nil {
		errs = append(errs, err)

		return changesMade, errs
	}

	changesMade = false
	cascadeState := false

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
					logger.Infof("CR %d has been approved", changeRequest.GetID())

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

					newApp := ApproverSet{Model: Model{ID: a.ID}}

					if err := newApp.PrepareShallow(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if targetState == StateActive || targetState == StateInactive {
						newApp.State = targetState
						cascadeState = true
					} else {
						errs = append(errs, fmt.Errorf("pending revision is in an invalid state: %s", targetState))

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

					newApp := ApproverSet{}
					if err := dbCache.FindByID(&newApp, a.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if a.CurrentRevisionID.Valid {
						curRev := ApproverSetRevision{}
						if err := dbCache.FindByID(&curRev, a.CurrentRevisionID.Int64); err != nil {
							errs = append(errs, err)

							return changesMade, errs
						}

						if curRev.RevisionState == StateBootstrap {
							newApp.State = StateActive
							cascadeState = true
						} else {
							newApp.State = curRev.RevisionState
							cascadeState = true
						}
					} else {
						newApp.State = StateNew
						cascadeState = true
					}

					if err := dbCache.Save(&newApp); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}
				}
			}
		} else {
			// If a pending revision was not found then we have to go back to
			// either active, inactive or bootstrap depending on the state
			if a.State == StatePendingBootstrap {
				changeObj.State = StateBootstrap
				a.State = StateBootstrap
				changesMade = true
				cascadeState = true
			} else if a.CurrentRevisionID.Valid {
				changeObj.State = a.CurrentRevision.DesiredState
				a.State = a.CurrentRevision.DesiredState
				changesMade = true
				cascadeState = true
			} else {
				changeObj.State = StateNew
				a.State = StateNew
				changesMade = true
				cascadeState = true
			}
		}
	default:
		// errs = append(errs, fmt.Errorf("UpdateState for Approver Set at state \"%s\" not implemented", a.State))
		logger.Warningf("UpdateState for Approver Set at state \"%s\" not implemented", a.State)
	}

	if changesMade {
		if err := dbCache.Save(&changeObj); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}
	}

	if cascadeState {
		appset := ApproverSet{}

		if err := dbCache.FindByID(&appset, a.ID); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}

		logger.Error("Cascade Update not configured")

		approvals, _ := appset.GetPendingApprovals(dbCache)

		for _, approval := range approvals {
			subChanges, subErrs := approval.UpdateState(dbCache, conf)
			errs = append(errs, subErrs...)
			changesMade = changesMade || subChanges
		}
	}

	return changesMade, errs
}

// GetPendingApprovals is used to get a list of approvals that are in
// the pendingapproval, inactiveapproverset or novalidapprovers states
// that have the passed approver set as their approver set.
//
// TODO: Consider moving the query into dbcache.
func (a *ApproverSet) GetPendingApprovals(dbCache *DBCache) (approvals []Approval, err error) {
	err = dbCache.DB.Where(&Approval{ApproverSetID: a.ID}).Where("state = ? or state = ? or state = ?", StatePendingApproval, StateNoValidApprovers, StateInactiveApproverSet).Find(&approvals).Error

	return
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
func (a *ApproverSet) Prepare(dbCache *DBCache) error {
	return PrepareBase(dbCache, a, func() (err error) {
		if a.CurrentRevisionID.Valid {
			a.CurrentRevision = ApproverSetRevision{}

			if err = dbCache.FindByID(&a.CurrentRevision, a.CurrentRevisionID.Int64); err != nil {
				return err
			}
		}

		// Grab the pending revison if it exists and prepare the revision
		if err = dbCache.GetNewAndPendingRevisions(a); err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return nil
			}

			return err
		}

		err = a.PendingRevision.Prepare(dbCache)

		return err
	})
}

// PrepareShallow populate all of the fields for a given object and
// not any of the linked object.
func (a *ApproverSet) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, a, FuncNOPErrFunc)
}

// GetPendingRevision implements the RegistrarParent interface and returns
// the pending revision pointer.
func (a *ApproverSet) GetPendingRevision() RegistrarObject {
	return &a.PendingRevision
}

// PrepareDisplayShallow populate all of the fields for a given object
// and the current revision but not any of the other linked object.
func (a *ApproverSet) PrepareDisplayShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, a, func() (err error) {
		if a.CurrentRevisionID.Valid {
			a.CurrentRevision.ID = a.CurrentRevisionID.Int64
			err = a.CurrentRevision.PrepareShallow(dbCache)
		}

		return
	})
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (a *ApproverSet) GetType() string {
	return ApproverSetType
}

// HasHappened is a helper function that will return true if the value
// of the timestamp who's name is passes has happened after the revision
// was created. This function is intended to be used with templates.
func (a ApproverSet) HasHappened(actionType string) bool {
	switch actionType {
	case EventUpdated:
		return !a.UpdatedAt.Before(a.CreatedAt)
	// TA: I think this was added in error
	// case EventApprovalStarted:
	// 	return !a.ApprovalStartTime.Before(a.CreatedAt)
	// case EventApprovalFailed:
	// 	return !a.ApprovalFailedTime.Before(a.CreatedAt)
	// case EventPromoted:
	// 	return !a.PromotedTime.Before(a.CreatedAt)
	// case EventSuperseded:
	// 	return !a.SupersededTime.Before(a.CreatedAt)
	default:
		logger.Errorf("Unknown actiontype: %s", actionType)

		return false
	}
}

// IsCancelled returns true iff the object has been canclled.
func (a *ApproverSet) IsCancelled() bool {
	return a.State == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (a *ApproverSet) IsEditable() bool {
	return a.State == StateNew
}

// GetPage will return an object that can be used to render the HTML
// template for the Approver Set.
func (a *ApproverSet) GetPage(dbCache *DBCache, username string, email string) (rpo RegistrarObjectPage, err error) {
	ret := &ApproverSetPage{Editable: true, IsNew: true}
	if a.ID != 0 {
		ret.Editable = a.IsEditable()
		ret.IsNew = false
	}

	ret.AppS = *a
	ret.PendingActions = make(map[string]string)

	if a.PendingRevision.ID != 0 {
		ret.PendingActions = a.PendingRevision.GetActions(false)
	}

	if a.CurrentRevisionID.Valid {
		rawPage, rawPageErr := a.CurrentRevision.GetPage(dbCache, username, email)

		if rawPageErr != nil {
			return ret, rawPageErr
		}

		ret.CurrentRevisionPage = rawPage.(*ApproverSetRevisionPage)
	}

	if a.HasPendingRevision() {
		rawPage, rawPageErr := a.PendingRevision.GetPage(dbCache, username, email)

		if rawPageErr != nil {
			return ret, rawPageErr
		}

		ret.PendingRevisionPage = rawPage.(*ApproverSetRevisionPage)
	} else {
		ret.PendingRevisionPage = &ApproverSetRevisionPage{IsEditable: true, IsNew: true}
	}

	ret.PendingRevisionPage.ParentApproverSet = a
	ret.PendingRevisionPage.SuggestedRequiredApprovers = make(map[int64]ApproverSetDisplayObject)
	ret.PendingRevisionPage.SuggestedInformedApprovers = make(map[int64]ApproverSetDisplayObject)
	ret.PendingRevisionPage.SuggestedApprovers = make(map[int64]string)
	ret.PendingRevisionPage.ValidApprovers, err = GetValidApproverMap(dbCache)

	if err != nil {
		return ret, err
	}

	ret.PendingRevisionPage.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	if err != nil {
		return ret, err
	}

	if a.CurrentRevision.ID != 0 {
		for _, appSet := range a.CurrentRevision.RequiredApproverSets {
			err = appSet.Prepare(dbCache)

			if err != nil {
				return ret, err
			}

			ret.PendingRevisionPage.SuggestedRequiredApprovers[appSet.ID] = appSet.GetDisplayObject()
		}

		for _, appSet := range a.CurrentRevision.InformedApproverSets {
			err = appSet.Prepare(dbCache)

			if err != nil {
				return ret, err
			}

			ret.PendingRevisionPage.SuggestedInformedApprovers[appSet.ID] = appSet.GetDisplayObject()
		}

		for _, approver := range a.CurrentRevision.Approvers {
			err = approver.Prepare(dbCache)

			if err != nil {
				return ret, err
			}

			ret.PendingRevisionPage.SuggestedApprovers[approver.ID] = approver.GetDisplayName()
		}
	} else {
		appSet, prepErr := GetDefaultApproverSet(dbCache)

		if prepErr != nil {
			return ret, errors.New("unable to find default approver - database probably not bootstrapped")
		}

		ret.PendingRevisionPage.SuggestedRequiredApprovers[1] = appSet.GetDisplayObject()
	}

	return ret, nil
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple Approver Sets.
//
// TODO: Add paging support.
// TODO: Add filtering.
func (a *ApproverSet) GetAllPage(dbCache *DBCache, _ string, _ string) (RegistrarObjectPage, error) {
	ret := &ApproverSetsPage{}

	err := dbCache.FindAll(&ret.ApproverSets)
	if err != nil {
		return ret, err
	}

	for idx := range ret.ApproverSets {
		err = ret.ApproverSets[idx].Prepare(dbCache)
		if err != nil {
			return ret, err
		}
	}

	return ret, nil
}

// IsValidApproverByEmail will check all of the approvers in the current
// approved revision to verify that the email address can be found.
//
// TODO: Check for approver being active.
// TODO: Check for special case of bootstrap.
func (a *ApproverSet) IsValidApproverByEmail(emailaddress string, dbCache *DBCache) (bool, error) {
	for _, approver := range a.CurrentRevision.Approvers {
		err := approver.Prepare(dbCache)
		if err != nil {
			return false, err
		}

		if approver.CurrentRevisionID.Valid {
			email := approver.GetCurrentValue(ApproverFieldEmailAddres)

			if email == emailaddress {
				return true, nil
			}
		}
	}

	return false, nil
}

// KeysById returns the set of keys that have the given key id. This
// method is part of the interface for []openpgp.Entities.
func (a ApproverSet) KeysById(keyID uint64) (keys []openpgp.Key) {
	for _, entity := range a.Keys {
		if entity.PrimaryKey.KeyId == keyID {
			var selfSig *packet.Signature

			for _, ident := range entity.Identities {
				if selfSig == nil {
					selfSig = ident.SelfSignature
				} else if ident.SelfSignature.IsPrimaryId != nil && *ident.SelfSignature.IsPrimaryId {
					selfSig = ident.SelfSignature

					break
				}
			}

			keys = append(keys, openpgp.Key{Entity: entity, PublicKey: entity.PrimaryKey, PrivateKey: entity.PrivateKey, SelfSignature: selfSig})
		}

		for _, subKey := range entity.Subkeys {
			if subKey.PublicKey.KeyId == keyID {
				keys = append(keys, openpgp.Key{Entity: entity, PublicKey: subKey.PublicKey, PrivateKey: subKey.PrivateKey, SelfSignature: subKey.Sig})
			}
		}
	}

	return keys
}

// KeysByIdUsage returns the set of keys with the given id that also
// meet the key usage given by requiredUsage.  The requiredUsage is
// expressed as the bitwise-OR of packet.KeyFlag* values. This method is
// part of the interface for []openpgp.Entities.
func (a ApproverSet) KeysByIdUsage(keyID uint64, requiredUsage byte) (keys []openpgp.Key) {
	for _, key := range a.KeysById(keyID) {
		if len(key.Entity.Revocations) > 0 {
			continue
		}

		if key.SelfSignature.RevocationReason != nil {
			continue
		}

		if key.SelfSignature.FlagsValid && requiredUsage != 0 {
			var usage byte
			if key.SelfSignature.FlagCertify {
				usage |= packet.KeyFlagCertify
			}

			if key.SelfSignature.FlagSign {
				usage |= packet.KeyFlagSign
			}

			if key.SelfSignature.FlagEncryptCommunications {
				usage |= packet.KeyFlagEncryptCommunications
			}

			if key.SelfSignature.FlagEncryptStorage {
				usage |= packet.KeyFlagEncryptStorage
			}

			if usage&requiredUsage != requiredUsage {
				continue
			}
		}

		keys = append(keys, key)
	}

	return keys
}

// DecryptionKeys returns all private keys that are valid for
// decryption. No private keys are stored by the system so it is always
// a noop. This method is part of the interface for []openpgp.Entities.
func (a ApproverSet) DecryptionKeys() (keys []openpgp.Key) {
	return
}

// PrepareGPGKeys will go over the list of current approvers and extract
// the GPG keys from the Approvers to prepare the Approver Set to be
// used as a verification keyring.
func (a *ApproverSet) PrepareGPGKeys(dbCache *DBCache) error {
	logger.Debugf("Preparing the GPG Public Keys for Approver Set %d", a.ID)

	if a.CurrentRevisionID.Valid && (a.CurrentRevision.RevisionState == StateBootstrap || a.CurrentRevision.RevisionState == StateActive) {
		for _, approver := range a.CurrentRevision.Approvers {
			if err := approver.Prepare(dbCache); err != nil {
				return err
			}

			block, err := approver.GetGPGKeyBlock()

			if err == nil {
				a.Keys = append(a.Keys, block)
			} else {
				logger.Warningf("%q", err)
			}
		}

		return nil
	}

	return errors.New("unable to find active key")
}

// ApproverFromIdentityName will iterate over the aprovers from the
// current revision and try to find the approver with a key that has
// the same Identity Name. If no Approver is found it will return an
// error.
func (a *ApproverSet) ApproverFromIdentityName(name string, dbCache *DBCache) (app Approver, err error) {
	if a.CurrentRevisionID.Valid && (a.CurrentRevision.RevisionState == StateBootstrap || a.CurrentRevision.RevisionState == StateActive) {
		for _, approver := range a.CurrentRevision.Approvers {
			if err = approver.Prepare(dbCache); err != nil {
				return app, err
			}

			block, getGPGKeyBlockErr := approver.GetGPGKeyBlock()

			if getGPGKeyBlockErr != nil {
				err = getGPGKeyBlockErr

				return app, err
			}

			for idenStr := range block.Identities {
				if idenStr == name {
					return approver, nil
				}
			}
		}
	}

	err = errors.New("unable to find key")

	return app, err
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
//
// TODO: Implement.
func (a *ApproverSet) TakeAction(response http.ResponseWriter, _ *http.Request, _ *DBCache, actionName string, _ bool, authMethod AuthType, _ Config) (errs []error) {
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
// required for the Approver Set (if a valid approver revision exists).
// If no approver revisions are found, a default of the infosec approver
// set will be returned.
func (a *ApproverSet) GetRequiredApproverSets(dbCache *DBCache) (approvers []ApproverSet, err error) {
	return GetRequiredApproverSets(dbCache, a)
}

// GetInformedApproverSets returns the list of approver sets that are
// informed for the Approver Set (if a valid approver revision exists).
// If no approver revisions are found, an empty list will be returned.
func (a *ApproverSet) GetInformedApproverSets(dbCache *DBCache) ([]ApproverSet, error) {
	return GetInformedApproverSets(dbCache, a)
}

// GetValidApproverSetMap will return a map containing the Approver set
// title and ID indexed by their Approver Set ID. Only Approver Sets
// with the state "active" or "activependingapproval" are returned.
//
// TODO: Consider moving the db call into the dbcache object.
func GetValidApproverSetMap(dbCache *DBCache) (map[int64]string, error) {
	var appSets []ApproverSet

	ret := make(map[int64]string)

	err := dbCache.DB.Where("state = ? or state = ?", StateActive, StateActivePendingApproval).Find(&appSets).Error
	if err != nil {
		return ret, err
	}

	for _, approverSet := range appSets {
		err = approverSet.PrepareDisplayShallow(dbCache)

		if err != nil {
			return ret, err
		}

		ret[approverSet.ID] = approverSet.GetDisplayName()
	}

	return ret, nil
}

// GetDefaultApproverSet - return the default approver.
func GetDefaultApproverSet(dbCache *DBCache) (approverSet ApproverSet, err error) {
	approverSet.ID = 1

	if err = dbCache.Find(&approverSet); err != nil {
		logger.Error("Unable to find approverSet 1 - database probably not bootstrapped")

		return
	}

	prepareErr := approverSet.PrepareDisplayShallow(dbCache)
	err = prepareErr

	return
}

type approverSetsByID []ApproverSet

func (a approverSetsByID) Len() int           { return len(a) }
func (a approverSetsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a approverSetsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// ParseApproverSets takes the http Request, a database connection and
// the html ID of the approver set list to parse and will return an
// array of Approver Sets that correspond to each of the id's from the
// http request's html element. If there are unparsable IDs in the list
// (could be strings or empty fields) and error is returned. Any valid
// approver sets that were found will be returned in the array even if
// an error occures.
//
// ApproverSets will be returned in ID order, regardless of the order requested.
func ParseApproverSets(request *http.Request, dbCache *DBCache, htmlID string, requireAppSet1 bool) (approverSets []ApproverSet, err error) {
	var unparsableIDs, invalidIDs string

	appSetMap := map[int64]*ApproverSet{}

	if requireAppSet1 {
		var das ApproverSet

		if das, err = GetDefaultApproverSet(dbCache); err != nil {
			invalidIDs = fmt.Sprintf("%s (DefaultApproverSet: %s)", invalidIDs, err.Error())
		} else {
			appSetMap[das.ID] = &das
		}
	}

	for _, appsetidChunk := range request.Form[htmlID] {
		for _, appsetidRaw := range strings.Split(appsetidChunk, " ") {
			appsetid, parseIntErr := strconv.ParseInt(appsetidRaw, 10, 64)

			if parseIntErr != nil {
				unparsableIDs = fmt.Sprintf("%s %s", unparsableIDs, appsetidRaw)

				continue
			}

			if _, ok := appSetMap[appsetid]; ok {
				// already seen it
				continue
			}

			appSetMap[appsetid] = &ApproverSet{}

			if err = dbCache.FindByID(appSetMap[appsetid], appsetid); err != nil {
				appSetMap[appsetid] = nil
				invalidIDs = fmt.Sprintf("%s (%d: %s)", invalidIDs, appsetid, err.Error())
			}
		}
	}

	errorStr := ""

	if unparsableIDs != "" {
		errorStr = fmt.Sprintf("%s These IDs did not parse: %s", errorStr, unparsableIDs)
	}

	if invalidIDs != "" {
		errorStr = fmt.Sprintf("%s These IDs failed to load: %s", errorStr, invalidIDs)
	}

	if errorStr != "" {
		err = fmt.Errorf("ParseApproverSets failed for %s: %s", htmlID, errorStr)

		return approverSets, err
	}

	err = nil

	for _, appSet := range appSetMap {
		if appSet != nil {
			approverSets = append(approverSets, *appSet)
		}
	}

	sort.Sort((approverSetsByID(approverSets)))

	return approverSets, err
}

// UpdateApproverSets will update a set of approvers for an object that
// is passed in to make the list reflect the list passed as the third
// parameter.
//
// TODO: Consider moving the db query into the dbcache object.
func UpdateApproverSets(object RegistrarObject, dbCache *DBCache, association string, approverSets []ApproverSet) error {
	err := dbCache.DB.Model(object).Association(association).Clear().Error
	dbCache.InvalidateObject(object)

	if err != nil {
		return err
	}

	for _, approverset := range approverSets {
		err = dbCache.DB.Model(object).Association(association).Append(approverset).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// GetApproverSetExportArr converts an array of ApproverSets to an array
// of ApproverSetExportShort objects for exporting.
func GetApproverSetExportArr(as []ApproverSet) []ApproverSetExportShort {
	export := make([]ApproverSetExportShort, len(as))

	for idx, set := range as {
		tmp := ApproverSetExportShort{
			ID: set.ID, State: set.State,
			CreatedAt: set.CreatedAt, CreatedBy: set.CreatedBy,
		}
		export[idx] = tmp
	}

	return export
}

// CompareToApproverSetListToExportShort compares a list of approver
// sets to a set of approver sets that were from an export version of an
// object. If the counts match and the IDs for the aprover sets match
// true is returned otherwise, false is returned.
func CompareToApproverSetListToExportShort(approverSet1 []ApproverSet, approverSet2 []ApproverSetExportShort) bool {
	exportShortCount := len(approverSet2)
	fullCount := len(approverSet1)

	if fullCount != exportShortCount {
		return false
	}

	for _, full := range approverSet1 {
		found := false

		for _, export := range approverSet2 {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	for _, export := range approverSet2 {
		found := false

		for _, full := range approverSet1 {
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

// CompareToApproverSetExportShortLists compares two lists of approver
// set export short objects that were from an export version of an
// object. If the counts match and the IDs for the aprover sets match
// true is returned otherwise, false is returned.
func CompareToApproverSetExportShortLists(approverSet1 []ApproverSetExportShort, approverSet2 []ApproverSetExportShort) bool {
	exportShortCount := len(approverSet2)
	fullCount := len(approverSet1)

	if fullCount != exportShortCount {
		return false
	}

	for _, full := range approverSet1 {
		found := false

		for _, export := range approverSet2 {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	for _, export := range approverSet2 {
		found := false

		for _, full := range approverSet1 {
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

// ApproverSetDisplayObject is used to bundle the dispaly name and the
// approver set ID as one object to be passed into a template.
type ApproverSetDisplayObject struct {
	DisplayName string
	ID          int64
}

// MigrateDBApproverSet will run the automigrate function for the
// Approver Set object.
func MigrateDBApproverSet(dbCache *DBCache) {
	dbCache.AutoMigrate(&ApproverSet{})
}
