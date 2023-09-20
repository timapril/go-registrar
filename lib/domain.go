package lib

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/timapril/go-registrar/epp"
	"github.com/timapril/go-registrar/whois-parse"

	"github.com/aryann/difflib"
	"github.com/jinzhu/gorm"
)

// Domain is an object that represents the state of a registrar domain
// object as defined by RFC 5731
// http://tools.ietf.org/html/rfc5731
type Domain struct {
	Model
	State string

	RegistryDomainID string `sql:"size:32"`

	DomainStatus                   string
	ClientDeleteProhibitedStatus   bool
	ServerDeleteProhibitedStatus   bool
	ClientHoldStatus               bool
	ServerHoldStatus               bool
	ClientRenewProhibitedStatus    bool
	ServerRenewProhibitedStatus    bool
	ClientTransferProhibitedStatus bool
	ServerTransferProhibitedStatus bool
	ClientUpdateProhibitedStatus   bool
	ServerUpdateProhibitedStatus   bool
	LinkedStatus                   bool
	OKStatus                       bool
	PendingCreateStatus            bool
	PendingDeleteStatus            bool
	PendingRenewStatus             bool
	PendingTransferStatus          bool
	PendingUpdateStatus            bool

	DomainName               string `sql:"size:256"`
	DomainROID               string `sql:"size:128"`
	DomainRegistrantROID     string `sql:"size:128"`
	DomainAdminContactROID   string `sql:"size:128"`
	DomainTechContactROID    string `sql:"size:128"`
	DomainBillingContactROID string `sql:"size:128"`
	DomainNSList             string `sql:"size:1024"`

	PreviewHostnames string `sql:"size:2048"`

	DSDataEntries []DSDataEntryEpp

	SponsoringClientID string `sql:"size:32"`
	CreateClientID     string `sql:"size:32"`
	CreateDate         time.Time
	UpdateClientID     string `sql:"size:32"`
	UpdateDate         time.Time
	TransferDate       time.Time
	ExpireDate         time.Time

	HoldActive bool
	HoldBy     string
	HoldAt     time.Time
	HoldReason string `sql:"size:1024"`

	CurrentRevision   DomainRevision
	CurrentRevisionID sql.NullInt64
	Revisions         []DomainRevision
	PendingRevision   DomainRevision `sql:"-"`

	WHOISRegistrar   string
	WHOISRegistrarID string
	WHOISServer      string
	WHOISDomainNS    string

	WHOISRegistrantName string
	WHOISAdminName      string
	WHOISTechName       string
	WHOISBillingName    string

	WHOISStatuses string

	WHOISDNSSSECSigned bool

	WHOISUpdatedDate        time.Time
	WHOISCreateDate         time.Time
	WHOISExpireDate         time.Time
	WHOISLastUpdatedAt      time.Time
	WHOISLastConfirmEmailAt time.Time

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

var (
	// EPPStatusProvisioned represents a domain that has been successfully
	// provisioned.
	EPPStatusProvisioned = "Provisioned"

	// EPPStatusServerFlagsMismatch represents the state where a domain has
	// incorrect server flags.
	EPPStatusServerFlagsMismatch = "Server Flags Mismatch"

	// EPPStatusAdditionalDSRecords represents the state where DS records exist in
	// the registry but not in the configured state.
	EPPStatusAdditionalDSRecords = "Additional DS Record(s) found"

	// EPPStatusMissingDSRecords represents the state where DS not all of the
	// requested DS records are present in the registry.
	EPPStatusMissingDSRecords = "Missing DS Record(s)"

	// EPPStatusHostMismatch represents the state where the list of hosts at the
	// registry does not match the requested state.
	EPPStatusHostMismatch = "Hosts mismatch"

	// EPPStatusClientFlagMismatch represents the state where client flags need
	// to be changed.
	EPPStatusClientFlagMismatch = "Client Flags Mismatch"

	// EPPStatusPendingChange represents the state where a change needs to be
	// made to a host.
	EPPStatusPendingChange = "Pending Change"

	// EPPStatusPendingRenew represents the state where a domain is pending
	// renewal.
	EPPStatusPendingRenew = "Pending Renew"
)

// DomainExport is an object that is used to export the current
// state of a Domain object. The full version of the export object
// also contains the current and pending revision (if either exist).
type DomainExport struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	DomainName string `json:"DomainName"`
	DomainROID string `json:"DomainROID"`

	CurrentRevision DomainRevisionExport `json:"CurrentRevision"`
	PendingRevision DomainRevisionExport `json:"PendingRevision"`

	CreateDate time.Time `json:"CreateDate"`
	UpdateDate time.Time `json:"UpdateDate"`
	ExpireDate time.Time `json:"ExpireDate"`

	ClientDeleteProhibitedStatus   bool `json:"ClientDeleteProhibitedStatus"`
	ServerDeleteProhibitedStatus   bool `json:"ServerDeleteProhibitedStatus"`
	ClientHoldStatus               bool `json:"ClientHoldStatus"`
	ServerHoldStatus               bool `json:"ServerHoldStatus"`
	ClientRenewProhibitedStatus    bool `json:"ClientRenewProhibitedStatus"`
	ServerRenewProhibitedStatus    bool `json:"ServerRenewProhibitedStatus"`
	ClientTransferProhibitedStatus bool `json:"ClientTransferProhibitedStatus"`
	ServerTransferProhibitedStatus bool `json:"ServerTransferProhibitedStatus"`
	ClientUpdateProhibitedStatus   bool `json:"ClientUpdateProhibitedStatus"`
	ServerUpdateProhibitedStatus   bool `json:"ServerUpdateProhibitedStatus"`
	LinkedStatus                   bool `json:"LinkedStatus"`
	OKStatus                       bool `json:"OKStatus"`
	PendingCreateStatus            bool `json:"PendingCreateStatus"`
	PendingDeleteStatus            bool `json:"PendingDeleteStatus"`
	PendingRenewStatus             bool `json:"PendingRenewStatus"`
	PendingTransferStatus          bool `json:"PendingTransferStatus"`
	PendingUpdateStatus            bool `json:"PendingUpdateStatus"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`

	HoldActive bool      `json:"HoldActive"`
	HoldBy     string    `json:"HoldBy"`
	HoldAt     time.Time `json:"HoldAt"`
	HoldReason string    `json:"HoldReason"`
}

