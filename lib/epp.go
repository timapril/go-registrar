package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/timapril/go-registrar/epp"
)

// EPPAction is used to store log infomration about an EPP action that has been
// taken.
type EPPAction struct {
	Model

	RegistrarServerTimestamp time.Time `json:"RegistrarServerTimestamp"`
	EPPClientTimestamp       time.Time `json:"EPPClientTimestamp"`

	RunID               int64  `json:"RunID"`
	ClientTransactionID string `json:"ClientTransactionID"`
	ServerTransactionID string `json:"ServerTransactionID"`
	ResponseCode        int    `json:"ResponseCode"`
	ResponseMessage     string `json:"ResponseMessage"`

	Action string `json:"Action"`
	Args   string `json:"Args"`
	Notes  string `json:"Notes"`

	Successful bool `json:"Successful"`
}

const (
	// EPPLogActionLogin represents the action where an EPP Login has happened.
	EPPLogActionLogin = "Login"

	// EPPLogActionLogout represents the action where an EPP Logout has happened.
	EPPLogActionLogout = "Logout"

	// EPPLogActionHello represents the action where an EPP Hello has happened.
	EPPLogActionHello = "Hello"

	// EPPLogActionPoll represents the action where an EPP Poll has happened.
	EPPLogActionPoll = "Poll"

	// EPPLogActionPollAck represents the action where an EPP Poll has been acked.
	EPPLogActionPollAck = "PollAck"

	// EPPLogActionDomainAvailable represents the action where an EPP Available
	// request has been made for a domain. The Argument provided is the domain
	// name that is queried.
	EPPLogActionDomainAvailable = "DomainAvailable"

	// EPPLogActionDomainInfo represents the action where an EPP Info
	// request has been made for a domain. The Argument provided is the domain
	// name that is queried.
	EPPLogActionDomainInfo = "DomainInfo"

	// EPPLogActionDomainCreate represents the action where an EPP Create
	// request has been made for a domain. The Argument provided is the domain
	// name that is queried.
	EPPLogActionDomainCreate = "DomainCreate"

	// EPPLogActionDomainDelete represents the action where an EPP Delete
	// request has been made for a domain. The Argument provided is the domain
	// name that is being created.
	EPPLogActionDomainDelete = "DomainDelete"

	// EPPLogActionDomainRenew represents the action where an EPP Renew
	// request has been made for a domain. The Argument provided is the domain
	// name that is being deleted.
	EPPLogActionDomainRenew = "DomainRenew"

	// EPPLogActionDomainAddHosts represents the action where an EPP Update
	// request has been made for a domain where the hosts are altered. The
	// Argument provided is the domain name that is queried. The notes will
	// have a list of the hosts that have been added.
	EPPLogActionDomainAddHosts = "DomainAddHosts"

	// EPPLogActionDomainRemoveHosts represents the action where an EPP Update
	// request has been made for a domain where the hosts are altered. The
	// Argument provided is the domain name that is queried. The notes will
	// have a list of the hosts that have been removed.
	EPPLogActionDomainRemoveHosts = "DomainRemoveHosts"

	// EPPLogActionDomainAddStatuses represents the action where an EPP Update
	// request has been made for a domain where the statuses are altered. The
	// Argument provided is the domain name that is queried. The notes will
	// have a list of the statuses that have been added.
	EPPLogActionDomainAddStatuses = "DomainAddStatuses"

	// EPPLogActionDomainRemoveStatuses represents the action where an EPP Update
	// request has been made for a domain where the statuses are altered. The
	// Argument provided is the domain name that is queried. The notes will
	// have a list of the statuses that have been removed.
	EPPLogActionDomainRemoveStatuses = "DomainRemoveStatuses"

	// EPPLogActionDomainAddDSRecord represents the action where an EPP Update
	// request has been made for a domain where the DS records are altered. The
	// Argument provided is the domain name that is queried. The notes will
	// have a list of the DS records that have been added.
	EPPLogActionDomainAddDSRecord = "DomainAddDSRecord"

	// EPPLogActionDomainRemoveDSRecord represents the action where an EPP Update
	// request has been made for a domain where the DS records are altered. The
	// Argument provided is the domain name that is queried. The notes will
	// have a list of the DS records that have been removed.
	EPPLogActionDomainRemoveDSRecord = "DomainRemoveDSRecord"

	// EPPLogActionDomainTransferRequest represents the action where an EPP
	// Transfer Request request has been made for a domain. The Argument provided
	// is the domain name that is requested.
	EPPLogActionDomainTransferRequest = "DomainTransferRequest"

	// EPPLogActionDomainTransferReject represents the action where an EPP
	// Transfer Reject request has been made for a domain. The Argument provided
	// is the domain name that is rejected.
	EPPLogActionDomainTransferReject = "DomainTransferReject"

	// EPPLogActionDomainTransferApprove represents the action where an EPP
	// Transfer Approve request has been made for a domain. The Argument provided
	// is the domain name that is approved.
	EPPLogActionDomainTransferApprove = "DomainTransferApprove"

	// EPPLogActionDomainTransferQuery represents the action where an EPP
	// Transfer Query request has been made for a domain. The Argument provided
	// is the domain name that is queried.
	EPPLogActionDomainTransferQuery = "DomainTransferQuery"

	// EPPLogActionDomainChangeAuthInfo represents the action where an EPP Update
	// request has been made for a domain to change its AuthInfo field. The
	// Argument provided is the domain name that is queried.
	EPPLogActionDomainChangeAuthInfo = "DomainChangeAuthInfo"

	// EPPLogActionDomainSync represents the action where an EPP Sync request has
	// been made for a domain to change its expiration date. The Argument provided
	// is the domain name that is queried. Notes are added for the day and month
	// of the new expiration date.
	EPPLogActionDomainSync = "DomainSync"

	// EPPLogActionHostAvailable represents the action where an EPP Available
	// request has been made for a host. The Argument provided is the host
	// name that is queried.
	EPPLogActionHostAvailable = "HostAvailable"

	// EPPLogActionHostInfo represents the action where an EPP Info
	// request has been made for a host. The Argument provided is the host
	// name that is queried.
	EPPLogActionHostInfo = "HostInfo"

	// EPPLogActionHostCreate represents the action where an EPP Create
	// request has been made for a host. The Argument provided is the host
	// name that is created.
	EPPLogActionHostCreate = "HostCreate"

	// EPPLogActionHostDelete represents the action where an EPP Delete
	// request has been made for a host. The Argument provided is the host
	// name that is deleted.
	EPPLogActionHostDelete = "HostDelete"

	// EPPLogActionHostUpdate represents the action where an EPP Update
	// request has been made for a host. The Argument provided is the host
	// name that is updated. The host IP addresses and statuses are included in
	// the action notes field.
	EPPLogActionHostUpdate = "HostUpdate"
)

