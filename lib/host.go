package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aryann/difflib"
	"github.com/jinzhu/gorm"

	"github.com/timapril/go-registrar/epp"
)

// Host is an object that represents the state of a registrar host
// object as defined by RFC 5732
// http://tools.ietf.org/html/rfc5732
type Host struct {
	Model
	State string

	HostStatus                     string
	ClientDeleteProhibitedStatus   bool
	ServerDeleteProhibitedStatus   bool
	ClientTransferProhibitedStatus bool
	ServerTransferProhibitedStatus bool
	ClientUpdateProhibitedStatus   bool
	ServerUpdateProhibitedStatus   bool
	LinkedStatus                   bool
	OKStatus                       bool
	PendingCreateStatus            bool
	PendingDeleteStatus            bool
	PendingTransferStatus          bool
	PendingUpdateStatus            bool

	HostName      string `sql:"size:256"`
	HostROID      string `sql:"size:128"`
	HostAddresses []HostAddressEpp

	PreviewIPs string `sql:"size:2048"`

	SponsoringClientID string `sql:"size:32"`
	CreateClientID     string `sql:"size:32"`
	CreateDate         time.Time
	UpdateClientID     string `sql:"size:32"`
	UpdateDate         time.Time
	TransferDate       time.Time

	HoldActive bool
	HoldBy     string
	HoldAt     time.Time
	HoldReason string `sql:"size:1024"`

	CurrentRevision   HostRevision
	CurrentRevisionID sql.NullInt64
	Revisions         []HostRevision
	PendingRevision   HostRevision `sql:"-"`

	EPPStatus     string
	EPPLastUpdate time.Time
	DNSStatus     string
	DNSLastUpdate time.Time

	CreatedAt     time.Time `json:"CreatedAt"`
	CreatedBy     string    `json:"CreatedBy"`
	UpdatedAt     time.Time `json:"UpdatedAt"`
	UpdatedBy     string    `json:"UpdatedBy"`
	CheckRequired bool
}

// HostExport is an object that is used to export the current
// state of a host object. The full version of the export object
// also contains the current and pending revision (if either exist).
type HostExport struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	HostName string `json:"HostName"`
	HostROID string `json:"HostROID"`

	CurrentRevision HostRevisionExport `json:"CurrentRevision"`
	PendingRevision HostRevisionExport `json:"PendingRevision"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`

	HoldActive bool      `json:"HoldActive"`
	HoldBy     string    `json:"HoldBy"`
	HoldAt     time.Time `json:"HoldAt"`
	HoldReason string    `json:"HoldReason" sql:"size:1024"`
}

// GetDiff will return a string containing a formatted diff of the
// current and pending revisions for the Host object. An empty
// string and an error are returned if an error occures during the
// processing
//
// TODO: Handle diff for objects that do not have a pending revision
// TODO: Handle diff for objects that do not have a current revision.
func (h HostExport) GetDiff() (string, error) {
	current, _ := h.CurrentRevision.ToJSON()
	pending, err2 := h.PendingRevision.ToJSON()

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
func (h HostExport) ToJSON() (string, error) {
	if h.ID <= 0 {
		return "", errors.New("ID not set")
	}

	byteArr, jsonErr := json.MarshalIndent(h, "", "  ")

	return string(byteArr), jsonErr
}

// HostExportShort is an object that is used to export the current state
// of a host object. The short version of the export object does not
// contain the current or pending revision.
type HostExportShort struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	HostName string `json:"HostName"`
	HostROID string `json:"HostROID"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`

	HoldActive bool      `json:"HoldActive"`
	HoldBy     string    `json:"HoldBy"`
	HoldAt     time.Time `json:"HoldAt"`
	HoldReason string    `json:"HoldReason" sql:"size:1024"`
}

// ToJSON will return a string containing the JSON representation of the
// object. An empty string and an error are returned if the JSON
// representation cannot be returned.
func (h HostExportShort) ToJSON() (string, error) {
	byteArr, err := json.MarshalIndent(h, "", "  ")

	return string(byteArr), err
}

