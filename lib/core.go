// Package lib provides the objects required to operate go-registrar
package lib

import (
	"bytes"
	//"crypto/md5".
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

var (
	// ErrNoCurrentRevision is returned if there is no current revision.
	ErrNoCurrentRevision = errors.New("unable to find a current revision")

	// ErrPendingRevisionInInvalidState indicates the object is in an unexpected state.
	ErrPendingRevisionInInvalidState = errors.New("pending revision is in an invalid state")

	// ErrUnableToHandleState indicates that the service is unable to handle
	// the current state.
	ErrUnableToHandleState = errors.New("unable to handle state")

	// ErrUpdateStateNotImplemented indicates that the current state of an object
	// is not supported by its updateState method.
	ErrUpdateStateNotImplemented = errors.New("updateState not impemented for object")

	// ErrNoDefaultApprover indicates that no default approver was located.
	ErrNoDefaultApprover = errors.New("unable to find default approver - database probably not bootstrapped")

	// ErrNoAPIUsersForKey indicates that no APIUser was found for the provided
	// key.
	ErrNoAPIUsersForKey = errors.New("no APIUsers found with the key provided")

	// ErrExcessAPIKeysLocated indicates that more than 1 apiuser was found with the same
	// API key.
	ErrExcessAPIKeysLocated = errors.New("more than one user found for an API key")
)

// TimeNow is the default now time function for all DB values, rounded to the second.
func TimeNow() time.Time {
	return time.Now().Round(time.Second)
}

func init() {
	// set gorm's NowFunc to our second rounded value
	gorm.NowFunc = TimeNow

	createFns = scopeFuncs{
		gorm.BeginTransaction,
		gorm.BeforeCreate,
		SaveBeforeAssociations,
		gorm.UpdateTimeStampWhenCreate,
		Create,
		gorm.ForceReloadAfterCreate,
		SaveAfterAssociations,
		gorm.AfterCreate,
		SetScopeHash,
	}
	updateFns = scopeFuncs{
		gorm.BeginTransaction,
		gorm.AssignUpdateAttributes,
		gorm.BeforeUpdate,
		SaveBeforeAssociations,
		gorm.UpdateTimeStampWhenUpdate,
		Update,
		SaveAfterAssociations,
		gorm.AfterUpdate,
		SetScopeHash,
	}
}

// Model is a base type for our gorm objects, it is the basis for many of the utility functions.
type Model struct {
	ID int64 `gorm:"primary_key:yes" json:"ID"`

	fieldHash  []byte `sql:"-"`
	inProgress bool   `sql:"-"`
	prepared   bool   `sql:"-"`
}

// FileCaller returns the file and line number of the first caller that is not in the same file as it's immediate caller
//func FileCaller() string {
//	_, baseFile, line, ok := runtime.Caller(1)
//	file := baseFile
//	for i := 2; ok && file == baseFile; i++ {
//		_, file, line, ok = runtime.Caller(i)
//	}
//	if ok {
//		return fmt.Sprintf(" at %s line %d", file, line)
//	}
//	return ""
//}

// GetID returns the ID of the object.
func (m *Model) GetID() int64 {
	return m.ID
}

// SetID sets the ID of the object if it is not already set. If the ID
// of the object has been set already an error will be returned.
func (m *Model) SetID(objectID int64) error {
	if m.ID != 0 {
		return errors.New("ID has already been set on this object")
	}

	if objectID <= 0 {
		return errors.New("IDs must be greater than 0")
	}

	m.ID = objectID

	return nil
}

// GetModel is an accessor method to get Model from any struct that embeds it.
func (m *Model) GetModel() *Model {
	return m
}

// Modeler interface for any struct implements GetModel, which is to say any struct that embeds Model.
type Modeler interface {
	GetModel() *Model
}

// Load will load rec from db iff rec is not already loaded.
func (m *Model) Load(dbCache *DBCache, rec Modeler) (err error) {
	if !m.IsLoaded() {
		if m.ID <= 0 {
			return fmt.Errorf("the ID of %T must be greater than 0 but is %d", rec, m.ID)
		}

		err = dbCache.DB.First(rec).Error
	}

	return
}

// Load will load rec from db iff rec is not already loaded.
func Load(dbCache *DBCache, rec Modeler) (err error) {
	m := rec.GetModel()

	return m.Load(dbCache, rec)
}

// PrepareBase implements the core logic of all of the Prepare* methods, returning immediately if the m.prepared is set,
// loading the base class only if is not already loaded, runs recursiveFunc, and if everything runs without errors,
// setting prepared to true.
func PrepareBase(dbCache *DBCache, rec Modeler, recursiveFunc func() error) (err error) {
	model := rec.GetModel()

	if model.prepared {
		return nil
	}

	// fmt.Printf("> PrepareBase %T %d\n", rec, m.ID)
	if err = model.Load(dbCache, rec); err != nil {
		return
	}

	if err = recursiveFunc(); err != nil {
		//		fmt.Printf("  PrepareBase %T %d recursive failed: %s\n", rec, m.ID, err.Error())
		return
	}

	//	fmt.Printf("< PrepareBase %T %+v\n", rec, m.ID)
	model.prepared = true

	return nil
}

// FuncNOPErrFunc simple noop callback for use with PrepareBase for PreprareShallow methods.
func FuncNOPErrFunc() error { return nil }

// A RevisionModeler is a abstration of a revision model to a general purpose GetRequiredApproverSets and
// GetInformedApproverSets to be written for all base types (e.g. APIUser, Approver, etc.)
type RevisionModeler interface {
	Modeler
	IsActive() bool
	GetRequiredApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error)
	GetInformedApproverSets(dbCache *DBCache) (approverSets []ApproverSet, err error)
}

