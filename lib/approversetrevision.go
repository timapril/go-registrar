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
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/ProtonMail/go-crypto/openpgp/packet"

	"github.com/jinzhu/gorm"
)

// More information about the ApproverSet Object and its States
// can be found in /doc/approversetrevision.md

// ApproverSetRevision represents an individual version of an Approver
// Set object.
type ApproverSetRevision struct {
	Model
	ApproverSetID int64
	RevisionState string

	DesiredState string

	Title       string
	Description string

	Approvers []Approver `gorm:"many2many:approver_to_revision_set;"`

	SavedNotes string `sql:"size:16384"`

	RequiredApproverSets []ApproverSet `gorm:"many2many:required_approverset_to_approversetrevision"`
	InformedApproverSets []ApproverSet `gorm:"many2many:informed_approverset_to_approversetrevision"`

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

// ApproverSetRevisionExport is an object that is used to export the
// current version of an Approver Set Revision.
type ApproverSetRevisionExport struct {
	ID            int64 `json:"ID"`
	ApproverSetID int64 `json:"ApproverSetID"`

	RevisionState string `json:"RevisionState"`
	DesiredState  string `json:"DesiredState"`

	Title       string `json:"Title"`
	Description string `json:"Description"`

	Approvers []ApproverExportShort `json:"Approvers"`

	SavedNotes string `json:"SavedNotes"`

	ChangeRequestID int64 `json:"ChangeRequestID"`

	IssueCR string `json:"IssueCR"`
	Notes   string `json:"Notes"`

	RequiredApproverSets []ApproverSetExportShort `json:"RequiredApproverSets"`
	InformedApproverSets []ApproverSetExportShort `json:"InformedApproverSets"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`

	// Unmarshaled entries
	verifiedApprovers []ApproverExportFull
	keys              []*openpgp.Entity
}

// AddVerifiedApprover appends the passed approver to the list of
// verified approvers for the approver set revision.
func (asre *ApproverSetRevisionExport) AddVerifiedApprover(app ApproverExportFull) error {
	asre.verifiedApprovers = append(asre.verifiedApprovers, app)

	return asre.addKey(app.CurrentRevision.PublicKey)
}

// HasVerifiedApprovers returns true iff there are any verified
// approvers for the approver set revision.
func (asre *ApproverSetRevisionExport) HasVerifiedApprovers() bool {
	return len(asre.verifiedApprovers) > 0
}

// AddKey is used to add a new key to a trust anchor set.
func (asre *ApproverSetRevisionExport) addKey(key string) error {
	decbuf := bytes.NewBufferString(key + "\n")

	block, err1 := armor.Decode(decbuf)
	if err1 != nil {
		return fmt.Errorf("error decoding block: %w", err1)
	}

	packetReader := packet.NewReader(block.Body)

	entity, err2 := openpgp.ReadEntity(packetReader)
	if err2 != nil {
		return fmt.Errorf("error reading entity: %w", err2)
	}

	asre.keys = append(asre.keys, entity)

	return nil
}

// KeysById returns the set of keys that have the given key id. This
// method is part of the interface for []openpgp.Entities.
func (asre ApproverSetRevisionExport) KeysById(entityID uint64) (keys []openpgp.Key) {
	for _, entity := range asre.keys {
		if entity.PrimaryKey.KeyId == entityID {
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
			if subKey.PublicKey.KeyId == entityID {
				keys = append(keys, openpgp.Key{Entity: entity, PublicKey: subKey.PublicKey, PrivateKey: subKey.PrivateKey, SelfSignature: subKey.Sig})
			}
		}
	}

	return
}

// KeysByIdUsage returns the set of keys with the given id that also
// meet the key usage given by requiredUsage.  The requiredUsage is
// expressed as the bitwise-OR of packet.KeyFlag* values. This method is
// part of the interface for []openpgp.Entities.
func (asre ApproverSetRevisionExport) KeysByIdUsage(id uint64, requiredUsage byte) (keys []openpgp.Key) {
	for _, key := range asre.KeysById(id) {
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
func (asre ApproverSetRevisionExport) DecryptionKeys() (keys []openpgp.Key) {
	return
}

// IsSignedBy will return true if the object is signed by one of the
// members of the TrustAnchors list.
func (asre ApproverSetRevisionExport) IsSignedBy(sig []byte) (valid bool, signedBody []byte) {
	block, _ := clearsign.Decode(sig)

	if block == nil {
		return false, signedBody
	}

	_, sigErr := openpgp.CheckDetachedSignature(asre, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body, nil)

	if sigErr == nil {
		return true, block.Bytes
	}

	return false, signedBody
}

// Compare is used to compare an export version of an object to the
// full revision to verify that all of the values are the same.
func (asre ApproverSetRevisionExport) Compare(asr ApproverSetRevision) (pass bool, errs []error) {
	pass = true

	if asre.ID != asr.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if asre.ApproverSetID != asr.ApproverSetID {
		errs = append(errs, fmt.Errorf("the ApproverSetID fields did not match"))
		pass = false
	}

	if asre.DesiredState != asr.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if asre.Title != asr.Title {
		errs = append(errs, fmt.Errorf("the Title fields did not match"))
		pass = false
	}

	if asre.Description != asr.Description {
		errs = append(errs, fmt.Errorf("the Title fields did not match"))
		pass = false
	}

	if asre.SavedNotes != asr.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if asre.IssueCR != asr.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if asre.Notes != asr.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	approversCheck := CompareToApproverListExportShortList(asr.Approvers, asre.Approvers)
	if !approversCheck {
		errs = append(errs, fmt.Errorf("the approvers list did not match"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetListToExportShort(asr.RequiredApproverSets, asre.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetListToExportShort(asr.InformedApproverSets, asre.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// CompareExport is used to compare an export version of an object to
// another export revision to verify that all of the values are the
// same.
func (asre ApproverSetRevisionExport) CompareExport(asr ApproverSetRevisionExport) (pass bool, errs []error) {
	pass = true

	if asre.ID != asr.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if asre.ApproverSetID != asr.ApproverSetID {
		errs = append(errs, fmt.Errorf("the ApproverSetID fields did not match"))
		pass = false
	}

	if asre.DesiredState != asr.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if asre.Title != asr.Title {
		errs = append(errs, fmt.Errorf("the Title fields did not match"))
		pass = false
	}

	if asre.Description != asr.Description {
		errs = append(errs, fmt.Errorf("the Title fields did not match"))
		pass = false
	}

	if asre.SavedNotes != asr.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if asre.IssueCR != asr.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if asre.Notes != asr.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	approversCheck := CompareToApproverExportShortLists(asr.Approvers, asre.Approvers)
	if !approversCheck {
		errs = append(errs, fmt.Errorf("the approvers list did not match"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetExportShortLists(asr.RequiredApproverSets, asre.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetExportShortLists(asr.InformedApproverSets, asre.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// ToJSON will return a string containing a JSON representation of the
// object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (asre ApproverSetRevisionExport) ToJSON() (string, error) {
	if asre.ID <= 0 {
		return "", errors.New("invalid revision ID")
	}

	byteArr, jsonErr := json.MarshalIndent(asre, "", "  ")

	return string(byteArr), jsonErr
}

// GetDiff will return an empty string and an error for a revision. A
// Diff is not available for revision objects.
func (asre ApproverSetRevisionExport) GetDiff() (string, error) {
	return "", errors.New("unable to diff a single revision")
}

// ApproverSetRevisionPage are used to hold all the information required
// to render the ApproverSetRevision HTML template.
type ApproverSetRevisionPage struct {
	IsEditable                 bool
	IsNew                      bool
	Revision                   ApproverSetRevision
	PendingActions             map[string]string
	ValidApprovers             map[int64]string
	ValidApproverSets          map[int64]string
	ParentApproverSet          *ApproverSet
	SuggestedRequiredApprovers map[int64]ApproverSetDisplayObject
	SuggestedInformedApprovers map[int64]ApproverSetDisplayObject
	SuggestedApprovers         map[int64]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApproverSetRevisionPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApproverSetRevisionPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// The ApproverSetRevisionsPage type is used to render the html template
// which lists all of the AppproverSetRevisions currently in the
// registrar system
//
// TODO: Add paging support.
type ApproverSetRevisionsPage struct {
	ApproverSetRevisions []ApproverSetRevision

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApproverSetRevisionsPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApproverSetRevisionsPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// GetExportVersion returns a export version of the ApproverSetRevision
// Object.
func (a *ApproverSetRevision) GetExportVersion() RegistrarObjectExport {
	export := ApproverSetRevisionExport{
		ID:            a.ID,
		Title:         a.Title,
		Description:   a.Description,
		ApproverSetID: a.ApproverSetID,
		SavedNotes:    a.SavedNotes,
		RevisionState: a.RevisionState,
		DesiredState:  a.DesiredState,
		IssueCR:       a.IssueCR,
		Notes:         a.Notes,
		CreatedAt:     a.CreatedAt,
		CreatedBy:     a.CreatedBy,
	}
	for idx := range a.Approvers {
		export.Approvers = append(export.Approvers, a.Approvers[idx].GetExportShortVersion())
	}
	// TODO: Add Approvers field
	export.RequiredApproverSets = GetApproverSetExportArr(a.RequiredApproverSets)
	export.InformedApproverSets = GetApproverSetExportArr(a.InformedApproverSets)

	if a.CRID.Valid {
		export.ChangeRequestID = a.CRID.Int64
	} else {
		export.ChangeRequestID = -1
	}

	return export
}

// GetExportVersionAt returns an export version of the Approver Set
// Revision Object at the timestamp provided if possible otherwise an
// error is returned. If a pending version existed at the time it will
// be excluded from the object.
func (a *ApproverSetRevision) GetExportVersionAt(_ *DBCache, _ int64) (obj RegistrarObjectExport, err error) {
	return obj, errors.New("GetExportVersionAt is not usable for revisions")
}

// GetState is used to verify that a state submitted in an HTTP request
// is either "active" or "inactive". If the submitted state does not
// match either of the options "active" is returned. "bootstrap" is not
// an allowed state via HTTP.
func (a *ApproverSetRevision) GetState(cleartextState string) string {
	if cleartextState == StateInactive {
		return StateInactive
	}

	return StateActive
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
func (a *ApproverSetRevision) ParseFromForm(request *http.Request, dbCache *DBCache) error {
	var err2, err3, err4 error

	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	setID, err1 := strconv.ParseInt(request.FormValue("revision_approver_set_id"), 10, 64)

	a.Title = request.FormValue("revision_title")
	a.Description = request.FormValue("revision_description")

	a.CreatedBy = runame
	a.UpdatedBy = runame

	a.Approvers, err2 = ParseApprovers(request, dbCache, "approver_id")

	a.ApproverSetID = setID
	a.SavedNotes = request.FormValue("revision_saved_notes")

	a.RevisionState = StateNew
	a.DesiredState = a.GetState(request.FormValue("revision_desiredstate"))

	a.IssueCR = request.FormValue("revision_issue_cr")
	a.Notes = request.FormValue("revision_notes")

	a.RequiredApproverSets, err3 = ParseApproverSets(request, dbCache, "approver_set_required_id", true)
	a.InformedApproverSets, err4 = ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

	if err1 != nil {
		return fmt.Errorf("error parsing approver set id: %w", err1)
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

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse ApproverSetRevision object with the changes that
// were made. An error object is always the second return value which
// is nil when no errors have occurred during parsing otherwise an error
// is returned.
//
// TODO: Consider moving the db query into the dbcache object.
func (a *ApproverSetRevision) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, _ Config) (err error) {
	if a.ID > 0 {
		runame, err0 := GetRemoteUser(request)
		if err0 != nil {
			return err0
		}

		if a.RevisionState == StateNew {
			a.UpdatedBy = runame

			a.Title = request.FormValue("revision_title")
			a.Description = request.FormValue("revision_description")

			a.SavedNotes = request.FormValue("revision_saved_notes")

			a.DesiredState = a.GetState(request.FormValue("revision_desiredstate"))

			a.IssueCR = request.FormValue("revision_issue_cr")
			a.Notes = request.FormValue("revision_notes")

			approvers, err2 := ParseApprovers(request, dbCache, "approver_id")

			RequiredApproverSets, err3 := ParseApproverSets(request, dbCache, "approver_set_required_id", true)
			InformedApproverSets, err4 := ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

			if err2 != nil {
				return err2
			}

			if err3 != nil {
				return err3
			}

			if err4 != nil {
				return err4
			}

			if err = dbCache.DB.Model(a).Association("Approvers").Clear().Error; err != nil {
				return err
			}

			for _, approver := range approvers {
				if err = dbCache.DB.Model(a).Association("Approvers").Append(approver).Error; err != nil {
					return err
				}
			}

			a.Approvers = approvers

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

	return errors.New("to update an approver set revision the ID must be greater than 0")
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the db query into the dbcache object.
func (a *ApproverSetRevision) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, a, func() (err error) {
		if err = dbCache.DB.Model(a).Related(&a.Approvers, "Approvers").Error; err != nil {
			return err
		}

		for idx := range a.Approvers {
			if err = a.Approvers[idx].Prepare(dbCache); err != nil {
				return err
			}
		}

		if err = dbCache.DB.Model(a).Related(&a.RequiredApproverSets, "RequiredApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return err
			}
		}

		if err = dbCache.DB.Model(a).Related(&a.InformedApproverSets, "InformedApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return err
			}
		}

		for idx := range a.RequiredApproverSets {
			if err = a.RequiredApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return err
			}
		}

		for idx := range a.InformedApproverSets {
			if err = a.InformedApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return err
			}
		}

		return err
	})
}

// PrepareShallow populate all of the fields for a given object and
// not any of the linked object.
func (a *ApproverSetRevision) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, a, FuncNOPErrFunc)
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (a *ApproverSetRevision) GetType() string {
	return ApproverSetRevisionType
}

// HasHappened is a helper function that will return true if the value
// of the timestamp who's name is passes has happened after the revision
// was created. This function is intended to be used with templates.
func (a ApproverSetRevision) HasHappened(actionType string) bool {
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
	default:
		logger.Errorf("Unknown actiontype: %s", actionType)

		return false
	}
}

// IsCancelled returns true iff the object has been canclled.
func (a *ApproverSetRevision) IsCancelled() bool {
	return a.RevisionState == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (a *ApproverSetRevision) IsEditable() bool {
	return a.RevisionState == StateNew
}

// IsDesiredState will return true iff the state passed in matches the
// desired state of the revision.
func (a ApproverSetRevision) IsDesiredState(state string) bool {
	return a.DesiredState == state
}

// GetPage will return an object that can be used to render the HTML
// template for the Approver Set Revision.
func (a *ApproverSetRevision) GetPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &ApproverSetRevisionPage{IsNew: true}
	ret.IsEditable = a.IsEditable()
	ret.PendingActions = a.GetActions(true)

	if a.ID != 0 {
		ret.IsNew = false
	}

	ret.Revision = *a
	ret.ParentApproverSet = &ApproverSet{}

	if err = dbCache.FindByID(ret.ParentApproverSet, a.ApproverSetID); err != nil {
		return
	}

	ret.ValidApprovers, err = GetValidApproverMap(dbCache)

	if err != nil {
		return
	}

	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	return ret, err
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple Approver Set Revisions.
//
// TODO: Implement
// TODO: Add paging support
// TODO: Add filtering.
func (a *ApproverSetRevision) GetAllPage(_ *DBCache, _ string, _ string) (RegistrarObjectPage, error) {
	return &ApproverSetRevisionsPage{}, nil
}

// Cancel will change the State of a revision from either "new" or
// "pendingapproval" to "cancelled"
//
// TODO: If in pending approval, cancel the change request and all
// approval objects
//
// TODO: Consider moving the db query into the dbcache object.
func (a *ApproverSetRevision) Cancel(dbCache *DBCache, conf Config) (errs []error) {
	if a.RevisionState == StateNew || a.RevisionState == StatePendingApproval {
		a.RevisionState = StateCancelled

		if err := dbCache.DB.Model(a).UpdateColumns(ApproverSetRevision{RevisionState: StateCancelled}).Error; err != nil {
			errs = append(errs, err)

			return errs
		}

		err := dbCache.Purge(a)
		if err != nil {
			errs = append(errs, err)

			return errs
		}

		parent := ApproverSet{}

		if err := dbCache.FindByID(&parent, a.ApproverSetID); err != nil {
			errs = append(errs, err)

			return errs
		}

		_, errs = parent.UpdateState(dbCache, conf)

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

// GetActions will return a list of possible actions that can be taken
// while in the current state
//
// TODO: handle all states.
func (a *ApproverSetRevision) GetActions(isSelf bool) map[string]string {
	ret := make(map[string]string)

	if a.RevisionState == StateNew {
		ret["Start Approval Process"] = fmt.Sprintf("/action/%s/%d/%s", ApproverSetRevisionType, a.ID, ApproverSetRevisionActionStartApproval)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/%s", ApproverSetRevisionType, a.ID, ApproverSetRevisionActionCancel)

		if isSelf {
			ret["View Parent Approver Set"] = fmt.Sprintf("/view/%s/%d", ApproverSetType, a.ApproverSetID)
		} else {
			ret["View/Edit Approver Set Revision"] = fmt.Sprintf("/view/%s/%d", ApproverSetRevisionType, a.ID)
		}

		ret["Update Object State"] = fmt.Sprintf("/action/%s/%d/%s", ApproverSetRevisionType, a.ID, ActionTriggerUpdate)
	}

	if a.RevisionState == StatePendingApproval {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", ApproverSetRevisionType, a.ID, ApproverSetRevisionActionGOTOChangeRequest)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/%s", ApproverSetRevisionType, a.ID, ApproverSetRevisionActionCancel)

		if isSelf {
			ret["View Parent Approver Set"] = fmt.Sprintf("/view/%s/%d", ApproverSetType, a.ApproverSetID)
		} else {
			ret["View Approver Set Revision"] = fmt.Sprintf("/view/%s/%d", ApproverSetRevisionType, a.ID)
		}

		ret["Update Object State"] = fmt.Sprintf("/action/%s/%d/%s", ApproverSetRevisionType, a.ID, ActionTriggerUpdate)
	}

	return ret
}

// ApproverSetRevisionActionStartApproval is the name of the action
// that will start the approval process for the Approver Set Revision.
const ApproverSetRevisionActionStartApproval string = "startapproval"

// ApproverSetRevisionActionCancel is the name of the action that will
// cancel an Approver Set Revision.
const ApproverSetRevisionActionCancel string = "cancel"

// ApproverSetRevisionActionGOTOChangeRequest is the name of the action
// that will trigger a redirect to the current CR of an object.
const ApproverSetRevisionActionGOTOChangeRequest string = "gotochangerequest"

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary
//
// TODO: Implement.
func (a *ApproverSetRevision) TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, validCSRF bool, authMethod AuthType, conf Config) (errs []error) {
	switch actionName {
	case ApproverSetRevisionActionCancel:
		if validCSRF {
			cancelErrs1 := a.Cancel(dbCache, conf)
			if cancelErrs1 != nil {
				errs = append(errs, cancelErrs1...)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ApproverSetRevisionType, a.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ApproverSetRevisionActionStartApproval:
		if validCSRF {
			appErr := a.StartApprovalProcess(request, dbCache, conf)
			if appErr != nil {
				errs = append(errs, appErr)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ApproverSetRevisionType, a.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ApproverSetRevisionActionGOTOChangeRequest:
		if a.CRID.Valid {
			http.Redirect(response, request, fmt.Sprintf("/view/changerequest/%d", a.CRID.Int64), http.StatusFound)
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(a.GetExportVersion()))

			return errs
		}
	case ActionTriggerUpdate:
		appSet := ApproverSet{}

		err := dbCache.FindByID(&appSet, a.ApproverSetID)
		if err != nil {
			errs = append(errs, err)

			return errs
		}

		appSet.UpdateState(dbCache, conf)
		http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ApproverSetRevisionType, a.ID), http.StatusFound)

		return errs
	default:
		errs = append(errs, fmt.Errorf("unknown action %s for %s", actionName, ApproverSetRevisionType))

		return errs
	}

	errs = append(errs, errors.New("unable to take action"))

	return errs
}

// StartApprovalProcess creates a change request to start the process of
// approvnig a new Change Request. If the Change Request was created
// no error is returned, otherwise an error will be returned.
//
// TODO: Check if a CR already exists for this object
// TODO: Ensure that if an error occures no changes are made.
func (a *ApproverSetRevision) StartApprovalProcess(request *http.Request, dbCache *DBCache, conf Config) (err error) {
	runame, ruerr := GetRemoteUser(request)
	if ruerr != nil {
		return errors.New("no username set")
	}

	if err = a.Prepare(dbCache); err != nil {
		return err
	}

	logger.Infof("starting approval for Approver Set Revision ID: %d", a.ID)

	approverSet := ApproverSet{}

	if err = dbCache.FindByID(&approverSet, a.ApproverSetID); err != nil {
		return err
	}

	currentRevision := approverSet.CurrentRevision

	logger.Debugf("Parent Approver Set ID: %d", approverSet.GetID())

	export := approverSet.GetExportVersion()

	changeRequestJSON, err1 := export.ToJSON()
	diff, err2 := export.GetDiff()

	if approverSet.PendingRevision.ID == approverSet.CurrentRevisionID.Int64 || err1 != nil || err2 != nil {
		return errors.New("unable to create diff for approval")
	}

	logger.Debugf("Diff Len: %d", len(diff))
	logger.Debugf("JSON Len: %d", len(changeRequestJSON))

	changeRequest := ChangeRequest{
		RegistrarObjectType: ApproverSetType,
		RegistrarObjectID:   a.ApproverSetID,
		ProposedRevisionID:  a.ID,
		State:               StateNew,
		InitialRevisionID:   approverSet.CurrentRevisionID,
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

	if len(currentRevision.RequiredApproverSets) == 0 {
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

	logger.Debugf("%d Required Approver Sets", len(currentRevision.RequiredApproverSets))

	for _, approverSet := range currentRevision.RequiredApproverSets {
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
		return fmt.Errorf("error finding change request: %w", err)
	}

	logger.Infof("Approver Set ID: %d", a.ID)

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

	// approverSet = ApproverSet{}
	// if err = db.FindByID(&approverSet, a.ApproverSetID); err != nil {
	approverSet = ApproverSet{Model: Model{ID: a.ApproverSetID}}

	if err = approverSet.PrepareDisplayShallow(dbCache); err != nil {
		return err
	}

	if approverSet.CurrentRevisionID.Valid {
		if approverSet.CurrentRevision.DesiredState != StateBootstrap {
			if approverSet.CurrentRevision.DesiredState == StateActive {
				approverSet.State = StateActivePendingApproval
			} else {
				approverSet.State = StateInactivePendingApproval
			}
		} else {
			approverSet.State = StatePendingBootstrap
		}
	} else {
		approverSet.State = StatePendingNew
	}

	approverSet.UpdatedBy = runame
	approverSet.UpdatedAt = TimeNow()

	if err = dbCache.Save(&approverSet); err != nil {
		return err
	}

	cr2 := ChangeRequest{}

	if err = dbCache.FindByID(&cr2, a.CR.ID); err != nil {
		return err
	}

	_, errs := cr2.UpdateState(dbCache, conf)
	if len(errs) != 0 {
		return errs[0]
	}

	return nil
}

// Promote will mark an ApproverSetRevision as the current revision for
// an Approver Set if it has not been cancelled or failed approval.
func (a *ApproverSetRevision) Promote(dbCache *DBCache) (err error) {
	tmp := ApproverSetRevision{Model: Model{ID: a.ID}}
	if err = tmp.PrepareShallow(dbCache); err != nil {
		return err
	}

	if tmp.RevisionState == StateCancelled || tmp.RevisionState == StateApprovalFailed {
		return errors.New("cannot promote revision in cancelled or approvalfailed state")
	}

	if a.CRID.Valid {
		changeRequest := ChangeRequest{}
		if err = dbCache.FindByID(&changeRequest, a.CRID.Int64); err != nil {
			return err
		}

		if changeRequest.State == StateApproved {
			tmp.RevisionState = a.DesiredState

			if tmp.PromotedTime == nil {
				tmp.PromotedTime = &time.Time{}
			}

			*tmp.PromotedTime = TimeNow()

			logger.Debug("Promoted ApproverSet Revision %d", a.ID)

			if err = dbCache.Save(&tmp); err != nil {
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

// Supersed will mark an ApproverSetRevision as a superseded revision for
// an ApproverSet.
func (a *ApproverSetRevision) Supersed(dbCache *DBCache) (err error) {
	tmp := ApproverSetRevision{Model: Model{ID: a.ID}}
	if err = tmp.PrepareShallow(dbCache); err != nil {
		return
	}

	if tmp.RevisionState != StateActive && tmp.RevisionState != StateInactive && tmp.RevisionState != StateBootstrap {
		return fmt.Errorf("cannot supersed a revision not in active, inactive or bootstrap state (in %s)", a.RevisionState)
	}

	tmp.RevisionState = StateSuperseded

	if tmp.SupersededTime == nil {
		tmp.SupersededTime = &time.Time{}
	}

	*tmp.SupersededTime = TimeNow()

	return dbCache.Save(&tmp)
}

// Decline will mark an ApproverSetRevision as decline for an ApproverSet.
func (a *ApproverSetRevision) Decline(dbCache *DBCache) (err error) {
	tmp := ApproverSetRevision{Model: Model{ID: a.ID}}

	if err = tmp.PrepareShallow(dbCache); err != nil {
		return
	}

	if tmp.RevisionState != StatePendingApproval {
		return errors.New("only revisions in pendingapproval may be declined")
	}

	tmp.RevisionState = StateApprovalFailed

	if tmp.ApprovalFailedTime == nil {
		tmp.ApprovalFailedTime = &time.Time{}
	}

	*tmp.ApprovalFailedTime = TimeNow()

	return dbCache.Save(&tmp)
}

// IsActive returns true if RevisionState is StateActive or StateBootstrap.
func (a *ApproverSetRevision) IsActive() bool {
	return a.RevisionState == StateActive || a.RevisionState == StateBootstrap
}

// GetRequiredApproverSets prepares object and returns the ApproverSets.
func (a *ApproverSetRevision) GetRequiredApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = a.Prepare(dbCache); err != nil {
		return
	}

	approverSets = a.RequiredApproverSets

	return
}

// GetInformedApproverSets prepares object and returns the ApproverSets.
func (a *ApproverSetRevision) GetInformedApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = a.Prepare(dbCache); err != nil {
		return
	}

	approverSets = a.InformedApproverSets

	return
}

// MigrateDBApproverSetRevision will run the automigrate function for
// the Approver Set Revision object.
func MigrateDBApproverSetRevision(dbCache *DBCache) {
	dbCache.AutoMigrate(&ApproverSetRevision{})
}