// GetDiff will return a string containing a formatted diff of the
// current and pending revisions for the Domain object. An empty
// string and an error are returned if an error occures during the
// processing
//
// TODO: Handle diff for objects that do not have a pending revision
// TODO: Handle diff for objects that do not have a current revision.
func (d DomainExport) GetDiff() (string, error) {
	current, _ := d.CurrentRevision.ToJSON()
	pending, err2 := d.PendingRevision.ToJSON()

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
func (d DomainExport) ToJSON() (string, error) {
	if d.ID <= 0 {
		return "", errors.New("ID not set")
	}

	byteArr, jsonErr := json.MarshalIndent(d, "", "  ")

	return string(byteArr), jsonErr
}

// DomainPage is used to hold all the information required to render
// the Domain HTML page.
type DomainPage struct {
	Editable            bool
	IsNew               bool
	Dom                 Domain
	CurrentRevisionPage *DomainRevisionPage
	PendingRevisionPage *DomainRevisionPage
	PendingActions      map[string]string
	ValidApproverSets   map[int64]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (d *DomainPage) GetCSRFToken() string {
	return d.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (d *DomainPage) SetCSRFToken(newToken string) {
	d.CSRFToken = newToken
	d.PendingRevisionPage.CSRFToken = newToken
}

// DomainsPage is used to hold all the information required to render
// the Domain HTML page.
type DomainsPage struct {
	Domains []Domain

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (d *DomainsPage) GetCSRFToken() string {
	return d.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (d *DomainsPage) SetCSRFToken(newToken string) {
	d.CSRFToken = newToken
}

// GetExportVersion returns a export version of the Domain Object.
func (d *Domain) GetExportVersion() RegistrarObjectExport {
	export := DomainExport{
		ID:              d.ID,
		State:           d.State,
		DomainName:      d.DomainName,
		DomainROID:      d.DomainROID,
		PendingRevision: (d.PendingRevision.GetExportVersion()).(DomainRevisionExport),
		CurrentRevision: (d.CurrentRevision.GetExportVersion()).(DomainRevisionExport),
		UpdatedAt:       d.UpdatedAt,
		UpdatedBy:       d.UpdatedBy,
		CreatedAt:       d.CreatedAt,
		CreatedBy:       d.CreatedBy,
		CreateDate:      d.CreateDate,
		UpdateDate:      d.UpdateDate,
		ExpireDate:      d.ExpireDate,

		ClientDeleteProhibitedStatus:   d.ClientDeleteProhibitedStatus,
		ServerDeleteProhibitedStatus:   d.ServerDeleteProhibitedStatus,
		ClientHoldStatus:               d.ClientHoldStatus,
		ServerHoldStatus:               d.ServerHoldStatus,
		ClientRenewProhibitedStatus:    d.ClientRenewProhibitedStatus,
		ServerRenewProhibitedStatus:    d.ServerRenewProhibitedStatus,
		ClientTransferProhibitedStatus: d.ClientTransferProhibitedStatus,
		ServerTransferProhibitedStatus: d.ServerTransferProhibitedStatus,
		ClientUpdateProhibitedStatus:   d.ClientUpdateProhibitedStatus,
		ServerUpdateProhibitedStatus:   d.ServerUpdateProhibitedStatus,
		LinkedStatus:                   d.LinkedStatus,
		OKStatus:                       d.OKStatus,
		PendingCreateStatus:            d.PendingCreateStatus,
		PendingDeleteStatus:            d.PendingDeleteStatus,
		PendingRenewStatus:             d.PendingRenewStatus,
		PendingTransferStatus:          d.PendingTransferStatus,
		PendingUpdateStatus:            d.PendingUpdateStatus,

		HoldActive: d.HoldActive,
		HoldAt:     d.HoldAt,
		HoldBy:     d.HoldBy,
		HoldReason: d.HoldReason,
	}

	return export
}

// GetExportVersionAt returns an export version of the Domain Object
// at the timestamp provided if possible otherwise an error is returned.
// If a pending version existed at the time it will be excluded from
// the object.
func (d *Domain) GetExportVersionAt(dbCache *DBCache, timestamp int64) (obj RegistrarObjectExport, err error) {
	domainRevision := DomainRevision{}

	// Grab the most recent promoted object before the time provided
	// where the promoted time is after the time stated above
	// If no objects were found meeting the criteria, return an error
	if err = dbCache.GetRevisionAtTime(&domainRevision, d.ID, timestamp); err != nil {
		return
	}

	// Otherwise, prepare the new object, fix the currnet object a little
	// and remove the pending revision if any
	if err = domainRevision.Prepare(dbCache); err != nil {
		return
	}

	d.CurrentRevision = domainRevision
	d.CurrentRevisionID.Int64 = domainRevision.ID
	d.CurrentRevisionID.Valid = true
	d.PendingRevision = DomainRevision{}

	// Return the export version and no error
	return d.GetExportVersion(), nil
}

// UnPreparedDomainError is the text of an error that is displayed
// when a domain has not been prepared before use.
const UnPreparedDomainError = "Error: Domain Not Prepared"

// GetCurrentValue is used to get the current value of a field in a
// revision if a current revision exists, otherwise an empty string is
// returned.
func (d *Domain) GetCurrentValue(field string) (ret string) {
	if !d.prepared {
		return UnPreparedDomainError
	}

	return d.SuggestedRevisionValue(field)
}

// DomainFieldName is a name that can be used to reference the
// name field of the current domain revision.
const DomainFieldName string = "Name"

// DomainFieldOwners is a name that can be used to reference the owners
// field of the current domain revision.
const DomainFieldOwners string = "Owners"

// DomainFieldClass is a name that can be used to reference the domain
// class of the current domain revision.
const DomainFieldClass string = "DomainClass"

// DomainFieldDomainRegistrant is a name that can be used to reference
// the domain registrant id field of the current domain revision.
const DomainFieldDomainRegistrant string = "DomainRegistrant"

// DomainFieldDomainAdminContact is a name that can be used to reference
// the domain admin contact id field of the current domain revision.
const DomainFieldDomainAdminContact string = "DomainAdminContact"

// DomainFieldDomainTechContact is a name that can be used to reference
// the domain technical contact id field of the current domain revision.
const DomainFieldDomainTechContact string = "DomainTechContact"

// DomainFieldDomainBillingContact is a name that can be used to
// reference the domain billing contact id field of the current domain
// revision.
const DomainFieldDomainBillingContact string = "DomainBillingContact"

// SuggestedRevisionValue takes a string naming the field that is being
// requested and returns a string containing the suggested value for
// the field in a new pending revision
//
// TODO: add other fields that have been added.
func (d Domain) SuggestedRevisionValue(field string) string {
	if d.CurrentRevisionID.Valid {
		switch field {
		case DomainFieldName:
			return d.DomainName
		case DomainFieldOwners:
			return d.CurrentRevision.Owners
		case DomainFieldClass:
			return d.CurrentRevision.Class
		case SavedObjectNote:
			return d.CurrentRevision.SavedNotes
		}
	}

	return ""
}

// SuggestedContactID takes a string naming the field that is being
// requested and returns an int64 containing the suggested value for the
// field in a new pending revions.
func (d Domain) SuggestedContactID(field string) int64 {
	if d.CurrentRevisionID.Valid {
		switch field {
		case DomainFieldDomainRegistrant:
			return d.CurrentRevision.DomainRegistrantID
		case DomainFieldDomainAdminContact:
			return d.CurrentRevision.DomainAdminContactID
		case DomainFieldDomainTechContact:
			return d.CurrentRevision.DomainTechContactID
		case DomainFieldDomainBillingContact:
			return d.CurrentRevision.DomainBillingContactID
		}
	}

	return 0
}

// SuggestedRevisionBool takes a string naming the flag that is being
// requested and returnes a bool containing the suggested value for the
// field in the new revision
//
// TODO: add other fields that have been added.
func (d Domain) SuggestedRevisionBool(field string) bool {
	if d.CurrentRevisionID.Valid {
		switch field {
		case ClientDeleteFlag:
			return d.CurrentRevision.ClientDeleteProhibitedStatus
		case ServerDeleteFlag:
			return d.CurrentRevision.ServerDeleteProhibitedStatus
		case ClientRenewFlag:
			return d.CurrentRevision.ClientRenewProhibitedStatus
		case ServerRenewFlag:
			return d.CurrentRevision.ServerRenewProhibitedStatus
		case ClientHoldFlag:
			return d.CurrentRevision.ClientHoldStatus
		case ServerHoldFlag:
			return d.CurrentRevision.ServerHoldStatus
		case ClientTransferFlag:
			return d.CurrentRevision.ClientTransferProhibitedStatus
		case ServerTransferFlag:
			return d.CurrentRevision.ServerTransferProhibitedStatus
		case ClientUpdateFlag:
			return d.CurrentRevision.ClientUpdateProhibitedStatus
		case ServerUpdateFlag:
			return d.CurrentRevision.ServerUpdateProhibitedStatus
		case DesiredStateActive:
			return d.CurrentRevision.DesiredState == StateActive
		case DesiredStateInactive:
			return d.CurrentRevision.DesiredState == StateInactive
		case DesiredStateExternal:
			return d.CurrentRevision.DesiredState == StateExternal

		case DomainClassHighValue:
			return d.CurrentRevision.Class == DomainClassHighValue
		case DomainClassInUse:
			return d.CurrentRevision.Class == DomainClassInUse
		case DomainClassParked:
			return d.CurrentRevision.Class == DomainClassParked
		case DomainClassOther:
			return d.CurrentRevision.Class != DomainClassHighValue &&
				d.CurrentRevision.Class != DomainClassInUse &&
				d.CurrentRevision.Class != DomainClassParked
		}
	} else {
		switch field {
		case DesiredStateExternal:
			return d.State == StateNewExternal
		case DesiredStateActive:
			return d.State != StateNewExternal
		case DomainClassParked:
			// Default doamin status is parked
			return true
		}
	}

	return false
}

// HasRevision returns true iff a current revision exists, otherwise
// false
//
// TODO: add a check to verify that the current revision has an approved
// change request.
func (d Domain) HasRevision() bool {
	return d.CurrentRevisionID.Valid
}

// HasPendingRevision returns true iff a pending revision exists for the
// Domain, otherwise false.
func (d Domain) HasPendingRevision() bool {
	return d.PendingRevision.ID != 0
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
//
// TODO: Consider moving the query into dbcache.
func (d *Domain) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, d, func() (err error) {
		// If there is a current revision, load the revision into the
		// CurrentRevision field
		if d.CurrentRevisionID.Valid {
			d.CurrentRevision = DomainRevision{}
			if err = dbCache.FindByID(&d.CurrentRevision, d.CurrentRevisionID.Int64); err != nil {
				return
			}
		}

		// Load the DSDataEntries from EPP
		if err = dbCache.DB.Where("domain_id = ?", d.ID).Find(&d.DSDataEntries).Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return
			}
		}

		// Grab the pending revison if it exists and prepare the revision
		if err = dbCache.DB.Where("domain_id = ? and revision_state in (?, ?)", d.ID, StateNew, StatePendingApproval).First(&d.PendingRevision).Error; err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return nil
			}

			return
		}

		err = d.PendingRevision.Prepare(dbCache)

		return
	})
}

// PrepareShallow populate all of the fields for a given object and
// not any of the linked object.
func (d *Domain) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, d, FuncNOPErrFunc)
}