// getCurrentRevision uses reflection with db to get a the current revision of rec as a RevisionModeler.
func getCurrentRevision(dbCache *DBCache, rec Modeler) (currRev RevisionModeler, err error) {
	var (
		fieldFound bool
		revField   *gorm.Field
	)

	if err = Load(dbCache, rec); err != nil {
		return currRev, err
	}

	scope := dbCache.DB.NewScope(rec)
	revField, fieldFound = scope.FieldByName("CurrentRevision")

	if !fieldFound {
		err = fmt.Errorf("couldn't get CurrentRevision for %T", rec)

		return currRev, err
	}

	if !revField.Field.CanInterface() {
		err = fmt.Errorf("currentRevision for %T can't interface", rec)

		return currRev, err
	}

	if currRev, fieldFound = reflect.Indirect(revField.Field).Addr().Interface().(RevisionModeler); !fieldFound {
		err = fmt.Errorf("currentRevision %T of %T is not a RevisionModeler", reflect.Indirect(revField.Field).Addr().Interface(), rec)

		return currRev, err
	}

	if currRev.GetModel().GetID() == 0 {
		err = dbCache.DB.Model(rec).Association("CurrentRevision").Find(currRev).Error
	} else {
		err = Load(dbCache, currRev)
	}

	return currRev, err
}

func getActiveCurrentRevision(dbCache *DBCache, rec Modeler) (currRev RevisionModeler, err error) {
	currRev, err = getCurrentRevision(dbCache, rec)

	if err == nil && currRev.IsActive() {
		return
	}

	err = ErrNoCurrentRevision

	return
}

// GetRequiredApproverSets is a general implementation of lookup for ApproverSets. It doesn't require that the object be
// loaded, only that it exist in the db. It returns the list of approver sets that are required for rec (if a
// valid approver revision exists). If no rec revisions are found, a default of the infosec approver set will be
// returned.
func GetRequiredApproverSets(dbCache *DBCache, rec Modeler) (approverSets []ApproverSet, err error) {
	var currRev RevisionModeler
	currRev, err = getActiveCurrentRevision(dbCache, rec)

	if err == nil {
		approverSets, err = currRev.GetRequiredApproverSets(dbCache)
	} else if errors.Is(err, ErrNoCurrentRevision) {
		return
	}

	if len(approverSets) == 0 {
		appSet, prepErr := GetDefaultApproverSet(dbCache)

		if prepErr == nil {
			approverSets = append(approverSets, appSet)
		}

		err = errors.New("unable to find a current revision")
	}

	return
}

