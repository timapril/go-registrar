// Package lib provides the objects required to operate registrar
package lib

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/op/go-logging"
)

// Creates the log handle for the entire package.
var logger = logging.MustGetLogger("reg")

const (
	// EventUpdated is used to represent when an object is update.
	EventUpdated string = "Updated"

	// EventApprovalStarted is used to represent when an object has
	// started the approval process.
	EventApprovalStarted string = "ApprovalStarted"

	// EventApprovalFailed is used to represent when an object has
	// failed the approval process.
	EventApprovalFailed string = "ApprovalFailed"

	// EventPromoted is used to represent when an object has completed
	// the approval process successfully.
	EventPromoted string = "Promoted"

	// EventSuperseded is used to represent when an object has been
	// superseded by a later object.
	EventSuperseded string = "Superseded"
)

// All constants prefixed with State are used to represent the state
// of an object. Not all objects can be in all states.
const (
	// StateActive is used to indicate that an object is currently active
	// or in the case that it is the value of DesiredState, the state
	// which the parent object should be in if approved.
	StateActive string = "active"

	// StateApproved is used to incidate that the approval for an object
	// has completed successfully.
	StateApproved string = "approved"

	// StateBootstrap is used to incidate that an object is being used to
	// start the system for the first time.
	StateBootstrap string = "bootstrap"

	// StatePendingBootstrap is used to indicate that an object is waiting
	// for a bootstrap approval to be completed.
	StatePendingBootstrap string = "pendingbootstrap"

	// StateCancelled is used to incidate that the approval process for
	// the object revision has been cancelled.
	StateCancelled string = "cancelled"

	// StateDeclined is used to indicate that the change request for a
	// revision has been declined by at least one Approver Set.
	StateDeclined string = "declined"

	// StateApprovalFailed is used to indicate that the approval process
	// for a revision has failed for some reason.
	StateApprovalFailed string = "approvalfailed"

	// StateInactive is used to indicate that an object is currently
	// inactive or in the case that it is the value of DesiredState, the
	// state which the parent object should be in if approved.
	StateInactive string = "inactive"

	// StateExternal is used to indicate that an object is currently
	// external or in the case that it is the value of DesiredState, the
	// state which the parent object should be in if approved.
	StateExternal string = "external"

	// StateNewExternal is used to indicate that an object is currently
	// new but will be external when the first approval is completed.
	StateNewExternal string = "new-external"

	// StateNew is used to indicate that an object has been created but
	// not submitted for approval (in the case of a revision), no
	// revisions have been submitted (in the case of a parent object) or
	// an approval that has not started (in the case of an Approval).
	StateNew string = "new"

	// StatePendingApproval is used to indicate that an object is waiting
	// on approvals to be completed.
	StatePendingApproval string = "pendingapproval"

	// StateActivePendingApproval is used to indicate that an object is
	// currently active but pending approval.
	StateActivePendingApproval string = "activependingapproval"

	// StateInactivePendingApproval is used to indicate that an object is
	// currently inactive but pending approval.
	StateInactivePendingApproval string = "inactivependingapproval"

	// StateExternalPendingApproval is used to indicate that an object is
	// currently external but pending approval.
	StateExternalPendingApproval string = "externalpendingapproval"

	// StatePendingNew is used to indicate parent objects that have a
	// revision that has been submitted and now current revision.
	StatePendingNew string = "pendingnew"

	// StatePendingNewExternal is used to indicate parent objects that
	// have a revision that has been submitted and now current revision.
	StatePendingNewExternal string = "pendingnewexternal"

	// StateNoValidApprovers is used to indicate that there are no valid
	// Approvers within the Approver Set that is required for an Approval.
	StateNoValidApprovers string = "novalidapprovers"

	// StateSkippedNoValidApprovers is used to indicate that there were no
	// valid approvers for an Approver Set when the Change Request was
	// fully approved (for a Change Request to be approved, at least one
	// Approval must be approved).
	StateSkippedNoValidApprovers string = "skippednovalidapprovers"

	// StateInactiveApproverSet is used to indicate that the Approver
	// set required for an Approval is in an Inactive state at the time.
	StateInactiveApproverSet string = "inactiveapproverset"

	// StateSkippedInactiveApproverSet is used to indicate that the
	// Approver set required for an Approval was not in an active state
	// after all other available approver sets had completed the approval
	// process. (for a Change Request to be approved, at least one
	// Approval must be approved).
	StateSkippedInactiveApproverSet string = "skippedinactiveapproverset"

	// StateImplemented is used to indicate that a Change Request has been
	// approved and all implementation steps have completed.
	StateImplemented string = "implemented"

	// StateSuperseded is used to indicate that a new revision has been
	// approved and is now the current revsion.
	StateSuperseded string = "superseded"
)

