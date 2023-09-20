package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// HostRevision represents individual versions of a Host Object.
type HostRevision struct {
	Model
	HostID int64

	RevisionState string
	DesiredState  string

	HostStatus                     string
	ClientDeleteProhibitedStatus   bool
	ServerDeleteProhibitedStatus   bool
	ClientTransferProhibitedStatus bool
	ServerTransferProhibitedStatus bool
	ClientUpdateProhibitedStatus   bool
	ServerUpdateProhibitedStatus   bool

	HostAddresses []HostAddress

	SavedNotes string `sql:"size:16384"`

	RequiredApproverSets []ApproverSet `gorm:"many2many:required_approverset_to_hostrevision"`
	InformedApproverSets []ApproverSet `gorm:"many2many:informed_approverset_to_hostrevision"`

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

// HostRevisionExport is an object that is used to export the
// current version of a Host Revision.
type HostRevisionExport struct {
	ID     int64 `json:"ID"`
	HostID int64 `json:"HostID"`

	RevisionState string `json:"RevisionState"`
	DesiredState  string `json:"DesiredState"`

	ClientDeleteProhibitedStatus   bool `json:"ClientDeleteProhibitedStatus"`
	ServerDeleteProhibitedStatus   bool `json:"ServerDeleteProhibitedStatus"`
	ClientTransferProhibitedStatus bool `json:"ClientTransferProhibitedStatus"`
	ServerTransferProhibitedStatus bool `json:"ServerTransferProhibitedStatus"`
	ClientUpdateProhibitedStatus   bool `json:"ClientUpdateProhibitedStatus"`
	ServerUpdateProhibitedStatus   bool `json:"ServerUpdateProhibitedStatus"`

	HostAddresses []HostAddress `json:"HostAddresses"`

	SavedNotes string `json:"SavedNotes"`

	ChangeRequestID int64 `json:"ChangeRequestID"`

	IssueCR string `json:"IssueCR"`
	Notes   string `json:"Notes"`

	RequiredApproverSets []ApproverSetExportShort `json:"RequiredApproverSets"`
	InformedApproverSets []ApproverSetExportShort `json:"InformedApproverSets"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// HostRevisionPage are used to hold all the information required to
// render the HostRevision HTML template.
type HostRevisionPage struct {
	IsEditable                 bool
	IsNew                      bool
	Revision                   HostRevision
	PendingActions             map[string]string
	ValidApproverSets          map[int64]string
	ParentHost                 *Host
	SuggestedRequiredApprovers map[int64]ApproverSetDisplayObject
	SuggestedInformedApprovers map[int64]ApproverSetDisplayObject
	SuggestedHostAddresses     []HostAddress

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (h *HostRevisionPage) GetCSRFToken() string {
	return h.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (h *HostRevisionPage) SetCSRFToken(newToken string) {
	h.CSRFToken = newToken
}

// The HostRevisionsPage type is used to render the html template
// which lists all of the HostRevisions currently in the
// registrar system
//
// TODO: Add paging support.
type HostRevisionsPage struct {
	HostRevisions []HostRevision

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (h *HostRevisionsPage) GetCSRFToken() string {
	return h.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (h *HostRevisionsPage) SetCSRFToken(newToken string) {
	h.CSRFToken = newToken
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (hre HostRevisionExport) ToJSON() (string, error) {
	if hre.ID <= 0 {
		return "", errors.New("invalid revision ID")
	}

	byteArr, jsonErr := json.MarshalIndent(hre, "", "  ")

	return string(byteArr), jsonErr
}

// GetDiff will return an empty string and an error for a revision. A
// Diff is not available for revision objects.
func (hre HostRevisionExport) GetDiff() (string, error) {
	return "", errors.New("unable to diff a single revision")
}

// GetExportVersion returns a export version of the HostRevision
// Object.
func (h *HostRevision) GetExportVersion() RegistrarObjectExport {
	export := HostRevisionExport{
		ID:                             h.ID,
		HostID:                         h.HostID,
		DesiredState:                   h.DesiredState,
		RevisionState:                  h.RevisionState,
		ClientDeleteProhibitedStatus:   h.ClientDeleteProhibitedStatus,
		ServerDeleteProhibitedStatus:   h.ServerDeleteProhibitedStatus,
		ClientTransferProhibitedStatus: h.ClientTransferProhibitedStatus,
		ServerTransferProhibitedStatus: h.ServerTransferProhibitedStatus,
		ClientUpdateProhibitedStatus:   h.ClientUpdateProhibitedStatus,
		ServerUpdateProhibitedStatus:   h.ServerUpdateProhibitedStatus,
		HostAddresses:                  h.HostAddresses,
		SavedNotes:                     h.SavedNotes,
		IssueCR:                        h.IssueCR,
		Notes:                          h.Notes,
		CreatedAt:                      h.CreatedAt,
		CreatedBy:                      h.CreatedBy,
	}
	export.RequiredApproverSets = GetApproverSetExportArr(h.RequiredApproverSets)
	export.InformedApproverSets = GetApproverSetExportArr(h.InformedApproverSets)

	if h.CRID.Valid {
		export.ChangeRequestID = h.CRID.Int64
	} else {
		export.ChangeRequestID = -1
	}

	return export
}

// GetExportVersionAt returns an export version of the Host Revision
// Object at the timestamp provided if possible otherwise an error is
// returned. If a pending version existed at the time it will be
// excluded from the object.
func (h *HostRevision) GetExportVersionAt(_ *DBCache, _ int64) (obj RegistrarObjectExport, err error) {
	return obj, errors.New("GetExportVersionAt is not usable for revisions")
}

// IsDesiredState will return true iff the state passed in matches the
// desired state of the revision.
func (h HostRevision) IsDesiredState(state string) bool {
	return h.DesiredState == state
}

// HasHappened is a helper function that will return true if the value
// of the timestamp who's name is passes has happened after the revision
// was created. This function is intended to be used with templates.
func (h HostRevision) HasHappened(actionType string) bool {
	switch actionType {
	case EventUpdated:
		return !h.UpdatedAt.Before(h.CreatedAt)
	case EventApprovalStarted:
		if h.ApprovalStartTime != nil {
			return !h.ApprovalStartTime.Before(h.CreatedAt)
		}

		return false
	case EventApprovalFailed:
		if h.ApprovalFailedTime != nil {
			return !h.ApprovalFailedTime.Before(h.CreatedAt)
		}

		return false
	case EventPromoted:
		if h.PromotedTime != nil {
			return !h.PromotedTime.Before(h.CreatedAt)
		}

		return false
	case EventSuperseded:
		if h.SupersededTime != nil {
			return !h.SupersededTime.Before(h.CreatedAt)
		}

		return false
	default:
		logger.Errorf("Unknown actiontype: %s", actionType)

		return false
	}
}

// GetPage will return an object that can be used to render the HTML
// template for the HostRevision.
func (h *HostRevision) GetPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &HostRevisionPage{IsNew: true}
	ret.IsEditable = h.IsEditable()
	ret.PendingActions = h.GetActions(true)

	if h.ID != 0 {
		ret.IsNew = false
	}

	ret.Revision = *h
	ret.ParentHost = &Host{}

	if err = dbCache.FindByID(ret.ParentHost, h.HostID); err != nil {
		return
	}

	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	return ret, err
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the query into dbcache.
func (h *HostRevision) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, h, func() (err error) {
		if err = dbCache.DB.Model(h).Related(&h.RequiredApproverSets, "RequiredApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return
			}
		}
		if err = dbCache.DB.Model(h).Related(&h.InformedApproverSets, "InformedApproverSets").Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return
			}
		}
		for idx := range h.RequiredApproverSets {
			if err = h.RequiredApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return
			}
		}
		for idx := range h.InformedApproverSets {
			if err = h.InformedApproverSets[idx].PrepareDisplayShallow(dbCache); err != nil {
				return
			}
		}

		return dbCache.DB.Where("host_revision_id = ?", h.ID).Find(&h.HostAddresses).Error
	})
}

// PrepareShallow populate all of the fields for a given object and
// not any of the linked object.
func (h *HostRevision) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, h, FuncNOPErrFunc)
}

// HostRevisionActionGOTOChangeRequest is the name of the action that
// will trigger a redirect to the current CR of an object.
const HostRevisionActionGOTOChangeRequest string = "gotochangerequest"

// GetActions will return a list of possible actions that can be taken
// while in the current state
//
// TODO: handle all states.
func (h *HostRevision) GetActions(isSelf bool) map[string]string {
	ret := make(map[string]string)

	if h.RevisionState == StateNew {
		ret["Start Approval Process"] = fmt.Sprintf("/action/%s/%d/startapproval", HostRevisionType, h.ID)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", HostRevisionType, h.ID)

		if isSelf {
			ret["View Parent Host"] = fmt.Sprintf("/view/%s/%d", HostType, h.HostID)
		} else {
			ret["View/Edit Host Revision"] = fmt.Sprintf("/view/%s/%d", HostRevisionType, h.ID)
		}
	}

	if h.RevisionState == StatePendingApproval {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", HostRevisionType, h.ID, HostRevisionActionGOTOChangeRequest)
		ret["Cancel Revision"] = fmt.Sprintf("/action/%s/%d/cancel", HostRevisionType, h.ID)

		if isSelf {
			ret["View Parent Host"] = fmt.Sprintf("/view/%s/%d", HostType, h.HostID)
		}
	}

	if h.RevisionState == StateCancelled {
		ret["View Change Request"] = fmt.Sprintf("/action/%s/%d/%s", HostRevisionType, h.ID, HostRevisionActionGOTOChangeRequest)

		if isSelf {
			ret["View Parent Host"] = fmt.Sprintf("/view/%s/%d", HostType, h.HostID)
		}
	}

	return ret
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (h *HostRevision) GetType() string {
	return HostRevisionType
}

// IsCancelled returns true iff the object has been canclled.
func (h *HostRevision) IsCancelled() bool {
	return h.RevisionState == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (h *HostRevision) IsEditable() bool {
	return h.RevisionState == StateNew
}

// GetPreview will generate and return the preview text for the associated
// with this host revision.
func (h *HostRevision) GetPreviewIPs() string {
	ret := ""

	for _, hos := range h.HostAddresses {
		ret += hos.IPAddress + "\n"
	}

	return ret
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple Host Revisions.
//
// TODO: Add paging support
// TODO: Add filtering.
func (h *HostRevision) GetAllPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &HostRevisionsPage{}

	err = dbCache.FindAll(&ret.HostRevisions)

	return ret, err
}

// Cancel will change the State of a revision from either "new" or
// "pendingapproval" to "cancelled"
//
// TODO: If in pending approval, cancel the change request and all
// approval objects
// TODO: Consider moving the query into dbcache.
func (h *HostRevision) Cancel(dbCache *DBCache, conf Config) (errs []error) {
	if h.RevisionState == StateNew || h.RevisionState == StatePendingApproval {
		h.RevisionState = StateCancelled

		if err := dbCache.DB.Model(h).UpdateColumns(HostRevision{RevisionState: StateCancelled}).Error; err != nil {
			errs = append(errs, err)

			return errs
		}

		err := dbCache.Purge(h)
		if err != nil {
			errs = append(errs, err)

			return errs
		}

		parent := Host{}

		if err := dbCache.FindByID(&parent, h.HostID); err != nil {
			errs = append(errs, err)

			return errs
		}

		_, parentErrs := parent.UpdateState(dbCache, conf)
		errs = append(errs, parentErrs...)

		if h.CRID.Valid {
			changeRequest := ChangeRequest{}

			if err := dbCache.FindByID(&changeRequest, h.CRID.Int64); err != nil {
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
func (h *HostRevision) TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, validCSRF bool, authMethod AuthType, conf Config) (errs []error) {
	switch actionName {
	case ActionCancel:
		if validCSRF {
			// TODO: Figure out who can cancel
			cancelErrs1 := h.Cancel(dbCache, conf)
			if cancelErrs1 != nil {
				errs = append(errs, cancelErrs1...)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", HostRevisionType, h.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case ActionStartApproval:
		if validCSRF {
			appErr := h.StartApprovalProcess(request, dbCache, conf)
			if appErr != nil {
				errs = append(errs, appErr)

				return errs
			}

			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", HostRevisionType, h.ID), http.StatusFound)

			return errs
		}

		errs = append(errs, ErrNoCSRFFound)

		return errs
	case HostRevisionActionGOTOChangeRequest:
		if h.CRID.Valid {
			http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", ChangeRequestType, h.CRID.Int64), http.StatusFound)

			return errs
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(h.GetExportVersion()))

			return errs
		}
	default:
		errs = append(errs, fmt.Errorf("unknown action %s for %s", actionName, HostRevisionType))

		return errs
	}

	errs = append(errs, errors.New("unable to take action"))

	return errs
}

// Promote will mark an HostRevision as the current revision for an
// Host if it has not been cancelled or failed approval.
func (h *HostRevision) Promote(dbCache *DBCache) (err error) {
	if err = h.Prepare(dbCache); err != nil {
		return err
	}

	if h.RevisionState == StateCancelled || h.RevisionState == StateApprovalFailed {
		return errors.New("cannot promote revision in cancelled or approvalfailed state")
	}

	if h.CRID.Valid {
		changeRequest := ChangeRequest{}

		if err = dbCache.FindByID(&changeRequest, h.CRID.Int64); err != nil {
			return err
		}

		if changeRequest.State == StateApproved {
			h.RevisionState = h.DesiredState

			if h.PromotedTime == nil {
				h.PromotedTime = &time.Time{}
			}

			*h.PromotedTime = TimeNow()

			if err = dbCache.Save(h); err != nil {
				return err
			}
		} else {
			return errors.New("cannot promote revision which has not been approved")
		}
	} else {
		return errors.New("no Change Request has been created for this host revision")
	}

	return err
}

// Supersed will mark an HostRevision as a superseded revision for
// an Host.
func (h *HostRevision) Supersed(dbCache *DBCache) (err error) {
	if err = h.Prepare(dbCache); err != nil {
		return
	}

	if h.RevisionState != StateActive && h.RevisionState != StateInactive && h.RevisionState != StateBootstrap {
		return fmt.Errorf("cannot supersed a revision not in active, inactive or bootstrap state (in %s)", h.RevisionState)
	}

	h.RevisionState = StateSuperseded
	if h.SupersededTime == nil {
		h.SupersededTime = &time.Time{}
	}

	*h.SupersededTime = TimeNow()

	return dbCache.Save(h)
}

// Decline will mark an HostRevision as decline for an Host.
func (h *HostRevision) Decline(dbCache *DBCache) (err error) {
	if err = h.Prepare(dbCache); err != nil {
		return
	}

	if h.RevisionState != StatePendingApproval {
		return errors.New("only revisions in pendingapproval may be declined")
	}

	h.RevisionState = StateApprovalFailed

	if h.ApprovalFailedTime == nil {
		h.ApprovalFailedTime = &time.Time{}
	}

	*h.ApprovalFailedTime = TimeNow()

	return dbCache.Save(h)
}

// StartApprovalProcess creates a change request to start the process of
// approvnig a new Change Request. If the Change Request was created
// no error is returned, otherwise an error will be returned.
//
// TODO: Check if a CR already exists for this object
// TODO: Ensure that if an error occures no changes are made.
func (h *HostRevision) StartApprovalProcess(request *http.Request, dbCache *DBCache, conf Config) (err error) {
	runame, ruerr := GetRemoteUser(request)
	if ruerr != nil {
		return errors.New("no username set")
	}

	logger.Infof("starting approval for ID: %d", h.ID)

	if h.RevisionState != StateNew {
		return fmt.Errorf("cannot start approval for %s %d, state is '%s' not 'new'", HostRevisionType, h.ID, h.RevisionState)
	}

	if err = h.Prepare(dbCache); err != nil {
		return err
	}

	host := Host{}
	if err = dbCache.FindByID(&host, h.HostID); err != nil {
		return err
	}

	currev := host.CurrentRevision
	logger.Debugf("Parent Host ID: %d", host.GetID())

	export := host.GetExportVersion()

	changeRequestJSON, err1 := export.ToJSON()
	diff, err2 := export.GetDiff()

	if host.PendingRevision.ID == host.CurrentRevisionID.Int64 || err1 != nil || err2 != nil {
		return errors.New("unable to create diff for approval")
	}

	logger.Debugf("Diff Len: %d", len(diff))
	logger.Debugf("JSON Len: %d", len(changeRequestJSON))

	changeRequest := ChangeRequest{
		RegistrarObjectType: HostType,
		RegistrarObjectID:   h.HostID,
		ProposedRevisionID:  h.ID,
		State:               StateNew,
		InitialRevisionID:   host.CurrentRevisionID,
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

	h.CR = changeRequest

	if err = h.CRID.Scan(changeRequest.ID); err != nil {
		return fmt.Errorf("error scananing change request id: %w", err)
	}

	h.RevisionState = StatePendingApproval
	h.UpdatedBy = runame
	h.UpdatedAt = TimeNow()

	if h.ApprovalStartTime == nil {
		h.ApprovalStartTime = &time.Time{}
	}

	*h.ApprovalStartTime = TimeNow()
	h.ApprovalStartBy = runame

	if err = dbCache.Save(h); err != nil {
		return err
	}

	host = Host{}

	if err = dbCache.FindByID(&host, h.HostID); err != nil {
		return err
	}

	if host.CurrentRevisionID.Valid {
		if host.CurrentRevision.DesiredState != StateBootstrap {
			if host.CurrentRevision.DesiredState == StateActive {
				host.State = StateActivePendingApproval
			} else {
				host.State = StateInactivePendingApproval
			}
		}
	} else {
		host.State = StatePendingNew
		sendErr := h.NewHostEmail(host.HostName, conf)
		if sendErr != nil {
			logger.Error(sendErr.Error())
		}
	}

	host.UpdatedBy = runame
	host.UpdatedAt = TimeNow()

	if err = dbCache.Save(&host); err != nil {
		return err
	}

	_, errs := changeRequest.UpdateState(dbCache, conf)

	if len(errs) != 0 {
		return errs[0]
	}

	return nil
}

// NewHostEmail will generate and send an email upon the creation of
// a new host
// TODO: Make this email a template and set the registrar name as a variable.
func (h *HostRevision) NewHostEmail(hostName string, conf Config) error {
	subject := fmt.Sprintf("Registrar: New Host Created - %s", hostName)
	message := fmt.Sprintf(`Hello,

This message is to inform you that a new host has been created in the
registrar system. You can view the host information at the link
below. If you feel that you should be part of one of the approver sets
associated with this host please send a message to
%s or respond to this thread.

%s/view/hostrevision/%d

Thank you,
The registrar Admins
`, conf.Email.FromEmail, conf.Server.AppURL, h.ID)

	return conf.SendAllEmail(subject, message, []string{conf.Email.Announce})
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
//
// TODO: add status flags
// TODO: Handle domain current status (new, transfer in).
func (h *HostRevision) ParseFromForm(request *http.Request, dbCache *DBCache) error {
	var err2, err3 error

	hostID, err1 := strconv.ParseInt(request.FormValue("revision_host_id"), 10, 64)

	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	h.CreatedBy = runame
	h.UpdatedBy = runame

	h.HostID = hostID
	h.RevisionState = StateNew

	h.SavedNotes = request.FormValue("revision_saved_notes")

	h.IssueCR = request.FormValue("revision_issue_cr")
	h.Notes = request.FormValue("revision_notes")

	hostAddresses, hostErrs := ParseHostAddresses(request, dbCache, "host_address")

	h.RequiredApproverSets, err2 = ParseApproverSets(request, dbCache, "approver_set_required_id", true)
	h.InformedApproverSets, err3 = ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

	h.DesiredState = GetActiveInactive(request.FormValue("revision_desiredstate"))

	//
	// HostStatus string
	//
	h.ClientDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_delete"))
	h.ServerDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_delete"))
	h.ClientTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_transfer"))
	h.ServerTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_transfer"))
	h.ClientUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_update"))
	h.ServerUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_update"))

	if err1 != nil {
		return fmt.Errorf("unable to parse revision host id: %w", err1)
	}

	if err2 != nil {
		return err2
	}

	if err3 != nil {
		return err3
	}

	if len(hostErrs) != 0 {
		var errStrs []string

		for _, err := range hostErrs {
			errStrs = append(errStrs, err.Error())
		}

		return errors.New(strings.Join(errStrs, "\n"))
	}

	h.HostAddresses = hostAddresses

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse HostRevision object with the changes that
// were made. An error object is always the second return value which
// is nil when no errors have occurred during parsing otherwise an error
// is returned.
//
// TODO: return error list rather than error
// TODO: Consider moving the query into dbcache.
func (h *HostRevision) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, _ Config) (err error) {
	if h.ID > 0 {
		runame, err := GetRemoteUser(request)
		if err != nil {
			return err
		}

		if h.RevisionState == StateNew {
			h.UpdatedBy = runame

			h.RevisionState = StateNew

			h.SavedNotes = request.FormValue("revision_saved_notes")

			h.IssueCR = request.FormValue("revision_issue_cr")
			h.Notes = request.FormValue("revision_notes")

			hostAddresses, hostErrs := ParseHostAddresses(request, dbCache, "host_address")

			RequiredApproverSets, err1 := ParseApproverSets(request, dbCache, "approver_set_required_id", true)
			InformedApproverSets, err2 := ParseApproverSets(request, dbCache, "approver_set_informed_id", false)

			h.DesiredState = GetActiveInactive(request.FormValue("revision_desiredstate"))

			h.ClientDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_delete"))
			h.ServerDeleteProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_delete"))
			h.ClientTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_transfer"))
			h.ServerTransferProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_transfer"))
			h.ClientUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_client_update"))
			h.ServerUpdateProhibitedStatus = GetCheckboxState(request.FormValue("revision_server_update"))

			if err1 != nil {
				return err1
			}

			if err2 != nil {
				return err2
			}

			if len(hostErrs) != 0 {
				var errStrs []string

				for _, err := range hostErrs {
					errStrs = append(errStrs, err.Error())
				}

				return errors.New(strings.Join(errStrs, "\n"))
			}

			updateAppSetErr := UpdateApproverSets(h, dbCache, "RequiredApproverSets", RequiredApproverSets)
			if updateAppSetErr != nil {
				return err
			}

			h.RequiredApproverSets = RequiredApproverSets

			updateAppSetErr = UpdateApproverSets(h, dbCache, "InformedApproverSets", InformedApproverSets)
			if updateAppSetErr != nil {
				return err
			}

			h.InformedApproverSets = InformedApproverSets

			// UpdateHostAddresses(h, db, "HostAddresses", hostAddresses)
			h.HostAddresses = hostAddresses

			if err := dbCache.DB.Where("host_revision_id = ?", h.ID).Delete(HostAddress{}).Error; err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot update an object not in the %s state", h.RevisionState)
		}

		return nil
	}

	return errors.New("to update an host revision the ID must be greater than 0")
}

// HostAddress is an object that will hold an IP address and the
// prodocol that the address is from (IPv4 / IPv6).
type HostAddress struct {
	ID             int64 `gorm:"primary_key:yes"`
	HostRevisionID int64
	IPAddress      string
	Protocol       int64
}

// ErrUnableToParseHostAddress defines the error returned when a host
// address is not able to be parsed.
var ErrUnableToParseHostAddress = errors.New("unable to parse Host Address")

// tokensCountInFormIPAddr is the number of tokens for an IP address form
// the web ui.
const tokensCountInFormIPAddr = 2

// ParseFromFormValue parses a value from a HTML form into a Host
// Address taking into account the encoding used by the web UI.
func (h *HostAddress) ParseFromFormValue(input string) error {
	tokens := strings.Split(input, "-")

	if len(tokens) == tokensCountInFormIPAddr {
		if tokens[0] == "v4" {
			h.Protocol = 4
		} else if tokens[0] == "v6" {
			h.Protocol = 6
		} else {
			return ErrUnableToParseHostAddress
		}

		_, addrErr := net.ResolveIPAddr("ip", tokens[1])

		if addrErr != nil {
			return ErrUnableToParseHostAddress
		}

		h.IPAddress = tokens[1]
	} else {
		return ErrUnableToParseHostAddress
	}

	return nil
}

// DisplayName formates a host address to be displayed as part of a
// HTML form.
func (h HostAddress) DisplayName() string {
	return fmt.Sprintf("%s - IPv%d", h.IPAddress, h.Protocol)
}

// FormValue will format the host address so it can be used as the
// value for a html form item.
func (h HostAddress) FormValue() string {
	return fmt.Sprintf("v%d-%s", h.Protocol, h.IPAddress)
}

// FormDivName creates a name that can be used as the ID for a div tag
// in the host selection forms.
func (h HostAddress) FormDivName() string {
	if h.Protocol == 4 || h.Protocol == 6 {
		return fmt.Sprintf("host_address_v%d-%s", h.Protocol, strings.Replace(strings.Replace(h.IPAddress, ".", "G", -1), ":", "S", -1))
	}

	return ""
}

// CompareHostAddressLists compares a list of HostAddresses to another
// set of HostAddresses that were from an export version of an object.
// If the counts match and the IDs for the host addresses match true is
// returned otherwise, false is returned.
func CompareHostAddressLists(halr []HostAddress, halre []HostAddress) bool {
	exportShortCount := len(halre)
	fullCount := len(halr)

	if fullCount != exportShortCount {
		return false
	}

	for _, full := range halr {
		found := false

		for _, export := range halre {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	for _, export := range halre {
		found := false

		for _, full := range halr {
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

// ParseHostAddresses takes a http Request, a database connection and
// the html ID of the host address list to parse and will return an
// array of Host Addresses that are represented in the http request. If
// an error occurs parsing the IP addresses a list of errors (one for
// each problem parsing) will be returned and the address will be
// excluded from the returned list.
func ParseHostAddresses(request *http.Request, _ *DBCache, htmlID string) ([]HostAddress, []error) {
	var (
		hosts []HostAddress
		errs  []error
	)

	for _, hostAddress := range request.Form[htmlID] {
		host := &HostAddress{}
		hostParseError := host.ParseFromFormValue(hostAddress)

		if hostParseError != nil {
			errs = append(errs, fmt.Errorf("unable to parse host address %s", hostAddress))
		} else {
			hosts = append(hosts, *host)
		}
	}

	return hosts, errs
}

// UpdateHostAddresses will update a set of host addresses for a host
//
//	that is passed in to make the list reflect the list passed as the
//
// third parameter.
//
// TODO: Consider moving the query into dbcache.
func UpdateHostAddresses(hostRev *HostRevision, dbCache *DBCache, association string, hostAddresses []HostAddress) error {
	err := dbCache.DB.Model(hostRev).Association(association).Clear().Error
	if err != nil {
		return err
	}

	for _, hostAddress := range hostAddresses {
		err = dbCache.DB.Model(hostRev).Association(association).Append(hostAddress).Error

		if err != nil {
			return err
		}
	}

	return nil
}

// Compare is used to compare an export version of an object to the
// full revision to verify that all of the values are the same.
func (hre HostRevisionExport) Compare(hostRevision HostRevision) (pass bool, errs []error) {
	pass = true

	if hre.ID != hostRevision.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if hre.HostID != hostRevision.HostID {
		errs = append(errs, fmt.Errorf("the HostID fields did not match"))
		pass = false
	}

	if hre.DesiredState != hostRevision.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if hre.ClientDeleteProhibitedStatus != hostRevision.ClientDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ServerDeleteProhibitedStatus != hostRevision.ServerDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ClientTransferProhibitedStatus != hostRevision.ClientTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ServerTransferProhibitedStatus != hostRevision.ServerTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ClientUpdateProhibitedStatus != hostRevision.ClientUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ServerUpdateProhibitedStatus != hostRevision.ServerUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.SavedNotes != hostRevision.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if hre.IssueCR != hostRevision.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if hre.Notes != hostRevision.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	// HostAddresses []HostAddress
	hostAddressCheck := CompareHostAddressLists(hostRevision.HostAddresses, hre.HostAddresses)
	if !hostAddressCheck {
		errs = append(errs, fmt.Errorf("the host address sets did not match"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetListToExportShort(hostRevision.RequiredApproverSets, hre.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetListToExportShort(hostRevision.InformedApproverSets, hre.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// CompareExport is used to compare an export version of an object to
// another export revision to verify that all of the values are the
// same.
func (hre HostRevisionExport) CompareExport(hostRevision HostRevisionExport) (pass bool, errs []error) {
	pass = true

	if hre.ID != hostRevision.ID {
		errs = append(errs, fmt.Errorf("the ID fields did not match"))
		pass = false
	}

	if hre.HostID != hostRevision.HostID {
		errs = append(errs, fmt.Errorf("the HostID fields did not match"))
		pass = false
	}

	if hre.DesiredState != hostRevision.DesiredState {
		errs = append(errs, fmt.Errorf("the DesiredState fields did not match"))
		pass = false
	}

	if hre.ClientDeleteProhibitedStatus != hostRevision.ClientDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ServerDeleteProhibitedStatus != hostRevision.ServerDeleteProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerDeleteProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ClientTransferProhibitedStatus != hostRevision.ClientTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ServerTransferProhibitedStatus != hostRevision.ServerTransferProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerTransferProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ClientUpdateProhibitedStatus != hostRevision.ClientUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ClientUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.ServerUpdateProhibitedStatus != hostRevision.ServerUpdateProhibitedStatus {
		errs = append(errs, fmt.Errorf("the ServerUpdateProhibitedStatus fields did not match"))
		pass = false
	}

	if hre.SavedNotes != hostRevision.SavedNotes {
		errs = append(errs, fmt.Errorf("the Saved Notes field did not match"))
		pass = false
	}

	if hre.IssueCR != hostRevision.IssueCR {
		errs = append(errs, fmt.Errorf("the Issue/CR fields did not macth"))
		pass = false
	}

	if hre.Notes != hostRevision.Notes {
		errs = append(errs, fmt.Errorf("the Notes fields did not macth"))
		pass = false
	}

	// HostAddresses []HostAddress
	hostAddressCheck := CompareHostAddressLists(hostRevision.HostAddresses, hre.HostAddresses)
	if !hostAddressCheck {
		errs = append(errs, fmt.Errorf("the host address sets did not match"))
		pass = false
	}

	requiredApproversCheck := CompareToApproverSetExportShortLists(hostRevision.RequiredApproverSets, hre.RequiredApproverSets)
	if !requiredApproversCheck {
		errs = append(errs, fmt.Errorf("the required approver sets did not match"))
		pass = false
	}

	informedApproversCheck := CompareToApproverSetExportShortLists(hostRevision.InformedApproverSets, hre.InformedApproverSets)
	if !informedApproversCheck {
		errs = append(errs, fmt.Errorf("the informed approver sets did not match"))
		pass = false
	}

	return pass, errs
}

// IsActive returns true if RevisionState is StateActive or StateBootstrap.
func (h *HostRevision) IsActive() bool {
	return h.RevisionState == StateActive || h.RevisionState == StateBootstrap
}

// GetRequiredApproverSets prepares object and returns the ApproverSets.
func (h *HostRevision) GetRequiredApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = h.Prepare(dbCache); err != nil {
		return
	}

	approverSets = h.RequiredApproverSets

	return
}

// GetInformedApproverSets prepares object and returns the ApproverSets.
func (h *HostRevision) GetInformedApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error) {
	if err = h.Prepare(dbCache); err != nil {
		return
	}

	approverSets = h.InformedApproverSets

	return
}

// MigrateDBHostRevision will run the automigrate function for the
// HostRevision object.
func MigrateDBHostRevision(dbCache *DBCache) {
	dbCache.AutoMigrate(&HostRevision{})
	dbCache.AutoMigrate(&HostAddress{})
	dbCache.DB.Model(&HostAddress{}).AddForeignKey("host_revision_id", "host_revisions(id)", "CASCADE", "RESTRICT")
}