// GetInformedApproverSets is a general implementation of lookup for ApproverSets. It doesn't require that the object
// be loaded, only that it exist in the db. It returns the list of approver sets that are
// informed for rec (if a valid approver revision exists).
func GetInformedApproverSets(dbCache *DBCache, rec Modeler) (approverSets []ApproverSet, err error) {
	var currRev RevisionModeler
	currRev, err = getActiveCurrentRevision(dbCache, rec)

	if err == nil {
		approverSets, err = currRev.GetInformedApproverSets(dbCache)
	}

	return
}

// HandleInProgress implements a simple loop prevention mechanism. If inProgress is not set it will set inProgress
// to true (setting it back to false with a defer) and run progressFunc. If inProgress is true it will return
// immediately.
func (m *Model) HandleInProgress(progressFunc func()) {
	if m.inProgress {
		//		if at := FileCaller(); at != "" {
		//			fmt.Printf("HandleInProgress blocked %s\n", at)
		//		}
		return
	}

	m.inProgress = true

	defer func() { m.inProgress = false }()

	progressFunc()
}

// ProgressHandler is a simple interface for the HandleInProgress method.
type ProgressHandler interface {
	HandleInProgress(func())
}

// HandleInProgress handles the case of an arbitrary interface. (Mostly DB objects at this point)
// If the interface implements ProgressHandler, it will use HandleInProgress to prevent loops. Otherwise
// it will always run progressFunc.
func HandleInProgress(rec interface{}, progressFunc func()) {
	if ph, ok := rec.(ProgressHandler); ok {
		ph.HandleInProgress(progressFunc)
	} else {
		progressFunc()
	}
}

// FieldHasher is the public interface field hashing routines in Model.
type FieldHasher interface {
	FieldHash() []byte
	IsLoaded() bool
	IsScopeUnchanged(*gorm.Scope) bool
	// IsUnchanged(*gorm.DB, interface{}) bool
	SetFieldHash(*gorm.DB, interface{})
	SetScopeFieldHash(*gorm.Scope)
	AfterFind(*gorm.Scope)
}

// FieldHash return the current hash value.
func (m *Model) FieldHash() []byte {
	return m.fieldHash
}

// calcScopeFieldHash generates a field hash for the Value interface of the provided scope
// additionally it returns a zero length, non-nil result on error (to distinguish it from a
// nil values which indicates no calculation has been done.)
//
// see IsScopeUnchanged/IsUnchanged for below for the context in which this is used.
func calcScopeFieldHash(scope *gorm.Scope) []byte {
	// fmt.Printf("scope.Value: %T\n", scope.Value)
	fields := make(map[string]interface{})

	for _, field := range scope.Fields() {
		// fmt.Printf("   field: %s %.30s\n", field.Name, fmt.Sprintf("%+v", field.Field))
		if field.IsNormal {
			if !field.Field.IsValid() {
				// usually means we're getting on an array ref; bail out
				return make([]byte, 0)
			}
			// Assign value of field to struct field name (not column name).
			fields[field.Name] = field.Field.Interface()
		}
	}

	// fmt.Printf("finished scope.Value: %T\n\n", scope.Value)
	hash := sha256.New()

	if err := json.NewEncoder(hash).Encode(fields); err != nil {
		return make([]byte, 0)
	}

	return hash.Sum(nil)
}

// IsLoaded checks to see if the fieldHash has been calculated (it is set to a non-nil 0 len value on error),
// since the AfterFind method should set fieldHash, we can tell if a given struct has not been loaded.
func (m *Model) IsLoaded() bool {
	return m.fieldHash != nil
}

// // calcFieldHash generates a field hash for dbStruct using the given a db handle
// func calcFieldHash(db *gorm.DB, dbStruct interface{}) []byte {
// 	scope := db.NewScope(dbStruct)
// 	return calcScopeFieldHash(scope)
// }

// SetScopeFieldHash uses scope to calculate and set fieldHash
// see IsScopeUnchanged/IsUnchanged for below for the context in which this is used.
func (m *Model) SetScopeFieldHash(scope *gorm.Scope) {
	m.fieldHash = calcScopeFieldHash(scope)
}

// SetFieldHash generates a scope to calculate the hash for dbStruct
// see IsScopeUnchanged/IsUnchanged for below for the context in which this is used.
func (m *Model) SetFieldHash(db *gorm.DB, dbStruct interface{}) {
	m.SetScopeFieldHash(db.NewScope(dbStruct))
}