// HostPage is used to hold all the information required to render
// the Host HTML page.
type HostPage struct {
	Editable            bool
	IsNew               bool
	Hos                 Host
	CurrentRevisionPage *HostRevisionPage
	PendingRevisionPage *HostRevisionPage
	PendingActions      map[string]string
	ValidApproverSets   map[int64]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (h *HostPage) GetCSRFToken() string {
	return h.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (h *HostPage) SetCSRFToken(newToken string) {
	h.CSRFToken = newToken
	h.PendingRevisionPage.CSRFToken = newToken
}

// HostsPage is used to hold all the information required to render
// the Host HTML page.
type HostsPage struct {
	Hosts []Host

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (c *HostsPage) GetCSRFToken() string {
	return c.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (c *HostsPage) SetCSRFToken(newToken string) {
	c.CSRFToken = newToken
}

// GetExportVersion returns an export version of the Host Object.
func (h *Host) GetExportVersion() RegistrarObjectExport {
	export := HostExport{
		ID:              h.ID,
		State:           h.State,
		PendingRevision: (h.PendingRevision.GetExportVersion()).(HostRevisionExport),
		CurrentRevision: (h.CurrentRevision.GetExportVersion()).(HostRevisionExport),
		HostName:        h.HostName,
		HostROID:        h.HostROID,
		UpdatedAt:       h.UpdatedAt,
		UpdatedBy:       h.UpdatedBy,
		CreatedAt:       h.CreatedAt,
		CreatedBy:       h.CreatedBy,
		HoldActive:      h.HoldActive,
		HoldBy:          h.HoldBy,
		HoldAt:          h.HoldAt,
		HoldReason:      h.HoldReason,
	}

	return export
}

// GetExportVersionAt returns an export version of the Host Object
// at the timestamp provided if possible otherwise an error is returned.
// If a pending version existed at the time it will be excluded from
// the object.
func (h *Host) GetExportVersionAt(dbCache *DBCache, timestamp int64) (obj RegistrarObjectExport, err error) {
	hostRevision := HostRevision{}

	// Grab the most recent promoted object before the time provided
	// where the promoted time is after the time stated above
	// If no objects were found meeting the criteria, return an error
	if err = dbCache.GetRevisionAtTime(&hostRevision, h.ID, timestamp); err != nil {
		return
	}

	// Otherwise, prepare the new object, fix the currnet object a little
	// and remove the pending revision if any
	if err = hostRevision.Prepare(dbCache); err != nil {
		return
	}

	h.CurrentRevision = hostRevision
	h.CurrentRevisionID.Int64 = hostRevision.ID
	h.CurrentRevisionID.Valid = true
	h.PendingRevision = HostRevision{}

	// Return the export version and no error
	return h.GetExportVersion(), nil
}

// GetExportShortVersion returned an export version of the Host object
// in its short form.
func (h *Host) GetExportShortVersion() HostExportShort {
	export := HostExportShort{
		ID:         h.ID,
		State:      h.State,
		HostName:   h.HostName,
		HostROID:   h.HostROID,
		CreatedAt:  h.CreatedAt,
		CreatedBy:  h.CreatedBy,
		HoldActive: h.HoldActive,
		HoldBy:     h.HoldBy,
		HoldAt:     h.HoldAt,
		HoldReason: h.HoldReason,
	}

	return export
}

// UnPreparedHostError is the text of an error that is displayed
// when a host has not been prepared before use.
const UnPreparedHostError = "Error: Host Not Prepared"

// GetDisplayName will return a name for the Host that can be used to
// display a shortened version of the invormation to users.
func (h *Host) GetDisplayName() string {
	return fmt.Sprintf("%d - %s", h.ID, h.HostName)
}

// GetCurrentValue is used to get the current value of a field in a
// revision if a current revision exists, otherwise an empty string is
// returned.
func (h *Host) GetCurrentValue(field string) (ret string) {
	if !h.prepared {
		return UnPreparedHostError
	}

	return h.SuggestedRevisionValue(field)
}

// GetCurrentHosts returnes an array of the display names for the Host
// Addresses of the current HostRevision.
func (h *Host) GetCurrentHosts() []string {
	var ret []string

	if h.CurrentRevisionID.Valid {
		for _, host := range h.CurrentRevision.HostAddresses {
			ret = append(ret, host.DisplayName())
		}
	}

	return ret
}

// HostFieldName is a name that can be used to reference the
// name field of the current host revision.
const HostFieldName string = "Name"

// SuggestedRevisionValue takes a string naming the field that is being
// requested and returns a string containing the suggested value for
// the field in a new pending revision
//
// TODO: add other fields that have been added.
func (h Host) SuggestedRevisionValue(field string) string {
	if h.CurrentRevisionID.Valid {
		switch field {
		case DomainFieldName:
			return h.HostName
		case SavedObjectNote:
			return h.CurrentRevision.SavedNotes
		}
	}

	return ""
}

// SuggestedRevisionBool takes a string naming the flag that is being
// requested and returnes a bool containing the suggested value for the
// field in the new revision
//
// TODO: add other fields that have been added.
func (h Host) SuggestedRevisionBool(field string) bool {
	if h.CurrentRevisionID.Valid {
		switch field {
		case ClientDeleteFlag:
			return h.CurrentRevision.ClientDeleteProhibitedStatus
		case ServerDeleteFlag:
			return h.CurrentRevision.ServerDeleteProhibitedStatus
		case ClientTransferFlag:
			return h.CurrentRevision.ClientTransferProhibitedStatus
		case ServerTransferFlag:
			return h.CurrentRevision.ServerTransferProhibitedStatus
		case ClientUpdateFlag:
			return h.CurrentRevision.ClientUpdateProhibitedStatus
		case ServerUpdateFlag:
			return h.CurrentRevision.ServerUpdateProhibitedStatus
		case DesiredStateActive:
			return h.CurrentRevision.DesiredState == StateActive
		case DesiredStateInactive:
			return h.CurrentRevision.DesiredState == StateInactive
		}
	}

	return false
}

// HasRevision returns true iff a current revision exists, otherwise
// false
//
// TODO: add a check to verify that the current revision has an approved
// change request.
func (h Host) HasRevision() bool {
	return h.CurrentRevisionID.Valid
}

// HasPendingRevision returns true iff a pending revision exists for the
// Host, otherwise false.
func (h Host) HasPendingRevision() bool {
	return h.PendingRevision.ID != 0
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the query into dbcache.
func (h *Host) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, h, func() (err error) {
		// If there is a current revision, load the revision into the
		// CurrentRevision field
		if h.CurrentRevisionID.Valid {
			h.CurrentRevision = HostRevision{}

			if err = dbCache.FindByID(&h.CurrentRevision, h.CurrentRevisionID.Int64); err != nil {
				return err
			}
		}

		// Load the HostAddresses from EPP
		if err = dbCache.DB.Where("host_id = ?", h.ID).Find(&h.HostAddresses).Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return err
			}
		}

		// Grab the pending revison if it exists and prepare the revision
		if err = dbCache.GetNewAndPendingRevisions(h); err == nil {
			if err = h.PendingRevision.Prepare(dbCache); err != nil {
				logger.Errorf("PendingRevision.Prepare error: %s\n", err.Error())

				return err
			}
		} else if errors.Is(err, gorm.RecordNotFound) {
			err = nil
		}

		return err
	})
}

// PrepareShallow populate all of the fields for a given object and
// not any of the linked object.
func (h *Host) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, h, FuncNOPErrFunc)
}