// All contants prefixed with Action are used to represent an action
// taken by an approver.
const (
	// ActionApproved is used to incidate when an approval acction has
	// taken place.
	ActionApproved string = "approve"

	// ActionDeclined is used to indicate when approval has been declined
	// for some action.
	ActionDeclined string = "decline"

	// ActionCancel is used to indicate when the cancel action has been
	// requested by a user.
	ActionCancel string = "cancel"

	// ActionStartApproval is used to indicate when the start approval
	// action has been requested by a user.
	ActionStartApproval string = "startapproval"

	// ActionGet is used to represent the API action where the current
	// selected object is returned to the requestor.
	ActionGet string = "get"

	// ActionUpdateEPPInfo is used to reprsent the API action where
	// the client is trying to update the EPP info of the object.
	ActionUpdateEPPInfo string = "updateEPPInfo"

	// ActionUpdateEPPCheckRequired is used to represent that API action where the
	// client is trying to unset the check_reqired field for an object.
	ActionUpdateEPPCheckRequired string = "updateEPPCheckRequired"

	// ActionTriggerUpdate is used to represent that a parent update should have
	// its update state process triggered.
	ActionTriggerUpdate string = "triggerUpdate"

	// ActionUpdatePreview is used to represent that the requestor would like to
	// update the preview fields for an object.
	ActionUpdatePreview string = "updatePreview"
)

// UnknownObjectTypeError indicates the passed type is not supported.
const UnknownObjectTypeError string = "Unknown object type"

// SavedObjectNote is the name that can be used to reference the
// saved object note for a revision.
const SavedObjectNote string = "SavedObjectNote"

// RegistrarParent is an interface that represents a parent object where the
// child revision can be found.
type RegistrarParent interface {
	GetID() int64
	GetPendingRevision() RegistrarObject
}

// RegistrarObject is defined to allow shared CRUD tasks within a
// single application.
type RegistrarObject interface {
	ParseFromForm(request *http.Request, dbCache *DBCache) error
	ParseFromFormUpdate(request *http.Request, dbCache *DBCache, conf Config) error
	Prepare(dbCache *DBCache) error
	SetID(int64) error
	GetID() int64
	GetType() string
	IsEditable() bool
	IsCancelled() bool
	GetPage(dbCache *DBCache, username string, email string) (RegistrarObjectPage, error)
	// TODO: Add paging support to getall page
	GetAllPage(dbCache *DBCache, username string, email string) (RegistrarObjectPage, error)
	TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, validCSRF bool, authMethod AuthType, conf Config) []error
	GetExportVersion() RegistrarObjectExport
	GetExportVersionAt(dbCache *DBCache, timestamp int64) (RegistrarObjectExport, error)
}

// RegistrarApprovalable is an interface that is used for objects
// that may have revisions and can be used to get information related
// to the current state of a revision.
type RegistrarApprovalable interface {
	GetRequiredApproverSets(dbCache *DBCache) ([]ApproverSet, error)
	GetInformedApproverSets(dbCache *DBCache) ([]ApproverSet, error)
	UpdateState(dbCache *DBCache, conf Config) (bool, []error)
	GetExportVersion() RegistrarObjectExport
	IsCancelled() bool
	HasPendingRevision() bool
	VerifyCR(dbCacheb *DBCache) (checksOut bool, errs []error)
}

// RegistrarObjectPage is an interface that is used to define
// methods that can be used to help generate HTML templates.
type RegistrarObjectPage interface {
	GetCSRFToken() string
	SetCSRFToken(string)
}

// RegistrarObjectExport is an interface that can be used to create a
// diff or json for export to be used in signing or verification
// operations. Export objects only include a restricted subset of object
// fields and prevent reference loops.
type RegistrarObjectExport interface {
	ToJSON() (string, error)
	GetDiff() (string, error)
}

// ErrNoCSRFFound is an error that is used when no valid CSRF token is
// found when a token is required for the action.
var ErrNoCSRFFound = errors.New("unable to find valid CSRF token")

// A NotExportableObject is used when trying to fully implement a
// Registrar object but do not want to create exportable versions
// of the object.
type NotExportableObject struct {
	Type string
}

// GetDiff on a NotExportableObject will result in an empty string and
// an error.
func (a NotExportableObject) GetDiff() (string, error) {
	return "", errors.New("unable to export this type")
}

// ToJSON on a NotExportableObject will result in an empty string and
// an error.
func (a NotExportableObject) ToJSON() (string, error) {
	return "", errors.New("unable to export this type")
}

// AuthType is a type that is used to describe the authentication type
// that was used for the request that has triggered the action.
type AuthType string

// RemoteUserAuthType indicates that the authentication type was a
// remote user header.
const RemoteUserAuthType AuthType = "remote_user_auth"