// AfterFind is the callback for model. It will be called for each object that embeds Model.
// We *almost* cleverly get around the problem with of only have Model instead of object that embeds it, by
// using scope.Value which points to what was getting loaded, thus allowing us to do our field hashing.
//
// There's one pretty significant problem with this: loading slices rather than single values. When a slice is
// loaded, scope.Value will be pointer to the slice, and AfterFind will be called for each element. Attempting to
// use the field values from scope in this case results in garbage.
//
// So when we see that scope.Value is a slice, we manually create a scope and invoke AfterFind for each element.
func (m *Model) AfterFind(scope *gorm.Scope) {
	// fmt.Printf("Calling afterFind for %T with hash %v\n", scope.Value, m.fieldHash)
	if m.fieldHash == nil {
		a := reflect.Indirect(reflect.ValueOf(scope.Value))
		if a.IsValid() && a.Kind() == reflect.Slice {
			// fmt.Printf("Calling afterFind array for %v with len %v\n", a.Type(), a.Len())
			for i := 0; i < a.Len(); i++ {
				value := a.Index(i).Addr().Interface()
				// fmt.Printf("value type %T\n", value)
				if fh, ok := value.(FieldHasher); ok {
					fh.AfterFind(scope.New(value))
				}
			}

			return
		}

		m.fieldHash = calcScopeFieldHash(scope)
		// fmt.Printf("Calling afterFind calculate for %T with hash %v\n", scope.Value, m.fieldHash)
		m.prepared = false
	}
}

// IsScopeUnchanged returns true iff m.fieldHash has been successfully calculated, and a hash for the provided scope
// can be succfully calculated, and these values as the same.
// This should be the case for any loaded struct embedding Model and running the AfterFind method.
func (m *Model) IsScopeUnchanged(scope *gorm.Scope) bool {
	if len(m.fieldHash) == 0 {
		return false
	}

	calced := calcScopeFieldHash(scope)

	if len(calced) == 0 {
		return false
	}

	return bytes.Equal(m.fieldHash, calced)
}

// IsUnchanged calculates a scope from db and dbStruct calls IsScopeUnchanged
//func (m *Model) IsUnchanged(db *gorm.DB, dbStruct interface{}) bool {
//	return m.IsScopeUnchanged(db.NewScope(dbStruct))
//}

// IsScopeUnchanged returns false if it is passed a scope that doesn't have a FieldHash compatible value, otherwise
// returns the value of IsScopeUnchanged.
func IsScopeUnchanged(scope *gorm.Scope) bool {
	if fh, ok := scope.Value.(FieldHasher); ok {
		return fh.IsScopeUnchanged(scope)
	}

	return false
}

// SetScopeHash will calculate a fieldHash for a scope containing a FieldHasher value.
func SetScopeHash(scope *gorm.Scope) {
	if fh, ok := scope.Value.(FieldHasher); ok {
		fh.SetScopeFieldHash(scope)
	}
}

// NullInt64ConfirmEqual returns match indicating if a and b are equal. If they are not, returns a string describing the
// difference.
func NullInt64ConfirmEqual(valA, valB sql.NullInt64) (match bool, diff string) {
	if match = valA.Valid == valB.Valid; match {
		if !valA.Valid { // Both none/null
			return
		}

		if match = valA.Int64 == valB.Int64; match {
			return
		}

		diff = fmt.Sprintf("Expected: %d Got: %d", valA.Int64, valB.Int64)
	} else {
		if valA.Valid { // Both none/null
			diff = fmt.Sprintf("Expected: %d Got: none", valA.Int64)
		} else {
			diff = fmt.Sprintf("Expected: none Got: %d", valB.Int64)
		}
	}

	return
}

// CompareLoadFn is a callback function type that loads values into the provided RegistrarObjectExport.
type CompareLoadFn func(RegistrarObjectExport)

// CompareReturnFn is callback function returned by ComparePendingToCallback. It represents a deferred comparison.
type CompareReturnFn func() (pass bool, errs []error)

