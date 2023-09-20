package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aryann/difflib"
	"github.com/jinzhu/gorm"

	"github.com/timapril/go-registrar/epp"
)

// Contact is an object that represents the state of a registrar contact
// object as defined by RFC 5733
// http://tools.ietf.org/html/rfc5733
type Contact struct {
	Model
	State string

	ContactRegistryID string `sql:"size:32"`
	ContactROID       string `sql:"size:32"`

	ContactStatus string

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

	Name        string `sql:"size:128"`
	Org         string `sql:"size:128"`
	Description string `sql:"size:2048"`

	AddressStreet1    string `sql:"size:128"`
	AddressStreet2    string `sql:"size:128"`
	AddressStreet3    string `sql:"size:128"`
	AddressCity       string `sql:"size:64"`
	AddressState      string `sql:"size:32"`
	AddressPostalCode string `sql:"size:32"`
	AddressCountry    string `sql:"size:3"`

	VoicePhoneNumber    string `sql:"size:32"`
	VoicePhoneExtension string `sql:"size:32"`
	FaxPhoneNumber      string `sql:"size:32"`
	FaxPhoneExtension   string `sql:"size:32"`

	EmailAddress string `sql:"size:128"`

	PreviewName    string `sql:"size:2048"`
	PreviewAddress string `sql:"size:2048"`
	PreviewPhone   string `sql:"size:2048"`
	PreviewEmail   string `sql:"size:2048"`

	SponsoringClientID string `sql:"size:32"`
	CreateClientID     string `sql:"size:32"`
	CreateDate         time.Time
	UpdateClientID     string `sql:"size:32"`
	UpdateDate         time.Time
	TransferDate       time.Time

	HoldActive bool
	HoldBy     string
	HoldAt     time.Time
	HoldReason string

	CurrentRevision   ContactRevision
	CurrentRevisionID sql.NullInt64
	Revisions         []ContactRevision
	PendingRevision   ContactRevision `sql:"-"`

	DisplayName string `sql:"-"`

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

// ContactExport is an object that is used to export the current
// state of a contact object. The full version of the export object
// also contains the current and pending revision (if either exist).
type ContactExport struct {
	ID    int64  `json:"ID"`
	State string `json:"State"`

	ContactRegistryID string `json:"ContactRegistryID"`
	ContactROID       string `json:"ContactROID"`

	CurrentRevision ContactRevisionExport `json:"CurrentRevision"`
	PendingRevision ContactRevisionExport `json:"PendingRevision"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy"`

	HoldActive bool      `json:"HoldActive"`
	HoldBy     string    `json:"HoldBy"`
	HoldAt     time.Time `json:"HoldAt"`
	HoldReason string    `json:"HoldReason"`
}

// ContactExportShort is an object that is used to expor the current
// snapshot of an Contact object. The short version of the export
// object does not contain any information about the current or pending
// revisions.
type ContactExportShort struct {
	ID                int64  `json:"ID"`
	State             string `json:"State"`
	Name              string `json:"Name"`
	ContactRegistryID string `json:"ContactRegistryID"`
	ContactROID       string `json:"ContactROID"`

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`

	HoldActive bool      `json:"HoldActive"`
	HoldBy     string    `json:"HoldBy"`
	HoldAt     time.Time `json:"HoldAt"`
	HoldReason string    `json:"HoldReason"`
}

// GetDiff will return a string containing a formatted diff of the
// current and pending revisions for the Contact object. An empty
// string and an error are returned if an error occures during the
// processing
//
// TODO: Handle diff for objects that do not have a pending revision
// TODO: Handle diff for objects that do not have a current revision.
func (c ContactExport) GetDiff() (string, error) {
	current, _ := c.CurrentRevision.ToJSON()
	pending, err2 := c.PendingRevision.ToJSON()

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
func (c ContactExport) ToJSON() (string, error) {
	if c.ID <= 0 {
		return "", errors.New("ID not set")
	}

	byteArr, jsonErr := json.MarshalIndent(c, "", "  ")

	return string(byteArr), jsonErr
}

// GetRegistryID will return what the current registry ID for the
// contact is if it is set or if the iteration is 0, it will return
// the first choice of regitry id of the string GOREG- followed by the
// contact ID or in subsequent calls the iteration number will be
// appended to the end of the first option.
func (c ContactExport) GetRegistryID(iteration int64) string {
	if c.ContactRegistryID != "" {
		return c.ContactRegistryID
	}

	if iteration == 0 {
		return fmt.Sprintf("GOREG-%d", c.ID)
	}

	return fmt.Sprintf("GOREG-%d-%d", c.ID, iteration)
}

// ContactPage is used to hold all the information required to render
// the Contact HTML page.
type ContactPage struct {
	Editable            bool
	IsNew               bool
	Con                 Contact
	CurrentRevisionPage *ContactRevisionPage
	PendingRevisionPage *ContactRevisionPage
	PendingActions      map[string]string
	ValidApproverSets   map[int64]string

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (c *ContactPage) GetCSRFToken() string {
	return c.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (c *ContactPage) SetCSRFToken(newToken string) {
	c.CSRFToken = newToken
	c.PendingRevisionPage.CSRFToken = newToken
}

// ContactsPage is used to hold all the information required to render
// the Contact HTML page.
type ContactsPage struct {
	Contacts []Contact

	CSRFToken string
}

// GetCSRFToken retrieves the CSRF token from an the Page Object.
func (c *ContactsPage) GetCSRFToken() string {
	return c.CSRFToken
}

// SetCSRFToken is used to set the CSRFToken for the Page Object.
func (c *ContactsPage) SetCSRFToken(newToken string) {
	c.CSRFToken = newToken
}

// ContactFieldName is a name that can be used to reference the
// name field of the current contact revision.
const ContactFieldName string = "Name"

// ContactFieldOrg is the name that is used to reference the org field
// of the current contact revision.
const ContactFieldOrg string = "Org"

// ContactFieldEmail is the name that is used to reference the email
// field of the current contact revision.
const ContactFieldEmail string = "Email"

// ContactFieldFullAddress is the name that is used to reference the
// full address of the current contact revision (made up of other
// address fields).
const ContactFieldFullAddress string = "FullAddress"

// ContactFieldAddressStreet1 is the name that is used to reference the
// address street 1 field of the current contact revision.
const ContactFieldAddressStreet1 string = "Street1"

// ContactFieldAddressStreet2 is the name that is used to reference the
// address street 2 field of the current contact revision.
const ContactFieldAddressStreet2 string = "Street2"

// ContactFieldAddressStreet3 is the name that is used to reference the
// address street 3 field of the current contact revision.
const ContactFieldAddressStreet3 string = "Street3"

// ContactFieldAddressCity is the name that is used to reference the
// address city field of the current contact revision.
const ContactFieldAddressCity string = "City"

// ContactFieldAddressState is the name that is used to reference the
// address state field of the current contact revision.
const ContactFieldAddressState string = "State"

// ContactFieldAddressPostalCode is the name that is used to reference
// the address postal code field of the current contact revision.
const ContactFieldAddressPostalCode string = "PostalCode"

// ContactFieldAddressCountry is the name that is used to reference the
// address country field of the current contact revision.
const ContactFieldAddressCountry string = "Country"

// ContactFieldVoiceNumber is the name that is used to reference the
// voice number field of the current contact revision.
const ContactFieldVoiceNumber string = "VoiceNumber"

// ContactFieldFaxNumber is the name that is used to reference the
// fax number field of the current contact revision.
const ContactFieldFaxNumber string = "FaxNumber"

// ContactFieldVoiceExtension is the name that is used to reference the
// voice phone number extension.
const ContactFieldVoiceExtension string = "VoiceExt"

// ContactFieldFaxExtension is the name that is used to reference the
// fax phone number extension.
const ContactFieldFaxExtension string = "FaxExt"

// GetExportVersion returns a export version of the Contact Object.
func (c *Contact) GetExportVersion() RegistrarObjectExport {
	export := ContactExport{
		ID:                c.ID,
		State:             c.State,
		PendingRevision:   (c.PendingRevision.GetExportVersion()).(ContactRevisionExport),
		CurrentRevision:   (c.CurrentRevision.GetExportVersion()).(ContactRevisionExport),
		ContactRegistryID: c.ContactRegistryID,
		ContactROID:       c.ContactROID,
		UpdatedAt:         c.UpdatedAt,
		UpdatedBy:         c.UpdatedBy,
		CreatedAt:         c.CreatedAt,
		CreatedBy:         c.CreatedBy,
		HoldActive:        c.HoldActive,
		HoldAt:            c.HoldAt,
		HoldBy:            c.HoldBy,
		HoldReason:        c.HoldReason,
	}

	return export
}

// GetExportVersionAt returns an export version of the Contact Object
// at the timestamp provided if possible otherwise an error is returned.
// If a pending version existed at the time it will be excluded from
// the object.
func (c *Contact) GetExportVersionAt(dbCache *DBCache, timestamp int64) (obj RegistrarObjectExport, err error) {
	changeRequest := ContactRevision{}

	// Grab the most recent promoted object before the time provided
	// where the promoted time is after the time stated above
	// If no objects were found meeting the criteria, return an error
	if err = dbCache.GetRevisionAtTime(&changeRequest, c.ID, timestamp); err != nil {
		return
	}

	// Otherwise, prepare the new object, fix the currnet object a little
	// and remove the pending revision if any
	if err = changeRequest.Prepare(dbCache); err != nil {
		return
	}

	c.CurrentRevision = changeRequest
	c.CurrentRevisionID.Int64 = changeRequest.ID
	c.CurrentRevisionID.Valid = true
	c.PendingRevision = ContactRevision{}

	// Return the export version and no error
	return c.GetExportVersion(), nil
}

// GetExportVersionShort returns a short export version of the Contact
// Object.
func (c *Contact) GetExportVersionShort() ContactExportShort {
	export := ContactExportShort{
		ID:                c.ID,
		State:             c.State,
		Name:              c.Name,
		ContactRegistryID: c.ContactRegistryID,
		ContactROID:       c.ContactROID,
		CreatedAt:         c.CreatedAt,
		CreatedBy:         c.CreatedBy,
		HoldActive:        c.HoldActive,
		HoldAt:            c.HoldAt,
		HoldBy:            c.HoldBy,
		HoldReason:        c.HoldReason,
	}

	return export
}

// UnPreparedContactError is the text of an error that is displayed
// when a contact has not been prepared before use.
const UnPreparedContactError = "Error: Contact Not Prepared"

// GetRegistryExtension is used to get the extension of the number that
// is present at the registry. If an extension exists, an "x " is
// prepended to the string to denote an extension.
func (c *Contact) GetRegistryExtension(field string) string {
	ext := ""

	switch field {
	case ContactFieldFaxExtension:
		ext = c.FaxPhoneExtension
	case ContactFieldVoiceExtension:
		ext = c.VoicePhoneExtension
	}

	if len(ext) != 0 {
		return fmt.Sprintf("x %s", ext)
	}

	return ""
}

// GetCurrentExtension is used to get the extension of the number in
// the current revision. If an extension exists, an "x " is prepended
// to the string to denote an extension.
func (c *Contact) GetCurrentExtension(field string) string {
	if !c.prepared {
		return UnPreparedContactError
	}

	ext := c.SuggestedRevisionValue(field)

	if len(ext) != 0 {
		return fmt.Sprintf("x %s", ext)
	}

	return ""
}

// GetDisplayName will return a name for the Contact that can be used to
// display a shortened version of the invormation to users.
func (c *Contact) GetDisplayName() string {
	return fmt.Sprintf("%d - %s", c.ID, c.GetCurrentValue(ContactFieldName))
}

// GetCurrentValue is used to get the current value of a field in a
// revision if a current revision exists, otherwise an empty string is
// returned.
func (c *Contact) GetCurrentValue(field string) (ret string) {
	if !c.prepared {
		return UnPreparedContactError
	}

	return c.SuggestedRevisionValue(field)
}

// SuggestedRevisionBool takes a string naming the flag that is being
// requested and returnes a bool containing the suggested value for the
// field in the new revision
//
// TODO: add other fields that have been added.
func (c Contact) SuggestedRevisionBool(field string) bool {
	if c.CurrentRevisionID.Valid {
		switch field {
		case ClientDeleteFlag:
			return c.CurrentRevision.ClientDeleteProhibitedStatus
		case ServerDeleteFlag:
			return c.CurrentRevision.ServerDeleteProhibitedStatus
		case ClientTransferFlag:
			return c.CurrentRevision.ClientTransferProhibitedStatus
		case ServerTransferFlag:
			return c.CurrentRevision.ServerTransferProhibitedStatus
		case ClientUpdateFlag:
			return c.CurrentRevision.ClientUpdateProhibitedStatus
		case ServerUpdateFlag:
			return c.CurrentRevision.ServerUpdateProhibitedStatus
		case DesiredStateActive:
			return c.CurrentRevision.DesiredState == StateActive
		case DesiredStateInactive:
			return c.CurrentRevision.DesiredState == StateInactive
		}
	}

	return false
}

// SuggestedRevisionValue takes a string naming the field that is being
// requested and returns a string containing the suggested value for
// the field in a new pending revision
//
// TODO: add other fields that have been added.
func (c Contact) SuggestedRevisionValue(field string) string {
	if c.CurrentRevisionID.Valid {
		switch field {
		case ContactFieldName:
			return c.CurrentRevision.Name
		case ContactFieldOrg:
			return c.CurrentRevision.Org
		case ContactFieldEmail:
			return c.CurrentRevision.EmailAddress
		case ContactFieldAddressStreet1:
			return c.CurrentRevision.AddressStreet1
		case ContactFieldAddressStreet2:
			return c.CurrentRevision.AddressStreet2
		case ContactFieldAddressStreet3:
			return c.CurrentRevision.AddressStreet3
		case ContactFieldAddressCity:
			return c.CurrentRevision.AddressCity
		case ContactFieldAddressState:
			return c.CurrentRevision.AddressState
		case ContactFieldAddressPostalCode:
			return c.CurrentRevision.AddressPostalCode
		case ContactFieldAddressCountry:
			return c.CurrentRevision.AddressCountry
		case ContactFieldVoiceNumber:
			return c.CurrentRevision.VoicePhoneNumber
		case ContactFieldVoiceExtension:
			return c.CurrentRevision.VoicePhoneExtension
		case ContactFieldFaxNumber:
			return c.CurrentRevision.FaxPhoneNumber
		case ContactFieldFaxExtension:
			return c.CurrentRevision.FaxPhoneExtension
		case ContactFieldFullAddress:
			return c.CurrentRevision.FullAddress()
		case SavedObjectNote:
			return c.CurrentRevision.SavedNotes
		}
	}

	return ""
}

// HasRevision returns true iff a current revision exists, otherwise
// false
//
// TODO: add a check to verify that the current revision has an approved
// change request.
func (c Contact) HasRevision() bool {
	return c.CurrentRevisionID.Valid
}

// HasPendingRevision returns true iff a pending revision exists for the
// Contact, otherwise false.
func (c Contact) HasPendingRevision() bool {
	return c.PendingRevision.ID != 0
}

// Prepare populate all of the fields for a given object as well as the
// linked objects.
func (c *Contact) Prepare(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, c, func() (err error) {
		// If there is a current revision, load the revision into the
		// CurrentRevision field
		if c.CurrentRevisionID.Valid {
			c.CurrentRevision = ContactRevision{}

			if err = dbCache.FindByID(&c.CurrentRevision, c.CurrentRevisionID.Int64); err != nil {
				return
			}
		}

		// Grab the pending revison if it exists and prepare the revision
		if err = dbCache.GetNewAndPendingRevisions(c); err != nil {
			if errors.Is(err, gorm.RecordNotFound) {
				return nil
			}

			return
		}
		if err = c.PendingRevision.Prepare(dbCache); err != nil {
			return
		}
		c.DisplayName = c.GetDisplayName()

		return nil
	})
}

// PrepareShallow populate all of the fields for a given object and
// not any of the linked object.
func (c *Contact) PrepareShallow(dbCache *DBCache) error {
	return PrepareBase(dbCache, c, FuncNOPErrFunc)
}

// GetPendingRevision implements the RegistrarParent interface and returns
// the pending revision pointer.
func (c *Contact) GetPendingRevision() RegistrarObject {
	return &c.PendingRevision
}

// PrepareDisplayShallow populate all of the fields for a given object
// and the current revision but not any of the other linked object.
func (c *Contact) PrepareDisplayShallow(dbCache *DBCache) (err error) {
	return PrepareBase(dbCache, c, func() (err error) {
		if c.CurrentRevisionID.Valid {
			c.CurrentRevision.ID = c.CurrentRevisionID.Int64

			if err = dbCache.Find(&c.CurrentRevision); err != nil {
				return
			}

			if err = c.CurrentRevision.PrepareShallow(dbCache); err != nil {
				return
			}
		}

		c.prepared = true
		c.DisplayName = c.GetDisplayName()

		return
	})
}

// GetAllPage will return an object that can be used to render a view
// Containing multiple contacts.
//
// TODO: Add paging support
// TODO: Add filtering.
func (c *Contact) GetAllPage(dbCache *DBCache, _ string, _ string) (rop RegistrarObjectPage, err error) {
	ret := &ContactsPage{}

	err = dbCache.FindAll(&ret.Contacts)
	if err != nil {
		return
	}

	return ret, nil
}

// IsCancelled returns true iff the object has been canclled.
func (c *Contact) IsCancelled() bool {
	return c.State == StateCancelled
}

// IsEditable returns true iff the object is editable.
func (c *Contact) IsEditable() bool {
	return c.State == StateNew
}

// GetPage will return an object that can be used to render the HTML
// template for the Contact.
func (c *Contact) GetPage(dbCache *DBCache, username string, email string) (rop RegistrarObjectPage, err error) {
	ret := &ContactPage{Editable: true, IsNew: true}

	if c.ID != 0 {
		ret.Editable = false
		ret.IsNew = false
	}

	ret.Con = *c
	ret.PendingActions = make(map[string]string)
	ret.ValidApproverSets, err = GetValidApproverSetMap(dbCache)

	if err != nil {
		return rop, err
	}

	if c.PendingRevision.ID != 0 {
		ret.PendingActions = c.PendingRevision.GetActions(false)
	}

	if c.CurrentRevisionID.Valid {
		rawPage, rawPageErr := c.CurrentRevision.GetPage(dbCache, username, email)

		if rawPageErr != nil {
			err = rawPageErr

			return rop, err
		}

		ret.CurrentRevisionPage = rawPage.(*ContactRevisionPage)
	}

	if c.HasPendingRevision() {
		rawPage, rawPageErr := c.PendingRevision.GetPage(dbCache, username, email)

		if rawPageErr != nil {
			err = rawPageErr

			return rop, err
		}

		ret.PendingRevisionPage = rawPage.(*ContactRevisionPage)
	} else {
		ret.PendingRevisionPage = &ContactRevisionPage{IsEditable: true, IsNew: true, ValidApproverSets: ret.ValidApproverSets}
	}

	ret.PendingRevisionPage.ParentContact = c
	ret.PendingRevisionPage.SuggestedRequiredApprovers = make(map[int64]ApproverSetDisplayObject)
	ret.PendingRevisionPage.SuggestedInformedApprovers = make(map[int64]ApproverSetDisplayObject)

	if c.CurrentRevision.ID != 0 {
		for _, appSet := range c.CurrentRevision.RequiredApproverSets {
			ret.PendingRevisionPage.SuggestedRequiredApprovers[appSet.ID] = appSet.GetDisplayObject()
		}

		for _, appSet := range c.CurrentRevision.InformedApproverSets {
			ret.PendingRevisionPage.SuggestedInformedApprovers[appSet.ID] = appSet.GetDisplayObject()
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

// GetType will return the object type string as defined in the
// RegistrarObject definition.
func (c Contact) GetType() string {
	return ContactType
}

// ParseFromForm takes a http Request and parses the field values and
// populates the acceptable values into the new object. An error is
// returned if there is difficulty parsing any of the fileds.
func (c *Contact) ParseFromForm(request *http.Request, _ *DBCache) error {
	runame, err := GetRemoteUser(request)
	if err != nil {
		return err
	}

	c.ContactRegistryID = request.FormValue("contact_registry_id")

	c.State = StateNew
	c.CreatedBy = runame
	c.UpdatedBy = runame

	return nil
}

// ParseFromFormUpdate takes a http Request and parses the form values
// and returns a sparse Contact object with the changes that were
// made. An error object is always the second return value which is nil
// when no errors have occurred during parsing otherwise an error is
// returned.
func (c *Contact) ParseFromFormUpdate(request *http.Request, _ *DBCache, _ Config) error {
	if c.ID > 0 {
		runame, err := GetRemoteUser(request)
		if err != nil {
			return err
		}

		if c.State == StateNew {
			c.UpdatedBy = runame
			c.ContactRegistryID = request.FormValue("contact_registry_id")

			return nil
		}

		return fmt.Errorf("cannot update contact in %s state", c.State)
	}

	return errors.New("to update a contact the ID must be greater than 0")
}

// GetRequiredApproverSets returns the list of approver sets that are
// required for the Contact (if a valid approver revision exists). If
// no approver revisions are found, a default of the infosec approver
// set will be returned.
func (c *Contact) GetRequiredApproverSets(dbCache *DBCache) (approvers []ApproverSet, err error) {
	return GetRequiredApproverSets(dbCache, c)
}

// GetInformedApproverSets returns the list of approver sets that are
// informed for the Contact (if a valid approver revision exists). If
// no approver revisions are found, an empty list will be returned.
func (c *Contact) GetInformedApproverSets(dbCache *DBCache) (as []ApproverSet, err error) {
	return GetInformedApproverSets(dbCache, c)
}

// TakeAction processes actions that are to be taken on the object and
// either display a resulting page, trigger a download or redirect to
// another page if necessary.
func (c *Contact) TakeAction(response http.ResponseWriter, request *http.Request, dbCache *DBCache, actionName string, _ bool, authMethod AuthType, _ Config) (errs []error) {
	switch actionName {
	case ActionUpdateEPPInfo:
		if authMethod == CertAuthType {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				errs = append(errs, err)

				return errs
			}

			if err := c.LoadEPPInfo(body, dbCache); err != nil {
				logger.Error(err)
				errs = append(errs, err)
			}

			return errs
		}
	case ActionUpdateEPPCheckRequired:
		if authMethod == CertAuthType {
			c.CheckRequired = false

			err := dbCache.Save(c)
			if err != nil {
				errs = append(errs, err)

				return errs
			}
		}
	case ActionGet:
		if authMethod == CertAuthType {
			APIRespond(response, GenerateObjectResponse(c.GetExportVersion()))

			return errs
		}
	case ActionUpdatePreview:
		newContact := Contact{Model: Model{ID: c.ID}}

		if err := newContact.PrepareShallow(dbCache); err != nil {
			errs = append(errs, err)

			return errs
		}

		newContact.PreviewName = c.CurrentRevision.GetPreviewName()
		newContact.PreviewAddress = c.CurrentRevision.GetPreviewAddress()
		newContact.PreviewPhone = c.CurrentRevision.GetPreviewPhone()
		newContact.PreviewEmail = c.CurrentRevision.GetPreviewEmail()

		err := dbCache.Save(&newContact)
		if err != nil {
			errs = append(errs, err)

			return errs
		}

		return errs
	case "setContactID":
		if authMethod == CertAuthType {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				errs = append(errs, err)

				return errs
			}

			err = c.SetRegistryID(body, dbCache)

			if err != nil {
				errs = append(errs, err)

				return errs
			}
		}
	}

	return errs
}

// SetRegistryID is used to try and set the RegistryID of the contact
// object if it has not already been set. An error is returned if the
// contact's RegistryID has been set already.
func (c *Contact) SetRegistryID(data []byte, dbCache *DBCache) error {
	if c.ContactRegistryID != "" {
		return errors.New("unable to set registry ID that has already been set")
	}

	var registryID string

	if err := json.Unmarshal(data, &registryID); err != nil {
		return fmt.Errorf("unable to unmarsal registryID: %w", err)
	}

	c.ContactRegistryID = strings.ToUpper(registryID)

	return dbCache.Save(c)
}

const (
	addressLine1 = 1
	addressLine2 = 2
	addressLine3 = 3
)

// LoadEPPInfo accepts an EPP response and attempts to marshall the data
// from the response into the object that was called.
func (c *Contact) LoadEPPInfo(data []byte, dbCache *DBCache) error {
	var resp epp.Response

	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("unable to unmarshal epp response: %w", err)
	}

	contactFound := false

	if resp.ResultData.GenericInfDataResp != nil {
		eppObj := resp.ResultData.GenericInfDataResp

		if c.ContactRegistryID != eppObj.ID {
			return fmt.Errorf("registry ID %s does not match %s", c.ContactRegistryID, eppObj.ID)
		}

		var contactStatus []string

		contactFound = true
		c.ContactROID = eppObj.ROID

		for _, status := range eppObj.Status {
			contactStatus = append(contactStatus, status.Value)

			switch status.Value {
			// TODO Switch the cases to constants
			case "OK":
				c.OKStatus = true
			case epp.StatusPendingCreate:
				c.PendingCreateStatus = true
			case epp.StatusPendingDelete:
				c.PendingDeleteStatus = true
			case epp.StatusPendingTransfer:
				c.PendingTransferStatus = true
			case epp.StatusPendingUpdate:
				c.PendingUpdateStatus = true
			case epp.StatusClientUpdateProhibited:
				c.ClientUpdateProhibitedStatus = true
			case epp.StatusServerUpdateProhibited:
				c.ServerUpdateProhibitedStatus = true
			case epp.StatusClientTransferProhibited:
				c.ClientTransferProhibitedStatus = true
			case epp.StatusServerTransferProhibited:
				c.ServerTransferProhibitedStatus = true
			case epp.StatusClientDeleteProhibited:
				c.ClientDeleteProhibitedStatus = true
			case epp.StatusServerDeleteProhibited:
				c.ServerDeleteProhibitedStatus = true
			case epp.StatusLinked:
				c.LinkedStatus = true
			default:
				logger.Errorf("Unknown Status: %s", status.Value)
			}
		}

		c.ContactStatus = strings.Join(contactStatus, " ")
		c.VoicePhoneNumber = eppObj.Voice.Number
		c.VoicePhoneExtension = eppObj.Voice.Extension
		c.FaxPhoneNumber = eppObj.Fax.Number
		c.FaxPhoneExtension = eppObj.Fax.Extension

		c.EmailAddress = eppObj.Email

		for _, con := range eppObj.PostalInfos {
			if con.Type == "loc" {
				c.Name = con.Name
				c.Org = con.Org
				c.AddressCity = con.Addr.City
				c.AddressState = con.Addr.StateProv
				c.AddressCountry = con.Addr.Country
				c.AddressPostalCode = con.Addr.PostalCode

				if len(con.Addr.Streets) >= addressLine1 {
					c.AddressStreet1 = con.Addr.Streets[0]
				}

				if len(con.Addr.Streets) >= addressLine2 {
					c.AddressStreet2 = con.Addr.Streets[1]
				}

				if len(con.Addr.Streets) >= addressLine3 {
					c.AddressStreet3 = con.Addr.Streets[2]
				}
			}
		}

		c.CreateClientID = eppObj.CreateID
		if tcr, crErr := time.Parse("2006-01-02T15:04:05.000Z", eppObj.CreateDate); crErr == nil {
			c.CreateDate = tcr
		}

		c.UpdateClientID = eppObj.UpdateID
		if tud, trErr := time.Parse("2006-01-02T15:04:05.000Z", eppObj.UpdateDate); trErr == nil {
			c.UpdateDate = tud
		}

		c.SponsoringClientID = eppObj.ClientID
	}

	if !contactFound {
		return fmt.Errorf("Contact %s not found", c.ContactRegistryID)
	}

	return dbCache.Save(c)
}

// VerifyCR Checks to make sure that all of the values and approvals
// within a change request match the approver that it is linked to
//
// TODO: more rigirous check on if the CR approved text matches.
func (c *Contact) VerifyCR(dbCache *DBCache) (checksOut bool, errs []error) {
	return VerifyCR(dbCache, c, nil)
}

// GetCurrentRevisionID will return the id of the current Contact
// Revision for the contact object.
func (c *Contact) GetCurrentRevisionID() sql.NullInt64 {
	return c.CurrentRevisionID
}

// GetPendingRevisionID will return the current pending revision for the
// Contact object if it exists. If no pending revision exists a 0 is
// returned.
func (c *Contact) GetPendingRevisionID() int64 {
	return c.PendingRevision.ID
}

// GetPendingCRID will return the current CR id if it is set, otherwise
// a nil will be returned (in the form of a sql.NullInt64).
func (c *Contact) GetPendingCRID() sql.NullInt64 {
	return c.PendingRevision.CRID
}

// ComparePendingToCallback will return a function that will compare the
// current revision object to itself after changes have been made.
func (c *Contact) ComparePendingToCallback(loadFn CompareLoadFn) (retFn CompareReturnFn) {
	exp := ContactExport{}
	loadFn(&exp)

	return func() (pass bool, errs []error) {
		return exp.PendingRevision.Compare(c.PendingRevision)
	}
}

// UpdateState can be called at any point to check the state of the
// Contact and update it if necessary
//
// TODO: Implement
// TODO: Make sure callers check errors.
func (c *Contact) UpdateState(dbCache *DBCache, _ Config) (changesMade bool, errs []error) {
	logger.Infof("UpdateState called on Contact %d (todo)", c.ID)

	changesMade = false

	if err := c.Prepare(dbCache); err != nil {
		errs = append(errs, err)

		return changesMade, errs
	}

	switch c.State {
	case StateNew, StateInactive, StateActive:
		logger.Infof("UpdateState for Contact at state \"%s\", Nothing to do", c.State)
	case StatePendingBootstrap, StatePendingNew, StateActivePendingApproval, StateInactivePendingApproval:
		if c.PendingRevision.ID != 0 && c.PendingRevision.CRID.Valid {
			crChecksOut, crcheckErrs := c.VerifyCR(dbCache)

			if len(crcheckErrs) != 0 {
				errs = append(errs, crcheckErrs...)
			}

			if crChecksOut {
				changeRequest := ChangeRequest{}

				if err := dbCache.FindByID(&changeRequest, c.PendingRevision.CRID.Int64); err != nil {
					errs = append(errs, err)

					return changesMade, errs
				}

				if changeRequest.State == StateApproved {
					// Promote the new revision
					targetState := c.PendingRevision.DesiredState

					if err := c.PendingRevision.Promote(dbCache); err != nil {
						errs = append(errs, fmt.Errorf("error promoting revision: %s", err.Error()))

						return changesMade, errs
					}

					if c.CurrentRevisionID.Valid {
						if err := c.CurrentRevision.Supersed(dbCache); err != nil {
							errs = append(errs, fmt.Errorf("error superseding revision: %s", err.Error()))

							return changesMade, errs
						}
					}

					newContact := Contact{Model: Model{ID: c.ID}}

					if err := newContact.PrepareShallow(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if targetState == StateActive || targetState == StateInactive {
						newContact.State = targetState
					} else {
						errs = append(errs, ErrPendingRevisionInInvalidState)
						logger.Errorf("Unexpected target state: %s", targetState)

						return changesMade, errs
					}

					var revision sql.NullInt64
					revision.Int64 = c.PendingRevision.ID
					revision.Valid = true

					newContact.CheckRequired = true
					newContact.CurrentRevisionID = revision
					newContact.PreviewName = c.PendingRevision.GetPreviewName()
					newContact.PreviewAddress = c.PendingRevision.GetPreviewAddress()
					newContact.PreviewPhone = c.PendingRevision.GetPreviewPhone()
					newContact.PreviewEmail = c.PendingRevision.GetPreviewEmail()

					if err := dbCache.Save(&newContact); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					pendingRevision := ContactRevision{}

					if err := dbCache.FindByID(&pendingRevision, c.PendingRevision.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					logger.Infof("Pending Revision State = %s", pendingRevision.RevisionState)
				} else if changeRequest.State == StateDeclined {
					logger.Infof("CR %d has been declined", changeRequest.GetID())

					if err := c.PendingRevision.Decline(dbCache); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					newContact := Contact{}
					if err := dbCache.FindByID(&newContact, c.ID); err != nil {
						errs = append(errs, err)

						return changesMade, errs
					}

					if c.CurrentRevisionID.Valid {
						curRev := ContactRevision{}

						if err := dbCache.FindByID(&curRev, c.CurrentRevisionID.Int64); err != nil {
							errs = append(errs, err)

							return changesMade, errs
						}

						if curRev.RevisionState == StateBootstrap {
							newContact.State = StateActive
						} else {
							newContact.State = curRev.RevisionState
						}
					} else {
						newContact.State = StateNew
					}
					if err := dbCache.Save(&newContact); err != nil {
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
			if c.State == StatePendingBootstrap {
				c.State = StateBootstrap
				changesMade = true
			} else if c.CurrentRevisionID.Valid {
				c.State = c.CurrentRevision.DesiredState
				changesMade = true
			} else {
				c.State = StateNew
				changesMade = true
			}
		}
	default:
		errs = append(errs, fmt.Errorf("updateState for Contact at state \"%s\" not implemented", c.State))
	}

	if changesMade {
		if err := dbCache.Save(c); err != nil {
			errs = append(errs, err)

			return changesMade, errs
		}
	}

	return changesMade, errs
}

// UpdateHoldStatus is used to adjust the object's Hold status after
// the user has been verified as an admin. An error will be returned
// if the hold reason is not set.
func (c *Contact) UpdateHoldStatus(holdActive bool, holdReason string, holdBy string) error {
	if !holdActive {
		c.HoldActive = false
		c.HoldAt = time.Unix(0, 0)
		c.HoldReason = ""
		c.HoldBy = ""
	} else {
		if len(holdReason) == 0 {
			return errors.New("a hold reason must be set")
		}

		c.HoldActive = true
		c.HoldAt = time.Now()
		c.HoldBy = holdBy
		c.HoldReason = holdReason
	}

	return nil
}

// GetValidContactMap will return a map container the Host's title
// indexed by the host ID. If an error occurs, it will be returned
//
// TODO: Consider moving the db query into the dbcache object.
func GetValidContactMap(dbCache *DBCache) (ret map[int64]string, err error) {
	var contactList []Contact

	ret = make(map[int64]string)

	err = dbCache.DB.Where("state = ? or state = ?", StateActive, StateActivePendingApproval).Find(&contactList).Error
	if err != nil {
		return
	}

	for _, contact := range contactList {
		err = contact.Prepare(dbCache)

		if err != nil {
			return ret, err
		}

		ret[contact.ID] = contact.GetDisplayName()
	}

	return ret, err
}

// GetWorkContacts returns a list of IDs for contacts that require
// attention in the form of an update to the registry or other related
// information. If an error occurs, it will be returned
//
// TODO: Consider moving the db query into the dbcache object.
func GetWorkContacts(dbCache *DBCache) (retlist []int64, revisions []APIRevisionHint, err error) {
	var contacts []Contact

	err = dbCache.DB.Where(map[string]interface{}{"check_required": true, "hold_active": false}).Find(&contacts).Error

	for _, contact := range contacts {
		retlist = append(retlist, contact.ID)

		if contact.CurrentRevisionID.Valid {
			rev := APIRevisionHint{}
			rev.ObjectID = contact.ID
			rev.RevisionID = contact.CurrentRevisionID.Int64
			rev.LastUpdate = contact.UpdatedAt
			revisions = append(revisions, rev)
		}
	}

	return
}

// GetAllContacts returns a list of IDs for contats that are in the
// active or activepending approval states or requires work to be done
// on the contact. If an error occurs, it will be returned
//
// TODO: Consider moving the db query into the dbcache object.
func GetAllContacts(dbCache *DBCache) (retlist []int64, revisions []APIRevisionHint, err error) {
	var contacts []Contact

	err = dbCache.DB.Where("state = ? or state = ?", StateActive, StateActivePendingApproval).Find(&contacts).Error
	if err != nil {
		return
	}

	for _, contact := range contacts {
		retlist = append(retlist, contact.ID)

		if contact.CurrentRevisionID.Valid {
			rev := APIRevisionHint{}
			rev.ObjectID = contact.ID
			rev.RevisionID = contact.CurrentRevisionID.Int64
			rev.LastUpdate = contact.UpdatedAt
			revisions = append(revisions, rev)
		}
	}

	return
}

// MigrateDBContact will run the automigrate function for the Contact
// object.
func MigrateDBContact(dbCache *DBCache) {
	dbCache.AutoMigrate(&Contact{})
}
