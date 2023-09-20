// Package lib provides the objects required to operate registrar
package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
)

// TODO: Fix deprecated golang.org/x/crypto/openpgp package inclusion.

// Approval holds the data associated with getting a Change Request
// approved by a member of an Approver Set.
type Approval struct {
	Model

	State           string `json:"State"`
	IsSigned        bool   `json:"IsSigned"        sql:"DEFAULT:false"`
	IsFinalApproval bool   `json:"IsFinalApproval"`

	ChangeRequestID int64 `json:"ChangeRequestID"`
	ApproverSetID   int64 `json:"ApproverSetID"`

	ApprovalApproverSet ApproverSet `json:"ApprovalAPproverSet" sql:"-"`
	Signature           []byte      `json:"Signature"           sql:"signature,type:text"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`
}

// ApprovalExport is an object that is used to export the current
// state of an Approval object.
type ApprovalExport struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	IsSigned        bool `json:"IsSigned"`
	IsFinalApproval bool `json:"IsFinalApproval"`

	ChangeRequestID int64 `json:"ChangeRequestID"`
	ApproverSetID   int64 `json:"ApproverSetID"`

	Signature []byte `json:"Signature"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (a ApprovalExport) ToJSON() (string, error) {
	if a.ID <= 0 {
		return "", errors.New("ID not set")
	}

	byteArr, jsonErr := json.MarshalIndent(a, "", "  ")

	return string(byteArr), jsonErr
}

// GetDiff will return an empty string and an error for a revision. A
// Diff is not available for revision objects.
func (a ApprovalExport) GetDiff() (string, error) {
	return "", errors.New("unable to diff a single revision")
}

// ApprovalPage is used to hold all the information required to render
// the Approval HTML Template.
type ApprovalPage struct {
	App             Approval
	CanApprove      bool
	IsEditable      bool
	IsSigned        bool
	IsFinalApproval bool
	SigLen          int

	HasSigner bool
	Signers   []Approver

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApprovalPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApprovalPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// ApprovalsPage is used to render the HTML template which lists a group
// of approvals.
//
// TODO: add paging support.
// TODO: add filtering support.
type ApprovalsPage struct {
	Approvals []Approval

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (a *ApprovalsPage) GetCSRFToken() string {
	return a.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (a *ApprovalsPage) SetCSRFToken(newToken string) {
	a.CSRFToken = newToken
}

// GetExportVersion returns a export version of the Approval Object.
func (a *Approval) GetExportVersion() RegistrarObjectExport {
	export := ApprovalExport{
		ID:              a.ID,
		State:           a.State,
		ApproverSetID:   a.ApproverSetID,
		IsSigned:        a.IsSigned,
		IsFinalApproval: a.IsFinalApproval,
		ChangeRequestID: a.ChangeRequestID,
		Signature:       a.Signature,
		CreatedAt:       a.CreatedAt,
		CreatedBy:       a.CreatedBy,
	}

	return export
}

// GetExportVersionAt returns an export version of the Approval Object
// at the timestamp provided if possible otherwise an error is returned.
// If a pending version existed at the time it will be excluded from
// the object.
// TODO: Implement.
func (a *Approval) GetExportVersionAt(_ *DBCache, _ int64) (obj RegistrarObjectExport, err error) {
	return obj, errors.New("GetExportVersionAt is not supported for approvals")
}

// ParseFromForm returns an error.
// Approvals may not be added using a web form.
func (a *Approval) ParseFromForm(_ *http.Request, _ *DBCache) error {
	return errors.New("approvals may not be created via a web form")
}

// GetApprovalAttestation will extract the approval attestation from the
// signed message stored in the object, if there is one and the
// signature is valid.
func (a *Approval) GetApprovalAttestation(dbCache *DBCache) (appatt ApprovalAttestationUnmarshal, validSig bool, err error) {
	validSig = false
	block, _ := clearsign.Decode(a.Signature)

	if block == nil {
		return appatt, validSig, errors.New("no signature found")
	}

	blocksErr := a.ApprovalApproverSet.PrepareGPGKeys(dbCache)

	if blocksErr == nil {
		// entity, err)
		_, sigErr := openpgp.CheckDetachedSignature(a.ApprovalApproverSet, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body, nil)
		if sigErr == nil {
			jsonerr := json.Unmarshal(block.Bytes, &appatt)

			if jsonerr == nil {
				validSig = true

				return appatt, validSig, err
			}

			err = jsonerr
			validSig = false

			return appatt, validSig, err
		}

		err = sigErr

		return appatt, validSig, err
	}

	err = blocksErr

	return appatt, validSig, err
}

// CheckSignature inspectes the signature of the approval object to
// see if the signature was created by one of the valid approvers in the
// linked Approver Set.
func (a *Approval) CheckSignature(dbCache *DBCache) (validSig bool, action string, err error) {
	var att ApprovalAttestationUnmarshal

	action = "none"

	att, validSig, err = a.GetApprovalAttestation(dbCache)

	if validSig && err == nil {
		if att.Action == ActionApproved {
			action = ActionApproved
		} else if att.Action == ActionDeclined {
			action = ActionDeclined
		} else {
			err = fmt.Errorf("unknown approval action %s", att.Action)
		}
	}

	return
}

// ParseFromFormUpdate takes a http Request and parses the field values
// and populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
func (a *Approval) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, conf Config) (err error) {
	runame, ruerr := GetRemoteUser(request)
	if ruerr != nil {
		return errors.New("no username set")
	}

	updateMade := false
	updateFile := true

	file, _, fileErr := request.FormFile("sig")
	if fileErr != nil {
		logger.Error(fileErr.Error())

		return fmt.Errorf("error reading from form: %w", fileErr)
	}

	if updateFile {
		logger.Debug("an updated signature was uploaded")

		defer func() {
			if err != nil {
				closeErr := file.Close()
				if closeErr != nil {
					err = closeErr
				}
			}
		}()

		data, err2 := io.ReadAll(file)

		if err2 == nil {
			var processErr error

			updateMade, processErr = a.processUploadedSignature(data, runame, a.ApprovalApproverSet, a.State, dbCache, conf)

			if processErr != nil {
				logger.Debug("ERROR")

				return processErr
			}
		}
	}

	if updateMade {
		a.UpdatedBy = runame
		a.UpdatedAt = TimeNow()
	}

	return err
}

func (a *Approval) processUploadedSignature(sig []byte, username string, appSet ApproverSet, state string, dbCache *DBCache, conf Config) (updateMade bool, err error) {
	changeRequest := ChangeRequest{}

	err = dbCache.FindByID(&changeRequest, a.ChangeRequestID)
	if err != nil {
		return updateMade, err
	}

	updateAppErrs := changeRequest.UpdateApprovals(dbCache, conf)
	if len(updateAppErrs) != 0 {
		err = updateAppErrs[0]

		return updateMade, err
	}

	app := Approval{}

	err = dbCache.FindByID(&app, a.ID)
	if err != nil {
		return updateMade, err
	}

	if app.State != StatePendingApproval {
		err = fmt.Errorf("cannot modify approval in state %s", state)
		logger.Error(err.Error())

		return false, err
	}

	if app.State != state {
		err = errors.New("the object is now in a new state")
		logger.Error(err.Error())

		return false, err
	}

	updateMade = false
	tmp := new(Approval)
	tmp.Signature = sig
	// logger.Debugf("Sig: %s", string(sig))
	tmp.ApprovalApproverSet = appSet
	validSig, _, sigErr := tmp.CheckSignature(dbCache)

	if validSig {
		logger.Debug("Valid Signature")

		a.Signature = sig
		a.IsSigned = true
		updateMade = true
	} else {
		logger.Debug("Invalid Signature")

		if sigErr != nil {
			err = errors.New("unable to accept signature")
			logger.Errorf("%s", sigErr)
		}

		return updateMade, err
	}

	if updateMade {
		a.UpdatedBy = username
		a.UpdatedAt = TimeNow()
	}

	return updateMade, nil
}

// PostUpdate is called on an object following a change to the object.
//
// TODO: finish implementing.
func (a *Approval) PostUpdate(dbCache *DBCache, conf Config) error {
	newApproval := Approval{}

	if err := dbCache.FindByID(&newApproval, a.ID); err != nil {
		return err
	}

	err := newApproval.Prepare(dbCache)
	if err != nil {
		return err
	}

	changesMade, errs := newApproval.UpdateState(dbCache, conf)
	if len(errs) != 0 {
		return errors.New("errors processing approval")
	}

	logger.Infof("PostUpdate called on Approval %d", newApproval.ID)

	if changesMade {
		newApproval = Approval{}
		if err := dbCache.FindByID(&newApproval, a.ID); err != nil {
			return err
		}

		cm2, _ := newApproval.UpdateState(dbCache, conf)

		if cm2 {
			changeRequest := ChangeRequest{}

			if err := dbCache.FindByID(&changeRequest, newApproval.ChangeRequestID); err != nil {
				return err
			}

			_, errs := changeRequest.UpdateState(dbCache, conf)

			if len(errs) != 0 {
				return errs[0]
			}
		}
	}

	return nil
}

// NewApprovalEmail will generate and send an email upon the creation of
// a new approval.
// TODO: Consider making this email a template and switching the registrar name to a variable.
func (a *Approval) NewApprovalEmail(conf Config, dbCache *DBCache) error {
	if a.State == StateNew {
		subject := fmt.Sprintf("Registrar: Approval %d PENDING", a.ID)
		message := fmt.Sprintf(`Hello,