// RegistrarCRObject is an implementation extension RegistrarApprovalable that includes
// methods for loading objects and other functions.
type RegistrarCRObject interface {
	RegistrarApprovalable
	SetID(ID int64) error
	GetID() int64
	GetType() string
	Prepare(*DBCache) error
	GetCurrentRevisionID() sql.NullInt64
	GetPendingRevisionID() int64
	GetPendingCRID() sql.NullInt64
	ComparePendingToCallback(loadFn CompareLoadFn) CompareReturnFn
}

// VerifyCR is a generic implemention of checks to make sure that all of the values and approvals within a change
// request match the approver that it is linked to, and that the CR is genreally well formed.
func VerifyCR(dbCache *DBCache, rec RegistrarCRObject, _ *ChangeRequest) (checksOut bool, errs []error) {
	checksOut = false
	objType := rec.GetType()
	objID := rec.GetID()
	pendingID := rec.GetPendingRevisionID()
	crID := rec.GetPendingCRID()

	if pendingID != 0 {
		// Check to see if the change request has been approved
		if crID.Valid {
			changeRequest := &ChangeRequest{}

			if err := dbCache.FindByID(changeRequest, crID.Int64); err != nil {
				errs = append(errs, err)

				return checksOut, errs
			}

			err := changeRequest.Prepare(dbCache)
			if err != nil {
				errs = append(errs, err)
			}

			checksOut = true

			// Check to make sure that the CR points to the correct object
			// type
			if changeRequest.RegistrarObjectType != objType {
				errs = append(errs, fmt.Errorf("CR %d does not have the correct Object Type. Expected: %s Got: %s", changeRequest.ID, objType, changeRequest.RegistrarObjectType))
				checksOut = false
			}

			// Check to make sure that the CR points to the correct object
			if changeRequest.RegistrarObjectID != objID {
				errs = append(errs, fmt.Errorf("CR %d does not have the correct Object ID. Expected: %d Got: %d", changeRequest.ID, objID, changeRequest.RegistrarObjectID))
				checksOut = false
			}

			// Check to make sure that the current revision matches in the CR
			if match, diff := NullInt64ConfirmEqual(rec.GetCurrentRevisionID(), changeRequest.InitialRevisionID); !match {
				errs = append(errs, fmt.Errorf("CR %d does not have the correct Initial Revision ID. %s", changeRequest.ID, diff))
				checksOut = false
			}

			// Check to make sure that the new revison ID matches the CR
			if changeRequest.ProposedRevisionID != pendingID {
				errs = append(errs, fmt.Errorf("CR %d does not have the correct Pending Revison ID, Expected: %d Got: %d", changeRequest.ID, pendingID, changeRequest.ProposedRevisionID))
				checksOut = false
			}

			if checksOut {
				for _, approval := range changeRequest.Approvals {
					err := approval.Prepare(dbCache)
					if err != nil {
						errs = append(errs, err)
					}

					att, validSig, attErr := approval.GetApprovalAttestation(dbCache)
					if validSig && attErr == nil {
						if att.ObjectType != objType {
							errs = append(errs, fmt.Errorf("incorrect Attestation Type found, expected %s, got %s", objType, att.ObjectType))
							checksOut = false
						}

						if att.ApprovalID != approval.ID {
							errs = append(errs, fmt.Errorf("incorrect Attestation Approval ID, expected %d, got %d", approval.ID, att.ApprovalID))
							checksOut = false
						}

						if att.ObjectType != objType {
							errs = append(errs, fmt.Errorf("incorrect Attestation Type found, expected %s, got %s", objType, att.ObjectType))
							checksOut = false
						} else {
							var jsonErr error

							returnFn := rec.ComparePendingToCallback(func(exp RegistrarObjectExport) {
								jsonErr = json.Unmarshal(att.ExportRev, &exp)
							})

							if jsonErr == nil {
								matches, compErrs := returnFn()
								if len(compErrs) != 0 {
									errs = append(errs, compErrs...)
								}

								if !matches {
									checksOut = false
								}
							} else {
								errs = append(errs, jsonErr)
								checksOut = false
							}
						}
					} else {
						if attErr != nil {
							checksOut = false
							errs = append(errs, attErr)
						} else {
							checksOut = false
						}
					}
				}
			}
		} else {
			errs = append(errs, fmt.Errorf("unable to find a CR for pending %s revision %d", objType, pendingID))
		}
	} else {
		errs = append(errs, fmt.Errorf("unable to find a pending revision for %s %d", objType, rec.GetID()))
	}

	return checksOut, errs
}