// PrepareDisplayShallow populate all of the fields for a given object
// and the current revision but not any of the other linked object.
func (h *Host) PrepareDisplayShallow(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, h, func() (err error) {
		if h.CurrentRevisionID.Valid {
			h.CurrentRevision.ID = h.CurrentRevisionID.Int64
			err = h.CurrentRevision.PrepareShallow(dbCache)
		}

		return
	})
}

// GetPendingRevision implements the RegistrarParent interface and returns
// the pending revision pointer.
func (h *Host) GetPendingRevision() RegistrarObject {
	return &h.PendingRevision
}

// FormDivName creates a name that can be used as the ID for a div tag
// in the host selection forms.
func (h Host) FormDivName() string {
	return fmt.Sprintf("hostname_%d", h.ID)
}

// DisplayName formates a host address to be displayed as part of a
// HTML form.
func (h Host) DisplayName() string {
	return fmt.Sprintf("%d - %s", h.ID, h.HostName)
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple hosts.
//
// TODO: Add paging support
// TODO: Add filtering.
func (h *Host) GetAllPage(ddbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &HostsPage{}

	err = ddbCache.FindAll(&ret.Hosts)
	if err != nil {
		return
	}

	return ret, nil
}

// IsCancelled returns true iff the object has been canclled.
func (h *Host) IsCancelled() bool {
	return h.State == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (h *Host) IsEditable() bool {
	return h.State == StateNew
}

// GetPage will return an object that can be used to render the HTML
// template for the Host.
func (h *Host) GetPage(dbCache *DBCache, username string, email string) (rop RegistrarObjectPage, err error) {
	ret := &HostPage{Editable: true, IsNew: true}

	if h.ID != 0 {
		ret.Editable = h.IsEditable()
		ret.IsNew = false
	}

	ret.Hos = *h

	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)
	if err != nil {
		return rop, err
	}

	ret.PendingActions = make(map[string]string)
	if h.PendingRevision.ID != 0 {
		ret.PendingActions = h.PendingRevision.GetActions(false)
	}

	var SuggestedHostAddresses []HostAddress

	if h.CurrentRevisionID.Valid {
		rawPage, rawPageErr := h.CurrentRevision.GetPage(dbCache, username, email)

		if rawPageErr != nil {
			err = rawPageErr

			return rop, err
		}

		ret.CurrentRevisionPage = rawPage.(*HostRevisionPage)
		SuggestedHostAddresses = h.CurrentRevision.HostAddresses
	}

	if h.HasPendingRevision() {
		rawPage, rawPageErr := h.PendingRevision.GetPage(dbCache, username, email)

		if rawPageErr != nil {
			err = rawPageErr

			return rop, err
		}

		ret.PendingRevisionPage = rawPage.(*HostRevisionPage)
	} else {
		ret.PendingRevisionPage = &HostRevisionPage{IsEditable: true, IsNew: true, ValidApproverSets: ret.ValidApproverSets}
	}

	ret.PendingRevisionPage.ParentHost = h
	ret.PendingRevisionPage.SuggestedRequiredApprovers = make(map[int64]ApproverSetDisplayObject)
	ret.PendingRevisionPage.SuggestedInformedApprovers = make(map[int64]ApproverSetDisplayObject)
	ret.PendingRevisionPage.SuggestedHostAddresses = SuggestedHostAddresses

	if h.CurrentRevision.ID != 0 {
		for _, appSet := range h.CurrentRevision.RequiredApproverSets {
			ret.PendingRevisionPage.SuggestedRequiredApprovers[appSet.ID] = appSet.GetDisplayObject()
		}

		for _, appSet := range h.CurrentRevision.InformedApproverSets {
			ret.PendingRevisionPage.SuggestedInformedApprovers[appSet.ID] = appSet.GetDisplayObject()
		}
	} else {
		appSet, prepErr := GetDefaultApproverSet(dbCache)

		if prepErr != nil {
			err = errors.New("unable to find default approver - database probably not bootstrapped")

			return rop, err
		}

		ret.PendingRevisionPage.SuggestedRequiredApprovers[1] = appSet.GetDisplayObject()
	}

	return ret, nil
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (h Host) GetType() string {
	return HostType
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
func (h *Host) ParseFromForm(request *http.Request, dbCache *DBCache) error {
	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	hostName := strings.ToUpper(request.FormValue("host_name"))

	he, heerr := HostnameExists(hostName, dbCache)
	if heerr != nil {
		return heerr
	}

	if !he {
		h.HostName = hostName
	} else {
		return fmt.Errorf("the host %s already exists", hostName)
	}

	h.State = StateNew
	h.CreatedBy = runame
	h.UpdatedBy = runame

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse Host object with the changes that were
// made. An error object is always the second return value which is nil
// when no errors have occurred during parsing otherwise an error is
// returned.
func (h *Host) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, _ Config) error {
	if h.ID > 0 {
		runame, err := GetRemoteUser(request)
		if err != nil {
			return err
		}

		if h.State == StateNew {
			h.UpdatedBy = runame

			hostName := strings.ToUpper(request.FormValue("host_name"))

			he, heerr := HostnameExists(hostName, dbCache)
			if heerr != nil {
				return heerr
			}

			if !he {
				h.HostName = hostName

				return nil
			}

			return fmt.Errorf("the host %s already exists", hostName)
		}

		return fmt.Errorf("cannot update host in %s state", h.State)
	}

	return errors.New("to update a host the ID must be greater than 0")
}

// GetRequiredApproverSets returns the list of approver sets that are
// required for the Host (if a valid approver revision exists). If
// no approver revisions are found, a default of the infosec approver
// set will be returned.
func (h *Host) GetRequiredApproverSets(dbCache *DBCache) (approvers []ApproverSet, err error) {
	return GetRequiredApproverSets(dbCache, h)
}

// GetInformedApproverSets returns the list of approver sets that are
// informed for the Host (if a valid approver revision exists). If
// no approver revisions are found, an empty list will be returned.
func (h *Host) GetInformedApproverSets(dbCache *DBCache) (as []ApproverSet, err error) {
	return GetInformedApproverSets(dbCache, h)
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (h *Host) TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, _ bool, authMethod AuthType, _ Config) (errs []error) {
	switch actionName {
	case ActionUpdateEPPInfo:
		if authMethod == CertAuthType {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				errs = append(errs, err)

				return errs
			}

			if err := h.LoadEPPInfo(body, dbCache); err != nil {
				logger.Error(err)

				errs = append(errs, err)
			}

			return errs
		}
	case ActionUpdateEPPCheckRequired:
		if authMethod == CertAuthType {
			h.CheckRequired = false

			err := dbCache.Save(h)
			if err != nil {
				errs = append(errs, err)

				return errs
			}
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(h.GetExportVersion()))

			return errs
		}
	case ActionUpdatePreview:
		newHost := Host{Model: Model{ID: h.ID}}

		if err := newHost.PrepareShallow(dbCache); err != nil {
			errs = append(errs, err)

			return errs
		}

		newHost.PreviewIPs = h.CurrentRevision.GetPreviewIPs()

		err := dbCache.Save(&newHost)
		if err != nil {
			errs = append(errs, err)

			return errs
		}

		return errs
	}

	return errs
}

// LoadEPPInfo accepts an EPP response and attempts to marshall the data
// from the response into the object that was called.
//
// TODO: Consider moving the query into dbcache.
func (h *Host) LoadEPPInfo(data []byte, dbCache *DBCache) (err error) {
	var resp epp.Response

	if unMarshalErr := json.Unmarshal(data, &resp); unMarshalErr != nil {
		return fmt.Errorf("unable to unmarshal epp rersponse: %w", unMarshalErr)
	}

	hostFound := false

	if resp.ResultData.HostInfDataResp != nil {
		eppObj := resp.ResultData.HostInfDataResp

		if h.HostName != strings.ToUpper(eppObj.Name) {
			return fmt.Errorf("%s does not match %s", strings.ToUpper(h.HostName), strings.ToUpper(eppObj.Name))
		}

		hostFound = true
		h.HostROID = eppObj.ROID

		h.OKStatus = false
		h.PendingCreateStatus = false
		h.PendingDeleteStatus = false
		h.PendingTransferStatus = false
		h.PendingUpdateStatus = false
		h.ClientDeleteProhibitedStatus = false
		h.ClientTransferProhibitedStatus = false
		h.ClientUpdateProhibitedStatus = false
		h.ServerDeleteProhibitedStatus = false
		h.ServerTransferProhibitedStatus = false
		h.ServerUpdateProhibitedStatus = false

		hostStatus := []string{}

		for _, status := range eppObj.Status {
			hostStatus = append(hostStatus, status.StatusFlag)

			switch status.StatusFlag {
			case "ok":
				h.OKStatus = true
			case epp.StatusPendingCreate:
				h.PendingCreateStatus = true
			case epp.StatusPendingDelete:
				h.PendingDeleteStatus = true
			case epp.StatusPendingTransfer:
				h.PendingTransferStatus = true
			case epp.StatusPendingUpdate:
				h.PendingUpdateStatus = true
			case epp.StatusClientUpdateProhibited:
				h.ClientUpdateProhibitedStatus = true
			case epp.StatusServerUpdateProhibited:
				h.ServerUpdateProhibitedStatus = true
			case epp.StatusClientTransferProhibited:
				h.ClientTransferProhibitedStatus = true
			case epp.StatusServerTransferProhibited:
				h.ServerTransferProhibitedStatus = true
			case epp.StatusClientDeleteProhibited:
				h.ClientDeleteProhibitedStatus = true
			case epp.StatusServerDeleteProhibited:
				h.ServerDeleteProhibitedStatus = true
			case epp.StatusLinked:
				h.LinkedStatus = true
			default:
				logger.Errorf("Unknown State %s", status.StatusFlag)
			}
		}

		h.HostStatus = strings.Join(hostStatus, " ")

		h.HostAddresses = []HostAddressEpp{}

		for _, haraw := range eppObj.Addresses {
			hostAddress := HostAddressEpp{}
			hostAddress.IPAddress = haraw.Address

			switch haraw.IPVersion {
			case epp.IPv4:
				hostAddress.Protocol = 4
			case epp.IPv6:
				hostAddress.Protocol = 6
			default:
				return fmt.Errorf("unknow IP version %s", haraw.IPVersion)
			}

			// Next line is broken, it needs to be updated to remove existing entries
			h.HostAddresses = append(h.HostAddresses, hostAddress)
		}

		if err = dbCache.DB.Where("host_id = ?", h.ID).Delete(HostAddressEpp{}).Error; err != nil {
			return err
		}

		h.CreateClientID = eppObj.CreateID
		if tcr, crErr := time.Parse("2006-01-02T15:04:05Z", eppObj.CreateDate); crErr == nil {
			h.CreateDate = tcr
		}

		h.UpdateClientID = eppObj.UpdateID
		if tud, trErr := time.Parse("2006-01-02T15:04:05Z", eppObj.UpdateDate); trErr == nil {
			h.UpdateDate = tud
		}

		h.SponsoringClientID = eppObj.ClientID
	}

	if !hostFound {
		return fmt.Errorf("hostname %s not found", h.HostName)
	}

	implemented, eppStatusFlag := h.EPPMatchesExpected(&resp)

	h.EPPStatus = eppStatusFlag
	h.EPPLastUpdate = time.Now().Truncate(time.Second)

	if implemented {
		h.CheckRequired = false
	}

	return dbCache.Save(h)
}

// EPPMatchesExpected takes a response from an EPP server and will compare the
// host object to the EPP response and return true iff the host is
// configured correctly or false otherwise. If false is returned, the string
// passed back represents the current state of the host.
func (h *Host) EPPMatchesExpected(resp *epp.Response) (bool, string) {
	if resp == nil || resp.ResultData == nil || resp.ResultData.HostInfDataResp == nil {
		return false, "Invalid EPP Domain Object"
	}

	hostInf := resp.ResultData.HostInfDataResp

	PendingCreateStatus := false
	PendingDeleteStatus := false
	PendingTransferStatus := false
	PendingUpdateStatus := false
	ClientDeleteProhibitedStatus := false
	ClientTransferProhibitedStatus := false
	ClientUpdateProhibitedStatus := false
	ServerDeleteProhibitedStatus := false
	ServerTransferProhibitedStatus := false
	ServerUpdateProhibitedStatus := false

	for _, status := range hostInf.Status {
		switch status.StatusFlag {
		case epp.StatusPendingCreate:
			PendingCreateStatus = true
		case epp.StatusPendingDelete:
			PendingDeleteStatus = true
		case epp.StatusPendingTransfer:
			PendingTransferStatus = true
		case epp.StatusPendingUpdate:
			PendingUpdateStatus = true
		case epp.StatusClientUpdateProhibited:
			ClientUpdateProhibitedStatus = true
		case epp.StatusServerUpdateProhibited:
			ServerUpdateProhibitedStatus = true
		case epp.StatusClientTransferProhibited:
			ClientTransferProhibitedStatus = true
		case epp.StatusServerTransferProhibited:
			ServerTransferProhibitedStatus = true
		case epp.StatusClientDeleteProhibited:
			ClientDeleteProhibitedStatus = true
		case epp.StatusServerDeleteProhibited:
			ServerDeleteProhibitedStatus = true
		}
	}

	if PendingCreateStatus || PendingDeleteStatus || PendingTransferStatus || PendingUpdateStatus {
		return false, "PendingChangeState"
	}

	if !(ClientUpdateProhibitedStatus == h.CurrentRevision.ClientUpdateProhibitedStatus &&
		ClientTransferProhibitedStatus == h.CurrentRevision.ClientTransferProhibitedStatus &&
		ClientDeleteProhibitedStatus == h.CurrentRevision.ClientDeleteProhibitedStatus) {
		return false, "Client Flags Mismatch"
	}

	if !(ServerUpdateProhibitedStatus == h.CurrentRevision.ServerUpdateProhibitedStatus &&
		ServerTransferProhibitedStatus == h.CurrentRevision.ServerTransferProhibitedStatus &&
		ServerDeleteProhibitedStatus == h.CurrentRevision.ServerDeleteProhibitedStatus) {
		return false, "Server Flags Mismatch"
	}

	eppIPs := []string{}
	revisionIps := []string{}

	for _, ip := range h.CurrentRevision.HostAddresses {
		revisionIps = append(revisionIps, ip.IPAddress)
	}

	for _, ip := range resp.ResultData.HostInfDataResp.Addresses {
		eppIPs = append(eppIPs, ip.Address)
	}

	add, remove := DiffIPLists(eppIPs, revisionIps)
	if len(add) != 0 {
		return false, fmt.Sprintf("Missing Host: %s", add[0])
	}

	if len(remove) != 0 {
		return false, fmt.Sprintf("Additional Host: %s", remove[0])
	}

	return true, "Provisioned"
}

// HostAddressEpp is an object that will hold an IP address and the
// prodocol that the address is from (IPv4 / IPv6).
type HostAddressEpp struct {
	ID        int64 `gorm:"primary_key:yes"`
	HostID    int64
	IPAddress string
	Protocol  int64
}

// DisplayName formates a host address to be displayed as part of a
// HTML form.
func (h HostAddressEpp) DisplayName() string {
	return fmt.Sprintf("%s - IPv%d", h.IPAddress, h.Protocol)
}

// FormValue will format the host address so it can be used as the
// value for a html form item.
func (h HostAddressEpp) FormValue() string {
	return fmt.Sprintf("v%d-%s", h.Protocol, h.IPAddress)
}

// FormDivName creates a name that can be used as the ID for a div tag
// in the host selection forms.
func (h HostAddressEpp) FormDivName() string {
	if h.Protocol == 4 || h.Protocol == 6 {
		return fmt.Sprintf("host_address_v%d-%s", h.Protocol, strings.Replace(strings.Replace(h.IPAddress, ".", "G", -1), ":", "S", -1))
	}

	return ""
}

// VerifyCR Checks to make sure that all of the values and approvals
// within a change request match the host that it is linked to
//
// TODO: more rigirous check on if the CR approved text matches.
func (h *Host) VerifyCR(dbCache *DBCache) (checksOut bool, errs []error) {
	return VerifyCR(dbCache, h, nil)
}

// GetCurrentRevisionID will return the id of the current Host
// Revision for the host object.
func (h *Host) GetCurrentRevisionID() sql.NullInt64 {
	return h.CurrentRevisionID
}

// GetPendingRevisionID will return the current pending revision for the
// Host object if it exists. If no pending revision exists a 0 is
// returned.
func (h *Host) GetPendingRevisionID() int64 {
	return h.PendingRevision.ID
}

// GetPendingCRID will return the current CR id if it is set, otherwise
// a nil will be returned (in the form of a sql.NullInt64).
func (h *Host) GetPendingCRID() sql.NullInt64 {
	return h.PendingRevision.CRID
}

// ComparePendingToCallback will return a function that will compare the
// current revision object to itself after changes have been made.
func (h *Host) ComparePendingToCallback(loadFn CompareLoadFn) (retFn CompareReturnFn) {
	exp := HostExport{}
	loadFn(&exp)

	return func() (pass bool, errs []error) {
		return exp.PendingRevision.Compare(h.PendingRevision)
	}
}

// UpdateState can be called at any point to check the state of the
// Host and update it if necessary
//
// TODO: Implement
// TODO: Make sure callers check errors.
func (h *Host) UpdateState(dbCache *DBCache, _ Config) (changesMade bool, errs []error) {
	logger.Infof("UpdateState called on Host %d (todo)", h.ID)

	if err := h.Prepare(dbCache); err != nil {
		errs = append(errs, err)

		return changesMade, errs
	}

	changesMade = false

	switch h.State {
	case StateNew, StateInactive, StateActive:
		logger.Infof("UpdateState for Host at state \"%s\", Nothing to do", h.State)
	case StatePendingBootstrap, StatePendingNew, StateActivePendingApproval, StateInactivePendingApproval:
		if h.PendingRevision.ID != 0 && h.PendingRevision.CRID.Valid {
			crChecksOut, crcheckErrs := h.VerifyCR(dbCache)

			if len(crcheckErrs) != 0 {
				errs = append(errs, crcheckErrs...)
			}

			if crChecksOut {
				changeRequest := ChangeRequest{}
				if err := dbCache.FindByID(&changeRequest, h.PendingRevision.CRID.Int64); err != nil {
					errs = append(errs, err)

					return changesMade, errs
				}

				if changeRequest.State == StateApproved {
					// Promote the new revision
					targetState := h.PendingRevision.DesiredState

					if err := h.PendingRevision.Promote(dbCache); err != nil {
						errs = append(errs, fmt.Errorf("error promoting revision: %s", err.Error()))

						return changesMade, errs
					}

					if h.CurrentRevisionID.Valid {
						if err := h.CurrentRevision.Supersed(dbCache); err != nil {
							errs = append(errs, fmt.Errorf("error superseding revision: %s", err.Error()))

							return changesMade, errs
						}
					}

					newHost := Host{Model: Model{ID: h.ID}}

					if err := newHost.PrepareShallow(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if targetState == StateActive || targetState == StateInactive {
						newHost.State = targetState
					} else {
						errs = append(errs, ErrPendingRevisionInInvalidState)
						logger.Errorf("Unexpected target state: %s", targetState)

						return changesMade, errs
					}

					var revision sql.NullInt64
					revision.Int64 = h.PendingRevision.ID
					revision.Valid = true

					newHost.CurrentRevisionID = revision
					newHost.CheckRequired = true
					newHost.EPPStatus = "Pending Change"
					newHost.EPPLastUpdate = time.Now()
					newHost.PreviewIPs = h.PendingRevision.GetPreviewIPs()

					if err := dbCache.Save(&newHost); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					pendingRevision := HostRevision{}

					if err := dbCache.FindByID(&pendingRevision, h.PendingRevision.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					logger.Infof("Pending Revision State = %s", pendingRevision.RevisionState)
				} else if changeRequest.State == StateDeclined {
					logger.Infof("CR %d has been declined", changeRequest.GetID())

					if err := h.PendingRevision.Decline(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					newHost := Host{}

					if err := dbCache.FindByID(&newHost, h.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if h.CurrentRevisionID.Valid {
						curRev := HostRevision{}

						if err := dbCache.FindByID(&curRev, h.CurrentRevisionID.Int64); err != nil {
							errs = append(errs, err)

							return changesMade, errs
						}

						if curRev.RevisionState == StateBootstrap {
							newHost.State = StateActive
						} else {
							newHost.State = curRev.RevisionState
						}
					} else {
						newHost.State = StateNew
					}

					if err := dbCache.Save(&newHost); err != nil {
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
			if h.State == StatePendingBootstrap {
				h.State = StateBootstrap
				changesMade = true
			} else if h.CurrentRevisionID.Valid {
				h.State = h.CurrentRevision.DesiredState
				changesMade = true
			} else {
				h.State = StateNew
				changesMade = true
			}
		}
	default:
		errs = append(errs, fmt.Errorf("updateState for Host at state \"%s\" not implemented", h.State))
	}

	if changesMade {
		if err := dbCache.Save(h); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}
	}

	return changesMade, errs
}

// UpdateHoldStatus is used to adjust the object's Hold status after
// the user has been verified as an admin. An error will be returned
// if the hold reason is not set.
func (h *Host) UpdateHoldStatus(holdActive bool, holdReason string, holdBy string) error {
	if !holdActive {
		h.HoldActive = false
		h.HoldAt = time.Unix(0, 0)
		h.HoldReason = ""
		h.HoldBy = ""
	} else {
		if len(holdReason) == 0 {
			return errors.New("a hold reason must be set")
		}

		h.HoldActive = true
		h.HoldAt = time.Now()
		h.HoldBy = holdBy
		h.HoldReason = holdReason
	}

	return nil
}

// GetValidHostMap will return a map container the Host's title indexed
// by the host ID. Only
//
// TODO: Consider moving the query into dbcache.
func GetValidHostMap(dbCache *DBCache) (ret map[int64]string, err error) {
	var hostList []Host

	ret = make(map[int64]string)

	err = dbCache.DB.Where("state = ? or state = ?", StateActive, StateActivePendingApproval).Find(&hostList).Error
	if err != nil {
		return
	}

	for _, host := range hostList {
		ret[host.ID] = host.HostName
	}

	return
}

// ParseHostList takes the http Request, a database connection and
// the HTML ID of the host list to parse and will return an array
// of Hosts that correspond to each of the IDs from the http
// request's html element. If there are unparsable IDs in the list
// (could be strings or empty fields) an error is returned. Any valid
// hosts that are found will be returned in the array, even if an
// error is found.
func ParseHostList(request *http.Request, dbCache *DBCache, htmlID string) (h []Host, err error) {
	unparsableIDs := ""
	failedParsing := false

	var (
		hostList []Host
		outErr   error
	)

	for _, hostidChunk := range request.Form[htmlID] {
		for _, hostidRaw := range strings.Split(hostidChunk, " ") {
			var hostid int64
			hostid, err = strconv.ParseInt(hostidRaw, 10, 64)

			if err == nil {
				tmpHost := Host{}

				if err = dbCache.FindByID(&tmpHost, hostid); err != nil {
					return h, err
				}

				hostList = append(hostList, tmpHost)
			} else {
				failedParsing = true
				unparsableIDs = fmt.Sprintf("%s %s", unparsableIDs, hostidRaw)
			}
		}
	}

	outErr = nil

	if failedParsing {
		outErr = fmt.Errorf("unable to parse the following IDs from %s: %s", htmlID, unparsableIDs)
	}

	return hostList, outErr
}

// UpdateHosts will update a set of hosts for an object that
// is passed in to make the list reflect the list passed as the third
// parameter.
//
// TODO: Consider moving the query into dbcache.
func UpdateHosts(object RegistrarObject, dbCache *DBCache, association string, hosts []Host) error {
	err := dbCache.DB.Model(object).Association(association).Clear().Error
	if err != nil {
		return err
	}

	for _, host := range hosts {
		err = dbCache.DB.Model(object).Association(association).Append(host).Error

		if err != nil {
			return err
		}
	}

	return nil
}

// CompareToHostListExportShortList compares a list of host list to a
// set of hosts that were from an export version of an object. If the
// counts match and the IDs for the host list match true is returned
// otherwise, false is returned.
func CompareToHostListExportShortList(host []Host, hostExport []HostExportShort) bool {
	exportShortCount := len(hostExport)
	fullCount := len(host)

	if fullCount != exportShortCount {
		return false
	}

	for _, full := range host {
		found := false

		for _, export := range hostExport {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	for _, export := range hostExport {
		found := false

		for _, full := range host {
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

// CompareToHostExportShortLists compares a list of host export objects
// to another list of host export objects. If the counts match and the
// IDs for the host list match true is returned otherwise, false is
// returned.
func CompareToHostExportShortLists(hostExport1 []HostExportShort, hostExport2 []HostExportShort) bool {
	exportShortCount := len(hostExport2)
	fullCount := len(hostExport1)

	if fullCount != exportShortCount {
		return false
	}

	for _, full := range hostExport1 {
		found := false

		for _, export := range hostExport2 {
			if export.ID == full.ID {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	for _, export := range hostExport2 {
		found := false

		for _, full := range hostExport1 {
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

// HostnameExists checks to see if the hostname passed is currently the
// hostname associated with a current revision. If there is an error
// querying the database, it will be returned otherwise true is returned
// indicating that the hostname is in use or not
//
// TODO: Consider moving the query into dbcache.
func HostnameExists(hostname string, dbCache *DBCache) (ret bool, err error) {
	var hostnameCount int64

	rows, rowsErr := dbCache.DB.Raw("select count(hosts.id) from hosts where host_name = ?", hostname).Rows()

	if rowsErr != nil {
		return false, fmt.Errorf("error querying database: %w", rowsErr)
	}

	err = rows.Err()
	if err != nil {
		return ret, fmt.Errorf("error interacting with database row: %w", err)
	}

	defer func() {
		if err != nil {
			err = rows.Close()
		}
	}()

	for rows.Next() {
		if err := rows.Scan(&hostnameCount); err != nil {
			return false, fmt.Errorf("error scanning hostname count: %w", err)
		}
	}

	return (hostnameCount != 0), nil
}

// GetWorkHosts returns a list of IDs for hosts that require
// attention in the form of an update to the registry or other related
// information. If an error occurs, it will be returned
//
// TODO: Consider moving the query into dbcache.
func GetWorkHosts(dbCache *DBCache) (retlist []int64, revisions []APIRevisionHint, err error) {
	var hosts []Host

	err = dbCache.DB.Where(map[string]interface{}{"check_required": true, "hold_active": false}).Find(&hosts).Error
	if err != nil {
		return
	}

	for _, host := range hosts {
		retlist = append(retlist, host.ID)

		if host.CurrentRevisionID.Valid {
			rev := APIRevisionHint{}
			rev.ObjectID = host.ID
			rev.RevisionID = host.CurrentRevisionID.Int64
			rev.LastUpdate = host.UpdatedAt
			revisions = append(revisions, rev)
		}
	}

	return
}

// GetWorkHostsFull returns a list of Hosts that require work to be done
// or an error.
func GetWorkHostsFull(dbCache *DBCache) (hosts []Host, err error) {
	err = dbCache.DB.Where(map[string]interface{}{"check_required": true}).Find(&hosts).Error

	return
}

// GetAllHosts returns a list of IDs for hosts that are in the
// active or activepending approval states or requires work to be done
// on the host. If an error occurs, it will be returned
//
// TODO: Consider moving the query into dbcache.
func GetAllHosts(dbCache *DBCache) (retlist []int64, revisions []APIRevisionHint, err error) {
	var hosts []Host

	err = dbCache.DB.Where("state = ? or state = ?", StateActive, StateActivePendingApproval).Find(&hosts).Error
	if err != nil {
		return
	}

	for _, host := range hosts {
		retlist = append(retlist, host.ID)

		if host.CurrentRevisionID.Valid {
			rev := APIRevisionHint{}
			rev.ObjectID = host.ID
			rev.RevisionID = host.CurrentRevisionID.Int64
			rev.LastUpdate = host.UpdatedAt
			revisions = append(revisions, rev)
		}
	}

	return
}

// GetAllHostNamess will return a map from hostnames to IDs for the related host.
func GetAllHostNamess(dbCache *DBCache) (hostnames map[string]int64, err error) {
	hostnames = make(map[string]int64)

	var hosts []Host
	err = dbCache.DB.Where("state != ?", StateNew).Find(&hosts).Error

	if err != nil {
		return
	}

	for _, host := range hosts {
		hostnames[host.HostName] = host.ID
	}

	return
}

const (
	ipVersion4Int = 4
	ipVersion6Int = 6
)

// DiffIPsExport will take an epp response for a host and a host object from the
// registrar system and generate lists of the IP addresses to both add and
// remove for IPv4 and IPv6. If an error occurs during the process, it will be
// returned.
func DiffIPsExport(registry *epp.Response, registrar *HostExport) (ipv4Add, ipv4Rem, ipv6Add, ipv6Rem []string, err error) {
	var (
		registryV4, registryV6   []string
		registrarV4, registrarV6 []string
	)

	if registry == nil || registry.ResultData == nil || registry.ResultData.HostInfDataResp == nil {
		err = errors.New("Host info section of registry response is empty")

		return ipv4Add, ipv4Rem, ipv6Add, ipv6Rem, err
	}

	for _, host := range registry.ResultData.HostInfDataResp.Addresses {
		switch host.IPVersion {
		case epp.IPv4:
			registryV4 = append(registryV4, host.Address)
		case epp.IPv6:
			registryV6 = append(registryV6, host.Address)
		}
	}

	for _, host := range registrar.CurrentRevision.HostAddresses {
		switch host.Protocol {
		case ipVersion4Int:
			registrarV4 = append(registrarV4, host.IPAddress)
		case ipVersion6Int:
			registrarV6 = append(registrarV6, host.IPAddress)
		}
	}

	ipv4Add, ipv4Rem = DiffIPLists(registryV4, registrarV4)
	ipv6Add, ipv6Rem = DiffIPLists(registryV6, registrarV6)

	return ipv4Add, ipv4Rem, ipv6Add, ipv6Rem, err
}

// DiffIPLists will take two lists of IP addresses represented as strings and
// compare the IPs to each other to deteremine which addresses need to be added
// or removed from the current list in order to match the expected list.
func DiffIPLists(currentList, expectedList []string) (add, remove []string) {
	for _, current := range currentList {
		found := false

		for _, expected := range expectedList {
			if net.ParseIP(current).Equal(net.ParseIP(expected)) {
				found = true

				break
			}
		}

		if !found {
			remove = append(remove, current)
		}
	}

	for _, expected := range expectedList {
		found := false

		for _, current := range currentList {
			if net.ParseIP(current).Equal(net.ParseIP(expected)) {
				found = true

				break
			}
		}

		if !found {
			add = append(add, expected)
		}
	}

	return add, remove
}

// MigrateDBHost will run the automigrate function for the Host
// object.
func MigrateDBHost(dbCache *DBCache) {
	dbCache.AutoMigrate(&Host{})
	dbCache.AutoMigrate(&HostAddressEpp{})
}