// NewEPPAction will gerenate and return a new EPPAction object.
func NewEPPAction(ClientTXID string) EPPAction {
	return EPPAction{
		ClientTransactionID: ClientTXID,
		Successful:          true,
		EPPClientTimestamp:  time.Now(),
	}
}

// AddNote will append a note to the list of notes for the action.
func (a *EPPAction) AddNote(note string) {
	if len(a.Notes) != 0 {
		a.Notes = a.Notes + "\n"
	}

	a.Notes += note
}

// HandleResponse will process an EPP response object and set the action's
// server transaction id and the error based on the response.
func (a *EPPAction) HandleResponse(resp epp.Epp) (err error) {
	a.ServerTransactionID, err = resp.GetServerTransactionID()
	a.ResponseCode = resp.ResponseObject.Result.Code
	a.ResponseMessage = resp.ResponseObject.Result.Msg
	a.SetError(err)

	if resp.ResponseObject != nil {
		if resp.ResponseObject.IsError() {
			a.SetError(resp.ResponseObject.GetError())
		}
	}

	return fmt.Errorf("unable to get server transaction id: %w", err)
}

// SetError will set the error of the action to the error provided and if the
// error is not nil, the successful bit will be set to false.
func (a *EPPAction) SetError(err error) {
	if err != nil {
		a.Successful = false
		a.AddNote(fmt.Sprintf("Error: %s", err.Error()))
	}
}