// GetPendingRevision implements the RegistrarParent interface and returns
// the pending revision pointer.
func (d *Domain) GetPendingRevision() RegistrarObject {
	return &d.PendingRevision
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple domains.
//
// TODO: Add paging support
// TODO: Add filtering.
func (d *Domain) GetAllPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &DomainsPage{}

	err = dbCache.FindAll(&ret.Domains)
	if err != nil {
		return
	}

	return ret, nil
}

// IsCancelled returns true iff the object has been canclled.
func (d *Domain) IsCancelled() bool {
	return d.State == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (d *Domain) IsEditable() bool {
	if d.State == StateNew || d.State == StateNewExternal {
		return true
	}

	return false
}

// GetPage will return an object that can be used to render the HTML
// template for the Domain.
func (d *Domain) GetPage(dbCache *DBCache, username string, email string) (rop RegistrarObjectPage, err error) {
	ret := &DomainPage{Editable: true, IsNew: true}

	if d.ID != 0 {
		ret.Editable = d.IsEditable()
		ret.IsNew = false
	}

	ret.Dom = *d
	ret.PendingActions = make(map[string]string)
	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	if err != nil {
		return rop, err
	}

	if d.PendingRevision.ID != 0 {
		ret.PendingActions = d.PendingRevision.GetActions(false)
	}

	var (
		SuggestedHostnames []Host
		SuggestedDSEntries []DSDataEntry
	)

	if d.CurrentRevisionID.Valid {
		rawPage, rawPageErr := d.CurrentRevision.GetPage(dbCache, username, email)
		if rawPageErr != nil {
			err = rawPageErr

			return rop, err
		}

		ret.CurrentRevisionPage = rawPage.(*DomainRevisionPage)

		SuggestedHostnames = d.CurrentRevision.Hostnames
		SuggestedDSEntries = d.CurrentRevision.DSDataEntries
	}

	if d.HasPendingRevision() {
		rawPage, rawPageErr := d.PendingRevision.GetPage(dbCache, username, email)

		if rawPageErr != nil {
			err = rawPageErr

			return rop, err
		}

		ret.PendingRevisionPage = rawPage.(*DomainRevisionPage)
	} else {
		hostMap, hostMapErr := GetValidHostMap(dbCache)

		if hostMapErr != nil {
			err = hostMapErr

			return rop, err
		}

		contactMap, contactMapErr := GetValidContactMap(dbCache)

		if contactMapErr != nil {
			err = contactMapErr

			return rop, err
		}

		ret.PendingRevisionPage = &DomainRevisionPage{
			IsEditable:        true,
			IsNew:             true,
			ValidApproverSets: ret.ValidApproverSets,
			ValidHosts:        hostMap,
			ValidContacts:     contactMap,
		}

		ret.PendingRevisionPage.Revision = DomainRevision{}

		if d.State == StateNewExternal {
			ret.PendingRevisionPage.Revision.DesiredState = StateExternal
		} else {
			ret.PendingRevisionPage.Revision.DesiredState = StateNew
		}
	}

	ret.PendingRevisionPage.ParentDomain = d
	ret.PendingRevisionPage.SuggestedRequiredApprovers = make(map[int64]ApproverSetDisplayObject)
	ret.PendingRevisionPage.SuggestedInformedApprovers = make(map[int64]ApproverSetDisplayObject)
	ret.PendingRevisionPage.SuggestedHostnames = SuggestedHostnames
	ret.PendingRevisionPage.SuggestedDSData = SuggestedDSEntries
	ret.PendingRevisionPage.DNSSECAlgorithms = DNSSECAlgorithms
	ret.PendingRevisionPage.DNSSECDigestTypes = DNSSECDigestTypes

	if d.CurrentRevision.ID != 0 {
		for _, appSet := range d.CurrentRevision.RequiredApproverSets {
			ret.PendingRevisionPage.SuggestedRequiredApprovers[appSet.ID] = appSet.GetDisplayObject()
		}

		for _, appSet := range d.CurrentRevision.InformedApproverSets {
			ret.PendingRevisionPage.SuggestedInformedApprovers[appSet.ID] = appSet.GetDisplayObject()
		}
	} else {
		appSet, prepErr := GetDefaultApproverSet(dbCache)

		if prepErr != nil {
			err = errors.New("unable to find default approver - database probably not bootstrapped")
		}

		ret.PendingRevisionPage.SuggestedRequiredApprovers[1] = appSet.GetDisplayObject()
	}

	return ret, err
}

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (d Domain) GetType() string {
	return DomainType
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
func (d *Domain) ParseFromForm(request *http.Request, dbCache *DBCache) error {
	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	domainName := strings.ToUpper(request.FormValue("domain_name"))
	registerable, regErr := IsRegisterableDomain(domainName)

	if regErr != nil {
		return regErr
	}

	if registerable {
		domExists, domExistsErr := DomainExists(domainName, dbCache)
		if domExistsErr != nil {
			return domExistsErr
		}

		if !domExists {
			d.DomainName = domainName
		} else {
			return fmt.Errorf("the domain name %s already exists", domainName)
		}
	} else {
		return fmt.Errorf("the domain name %s is not in a valid zone", domainName)
	}

	d.State = GetActiveNewExternal(request.FormValue("domain_state"))

	d.CreatedBy = runame
	d.UpdatedBy = runame

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse Domain object with the changes that were
// made. An error object is always the second return value which is nil
// when no errors have occurred during parsing otherwise an error is
// returned.
func (d *Domain) ParseFromFormUpdate(request *http.Request, dbCache *DBCache, _ Config) error {
	if d.ID > 0 {
		runame, err := GetRemoteUser(request)
		if err != nil {
			return err
		}

		if d.State == StateNew || d.State == StateNewExternal {
			d.UpdatedBy = runame

			domainName := strings.ToUpper(request.FormValue("domain_name"))

			registerable, regErr := IsRegisterableDomain(domainName)
			if regErr != nil {
				return regErr
			}

			if registerable {
				domExists, domExistsErr := DomainExists(domainName, dbCache)

				if domExistsErr != nil {
					return domExistsErr
				}

				if d.DomainName == domainName || !domExists {
					d.DomainName = domainName
				} else {
					return fmt.Errorf("the domain name %s already exists", domainName)
				}
			} else {
				return fmt.Errorf("the domain name %s is not in a valid zone", domainName)
			}

			d.State = GetActiveNewExternal(request.FormValue("domain_state"))
		} else {
			return fmt.Errorf("cannot update an object not in the %s state", d.State)
		}

		return nil
	}

	return errors.New("to update an domain the ID must be greater than 0")
}

// GetRequiredApproverSets returns the list of approver sets that are
// required for the Domain (if a valid approver revision exists). If
// no approver revisions are found, a default of the infosec approver
// set will be returned.
func (d *Domain) GetRequiredApproverSets(dbCache *DBCache) (approvers []ApproverSet, err error) {
	return GetRequiredApproverSets(dbCache, d)
}

// GetInformedApproverSets returns the list of approver sets that are
// informed for the Domain (if a valid approver revision exists). If
// no approver revisions are found, an empty list will be returned.
func (d *Domain) GetInformedApproverSets(dbCache *DBCache) (as []ApproverSet, err error) {
	return GetInformedApproverSets(dbCache, d)
}

// WHOISNameServers turns the list of name servers into a n array of
// name servers by splitting on newlines.
func (d Domain) WHOISNameServers() []string {
	return strings.Split(d.WHOISDomainNS, "\n")
}

// WHOISStatusFlags turns a string of status flags into a list of status
// flags by splitting on new lines.
func (d Domain) WHOISStatusFlags() []string {
	return strings.Split(d.WHOISStatuses, "\n")
}

// WHOISHasStatus will check to see if a status exists in the list of
// statuses found in the WHOIS data. Comparisons are done on a lower
// case verson of both strings. If the status is found, true is returned
// otherwise, false is retuned.
func (d Domain) WHOISHasStatus(status string) bool {
	statuses := d.WHOISStatusFlags()

	for _, flag := range statuses {
		if strings.EqualFold(status, flag) {
			return true
		}
	}

	return false
}

// UpdateWHOISFromResponse takes a response from a WHOIS query and puts
// the resulting data into the domain object and updates the
// LastUpdateAt field to the current timestamp.
func (d *Domain) UpdateWHOISFromResponse(resp whois.Response, dbCache *DBCache) (errs []error) {
	if resp.ObjectType == "domain" && strings.EqualFold(d.DomainName, resp.DomainName) {
		d.WHOISRegistrarID = resp.SponsoringRegistrarIANAID
		d.WHOISServer = resp.WhoisServer

		d.WHOISDomainNS = strings.Join(resp.NameServers, "\n")
		d.WHOISStatuses = strings.Join(resp.Statuses, "\n")

		d.WHOISRegistrantName = resp.RegistrantContactName
		d.WHOISAdminName = resp.AdminContactName
		d.WHOISTechName = resp.TechContactName
		d.WHOISBillingName = resp.BillingContactName

		d.WHOISUpdatedDate = resp.UpdatedDate
		d.WHOISCreateDate = resp.CreationDate
		d.WHOISExpireDate = resp.ExpirationDate

		d.WHOISLastUpdatedAt = TimeNow()

		d.WHOISDNSSSECSigned = resp.DNSSECSigned

		if err := dbCache.Save(d); err != nil {
			errs = append(errs, err)

			return errs
		}
	} else {
		errs = append(errs, fmt.Errorf("whois information is for %s rather than %s", resp.DomainName, d.DomainName))
	}

	return errs
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (d *Domain) TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, _ bool, authMethod AuthType, conf Config) (errs []error) {
	switch actionName {
	case "updatewhois":
		// TODO
		resp, queryErrs := whois.Query(d.DomainName)
		if len(queryErrs) != 0 {
			errs = append(errs, queryErrs...)

			return errs
		}

		updateErrs := d.UpdateWHOISFromResponse(resp, dbCache)

		if len(updateErrs) != 0 {
			errs = append(errs, updateErrs...)

			return errs
		}

		http.Redirect(response, request, fmt.Sprintf("/view/%s/%d", DomainType, d.ID), http.StatusFound)
	case ActionUpdateEPPInfo:
		if authMethod == CertAuthType {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				errs = append(errs, err)

				return errs
			}

			if err := d.LoadEPPInfo(body, dbCache, conf); err != nil {
				logger.Error(err)

				errs = append(errs, err)
			}

			return errs
		}
	case ActionUpdateEPPCheckRequired:
		if authMethod == CertAuthType {
			d.CheckRequired = false

			err := dbCache.Save(d)
			if err != nil {
				errs = append(errs, err)

				return errs
			}
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(d.GetExportVersion()))

			return errs
		}
	case ActionUpdatePreview:
		newDomain := Domain{Model: Model{ID: d.ID}}

		if err := newDomain.PrepareShallow(dbCache); err != nil {
			errs = append(errs, err)

			return errs
		}

		newDomain.PreviewHostnames = d.CurrentRevision.GetPreviewHostnames()

		err := dbCache.Save(&newDomain)
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
func (d *Domain) LoadEPPInfo(data []byte, dbCache *DBCache, conf Config) (err error) {
	var resp epp.Response
	if unMarshalErr := json.Unmarshal(data, &resp); unMarshalErr != nil {
		return fmt.Errorf("unable to unmarshal response: %w", unMarshalErr)
	}

	if resp.ResultData == nil || resp.ResultData.DomainInfDataResp == nil {
		return errors.New("no domain infomration found")
	}

	eppObj := resp.ResultData.DomainInfDataResp
	if strings.EqualFold(d.DomainName, eppObj.Name) {
		return fmt.Errorf("%s does not match %s", strings.ToUpper(d.DomainName), strings.ToUpper(eppObj.Name))
	}

	domainFound := true
	d.DomainROID = eppObj.ROID

	d.OKStatus = false
	d.PendingCreateStatus = false
	d.PendingDeleteStatus = false
	d.PendingRenewStatus = false
	d.PendingTransferStatus = false
	d.PendingUpdateStatus = false
	d.ClientDeleteProhibitedStatus = false
	d.ClientHoldStatus = false
	d.ClientRenewProhibitedStatus = false
	d.ClientTransferProhibitedStatus = false
	d.ClientUpdateProhibitedStatus = false
	d.ServerDeleteProhibitedStatus = false
	d.ServerHoldStatus = false
	d.ServerRenewProhibitedStatus = false
	d.ServerTransferProhibitedStatus = false
	d.ServerUpdateProhibitedStatus = false

	domainStatuses := []string{}

	for _, status := range eppObj.Status {
		domainStatuses = append(domainStatuses, status.StatusFlag)

		switch status.StatusFlag {
		case "ok":
			d.OKStatus = true
		case epp.StatusPendingCreate:
			d.PendingCreateStatus = true
		case epp.StatusPendingDelete:
			d.PendingDeleteStatus = true
		case epp.StatusPendingRenew:
			d.PendingRenewStatus = true
		case epp.StatusPendingTransfer:
			d.PendingTransferStatus = true
		case epp.StatusPendingUpdate:
			d.PendingUpdateStatus = true
		case epp.StatusClientUpdateProhibited:
			d.ClientUpdateProhibitedStatus = true
		case epp.StatusServerUpdateProhibited:
			d.ServerUpdateProhibitedStatus = true
		case epp.StatusClientTransferProhibited:
			d.ClientTransferProhibitedStatus = true
		case epp.StatusServerTransferProhibited:
			d.ServerTransferProhibitedStatus = true
		case epp.StatusClientRenewProhibited:
			d.ClientRenewProhibitedStatus = true
		case epp.StatusServerRenewProhibited:
			d.ServerRenewProhibitedStatus = true
		case epp.StatusClientHold:
			d.ClientHoldStatus = true
		case epp.StatusServerHold:
			d.ServerHoldStatus = true
		case epp.StatusClientDeleteProhibited:
			d.ClientDeleteProhibitedStatus = true
		case epp.StatusServerDeleteProhibited:
			d.ServerDeleteProhibitedStatus = true
		default:
			logger.Errorf("Unknown State %s", status.StatusFlag)
		}
	}

	d.DomainStatus = strings.Join(domainStatuses, " ")
	d.DomainRegistrantROID = eppObj.RegistrantID

	for _, con := range eppObj.Contacts {
		switch con.Type {
		case "admin":
			d.DomainAdminContactROID = con.Value
		case "billing":
			d.DomainBillingContactROID = con.Value
		case "tech":
			d.DomainTechContactROID = con.Value
		}
	}

	hosts := []string{}

	for _, host := range eppObj.NSHosts.Hosts {
		hosts = append(hosts, host.Value)
	}

	d.DomainNSList = strings.Join(hosts, "\n")

	d.CreateClientID = eppObj.CreateID

	if tcr, crErr := time.Parse("2006-01-02T15:04:05Z", eppObj.CreateDate); crErr == nil {
		d.CreateDate = tcr
	}

	d.UpdateClientID = eppObj.UpdateID

	if tud, trErr := time.Parse("2006-01-02T15:04:05Z", eppObj.UpdateDate); trErr == nil {
		d.UpdateDate = tud
	}

	if tex, exErr := time.Parse("2006-01-02T15:04:05Z", eppObj.ExpireDate); exErr == nil {
		d.ExpireDate = tex
	}

	d.SponsoringClientID = eppObj.ClientID

	if !domainFound {
		return fmt.Errorf("Domain %s not found", d.DomainName)
	}

	d.DSDataEntries = []DSDataEntryEpp{}

	if resp.Extension != nil {
		if resp.Extension.SecDNSInfData != nil {
			for _, rawdsdata := range resp.Extension.SecDNSInfData.DSData {
				newds := DSDataEntryEpp{}
				newds.Algorithm = int64(rawdsdata.Alg)
				newds.DigestType = int64(rawdsdata.DigestType)
				newds.KeyTag = int64(rawdsdata.KeyTag)
				newds.Digest = rawdsdata.Digest
				d.DSDataEntries = append(d.DSDataEntries, newds)
			}
		}
	}

	implemented, eppStatusFlag := d.EPPMatchesExpected(&resp)

	initialFlag := d.EPPStatus

	d.EPPStatus = eppStatusFlag
	d.EPPLastUpdate = time.Now().Truncate(time.Second)

	if d.CurrentRevision.DesiredState == StateExternal && conf.Registrar.ID != resp.ResultData.DomainInfDataResp.ClientID {
		d.EPPStatus = "External"
		// The domain is now out of our control, we dont need to check it
		implemented = true
	}

	if initialFlag != d.EPPStatus && eppStatusFlag == EPPStatusProvisioned {
		msg := fmt.Sprintf("The domain changes you requested for %s have been fully provisioned", d.DomainName)

		err := conf.SendAllEmail(fmt.Sprintf("%s Provisioned", d.DomainName), msg, []string{conf.Email.FromEmail})
		if err != nil {
			logger.Errorf("error sending email: %s", err.Error())
		}
	}

	if implemented {
		d.CheckRequired = false
	}

	if err = dbCache.DB.Where("domain_id = ?", d.ID).Delete(DSDataEntryEpp{}).Error; err != nil {
		return err
	}

	return dbCache.Save(d)
}

// EPPMatchesExpected takes a response from an EPP server and will compare the
// domain object to the EPP response and return true iff the domain is
// configured correctly or false otherwise. If false is returned, the string
// passed back represents the current state of the domain.
func (d *Domain) EPPMatchesExpected(resp *epp.Response) (bool, string) {
	if resp == nil || resp.ResultData == nil || resp.ResultData.DomainInfDataResp == nil {
		return false, "Invalid EPP Domain Object"
	}

	domInf := resp.ResultData.DomainInfDataResp

	// Start with statuses
	PendingCreateStatus := false
	PendingDeleteStatus := false
	PendingRenewStatus := false
	PendingTransferStatus := false
	PendingUpdateStatus := false
	ClientDeleteProhibitedStatus := false
	ClientHoldStatus := false
	ClientRenewProhibitedStatus := false
	ClientTransferProhibitedStatus := false
	ClientUpdateProhibitedStatus := false
	ServerDeleteProhibitedStatus := false
	ServerHoldStatus := false
	ServerRenewProhibitedStatus := false
	ServerTransferProhibitedStatus := false
	ServerUpdateProhibitedStatus := false

	for _, status := range domInf.Status {
		switch status.StatusFlag {
		case epp.StatusPendingCreate:
			PendingCreateStatus = true
		case epp.StatusPendingDelete:
			PendingDeleteStatus = true
		case epp.StatusPendingRenew:
			PendingRenewStatus = true
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
		case epp.StatusClientRenewProhibited:
			ClientRenewProhibitedStatus = true
		case epp.StatusServerRenewProhibited:
			ServerRenewProhibitedStatus = true
		case epp.StatusClientHold:
			ClientHoldStatus = true
		case epp.StatusServerHold:
			ServerHoldStatus = true
		case epp.StatusClientDeleteProhibited:
			ClientDeleteProhibitedStatus = true
		case epp.StatusServerDeleteProhibited:
			ServerDeleteProhibitedStatus = true
		}
	}

	if PendingCreateStatus || PendingDeleteStatus || PendingRenewStatus || PendingTransferStatus || PendingUpdateStatus {
		return false, "PendingChangeState"
	}

	if !(ClientUpdateProhibitedStatus == d.CurrentRevision.ClientUpdateProhibitedStatus &&
		ClientTransferProhibitedStatus == d.CurrentRevision.ClientTransferProhibitedStatus &&
		ClientRenewProhibitedStatus == d.CurrentRevision.ClientRenewProhibitedStatus &&
		ClientHoldStatus == d.CurrentRevision.ClientHoldStatus &&
		ClientDeleteProhibitedStatus == d.CurrentRevision.ClientDeleteProhibitedStatus) {
		return false, EPPStatusClientFlagMismatch
	}

	// Check the hostnames
	eppHosts := []string{}
	revisionHosts := []string{}

	for _, host := range domInf.NSHosts.Hosts {
		eppHosts = append(eppHosts, host.Value)
	}

	for _, host := range d.CurrentRevision.Hostnames {
		revisionHosts = append(revisionHosts, host.HostName)
	}

	// Check hosts in both directions to make sure that there is not some fluke
	// where there is a duplicate
	if len(revisionHosts) != len(eppHosts) {
		return false, EPPStatusHostMismatch
	}

	for _, hos1 := range revisionHosts {
		found := false

		for _, hos2 := range eppHosts {
			if hos1 == hos2 {
				found = true

				break
			}
		}

		if !found {
			return false, fmt.Sprintf("Missing Host: %s", hos1)
		}
	}

	for _, hos1 := range eppHosts {
		found := false

		for _, hos2 := range revisionHosts {
			if hos1 == hos2 {
				found = true

				break
			}
		}

		if !found {
			return false, fmt.Sprintf("Additional Host: %s", hos1)
		}
	}

	// TODO Check dnssec

	addDS, remDS, dsDiffErr := DiffDomainDSData(resp, d.CurrentRevision.DSDataEntries)

	if dsDiffErr != nil {
		return false, fmt.Sprintf("Error diffing DS records: %s", dsDiffErr)
	}

	if len(addDS) != 0 {
		return false, EPPStatusMissingDSRecords
	}

	if len(remDS) != 0 {
		return false, EPPStatusAdditionalDSRecords
	}

	// This is done last to make sure that all other changes propagate first
	if !(ServerUpdateProhibitedStatus == d.CurrentRevision.ServerUpdateProhibitedStatus &&
		ServerTransferProhibitedStatus == d.CurrentRevision.ServerTransferProhibitedStatus &&
		ServerRenewProhibitedStatus == d.CurrentRevision.ServerRenewProhibitedStatus &&
		ServerHoldStatus == d.CurrentRevision.ServerHoldStatus &&
		ServerDeleteProhibitedStatus == d.CurrentRevision.ServerDeleteProhibitedStatus) {
		return false, EPPStatusServerFlagsMismatch
	}

	return true, EPPStatusProvisioned
}

// DiffDomainDSData will take the DS records in the EPP response from the
// registry and the list of DS records that the registrar expects to be there
// and will return a list of the DS records that need to be added or removed
// from the from the registry.
func DiffDomainDSData(registry *epp.Response, registrar []DSDataEntry) (addDSRecords, remDSRecords []epp.DSData, err error) {
	if registry == nil {
		err = errors.New("Domain info section of registry response is empty")

		return addDSRecords, remDSRecords, err
	}

	currentDSData := make(map[string]epp.DSData)
	expectedDSData := make(map[string]epp.DSData)

	if registry.Extension != nil && registry.Extension.SecDNSInfData != nil {
		for _, entry := range registry.Extension.SecDNSInfData.DSData {
			strRep := fmt.Sprintf("%d:%d:%d:%s", entry.KeyTag, entry.Alg, entry.DigestType, entry.Digest)
			currentDSData[strRep] = entry
		}
	}

	for _, entry := range registrar {
		strRep := fmt.Sprintf("%d:%d:%d:%s", entry.KeyTag, entry.Algorithm, entry.DigestType, entry.Digest)
		dsdata := epp.DSData{Alg: int(entry.Algorithm), Digest: entry.Digest, DigestType: int(entry.DigestType), KeyTag: int(entry.KeyTag)}
		expectedDSData[strRep] = dsdata
	}

	for current, currentVal := range currentDSData {
		found := false

		for expected := range expectedDSData {
			if current == expected {
				found = true
			}
		}

		if !found {
			remDSRecords = append(remDSRecords, currentVal)
		}
	}

	for expected, expectedVal := range expectedDSData {
		found := false

		for current := range currentDSData {
			if current == expected {
				found = true
			}
		}

		if !found {
			addDSRecords = append(addDSRecords, expectedVal)
		}
	}

	return addDSRecords, remDSRecords, err
}

// VerifyCR Checks to make sure that all of the values and approvals
// within a change request match the domain that it is linked to
//
// TODO: more rigirous check on if the CR approved text matches.
func (d *Domain) VerifyCR(dbCache *DBCache) (checksOut bool, errs []error) {
	return VerifyCR(dbCache, d, nil)
}

// GetCurrentRevisionID will return the id of the current Domain
// Revision for the domain object.
func (d *Domain) GetCurrentRevisionID() sql.NullInt64 {
	return d.CurrentRevisionID
}

// GetPendingRevisionID will return the current pending revision for the
// Domain object if it exists. If no pending revision exists a 0 is
// returned.
func (d *Domain) GetPendingRevisionID() int64 {
	return d.PendingRevision.ID
}

// GetPendingCRID will return the current CR id if it is set, otherwise
// a nil will be returned (in the form of a sql.NullInt64).
func (d *Domain) GetPendingCRID() sql.NullInt64 {
	return d.PendingRevision.CRID
}

// ComparePendingToCallback will return a function that will compare the
// current revision object to itself after changes have been made.
func (d *Domain) ComparePendingToCallback(loadFn CompareLoadFn) (retFn CompareReturnFn) {
	exp := DomainExport{}
	loadFn(&exp)

	return func() (pass bool, errs []error) {
		return exp.PendingRevision.Compare(d.PendingRevision)
	}
}

// UpdateState can be called at any point to check the state of the
// Domain and update it if necessary
//
// TODO: Implement
// TODO: Make sure callers check errors.
func (d *Domain) UpdateState(dbCache *DBCache, _ Config) (changesMade bool, errs []error) {
	logger.Infof("UpdateState called on %s %d (todo)", DomainType, d.ID)

	if err := d.Prepare(dbCache); err != nil {
		errs = append(errs, err)

		return changesMade, errs
	}

	changesMade = false

	switch d.State {
	case StateNew, StateInactive, StateActive, StateExternal, StateNewExternal:
		logger.Infof("UpdateState for Domain at state \"%s\", Nothing to do", d.State)
	case StatePendingBootstrap, StatePendingNew, StateActivePendingApproval, StateInactivePendingApproval, StateExternalPendingApproval, StatePendingNewExternal:
		if d.PendingRevision.ID != 0 && d.PendingRevision.CRID.Valid {
			crChecksOut, crcheckErrs := d.VerifyCR(dbCache)

			if len(crcheckErrs) != 0 {
				errs = append(errs, crcheckErrs...)
			}

			if crChecksOut {
				changeRequest := ChangeRequest{}

				if err := dbCache.FindByID(&changeRequest, d.PendingRevision.CRID.Int64); err != nil {
					errs = append(errs, err)

					return changesMade, errs
				}

				if changeRequest.State == StateApproved {
					// Promote the new revision
					targetState := d.PendingRevision.DesiredState

					if err := d.PendingRevision.Promote(dbCache); err != nil {
						errs = append(errs, fmt.Errorf("error promoting revision: %s", err.Error()))

						return changesMade, errs
					}

					if d.CurrentRevisionID.Valid {
						if err := d.CurrentRevision.Supersed(dbCache); err != nil {
							errs = append(errs, fmt.Errorf("error superseding revision: %s", err.Error()))

							return changesMade, errs
						}
					}

					newDomain := Domain{Model: Model{ID: d.ID}}
					if err := newDomain.PrepareShallow(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if targetState == StateActive || targetState == StateInactive || targetState == StateExternal {
						newDomain.State = targetState
					} else {
						errs = append(errs, ErrPendingRevisionInInvalidState)
						logger.Errorf("Unexpected target state: %s", targetState)

						return changesMade, errs
					}

					var revision sql.NullInt64
					revision.Int64 = d.PendingRevision.ID
					revision.Valid = true

					newDomain.CurrentRevisionID = revision
					newDomain.CheckRequired = true
					newDomain.EPPStatus = EPPStatusPendingChange
					newDomain.EPPLastUpdate = time.Now()
					newDomain.PreviewHostnames = d.PendingRevision.GetPreviewHostnames()

					if err := dbCache.Save(&newDomain); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					pendingRevision := DomainRevision{}

					if err := dbCache.FindByID(&pendingRevision, d.PendingRevision.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					logger.Infof("Pending Revision State = %s", pendingRevision.RevisionState)
				} else if changeRequest.State == StateDeclined {
					logger.Infof("CR %d has been declined", changeRequest.GetID())

					if err := d.PendingRevision.Decline(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					newDomain := Domain{}

					if err := dbCache.FindByID(&newDomain, d.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if d.CurrentRevisionID.Valid {
						curRev := DomainRevision{}

						if err := dbCache.FindByID(&curRev, d.CurrentRevisionID.Int64); err != nil {
							errs = append(errs, err)

							return changesMade, errs
						}

						if curRev.RevisionState == StateBootstrap {
							newDomain.State = StateActive
						} else {
							newDomain.State = curRev.RevisionState
						}
					} else {
						if d.State == StateExternalPendingApproval {
							newDomain.State = StateNewExternal
						} else {
							newDomain.State = StateNew
						}
					}

					if err := dbCache.Save(&newDomain); err != nil {
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
			if d.State == StatePendingBootstrap {
				d.State = StateBootstrap
			} else if d.CurrentRevisionID.Valid {
				d.State = d.CurrentRevision.DesiredState
			} else if d.State == StatePendingNewExternal {
				d.State = StateNewExternal
			} else {
				d.State = StateNew
			}
			changesMade = true
		}
	default:
		errs = append(errs, fmt.Errorf("updateState for %s at state \"%s\" not implemented", DomainType, d.State))
	}

	if changesMade {
		if err := dbCache.Save(d); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}
	}

	return changesMade, errs
}

// UpdateHoldStatus is used to adjust the object's Hold status after
// the user has been verified as an admin. An error will be returned
// if the hold reason is not set.
func (d *Domain) UpdateHoldStatus(holdActive bool, holdReason string, holdBy string) error {
	if !holdActive {
		d.HoldActive = false
		d.HoldAt = time.Unix(0, 0)
		d.HoldReason = ""
		d.HoldBy = ""
	} else {
		if len(holdReason) == 0 {
			return errors.New("a hold reason must be set")
		}
		d.HoldActive = true
		d.HoldAt = time.Now()
		d.HoldBy = holdBy
		d.HoldReason = holdReason
	}

	return nil
}

// RegisterableDomainSuffix will check to see if the domain is part of
// a zone that can be registered with registrar.
func RegisterableDomainSuffix(domainName string) string {
	longestSuffix := ""

	for _, suffix := range ValidSuffixList {
		if strings.HasSuffix(domainName, suffix) {
			if len(suffix) > len(longestSuffix) {
				longestSuffix = suffix
			}
		}
	}

	return longestSuffix
}

// IsRegisterableDomain will determine if the domain can be registered
// through the registrar system.
func IsRegisterableDomain(domainName string) (bool, error) {
	suffix := RegisterableDomainSuffix(domainName)

	if len(suffix) != 0 {
		subdomainEndIDX := strings.LastIndex(domainName, suffix)
		subdomain := domainName[0:subdomainEndIDX]

		if strings.Contains(subdomain, ".") {
			return false, errors.New("cannot register a subdomain")
		}

		return true, nil
	}

	return false, errors.New("no valid zone found")
}

// DomainExists will check if a domain name exists in the registrar
// system or not.
func DomainExists(domainName string, dbCache *DBCache) (bool, error) {
	var domainCount int64

	err := dbCache.DB.Model(Domain{}).Where("domain_name = ?", domainName).Count(&domainCount).Error

	if domainCount == 0 {
		return false, err
	}

	return true, err
}

// GetWorkDomains returns a list of IDs for domains that require
// attention in the form of an update to the registry or other related
// information. If an error occurs, it will be returned.
func GetWorkDomains(dbCache *DBCache) (retlist []int64, revisions []APIRevisionHint, err error) {
	var domains []Domain

	err = dbCache.DB.Where(map[string]interface{}{"check_required": true, "hold_active": false}).Find(&domains).Error
	if err != nil {
		return
	}

	for _, domain := range domains {
		retlist = append(retlist, domain.ID)

		if domain.CurrentRevisionID.Valid {
			rev := APIRevisionHint{}
			rev.ObjectID = domain.ID
			rev.RevisionID = domain.CurrentRevisionID.Int64
			rev.LastUpdate = domain.UpdatedAt
			revisions = append(revisions, rev)
		}
	}

	return
}

// GetWorkDomainsFull returns a list of Doamins that require work to be done
// or an error.
func GetWorkDomainsFull(dbCache *DBCache) (domains []Domain, err error) {
	err = dbCache.DB.Where(map[string]interface{}{"check_required": true}).Find(&domains).Error

	return
}

// GetWorkDomainsPrepared will try and find all domains that require work to be
// done and then will prepare all of the domain objects. If an error occurs in
// the process, the error will be returned.
func GetWorkDomainsPrepared(dbCache *DBCache) (domains []Domain, err error) {
	domains, err = GetWorkDomainsFull(dbCache)
	if err != nil {
		return
	}

	for idx := range domains {
		dom := domains[idx]
		err = dom.Prepare(dbCache)

		if err != nil {
			return domains, err
		}

		domains[idx] = dom
	}

	return
}

// GetServerLockChanges will attempt to generate a list of domains and server
// flags that need to be added or removed in order to complete changes. If an
// error occurs, it will be returned.
func GetServerLockChanges(dbCache *DBCache) (unlock map[string][]string, lock map[string][]string, err error) {
	logger.Info("Running Server Lock Check")

	var domains []Domain

	unlock = make(map[string][]string)
	lock = make(map[string][]string)
	domains, err = GetWorkDomainsPrepared(dbCache)

	if err != nil {
		return unlock, lock, err
	}

	for _, dom := range domains {
		var addFlags, removeFlags []string
		// Step 1: see if anything needs to be removed

		// If the client flags, hosts or DS records need to change, the server
		// change prohibited flag needs to be removed
		if dom.ServerDeleteProhibitedStatus && (dom.EPPStatus == EPPStatusClientFlagMismatch ||
			dom.EPPStatus == EPPStatusHostMismatch ||
			dom.EPPStatus == EPPStatusMissingDSRecords ||
			dom.EPPStatus == EPPStatusAdditionalDSRecords ||
			strings.HasPrefix(dom.EPPStatus, "Missing Host: ") ||
			strings.HasPrefix(dom.EPPStatus, "Additional Host: ") ||
			dom.EPPStatus == EPPStatusPendingChange) {
			removeFlags = append(removeFlags, epp.StatusServerUpdateProhibited)
		}

		// Remove server flags that are set on the server but have been removed from
		// the current revision
		if dom.ServerDeleteProhibitedStatus && !dom.CurrentRevision.ServerDeleteProhibitedStatus {
			removeFlags = append(removeFlags, epp.StatusServerDeleteProhibited)
		}

		if dom.ServerRenewProhibitedStatus && !dom.CurrentRevision.ServerRenewProhibitedStatus {
			removeFlags = append(removeFlags, epp.StatusServerRenewProhibited)
		}

		if dom.ServerTransferProhibitedStatus && !dom.CurrentRevision.ServerTransferProhibitedStatus {
			removeFlags = append(removeFlags, epp.StatusServerTransferProhibited)
		}

		if dom.ServerUpdateProhibitedStatus && !dom.CurrentRevision.ServerUpdateProhibitedStatus {
			removeFlags = append(removeFlags, epp.StatusServerUpdateProhibited)
		}

		// Step 2: See what needs to be added
		if dom.EPPStatus != EPPStatusPendingChange {
			if !dom.ServerDeleteProhibitedStatus && dom.CurrentRevision.ServerDeleteProhibitedStatus {
				addFlags = append(addFlags, epp.StatusServerDeleteProhibited)
			}

			if !dom.ServerRenewProhibitedStatus && dom.CurrentRevision.ServerRenewProhibitedStatus {
				addFlags = append(addFlags, epp.StatusServerRenewProhibited)
			}

			if !dom.ServerTransferProhibitedStatus && dom.CurrentRevision.ServerTransferProhibitedStatus {
				addFlags = append(addFlags, epp.StatusServerTransferProhibited)
			}

			if !dom.ServerUpdateProhibitedStatus && dom.CurrentRevision.ServerUpdateProhibitedStatus {
				addFlags = append(addFlags, epp.StatusServerUpdateProhibited)
			}
		}

		if len(addFlags) != 0 {
			lock[dom.DomainName] = addFlags
		}

		if len(removeFlags) != 0 {
			unlock[dom.DomainName] = removeFlags
		}
	}

	return unlock, lock, err
}

// FlagDomainsRequiringRenewal will review all domains in the system that are
// active and will flag all domains that require renewal. If an error occurs
// during the processing it will be returned.
func FlagDomainsRequiringRenewal(dbCache *DBCache) (err error) {
	// flag domains that are not inactive or external that have a expire date
	// within 1 year by setting the EPP Status to Pending Renew and flipping the
	// check_required bit
	logger.Info("Running Renewal Check")
	dbCache.DB.Model(Domain{}).Where("expire_date < ? and state != ? and state != ? and state != ?", time.Now().AddDate(1, 0, 0), StateInactive, StateExternal, StateNew).Updates(map[string]interface{}{"e_p_p_status": EPPStatusPendingRenew, "check_required": true})

	return dbCache.DB.Error
}

type whoisEmailDomainResult struct {
	ID                             int64
	DomainName                     string
	DomainROID                     string
	ClientDeleteProhibitedStatus   bool
	ServerDeleteProhibitedStatus   bool
	ClientHoldStatus               bool
	ServerHoldStatus               bool
	ClientRenewProhibitedStatus    bool
	ServerRenewProhibitedStatus    bool
	ClientTransferProhibitedStatus bool
	ServerTransferProhibitedStatus bool
	ClientUpdateProhibitedStatus   bool
	ServerUpdateProhibitedStatus   bool
	LinkedStatus                   bool
	OKStatus                       bool
	DomainNSList                   string
	CreateDate                     time.Time
	UpdateDate                     time.Time
	TransferDate                   time.Time
	ExpireDate                     time.Time
	CurrentRevisionID              int64
	DomainRegistrantID             int64
	DomainAdminContactID           int64
	DomainTechContactID            int64
	DomainBillingContactID         int64

	Statuses         []string
	NameServers      []string
	TransferHappened bool
	UpdateHappened   bool
}

func (wd *whoisEmailDomainResult) Prep() {
	epoch := time.Unix(0, 0)
	wd.TransferHappened = wd.TransferDate.After(epoch)
	wd.UpdateHappened = wd.UpdateDate.After(epoch)
	wd.NameServers = strings.Split(wd.DomainNSList, "\n")
}

type whoisEmailContactResult struct {
	ContactID           int64
	RevisionID          int64
	Name                string
	Org                 string
	AddressStreet1      string
	AddressStreet2      string
	AddressStreet3      string
	AddressCity         string
	AddressState        string
	AddressPostalCode   string
	AddressCountry      string
	VoicePhoneNumber    string
	VoicePhoneExtension string
	FaxPhoneNumber      string
	FaxPhoneExtension   string
	EmailAddress        string

	Street1Set bool
	Street2Set bool
	Street3Set bool
}

func (wc *whoisEmailContactResult) Prep() {
	wc.Street1Set = len(wc.AddressStreet1) > 0
	wc.Street2Set = len(wc.AddressStreet2) > 0
	wc.Street3Set = len(wc.AddressStreet3) > 0
}

type whoisEmailData struct {
	RegistrantID int64
	Registrant   whoisEmailContactResult
	Domains      []whoisEmailDomainResult
	Contacts     map[int64]whoisEmailContactResult
}

// WHOISConfirmEmail will collect the required information to generate the
// WHOIS confimation emails as required by ICANN and then send the email. Once
// the email has been sent, all of the domains that were processed will be
// marked as processed to prevent the email from being sent for another year.
func WHOISConfirmEmail(dbCache *DBCache, con Config) (err error) {
	domainsToContact := make(map[int64]whoisEmailData)
	contactList := make(map[int64]bool)
	contacts := make(map[int64]whoisEmailContactResult)

	// Get Domains that have not had a whois confirmation email in the last 330
	// days
	rows, err := dbCache.DB.Raw(`select
	domains.id,
	domains.domain_name,
	domains.client_delete_prohibited_status,
	domains.client_hold_status,
	domains.client_renew_prohibited_status,
	domains.client_transfer_prohibited_status,
	domains.client_update_prohibited_status,
	domains.server_delete_prohibited_status,
	domains.server_hold_status,
	domains.server_renew_prohibited_status,
	domains.server_transfer_prohibited_status,
	domains.server_update_prohibited_status,
	domains.linked_status,
	domains.o_k_status,
	domains.domain_r_o_id,
	domains.domain_n_s_list,
	domains.create_date,
	domains.update_date,
	domains.transfer_date,
	domains.expire_date,
	domains.current_revision_id,
	domain_revisions.domain_registrant_id,
	domain_revisions.domain_admin_contact_id,
	domain_revisions.domain_tech_contact_id,
	domain_revisions.domain_billing_contact_id
from domains, domain_revisions
where
	domains.current_revision_id = domain_revisions.id and
	(
			w_h_o_i_s_last_confirm_email_at < ? or
			w_h_o_i_s_last_confirm_email_at is null
	) and domains.state != 'external'
`, time.Now().Add(time.Hour*24*-330)).Rows()
	if err != nil {
		return fmt.Errorf("unable to query whois info for the provided domain: %w", err)
	}

	for rows.Next() {
		var who whoisEmailDomainResult

		err = rows.Scan(&who.ID, &who.DomainName, &who.ClientDeleteProhibitedStatus,
			&who.ClientHoldStatus, &who.ClientRenewProhibitedStatus,
			&who.ClientTransferProhibitedStatus,
			&who.ClientUpdateProhibitedStatus, &who.ServerDeleteProhibitedStatus,
			&who.ServerHoldStatus, &who.ServerRenewProhibitedStatus,
			&who.ServerTransferProhibitedStatus,
			&who.ServerUpdateProhibitedStatus, &who.LinkedStatus, &who.OKStatus,
			&who.DomainROID, &who.DomainNSList, &who.CreateDate, &who.UpdateDate,
			&who.TransferDate, &who.ExpireDate, &who.CurrentRevisionID,
			&who.DomainRegistrantID, &who.DomainAdminContactID,
			&who.DomainTechContactID, &who.DomainBillingContactID)

		if err != nil {
			return fmt.Errorf("error scanning data fields: %w", err)
		}

		who.Prep()

		if who.ClientDeleteProhibitedStatus {
			who.Statuses = append(who.Statuses, epp.StatusClientDeleteProhibited)
		}

		if who.ServerDeleteProhibitedStatus {
			who.Statuses = append(who.Statuses, epp.StatusServerDeleteProhibited)
		}

		if who.ClientHoldStatus {
			who.Statuses = append(who.Statuses, epp.StatusClientHold)
		}

		if who.ServerHoldStatus {
			who.Statuses = append(who.Statuses, epp.StatusServerHold)
		}

		if who.ClientRenewProhibitedStatus {
			who.Statuses = append(who.Statuses, epp.StatusClientRenewProhibited)
		}

		if who.ServerRenewProhibitedStatus {
			who.Statuses = append(who.Statuses, epp.StatusServerRenewProhibited)
		}

		if who.ClientTransferProhibitedStatus {
			who.Statuses = append(who.Statuses, epp.StatusClientTransferProhibited)
		}

		if who.ServerTransferProhibitedStatus {
			who.Statuses = append(who.Statuses, epp.StatusServerTransferProhibited)
		}

		if who.ClientUpdateProhibitedStatus {
			who.Statuses = append(who.Statuses, epp.StatusClientUpdateProhibited)
		}

		if who.ServerUpdateProhibitedStatus {
			who.Statuses = append(who.Statuses, epp.StatusServerUpdateProhibited)
		}

		if who.LinkedStatus {
			who.Statuses = append(who.Statuses, epp.StatusLinked)
		}

		if who.OKStatus {
			who.Statuses = append(who.Statuses, epp.StatusOK)
		}

		contactList[who.DomainRegistrantID] = true
		contactList[who.DomainBillingContactID] = true
		contactList[who.DomainAdminContactID] = true
		contactList[who.DomainTechContactID] = true

		var wed whoisEmailData
		if wedin, ok := domainsToContact[who.DomainRegistrantID]; ok {
			wed = wedin
		}

		wed.Domains = append(wed.Domains, who)
		wed.RegistrantID = who.DomainRegistrantID

		domainsToContact[who.DomainRegistrantID] = wed
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("error querying db: %w", err)
	}

	for contactID := range contactList {
		logger.Infof("get contact %d", contactID)

		rows, err = dbCache.DB.Raw(`select
		contacts.id,
		contact_revisions.id,
		contact_revisions.name,
		contact_revisions.org,
		contact_revisions.address_street1,
		contact_revisions.address_street2,
		contact_revisions.address_street3,
		contact_revisions.address_city,
		contact_revisions.address_state,
		contact_revisions.address_postal_code,
		contact_revisions.address_country,
		contact_revisions.voice_phone_number,
		contact_revisions.voice_phone_extension,
		contact_revisions.fax_phone_number,
		contact_revisions.fax_phone_extension,
		contact_revisions.email_address
	from contacts, contact_revisions
	where
		contacts.id = ?`,
			contactID).Rows()

		defer func() {
			err = rows.Close()
			if err != nil {
				logger.Error(err)
			}
		}()

		if err != nil {
			return fmt.Errorf("error closing db row: %w", err)
		}

		err = rows.Err()
		if err != nil {
			return fmt.Errorf("error querying database: %w", err)
		}

		for rows.Next() {
			var con whoisEmailContactResult

			err = rows.Scan(&con.ContactID, &con.RevisionID, &con.Name, &con.Org,
				&con.AddressStreet1, &con.AddressStreet2, &con.AddressStreet3,
				&con.AddressCity, &con.AddressState, &con.AddressPostalCode,
				&con.AddressCountry, &con.VoicePhoneNumber, &con.VoicePhoneExtension,
				&con.FaxPhoneNumber, &con.FaxPhoneExtension, &con.EmailAddress)

			if err != nil {
				return fmt.Errorf("unable to scan data fields: %w", err)
			}

			con.Prep()
			contacts[contactID] = con
		}
	}

	template, err := template.New("whoisemail").Parse(whoisEmailFormat)
	if err != nil {
		return fmt.Errorf("error parsing email template: %w", err)
	}

	for _, domainContact := range domainsToContact {
		domainContact.Contacts = contacts
		domainContact.Registrant = contacts[domainContact.RegistrantID]

		buf := bytes.Buffer{}
		writer := bufio.NewWriter(&buf)

		domToRender := domainContact

		err := template.Execute(writer, &domToRender)
		if err != nil {
			return fmt.Errorf("error executing teamplte: %w", err)
		}

		writer.Flush()

		message := buf.String()

		if len(domainContact.Registrant.EmailAddress) != 0 && strings.Contains(domainContact.Registrant.EmailAddress, "@") {
			logger.Infof("Sending WHOIS Data Confirmation to %s", domainContact.Registrant.EmailAddress)
			logger.Debug(message)

			err := con.SendAllEmail("WHOIS Data Confirmation", buf.String(), []string{domainContact.Registrant.EmailAddress, con.Email.CC})
			if err != nil {
				return err
			}

			for _, dom := range domainContact.Domains {
				logger.Infof("Mark %d as processed", dom.ID)

				err := dbCache.DB.Model(Domain{}).Where("id = ?", dom.ID).Updates(map[string]interface{}{"w_h_o_i_s_last_confirm_email_at": time.Now()}).Error
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GetAllDomains returns a list of IDs for domains that are in the
// active or activepending approval states or requires work to be done
// on the domain.
func GetAllDomains(dbCache *DBCache) (retlist []int64, revisions []APIRevisionHint, err error) {
	var domains []Domain

	err = dbCache.DB.Where("state = ? or state = ?", StateActive, StateActivePendingApproval).Find(&domains).Error

	if err != nil {
		return
	}

	for _, domain := range domains {
		retlist = append(retlist, domain.ID)

		if domain.CurrentRevisionID.Valid {
			rev := APIRevisionHint{}
			rev.ObjectID = domain.ID
			rev.RevisionID = domain.CurrentRevisionID.Int64
			rev.LastUpdate = domain.UpdatedAt
			revisions = append(revisions, rev)
		}
	}

	return
}

// GetDomainIDFromDomainName will attempt to find the domain ID for the domain
// name passed and return the domain ID. If no domain was found, then an error
// will be returned.
func GetDomainIDFromDomainName(dbCache *DBCache, domainName string) (domainID int64, err error) {
	domain := Domain{}

	err = dbCache.DB.Model(Domain{}).Where("domain_name = ?", domainName).First(&domain).Error
	if err != nil {
		return -1, err
	}

	if domain.ID == 0 {
		return -1, errors.New("Domain not found")
	}

	return domain.ID, err
}

// DSDataEntryEpp is an object that will hold a single DS Data entry
// used to indicate how a domain is signed with DNSSEC.
type DSDataEntryEpp struct {
	ID         int64 `gorm:"primary_key:yes"`
	DomainID   int64
	KeyTag     int64
	Algorithm  int64
	DigestType int64
	Digest     string `sql:"size:256"`
}

// DisplayName formates a DSData Entry to be displayed as part of a
// HTML form.
func (ds DSDataEntryEpp) DisplayName() string {
	return ds.FormValue()
}

// FormValue will format the DSData Entry so it can be used as the
// value for a html form item.
func (ds DSDataEntryEpp) FormValue() string {
	return fmt.Sprintf("%d:%d:%d:%s", ds.KeyTag, ds.Algorithm, ds.DigestType, ds.Digest)
}

// FormDivName creates a name that can be used as the ID for a div tag
// in the domain selection forms.
func (ds DSDataEntryEpp) FormDivName() string {
	return fmt.Sprintf("%d-%d-%d-%s", ds.KeyTag, ds.Algorithm, ds.DigestType, ds.Digest)
}

// MigrateDBDomain will run the automigrate function for the Domain
// object.
func MigrateDBDomain(dbCache *DBCache) {
	dbCache.AutoMigrate(&Domain{})
	dbCache.AutoMigrate(&DSDataEntryEpp{})
}

var whoisEmailFormat = `
Hello {{.Registrant.Name}},

ICANN RAA 2013 requires a yearly email, as well as an email at doamin
registration time, for all domains to confirm the WHOIS information
related to your domain(s). Our records indicate that you are the owner of the
domains below. If you see errors in the WHOIS information below please contact
the registrar to correct the information. The domain information will appear
first followed by the contact information.

Additional Notices:
	Also, as required by the RAA 2013, here are links and information that we are
	required to provide to you:

	Registrants' Benefits and Responsibilities:
			https://www.icann.org/resources/pages/benefits-2013-09-16-en?

	Auto-Renew Policy: All domains will be auto renewed unless otherwise requested

Domains:
{{range $dom := .Domains}}
 Domain Name: {{$dom.DomainName}}
	Domain ID: {{$dom.DomainROID}}
	{{if $dom.UpdateHappened}}Updated Date: {{$dom.UpdateDate}}
	{{end}}Creation Date: {{$dom.CreateDate}}
	{{if $dom.TransferHappened}}Transfer Date: {{$dom.TransferDate}}
	{{end}}Expire Date: {{$dom.ExpireDate}}{{ $stats := $dom.Statuses }}
	{{range $stat := $stats}}Domain Status: {{$stat}} https://icann.org/epp#{{$stat}}
	{{end}}{{ $nses := $dom.NameServers}}{{range $ns := $nses}}Name Server: {{$ns}}
	{{end}}Registrant ID: {{$dom.DomainRegistrantID}}
	Admin ID: {{$dom.DomainAdminContactID}}
	Tech ID: {{$dom.DomainTechContactID}}
	Billing ID: {{$dom.DomainBillingContactID}}
{{end}}

Contacts:
{{range $id, $con := .Contacts}}
 Contact ID: {{$id}}
	Name: {{$con.Name}}
	Organication: {{$con.Org}}
	{{if $con.Street1Set}}Street: {{$con.AddressStreet1}}
	{{end}}{{if $con.Street2Set}}Street: {{$con.AddressStreet2}}
	{{end}}{{if $con.Street3Set}}Street: {{$con.AddressStreet3}}
	{{end}}City: {{$con.AddressCity}}
	State / Province: {{$con.AddressState}}
	Postal Code: {{$con.AddressPostalCode}}
	Country: {{$con.AddressCountry}}
	Phone: {{$con.VoicePhoneNumber}}
	Phone Ext: {{$con.VoicePhoneExtension}}
	Fax: {{$con.FaxPhoneNumber}}
	Fax Ext: {{$con.FaxPhoneExtension}}
	Email: {{$con.EmailAddress}}
{{end}}
`