// CertAuthType indicates that the authentication type was a client
// certificate.
const CertAuthType AuthType = "cert_user_auth"

// ApproverType is the string used to represent the Approver object.
const ApproverType string = "approver"

// ApproverRevisionType is the string used to represent the Approver
// Revision object.
const ApproverRevisionType string = "approverrevision"

// ApproverSetType is the string used to represent the Approver Set
// object.
const ApproverSetType string = "approverset"

// ApproverSetRevisionType is the string used to represent the Approver
// Set Revision object.
const ApproverSetRevisionType string = "approversetrevision"

// ChangeRequestType is the string used to represent the Change Request
// object.
const ChangeRequestType string = "changerequest"

// ApprovalType is the string used to represent the Approval object.
const ApprovalType string = "approval"

// ContactType is the string used to represent the Contact object.
const ContactType string = "contact"

// ContactRevisionType is the string used to represent the Contact
// Revision object.
const ContactRevisionType string = "contactrevision"

// HostType is the string used to represent the Host object.
const HostType string = "host"

// HostRevisionType is the string used to represent the Host Revision
// object.
const HostRevisionType string = "hostrevision"

// DomainType is the string used to represent the Host object.
const DomainType string = "domain"

// DomainRevisionType is the string used to represent the Host Revision
// object.
const DomainRevisionType string = "domainrevision"

// APIUserType is a string used to represent the API User object.
const APIUserType string = "apiuser"

// APIUserRevisionType is a string used to represent the API User
// revision object.
const APIUserRevisionType string = "apiuserrevision"

// ApproverFieldEmailAddres is a name that can be used to reference the
// email address field of the current approver revision.
const ApproverFieldEmailAddres string = "EmailAddress"

// ApproverFieldRole is a name that can be used to reference the
// role field of the current approver revision.
const ApproverFieldRole string = "Role"

// ApproverFieldName is a name that can be used to reference the
// name field of the current approver revision.
const ApproverFieldName string = "Name"

// ApproverFieldUsername is a name that can be used to reference the
// username field of the current approver revision.
const ApproverFieldUsername string = "Username"

// ApproverFieldEmployeeID is a name that can be used to reference the
// EmployeeID field of the current approver revision.
const ApproverFieldEmployeeID string = "EmployeeID"

// ApproverFieldDepartment is a name that can be used to reference the
// Department field of the current approver revision.
const ApproverFieldDepartment string = "Department"

// ApproverFieldFingerprint is a name that can be used to reference the
// Fingerprint field of the current approver revision.
const ApproverFieldFingerprint string = "Fingerprint"

// ApproverFieldPublicKey is a name that can be used to reference the
// PublicKey field of the current approver revision.
const ApproverFieldPublicKey string = "PublicKey"

// ApproverSetFieldTitle is the name that can be used to reference the
// Title field of the current approver set revision.
const ApproverSetFieldTitle string = "Title"

// ApproverSetFieldDescription is the name that can be used to reference
// the Description field of the current approver set revision.
const ApproverSetFieldDescription string = "Description"

// DesiredStateActive is the name that can be used to reference the
// desired state of active when checking for a suggested value.
const DesiredStateActive string = "DesiredStateActive"

// DesiredStateInactive is the name that can be used to reference the
// desired state of inactive when checking for a suggested value.
const DesiredStateInactive string = "DesiredStateInactive"

// DesiredStateExternal is the name that can be used to reference the
// desired state of external when checking for a suggested value.
const DesiredStateExternal string = "DesiredStateExternal"

// ValidSuffixList contains a list of all valid zones for which domains
// can be registered.
var ValidSuffixList = []string{".COM", ".NET"}

// NewRegistrarObject will return the object that is of the type of
// the object type passed.
func NewRegistrarObject(objectType string) (obj RegistrarObject, err error) {
	switch objectType {
	case ApproverType:
		obj = &Approver{}
	case ApproverRevisionType:
		obj = &ApproverRevision{}
	case ApproverSetType:
		obj = &ApproverSet{}
	case ApproverSetRevisionType:
		obj = &ApproverSetRevision{}
	case ChangeRequestType:
		obj = &ChangeRequest{}
	case ApprovalType:
		obj = &Approval{}
	case ContactType:
		obj = &Contact{}
	case ContactRevisionType:
		obj = &ContactRevision{}
	case HostType:
		obj = &Host{}
	case HostRevisionType:
		obj = &HostRevision{}
	case DomainType:
		obj = &Domain{}
	case DomainRevisionType:
		obj = &DomainRevision{}
	case APIUserType:
		obj = &APIUser{}
	case APIUserRevisionType:
		obj = &APIUserRevision{}
	default:
		err = fmt.Errorf("%s %q", UnknownObjectTypeError, objectType)
	}

	return obj, err
}