type scopeFuncs []func(*gorm.Scope)

var createFns, updateFns scopeFuncs

// Create is a simple wrapper around gorm.Create. It skips the create if an object is unchanged.
func Create(scope *gorm.Scope) {
	if IsScopeUnchanged(scope) {
		return
	}

	gorm.Create(scope)
}

// Update is a simple wrapper around gorm.Update. It skips the update if an object is unchanged.
func Update(scope *gorm.Scope) {
	if IsScopeUnchanged(scope) {
		return
	}

	gorm.Update(scope)
}

func getScopeFuncs(scope *gorm.Scope) scopeFuncs {
	if scope.PrimaryKeyZero() {
		return createFns
	}

	return updateFns
}

// Save implements save logic broadly equivalent to gorm.DB.Save, but with smarter loop preventing logic.
func Save(db *gorm.DB, dbStruct interface{}) *gorm.DB {
	scope := db.NewScope(dbStruct)
	HandleInProgress(dbStruct, func() {
		for _, f := range getScopeFuncs(scope) {
			f(scope)
			if scope.HasError() {
				break
			}
		}
		scope.CommitOrRollback()
	})

	return scope.DB()
}

// GetRemoteUser extracts the REMOTE_USER header from a HTTP Request
// and returns the username found. The function assumes that a proxy
// in front of the application verifies the authenticity of the header.
// If the header is not set an error is returned.
func GetRemoteUser(req *http.Request) (string, error) {
	var err error

	remoteUser := getCaseInsesitiveHeader(req, "REMOTE_USER")

	if remoteUser == "" {
		err = errors.New("no user set")
	}

	return remoteUser, err
}

// GetRemoteUserEmail extracts the REMOTE_USER header from a HTTP
// request and returns the email address for the user. If no
// REMOTE_USER_ORG is not set, the org is assumed to be the default user
// domain configured otherwise the value in REMOTE_USER_ORG is appended
// to the REMOTE_USER. REMOTE_USER_ORG and REMOTE_USER are request headers
// that may be set.
func GetRemoteUserEmail(req *http.Request, conf Config) (string, error) {
	runame, err := GetRemoteUser(req)
	if err != nil {
		return "", err
	}

	remoteUserOrg := getCaseInsesitiveHeader(req, "REMOTE_USER_ORG")

	if remoteUserOrg == "" {
		return runame + "@" + conf.Server.DefaultUserDomain, nil
	}

	return runame + remoteUserOrg, nil
}

// getCaseInsesitiveHeader will inspect the request headers and return
// the first header that matches the headerName passed ignoring case.
func getCaseInsesitiveHeader(req *http.Request, headerName string) string {
	expectedHeader := strings.ToLower(headerName)

	for header, val := range req.Header {
		if len(val) >= 1 {
			if strings.ToLower(header) == expectedHeader {
				return val[0]
			}
		}
	}

	return ""
}

// GetAPIUser extracts the certificate from the headers provided by
// apache and will look for a current user that has a matching
// certificate. If no certificate is found, or there is no API users
// associated with the certificate then an error is returned.
func GetAPIUser(req *http.Request, dbCache *DBCache, conf Config) (user *APIUser, err error) {
	certPem := getCaseInsesitiveHeader(req, conf.Server.CertHeader)

	if certPem == "" {
		err = errors.New("no cert set")
	}

	newlines := strings.Replace(certPem, " ", "\n", -1)
	fixedCerts := strings.Replace(newlines, "\nCERTIFICATE-----", " CERTIFICATE-----", -1)

	user, apiGetErr := GetAPIUserFromPEM(dbCache, []byte(fixedCerts))
	if apiGetErr != nil {
		return user, apiGetErr
	}

	return user, err
}

// GetActiveInactive is used to verify that a state submitted in an HTTP
// request is either "active" or "inactive". If the submitted state does
// not match either of the options "active" is returned. "bootstrap" is
// not an allowed state via HTTP.
func GetActiveInactive(cleartextState string) string {
	if cleartextState == StateInactive {
		return StateInactive
	}

	return StateActive
}