This message is to inform you that a new approval has been created in
the registrar system that requires approval by an Approver set that
you are a member of. You can view the Approval information at the link
below. If you have any questions about this approval, please send a
message to %s or respond to this thread.

%s/view/approval/%d

Thank you,
The registrar Admins
`, conf.Email.FromEmail, conf.Server.AppURL, a.ID)

		emails, err := a.GetApproverEmails(dbCache)
		if err != nil {
			return err
		}

		return conf.SendAllEmail(subject, message, emails)
	}

	return nil
}

// ApprovalUpdateEmail will generate and send an email following the
// update of an approval
// TODO: Consider making this email a template and switching the registrar name to a variable.
func (a *Approval) ApprovalUpdateEmail(newState string, conf Config, dbCache *DBCache) error {
	subject := fmt.Sprintf("Registrar: Approval %d %s", a.ID, newState)
	message := fmt.Sprintf(`Hello,

This message is to inform you that approval %d has been %s in
the registrar system. This approval is now completed. You can view
the Approval information at the link below. If you have any questions
about this approval, please send a message to
%s or respond to this thread.

%s/view/approval/%d

Thank you,
The registrar Admins
`, a.ID, newState, conf.Email.FromEmail, conf.Server.AppURL, a.ID)

	emails, err := a.GetApproverEmails(dbCache)
	if err != nil {
		return err
	}

	return conf.SendAllEmail(subject, message, emails)
}

// GetApproverEmails will extract the email addresses from each of the
// users of the approvers in the linked approver set.
func (a *Approval) GetApproverEmails(dbCache *DBCache) (emails []string, err error) {
	if !a.prepared {
		err = a.Prepare(dbCache)

		if err != nil {
			return emails, err
		}
	}

	if !a.ApprovalApproverSet.prepared {
		err = a.ApprovalApproverSet.Prepare(dbCache)

		if err != nil {
			return emails, err
		}
	}

	if !a.ApprovalApproverSet.CurrentRevision.prepared {
		err = a.ApprovalApproverSet.CurrentRevision.Prepare(dbCache)

		if err != nil {
			return emails, err
		}
	}

	for _, app := range a.ApprovalApproverSet.CurrentRevision.Approvers {
		if !app.prepared {
			err = app.Prepare(dbCache)

			if err != nil {
				return emails, err
			}
		}

		emails = append(emails, app.CurrentRevision.EmailAddress)
	}

	return emails, nil
}

// UpdateState can be called at any point to check the state of the
// Approver Set Revision and update it if necessary.
//
// TODO: Implement.
func (a *Approval) UpdateState(dbCache *DBCache, conf Config) (changesMade bool, errs []error) {
	changesMade = false
	cascadeState := false

	switch a.State {
	case StateNew:
		changeRequest := ChangeRequest{}

		if err := dbCache.FindByID(&changeRequest, a.ChangeRequestID); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}

		if changeRequest.IsCancelled() {
			a.State = StateCancelled
			changesMade = true
			cascadeState = true
		} else {
			changeRequest := ChangeRequest{}
			logger.Debugf("Change Request ID: %d", a.ChangeRequestID)

			if err := dbCache.FindByID(&changeRequest, a.ChangeRequestID); err != nil {
				errs = append(errs, err)

				return changesMade, errs
			}

			logger.Infof("Parent Change request state %s", changeRequest.State)

			if changeRequest.State == StatePendingApproval {
				newState, newStateErr := a.CheckValidityOfApproverSet(dbCache)

				if newStateErr != nil {
					logger.Errorf("Approval Error: %s", newState)

					errs = append(errs, newStateErr)

					return changesMade, errs
				}

				if newState == StatePendingApproval && a.State != newState {
					err := a.NewApprovalEmail(conf, dbCache)
					if err != nil {
						logger.Error(err.Error())
					}
				}

				if a.State != newState {
					a.State = newState
					changesMade = true
					cascadeState = true
				}
			}
		}
	case StatePendingApproval:
		changeRequest := ChangeRequest{}

		if err := dbCache.FindByID(&changeRequest, a.ChangeRequestID); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}

		if changeRequest.IsCancelled() {
			a.State = StateCancelled

			if err := a.ApprovalUpdateEmail("CANCELLED", conf, dbCache); err != nil {
				errs = append(errs, err)
			}

			changesMade = true
			cascadeState = true
		} else if len(a.Signature) != 0 {
			validSig, action, sigErr := a.CheckSignature(dbCache)

			if sigErr == nil && validSig {
				if changeRequest.Object != nil {
					verifies, verifyErrs := changeRequest.Object.VerifyCR(dbCache)

					if verifies {
						if action == ActionApproved {
							a.State = StateApproved

							if err := a.ApprovalUpdateEmail("APPROVED", conf, dbCache); err != nil {
								errs = append(errs, err)
							}

							cascadeState = true
						} else if action == ActionDeclined {
							a.State = StateDeclined

							if err := a.ApprovalUpdateEmail("DECLINED", conf, dbCache); err != nil {
								errs = append(errs, err)
							}

							cascadeState = true
						}
					} else {
						if len(verifyErrs) != 0 {
							errs = append(errs, verifyErrs...)
							a.Signature = []byte("")
						}
					}
				}
				changesMade = true
			}
		} else {
			newState, newStateErr := a.CheckValidityOfApproverSet(dbCache)
			if newStateErr != nil {
				errs = append(errs, newStateErr)

				return changesMade, errs
			}
			if newState == StatePendingApproval && a.State != newState {
				err := a.NewApprovalEmail(conf, dbCache)
				if err != nil {
					logger.Error(err.Error())
				}
			}
			if newState != a.State {
				a.State = newState
				changesMade = true
				cascadeState = true
			}
		}
	case StateNoValidApprovers, StateInactiveApproverSet:
		newState, newStateErr := a.CheckValidityOfApproverSet(dbCache)
		if newStateErr != nil {
			errs = append(errs, newStateErr)

			return changesMade, errs
		}

		if newState == StatePendingApproval && a.State != newState {
			err := a.NewApprovalEmail(conf, dbCache)
			if err != nil {
				logger.Error(err.Error())
			}
		}

		if newState != a.State {
			a.State = newState
			changesMade = true
		}
	case StateCancelled, StateApproved, StateDeclined, StateSkippedNoValidApprovers, StateSkippedInactiveApproverSet:
		// TerminalState, Nothing to do
	default:
		errs = append(errs, fmt.Errorf("don't know how to process state %s", a.State))
	}

	if changesMade {
		if err := dbCache.Save(a); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}
	}

	if cascadeState {
		changeRequest := ChangeRequest{}

		if err := dbCache.FindByID(&changeRequest, a.ChangeRequestID); err != nil {
			errs = append(errs, fmt.Errorf("unable to prepare CR %d", a.ChangeRequestID))

			return changesMade, errs
		}

		subChanges, subErrs := changeRequest.UpdateState(dbCache, conf)
		errs = append(errs, subErrs...)
		changesMade = changesMade || subChanges
	}

	return changesMade, errs
}

// CheckValidityOfApproverSet will return StateInactiveApproverSet if
// the approver set is not valid, StateNoValidApprovers if there were
// no valid approvers for the approverset and StatePendingApproval if
// the approver set is valid and has valid approvers.
func (a *Approval) CheckValidityOfApproverSet(dbCache *DBCache) (state string, err error) {
	approverSet := ApproverSet{}

	if err = dbCache.FindByID(&approverSet, a.ApproverSetID); err != nil {
		return state, err
	}

	changeRequest := ChangeRequest{}

	if err = dbCache.FindByID(&changeRequest, a.ChangeRequestID); err != nil {
		return state, err
	}

	if changeRequest.InitialRevisionID.Int64 == 1 && (changeRequest.RegistrarObjectType == ApproverType || changeRequest.RegistrarObjectType == ApproverSetType) {
		if changeRequest.RegistrarObjectID == 1 {
			return StatePendingApproval, nil
		}
	}

	if approverSet.State == StateInactive || approverSet.State == StateInactivePendingApproval {
		return StateInactiveApproverSet, nil
	}

	foundValidApprover := false

	for _, approver := range approverSet.CurrentRevision.Approvers {
		if approver.State == StateActive || approver.State == StateActivePendingApproval {
			foundValidApprover = true
		}
	}

	if !foundValidApprover {
		return StateNoValidApprovers, nil
	}

	if !a.IsFinalApproval || changeRequest.ReadyForFinalApproval() {
		return StatePendingApproval, nil
	}

	return StateNew, nil
}

// GetSigner is used to get the Approver(s) who have signed an approval
// returning an error if when an error occures trying to find Approvers
// from the signed object information.
func (a *Approval) GetSigner(dbCache *DBCache) (signers []Approver, err error) {
	if len(a.Signature) > 0 {
		block, _ := clearsign.Decode(a.Signature)
		blocksErr := a.ApprovalApproverSet.PrepareGPGKeys(dbCache)

		if blocksErr == nil {
			entity, err1 := openpgp.CheckDetachedSignature(a.ApprovalApproverSet, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body, nil)

			if entity != nil {
				for idenStr := range entity.Identities {
					app, errApp := a.ApprovalApproverSet.ApproverFromIdentityName(idenStr, dbCache)

					if errApp == nil {
						signers = append(signers, app)
					}
				}
			}

			err = err1

			return signers, err
		}

		err = blocksErr

		return signers, err
	}

	err = errors.New("signature Not Found or Invalid")

	return signers, err
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
func (a *Approval) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, a, func() (err error) {
		a.ApprovalApproverSet.ID = a.ApproverSetID

		return a.ApprovalApproverSet.Prepare(dbCache)
	})
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (a *Approval) GetType() string {
	return ApprovalType
}

// IsCancelled returns true iff the object has been canclled.
func (a *Approval) IsCancelled() bool {
	return a.State == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (a *Approval) IsEditable() bool {
	return a.State != StateApproved && a.State != StateDeclined &&
		a.State != StateCancelled && a.State != StateSkippedNoValidApprovers &&
		a.State != StateSkippedInactiveApproverSet
}

// GetPage will return an object that can be used to render the HTML
// template for the Approval.
func (a *Approval) GetPage(dbCache *DBCache, _ string, email string) (rop RegistrarObjectPage, err error) {
	ret := &ApprovalPage{}
	ret.App = *a
	err = a.ApprovalApproverSet.Prepare(dbCache)

	if err != nil {
		return ret, err
	}

	ret.CanApprove, err = a.ApprovalApproverSet.IsValidApproverByEmail(email, dbCache)

	if err != nil {
		return ret, err
	}

	ret.IsEditable = a.IsEditable()

	if a.IsSigned {
		ret.SigLen = len(a.Signature)
	} else {
		ret.SigLen = -1
	}

	err = a.ApprovalApproverSet.Prepare(dbCache)

	if err != nil {
		return ret, err
	}

	signers, err := a.GetSigner(dbCache)

	if err != nil {
		ret.HasSigner = false
	} else {
		ret.HasSigner = true
		ret.Signers = signers
	}

	return ret, nil
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple Approvals.
//
// TODO: Implement.
// TODO: Add paging support.
// TODO: Add filtering.
func (a *Approval) GetAllPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &ApprovalsPage{}

	err = dbCache.FindAll(&ret.Approvals)
	if err != nil {
		return ret, err
	}

	for idx := range ret.Approvals {
		err = ret.Approvals[idx].Prepare(dbCache)

		if err != nil {
			return ret, err
		}
	}

	return ret, nil
}

// ApprovalAttestationUnmarshal allows the Approval Attestations to be
// unpacked knowing that multiple types may be present.
type ApprovalAttestationUnmarshal struct {
	ApprovalID int64           `json:"ApprovalID"`
	ExportRev  json.RawMessage `json:"ExportRev"`
	Username   string          `json:"Username"`
	Action     string          `json:"Action"`
	ObjectType string          `json:"ObjectType"`
	Signatures [][]byte        `json:"Signature"`
}

// ApprovalAttestation is used to create the object that is exported
// and presented to an Approver so they can either approve or decline
// an approval. The exported object is then signed using a GPG key
// and then resubmitted to the registrar system.
type ApprovalAttestation struct {
	ApprovalID int64                 `json:"ApprovalID"`
	ExportRev  RegistrarObjectExport `json:"ExportRev"`
	Username   string                `json:"Username"`
	Action     string                `json:"Action"`
	ObjectType string                `json:"ObjectType"`
	Signatures [][]byte              `json:"Signature"`
}

// ToJSON will return a string containing a JSON representation of the
// object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (a ApprovalAttestation) ToJSON() (string, error) {
	if a.ApprovalID <= 0 {
		return "", errors.New("unable to export an assertion that has no approval id")
	}

	byteArr, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return string(byteArr), err
	}

	return string(byteArr), nil
}

// GetDownload is used to generate an approval assertion that can be
// used to approve or decline the Approval.
//
// TODO: add an error to the return.
func (a *Approval) GetDownload(dbCache *DBCache, username string, method string) string {
	aa, err := a.GetDownloadAttestation(dbCache, username, method)
	if err != nil {
		return ""
	}

	json, _ := aa.ToJSON()

	return fmt.Sprint(json)
}

// GetDownloadAttestation will create and return an ApprovalAttestation
// object for the calling Approval. If the method is not valid an error
// will be returned.
func (a *Approval) GetDownloadAttestation(dbCache *DBCache, username string, method string) (aa ApprovalAttestation, err error) {
	if method != ActionApproved && method != ActionDeclined {
		err = fmt.Errorf("invalid approval method %s", method)

		return aa, err
	}

	changeRequest := ChangeRequest{}

	if err = dbCache.FindByID(&changeRequest, a.ChangeRequestID); err != nil {
		return aa, err
	}

	aa = ApprovalAttestation{
		ApprovalID: a.ID,
		ExportRev:  changeRequest.Object.GetExportVersion(),
		Username:   username,
		Action:     method,
		ObjectType: changeRequest.RegistrarObjectType,
	}

	if a.IsFinalApproval {
		aa.Signatures = make([][]byte, 0)

		for _, app := range changeRequest.Approvals {
			if app.ID != a.ID {
				aa.Signatures = append(aa.Signatures, app.Signature)
			}
		}
	}

	return aa, nil
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
//
// TODO: Implement.
func (a *Approval) TakeAction(responseWriter http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, validCSRF bool, authMethod AuthType, conf Config) (errs []error) {
	switch actionName {
	// add case "downloadsig". Dont forget to check for no sig first
	case SignatureDownloadType:
		// downloadsig is used to download the existing signiture object
		// that was submitted for an approval
		if a.IsSigned {
			if authMethod == RemoteUserAuthType {
				responseWriter.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=approval%d.sig", a.ID))
				responseWriter.Header().Set("Content-Type", request.Header.Get("Content-Type"))
				fmt.Fprintf(responseWriter, "%s", a.Signature)
			} else if authMethod == CertAuthType {
				logger.Debugf("Downloading signature for Approval %d", a.ID)

				APIRespond(responseWriter, GenerateSignatureResponse(a.Signature))
			}

			return errs
		}

		errs = append(errs, fmt.Errorf("Approval %d is not signed", a.ID))

		return errs
	case "download":
		// download is used to download the attestation that must be signed
		// in order to approve a request. Setting the "approver" query param
		// can be used to override the remote username
		ruemail, _ := GetRemoteUserEmail(request, conf)
		runame, _ := GetRemoteUser(request)

		if err := request.ParseForm(); err != nil {
			errs = append(errs, err)

			return errs
		}

		queryApproverIDs := request.Form["approverid"]

		if len(queryApproverIDs) == 1 {
			objIDparsed, parseError := strconv.ParseInt(queryApproverIDs[0], 10, 64)

			if parseError != nil {
				errs = append(errs, fmt.Errorf("unable to parse ID %s", queryApproverIDs[0]))

				return errs
			}

			approver := Approver{}

			if err := dbCache.FindByID(&approver, objIDparsed); err != nil {
				errs = append(errs, fmt.Errorf("Approver %d was not found", objIDparsed))

				return errs
			}

			if !approver.CurrentRevisionID.Valid {
				errs = append(errs, fmt.Errorf("Approver %d has no current revision", objIDparsed))

				return errs
			}

			runame = approver.CurrentRevision.Username
			ruemail = approver.GetCurrentValue(ApproverFieldEmailAddres)
		} else if len(queryApproverIDs) > 1 {
			errs = append(errs, errors.New("error creating download. Only one approver may be set"))

			return errs
		}

		valid, err := a.ApprovalApproverSet.IsValidApproverByEmail(ruemail, dbCache)
		if err != nil {
			return []error{err}
		}

		if !valid {
			errs = append(errs, errors.New("you are not authorized to approve this request "))

			return errs
		}

		action := ActionApproved
		actions := request.Form["action"]

		if len(actions) == 1 {
			if actions[0] == ActionDeclined {
				action = ActionDeclined
			} else if actions[0] == ActionApproved {
				action = ActionApproved
			} else {
				errs = append(errs, fmt.Errorf("unable to treat %s as an action (%s or %s)", actions[0], ActionApproved, ActionDeclined))

				return errs
			}
		} else if len(actions) > 1 {
			errs = append(errs, errors.New("error creating download. Only one method may be set"))

			return errs
		}

		if authMethod == RemoteUserAuthType {
			logger.Debugf("Downloading approval for %s", runame)
			output := a.GetDownload(dbCache, runame, action)

			responseWriter.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=approval%d-%s.txt", a.ID, runame))
			responseWriter.Header().Set("Content-Type", request.Header.Get("Content-Type"))

			fmt.Fprint(responseWriter, output)
		} else if authMethod == CertAuthType {
			logger.Debugf("Downloading approval for %s", runame)
			output, err := a.GetDownloadAttestation(dbCache, runame, action)
			if err != nil {
				errs = append(errs, err)

				return errs
			}

			response, marshAllErr := json.MarshalIndent(output, "", "  ")

			if marshAllErr != nil {
				errs = append(errs, marshAllErr)

				return errs
			}

			APIRespond(responseWriter, GenerateApprovalDownload(response, []error{err}))
		}

		return errs
	case SignatureUploadType:
		if authMethod == CertAuthType && validCSRF {
			requestBody, reqErr := io.ReadAll(request.Body)

			if reqErr != nil {
				errs = append(errs, reqErr)

				return errs
			}

			user, ruerr := GetAPIUser(request, dbCache, conf)

			if ruerr != nil {
				errs = append(errs, ruerr)

				return errs
			}

			if len(strings.TrimSpace(string(requestBody))) != 0 {
				var reqObj APIRequest
				unmarshalErr := json.Unmarshal(requestBody, &reqObj)

				if unmarshalErr != nil {
					errs = append(errs, unmarshalErr)

					return errs
				}

				if reqObj.MessageType == SignatureUploadType && reqObj.Signature != nil {
					ret := new(Approval)
					setErr := ret.SetID(a.ID)

					if setErr != nil {
						errs = append(errs, setErr)

						return errs
					}

					prepErr := ret.Prepare(dbCache)

					if prepErr != nil {
						errs = append(errs, prepErr)

						return errs
					}

					updateMade, processErr := ret.processUploadedSignature(reqObj.Signature.Signature, user.GetCertName(), a.ApprovalApproverSet, a.State, dbCache, conf)

					if processErr != nil {
						errs = append(errs, processErr)

						return errs
					}

					if updateMade {
						if err := dbCache.Update(a, ret); err != nil {
							errs = append(errs, err)

							return errs
						}

						updateApp := Approval{}

						if err := dbCache.FindByID(&updateApp, a.ID); err != nil {
							errs = append(errs, err)

							return errs
						}

						err := updateApp.PostUpdate(dbCache, conf)
						if err != nil {
							errs = append(errs, err)
						}
					}
				}
			} else {
				errs = append(errs, errors.New("no signature found"))
			}
		} else {
			if authMethod == CertAuthType {
				errs = append(errs, errors.New("a valid CSRF token is required to take that action"))
			} else {
				errs = append(errs, errors.New("unable to preform requested action"))
			}
		}

		return errs
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(responseWriter, GenerateObjectResponse(a.GetExportVersion()))

			return errs
		}

		return errs
	default:
		errs = append(errs, fmt.Errorf("unknown action %s for %s", actionName, ApprovalType))

		return errs
	}
}

// AfterSave is used to hook specific functions that are required for
// propogating the approval to the change request and attached objects
// after an approval has changed.
func (a *Approval) AfterSave(dbin *gorm.DB) error {
	dbCache := NewDBCache(dbin)
	app := Approval{}

	if err := dbCache.FindByID(&app, a.ID); err != nil {
		return err
	}

	return app.PostUpdate(&dbCache, GetConfig())
}

// MigrateDBApproval will run the automigrate function for the Approval
// object.
func MigrateDBApproval(dbCache *DBCache) {
	dbCache.AutoMigrate(&Approval{})
}
