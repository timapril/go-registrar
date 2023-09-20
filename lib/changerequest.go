// Package lib provides the objects required to operate registrar
package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// More information about the ApproverSet Object and its States
// can be found in /doc/changerequest.md

// ChangeRequest is an object that represents a desired change in an
// object. It contains a link back to the originating object, revisions
// and Approvals that are required for the change request to be approved.
type ChangeRequest struct {
	Model
	RegistrarObjectType string `json:"RegistrarObjectType"`
	RegistrarObjectID   int64  `json:"RegistrarObjectID"`

	State string `json:"State"`

	Object RegistrarApprovalable `json:"Object" sql:"-"`

	InitialRevisionID  sql.NullInt64 `json:"InitialRevisionID"`
	ProposedRevisionID int64         `json:"ProposedRevisionID"`

	ChangeJSON string `json:"ChangeJSON" sql:"type:text;"`
	ChangeDiff string `json:"ChangeDiff" sql:"type:text;"`

	Approvals []Approval `json:"Approvals"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`
}

// ChangeRequestExport is an object that is used to export the current
// state of a ChangeRequest object.
type ChangeRequestExport struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	RegistrarObjectType string `json:"RegistrarObjectType"`
	RegistrarObjectID   int64  `json:"RegistrarObjectID"`

	InitialRevisionID  int64 `json:"InitialRevisionID"`
	ProposedRevisionID int64 `json:"ProposedRevisionID"`

	ChangeJSON string `json:"ChangeJSON"`
	ChangeDiff string `json:"ChangeDiff"`

	Approvals []ApprovalExport `json:"Approvals"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// ToJSON will return a string containing a JSON representation
// of the object. An empty string and an error are returned if a JSON
// representation cannot be returned.
func (cre ChangeRequestExport) ToJSON() (string, error) {
	if cre.ID <= 0 {
		return "", errors.New("id not set")
	}

	byteArr, jsonErr := json.MarshalIndent(cre, "", "  ")

	return string(byteArr), jsonErr
}

// GetDiff will return an empty string and an error for a revision. A
// Diff is not available for revision objects.
func (cre ChangeRequestExport) GetDiff() (string, error) {
	return "", errors.New("unable to diff a single revision")
}

// ChangeRequestPage is used to hold all the information required to
// render the Change Reuqest HTML template.
type ChangeRequestPage struct {
	CR             ChangeRequest
	PendingActions map[string]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (c *ChangeRequestPage) GetCSRFToken() string {
	return c.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (c *ChangeRequestPage) SetCSRFToken(newToken string) {
	c.CSRFToken = newToken
}

// SplitJSON is a helper function used by the ChangeRequest template
// to make a JSON object more readable when being displayed.
func (c ChangeRequestPage) SplitJSON(json string) []string {
	return strings.Split(json, "\n")
}

// ChangeRequestsPage is used to render the html template which lists
// all of the Change Requests currently in the registrar system
//
// TODO: Add paging support
// TODO: Add filtering support.
type ChangeRequestsPage struct {
	CRs []ChangeRequest

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (c *ChangeRequestsPage) GetCSRFToken() string {
	return c.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (c *ChangeRequestsPage) SetCSRFToken(newToken string) {
	c.CSRFToken = newToken
}

// GetExportVersion returns a export version of the ChangeRequest Object.
func (c *ChangeRequest) GetExportVersion() RegistrarObjectExport {
	export := ChangeRequestExport{
		ID:                  c.ID,
		State:               c.State,
		RegistrarObjectType: c.RegistrarObjectType,
		RegistrarObjectID:   c.RegistrarObjectID,
		ProposedRevisionID:  c.ProposedRevisionID,
		ChangeJSON:          c.ChangeJSON,
		ChangeDiff:          c.ChangeDiff,
		CreatedAt:           c.CreatedAt,
		CreatedBy:           c.CreatedBy,
	}

	if c.InitialRevisionID.Valid {
		export.InitialRevisionID = c.InitialRevisionID.Int64
	} else {
		export.InitialRevisionID = -1
	}

	for idx := range c.Approvals {
		export.Approvals = append(export.Approvals, (c.Approvals[idx].GetExportVersion()).(ApprovalExport))
	}

	return export
}

// GetExportVersionAt returns an export version of the Changer Request
// Object at the timestamp provided if possible otherwise an error is
// returned. If a pending version existed at the time it will be
// excluded from the object.
// TODO: Implement.
func (c *ChangeRequest) GetExportVersionAt(_ *DBCache, _ int64) (obj RegistrarObjectExport, err error) {
	return obj, errors.New("getExportVersionAt is not supported for change requests")
}

// ParseFromForm returns an error.
// Change Requests may not be added using a web form.
func (c *ChangeRequest) ParseFromForm(_ *http.Request, _ *DBCache) error {
	return errors.New("change Requests cannot be created thought the web UI")
}

// ParseFromFormUpdate returns an error.
// Change Requests may not be updated using a web form.
func (c *ChangeRequest) ParseFromFormUpdate(_ *http.Request, _ *DBCache, _ Config) error {
	return errors.New("change Requests cannot be updated thought the web UI")
}

// UpdateState can be called at any point to check the state of the
// Change Request and update it if necessary
//
// TODO: Implement.
func (c *ChangeRequest) UpdateState(dbCache *DBCache, conf Config) (changesMade bool, errs []error) {
	logger.Infof("Updating the state of Change Request %d", c.ID)

	cascadeUpdate := false
	changesMade = false

	if err := c.Prepare(dbCache); err != nil {
		errs = append(errs, err)

		return changesMade, errs
	}

	// When in the Pending Approval State
	switch c.State {
	case StateNew:
		if !c.Object.HasPendingRevision() {
			c.State = StateCancelled
			changesMade = true
			cascadeUpdate = true
		} else {
			appSets, err := c.Object.GetRequiredApproverSets(dbCache)
			if err != nil {
				logger.Debugf("Unable to find required approvers for %s %d, using default", c.RegistrarObjectType, c.RegistrarObjectID)
			}

			allFound := true
			for _, appSet := range appSets {
				setFound := false

				for _, app := range c.Approvals {
					if app.ApproverSetID == appSet.GetID() {
						setFound = true

						break
					}
				}
				if !setFound {
					errs = append(errs, fmt.Errorf("unable to find approval for Approver Set %d", appSet.GetID()))

					return changesMade, errs
				}
			}

			if allFound {
				logger.Infof("Found all required approvals, upgrading to %s", StatePendingApproval)
				c.State = StatePendingApproval
				changesMade = true
				cascadeUpdate = true
			}

			logger.Info("Looking for final approver set")

			for idx, app := range c.Approvals {
				if app.ApproverSetID == 1 {
					logger.Infof("Found Approval %d to mark as final approver set", app.ID)
					tmpApp := Approval{}

					if err := dbCache.FindByID(&tmpApp, app.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					tmpApp.IsFinalApproval = true
					tmpApp.State = StateNew

					if err := dbCache.Save(&tmpApp); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					c.Approvals[idx] = Approval{}
					if err := dbCache.FindByID(&c.Approvals[idx], tmpApp.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					changesMade = true
					cascadeUpdate = true
				}
			}
		}
		// TODO: Verify the correct set of approvers exist
		// If so, move to StatePendingApproval, Otherwise throw error
	case StatePendingApproval:
		if !c.Object.HasPendingRevision() {
			c.State = StateCancelled
			changesMade = true
			cascadeUpdate = true
		} else {
			var finalApproval Approval
			finalApprovalFound := false

			numApp := 0
			numApproved := 0
			numDeclined := 0
			numNoValidApprovers := 0
			numInactiveApproverSet := 0

			for _, app := range c.Approvals {
				numApp = numApp + 1

				if app.State == StateApproved {
					numApproved = numApproved + 1
				}

				if app.State == StateDeclined {
					numDeclined = numDeclined + 1
				}

				if app.State == StateNoValidApprovers {
					numNoValidApprovers = numNoValidApprovers + 1
				}

				if app.State == StateInactiveApproverSet {
					numInactiveApproverSet = numInactiveApproverSet + 1
				}

				if app.IsFinalApproval {
					finalApprovalFound = true
					finalApproval = app
				}
			}

			logger.Infof("Number of Approvals: %d", numApp)
			logger.Infof("Number of Approved Approvals: %d", numApproved)
			logger.Infof("Number of Declined Approvals: %d", numDeclined)
			logger.Infof("Number of No Valid Approver Approvals: %d", numNoValidApprovers)
			logger.Infof("Number of Inactive Approver Set Approvals: %d", numInactiveApproverSet)

			// If there has been 1 declined approval, the Change Request is denied
			if numDeclined >= 1 {
				logger.Infof("Change Request %d has been declined", c.ID)
				c.State = StateDeclined
				changesMade = true
				cascadeUpdate = true
			}

			if numApp == (numApproved + numNoValidApprovers + numInactiveApproverSet + 1) {
				logger.Info("All but final approver set done, time to mark final a ready")

				if finalApprovalFound {
					logger.Infof("Found approval = %d", finalApproval.ID)
					changesMade = true
					cascadeUpdate = true
				}
			}

			// If there is at least one approval and all valid approvals have
			// succeeded marke the Change Request as Approved
			if numApproved >= 1 && numApp == (numApproved+numNoValidApprovers+numInactiveApproverSet) {
				logger.Infof("Change Request %d has been approved", c.ID)
				// TODO: Check for implementation steps
				c.State = StateApproved
				changesMade = true
				cascadeUpdate = true
			}
		}
		// case StateCancelled:
		// // Do Nothing, Cancelled is a terminal State
		// case StateImplemented:
		// // Do Nothing, Implemented is a terminal State
		// case StateDeclined:
		// // Do Nothing, Declined is a terminal State
		// case StateApproved:
		// 	// Start Implementation Steps
	default:
		errs = append(errs, fmt.Errorf("UpdateState for Change Request at state \"%s\" not implemented", c.State))
	}

	if changesMade {
		c.Approvals = []Approval{}
		if err := dbCache.Save(c); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}
	}

	if cascadeUpdate {
		logger.Infof("Nofitify the interested object related to Change Request %d", c.ID)
		// Trigger all approvals to update their state as required
		updateErrs := c.UpdateApprovals(dbCache, conf)

		if len(updateErrs) != 0 {
			errs = append(errs, updateErrs...)
		}
	}

	return changesMade, errs
}

// UpdateApprovals will cycle through all of the change request
// approvals to make sure that they are in the correct state.
func (c *ChangeRequest) UpdateApprovals(dbCache *DBCache, conf Config) (errs []error) {
	var finalApproval Approval

	finalApprovalFound := false

	numApp := 0
	numApproved := 0
	numDeclined := 0
	numPendingApproval := 0
	numNoValidApprovers := 0
	numInactiveApproverSet := 0

	changeRequest := ChangeRequest{}

	if err := dbCache.FindByID(&changeRequest, c.ID); err != nil {
		errs = append(errs, err)

		return errs
	}

	for _, appin := range changeRequest.Approvals {
		app := Approval{}
		if err := dbCache.FindByID(&app, appin.ID); err != nil {
			errs = append(errs, err)

			return errs
		}

		app.UpdateState(dbCache, conf)

		numApp = numApp + 1

		switch app.State {
		case StateApproved:
			numApproved = numApproved + 1
		case StateDeclined:
			numDeclined = numDeclined + 1
		case StatePendingApproval:
			numPendingApproval = numPendingApproval + 1
		case StateNoValidApprovers:
			numNoValidApprovers = numNoValidApprovers + 1
		case StateInactiveApproverSet:
			numInactiveApproverSet = numInactiveApproverSet + 1
		}

		if app.IsFinalApproval {
			finalApprovalFound = true
			finalApproval = app
		}

		// Lets move the final approval back to new if there is still a
		// pending approval
		if changeRequest.State != StateApproved {
			if numPendingApproval > 1 && finalApprovalFound {
				if finalApproval.State == StatePendingApproval {
					finalApproval.State = StateNew

					err := dbCache.Save(&finalApproval)
					if err != nil {
						errs = append(errs, err)

						return errs
					}
				}
			}
		}
	}

	// Trigger the Object to update its state
	c.Object.UpdateState(dbCache, conf)

	return errs
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the db query into the dbcache object.
func (c *ChangeRequest) Prepare(dbCache *DBCache) (err error) {
	// db.DB = db.DB.Debug()
	return PrepareBase(dbCache, c, func() (err error) {
		if err = dbCache.DB.Where("change_request_id = ?", c.ID).Find(&c.Approvals).Error; err != nil {
			// fmt.Printf("ChangeRequest get Approvals failed: %s \n", err.Error())
			return err
		}

		for i := range c.Approvals {
			if err = c.Approvals[i].Prepare(dbCache); err != nil {
				return err
			}
		}

		obj, objErr := NewRegistrarObject(c.RegistrarObjectType)

		if objErr != nil {
			logger.Errorf("ERROR: Unknonwn object type %s", c.RegistrarObjectType)

			return errors.New("unknown object type")
		}
		crObj, ok := obj.(RegistrarCRObject)

		if !ok {
			logger.Errorf("ERROR: Object type %s is not valid for a change request", c.RegistrarObjectType)

			err = fmt.Errorf("object type %s is not valid for a change request", c.RegistrarObjectType)

			return err
		}

		if err = crObj.SetID(c.RegistrarObjectID); err != nil {
			return fmt.Errorf("cannot set registrar object id: %w", err)
		}

		if err = obj.Prepare(dbCache); err != nil {
			return fmt.Errorf("unable to prepare object: %w", err)
		}

		c.Object = crObj

		return nil
	})
}

// ReadyForFinalApproval determines if the change request has all of
// the required approvals other than the final approval.
func (c *ChangeRequest) ReadyForFinalApproval() bool {
	finalApprovalFound := false

	numApp := 0
	numApproved := 0
	numDeclined := 0
	numNoValidApprovers := 0
	numInactiveApproverSet := 0

	for _, app := range c.Approvals {
		numApp = numApp + 1

		if app.State == StateApproved {
			numApproved = numApproved + 1
		}

		if app.State == StateDeclined {
			numDeclined = numDeclined + 1
		}

		if app.State == StateNoValidApprovers {
			numNoValidApprovers = numNoValidApprovers + 1
		}

		if app.State == StateInactiveApproverSet {
			numInactiveApproverSet = numInactiveApproverSet + 1
		}

		if app.IsFinalApproval {
			finalApprovalFound = true
		}
	}

	if numApp == (numApproved + numNoValidApprovers + numInactiveApproverSet + 1) {
		logger.Info("All but final approver set done, time to mark final a ready")

		if finalApprovalFound {
			return true
		}
	}

	return false
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (c *ChangeRequest) GetType() string {
	return ChangeRequestType
}

// IsCancelled returns true iff the object has been canclled.
func (c *ChangeRequest) IsCancelled() bool {
	return c.State == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (c *ChangeRequest) IsEditable() bool {
	return false
}

// GetPage will return an object that can be used to render the HTML
// template for the Change Request.
func (c *ChangeRequest) GetPage(_ *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &ChangeRequestPage{}
	ret.CR = *c

	return ret, nil
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple Change Requests.
//
// TODO: Add paging support
// TODO: Add filtering.
func (c *ChangeRequest) GetAllPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &ChangeRequestsPage{}

	err = dbCache.FindAll(&ret.CRs)

	return ret, err
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary
//
// TODO: Implement.
func (c *ChangeRequest) TakeAction(response http.ResponseWriter, _ *http.Request, _ *DBCache, actionName string, _ bool, authMethod AuthType, _ Config) (errs []error) {
	switch actionName {
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(c.GetExportVersion()))

			return
		}
	}

	return
}

// MigrateDBChangeRequest will run the automigrate function for the
// Change Request and Approval objects.
func MigrateDBChangeRequest(dbCache *DBCache) {
	dbCache.AutoMigrate(&ChangeRequest{})
}