// GetActiveInactiveExternal is used to verify that a state submitted in
// a HTTP request is either "active", "inactive" or external. If the
// submitted state does not match either of the options, "active" is
// returned.
func GetActiveInactiveExternal(cleartextState string) string {
	if cleartextState == StateInactive {
		return StateInactive
	}

	if cleartextState == StateExternal {
		return StateExternal
	}

	return StateActive
}

// GetActiveNewExternal is used to verify that a state submitted in
// a HTTP request is either "new" or "newexternal". If the submitted
// state does not match either of the options, "new" is returned.
func GetActiveNewExternal(cleartextState string) string {
	if cleartextState == StateNewExternal {
		return StateNewExternal
	}

	return StateNew
}

// GetCheckboxState is used to turn a checkbox value into a boolean. If
// the form value is "on", "On" or "ON" true is returned, otherwise
// false is returned.
func GetCheckboxState(cleartextForm string) bool {
	if cleartextForm == "on" || cleartextForm == "On" || cleartextForm == "ON" {
		return true
	}

	return false
}

// ClientDeleteFlag is a name that can be used to reference the
// Client Delete field of the current revision.
const ClientDeleteFlag = "ClientDelete"

// ServerDeleteFlag is a name that can be used to reference the
// Server Delete field of the current revision.
const ServerDeleteFlag = "ServerDelete"

// ClientTransferFlag is a name that can be used to reference the
// Client Transfer field of the current revision.
const ClientTransferFlag = "ClientTransfer"

// ServerTransferFlag is a name that can be used to reference the
// Server Transfer field of the current revision.
const ServerTransferFlag = "ServerTransfer"

// ClientUpdateFlag is a name that can be used to reference the
// Client Transfer field of the current revision.
const ClientUpdateFlag = "ClientUpdate"

// ServerUpdateFlag is a name that can be used to reference the
// Server Transfer field of the current revision.
const ServerUpdateFlag = "ServerUpdate"

// ClientRenewFlag is a name that can be used to reference the
// Client Renew field of the current revision.
const ClientRenewFlag = "ClientRenew"

// ServerRenewFlag is a name that can be used to reference the
// Server Renew field of the current revision.
const ServerRenewFlag = "ServerRenew"

// ClientHoldFlag is a name that can be used to reference the
// Client Hold field of the current revision.
const ClientHoldFlag = "ClientHold"

// ServerHoldFlag is a name that can be used to reference the
// Server Hold field of the current revision.
const ServerHoldFlag = "ServerHold"

// LivenessCheck is a table that is used to get a known value when the GTM
// liveness check is issued.
type LivenessCheck struct {
	ID         int64
	UnusedData int64
}

// DBPing will time the round trip to the database for a connection and return
// the time. If an error occurs or the database cannot be contacted, a time
// of -1 second and an error will be returned.
func DBPing(db *gorm.DB) (time.Duration, error) {
	startTime := time.Now()
	rows, err := db.Raw("select id from liveness_checks limit 1").Rows()

	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Error(err)
		}
	}()

	err2 := rows.Err()

	endTime := time.Now()

	if err != nil {
		return time.Second * -1, fmt.Errorf("error querying liveness table: %w", err)
	}

	if err2 != nil {
		return time.Second * -1, fmt.Errorf("error querying liveness table: %w", err2)
	}

	defer closeRowError(rows)

	if rows.Next() {
		return endTime.Sub(startTime), nil
	}

	return time.Second * -1, errors.New("unable to find record in the test table")
}

// closeRowError will log an error if the row closing does not work for some
// reason.
func closeRowError(rows io.Closer) {
	err := rows.Close()
	if err != nil {
		logger.Errorf("Error closing rows: %s", err.Error())
	}
}

// MigrateDBLivenessCheck will run the automigrate function for the Liveness
// Check table and will also ensure that there is at least one row in the
// liveness check table.
func MigrateDBLivenessCheck(dbCache *DBCache) {
	dbCache.AutoMigrate(&LivenessCheck{})

	var count int64

	dbCache.DB.Table("liveness_checks").Count(&count)

	if count == 0 {
		lc := LivenessCheck{UnusedData: 1}
		dbCache.DB.Save(&lc)
	}
}