// SetAction is used to set the action that is taking place and any of the
// arguments passed.
func (a *EPPAction) SetAction(Action, Args string) {
	a.Action = Action
	a.Args = Args
}

// AppendEPPActionLog will attempt to add the EPP log message to the list of log
// messages. In the event that the appending fails, an error will be returned.
func AppendEPPActionLog(dbCache *DBCache, request *http.Request) error {
	body, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		return fmt.Errorf("error reading request: %w", readErr)
	}
	defer request.Body.Close()

	var action EPPAction
	if unMarshallErr := json.Unmarshal(body, &action); unMarshallErr != nil {
		return fmt.Errorf("error unmarshaling epp action log: %w", unMarshallErr)
	}

	action.RegistrarServerTimestamp = time.Now()

	err := dbCache.Save(&action)
	if err != nil {
		return err
	}

	return errors.New("not implemented")
}

// EPPRun is a record that is used to identify unique EPP sessions to ensure
// that the client transaction IDs for each epp command are unique.
type EPPRun struct {
	Model

	StartTime time.Time
	EndTime   time.Time

	StartClientID string
	EndClientID   string
}

// CreateEPPRunRecord will attempt to add a new EPP run to the database and
// then return the Unique ID for that run to the client. If an error occurs
// when creating the run it will be returned.
func CreateEPPRunRecord(dbCache *DBCache, clientID string) (id int64, err error) {
	run := EPPRun{}
	run.StartTime = time.Now()
	run.StartClientID = clientID
	err = dbCache.Save(&run)

	return run.GetID(), err
}

// EndEPPRunRecord will attempt to mark an EPPRun record as completed in the
// database and return an error if the completion time could not be logged.
func EndEPPRunRecord(dbCache *DBCache, runid int64, clientID string) (err error) {
	return dbCache.DB.Table("e_p_p_runs").Where("id IN (?)", []int64{runid}).Updates(map[string]interface{}{"end_client_id": clientID, "end_time": time.Now()}).Error
}

// EPPEncryptedPassphrase is used to store the encrypted passphrases for the EPP
// accounts associated with the registrar.
type EPPEncryptedPassphrase struct {
	Model

	Username            string
	EncryptedPassphrase string `gorm:"size:16384"`
	LastUpdate          time.Time
}

// SetEPPEncryptedPassphrase is used to set the encrypted passphrase for the
// provided user to the encrypted passphrase included. If an error occurs
// setting the passphrase for the user, it will be returned.
func SetEPPEncryptedPassphrase(dbCache *DBCache, username string, encPassphrase string) (err error) {
	var count int64

	dbCache.DB.Model(&EPPEncryptedPassphrase{}).Where("username = ?", username).Count(&count)

	if count != 0 {
		// insert
		return dbCache.DB.Table("e_p_p_encrypted_passphrases").Where("username = ?", username).Updates(map[string]interface{}{"encrypted_passphrase": encPassphrase, "last_update": time.Now()}).Error
	}

	// update
	pass := EPPEncryptedPassphrase{}
	pass.LastUpdate = time.Now()
	pass.Username = username
	pass.EncryptedPassphrase = encPassphrase
	err = dbCache.DB.Save(&pass).Error

	if err != nil {
		logger.Error(err)
	}

	return
}

// GetEPPEncryptedPassphrase is used to get the encrypted passphrase for the
// username provided. If an error occurs finding the username, the error will
// be returned.
func GetEPPEncryptedPassphrase(dbCache *DBCache, username string) (encPassphrase string, err error) {
	pass := EPPEncryptedPassphrase{}
	err = dbCache.DB.Where("username = ?", username).First(&pass).Error

	if err != nil {
		return "", err
	}

	return pass.EncryptedPassphrase, nil
}

// MigrateEPPActionLog will run the automigrate function for the separate epp
// actions that have been added.
func MigrateEPPActionLog(dbCache *DBCache) {
	dbCache.AutoMigrate(&EPPRun{})
	dbCache.AutoMigrate(&EPPAction{})
	dbCache.AutoMigrate(&EPPEncryptedPassphrase{})
}
