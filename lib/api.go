package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"
)

const (
	// ErrorResponseType is used to identify an APIResponse containing an
	// error(s).
	ErrorResponseType string = "error"

	// TokenResponseType is used to identify an APIResponse containing a
	// TokenResponse object.
	TokenResponseType string = "token"

	// DomainIDListType is used to identify an APIResponse containing a list
	// of DomainIDs.
	DomainIDListType string = "domainidlist"

	// HostIDListType is used to identify an APIResponse containing a list
	// of HostIDs.
	HostIDListType string = "hostidlist"

	// ContactIDListType is used to identify an APIResponse containing a list
	// of ContactIDs.
	ContactIDListType string = "contactidlist"

	// HostnameListType is used to identify an APIResponse containing a map of
	// hostnames to IDs.
	HostnameListType string = "hostnamelist"

	// HostIPAllowList is used to identify an APIResponse containing a list of
	// IPs which correspond to registrar controlled nameserver IPs
	// TODO: inclusive language edit.
	HostIPAllowList string = "hostipallowlist"

	// ProtectedDomainList is used to identity an APIResponse containing a list of
	// domains which are protected and suggest extra review before provisioning.
	ProtectedDomainList string = "protecteddomainlist"

	// DomainObjectType is used to identify an APIResponse containing a domain
	// object.
	DomainObjectType string = "domainobject"

	// DomainRevisionObjectType is used to identify an APIResponse containing a
	// domain revision object.
	DomainRevisionObjectType string = "domainrevisionobject"

	// HostObjectType is used to identify an APIResponse containing a host
	// object.
	HostObjectType string = "hostobject"

	// HostRevisionObjectType is used to identify an APIResponse containing a
	// host revision object.
	HostRevisionObjectType string = "hostrevisionobject"

	// ContactObjectType is used to identify an APIResponse containing a contact
	// object.
	ContactObjectType string = "contactobject"

	// ContactRevisionObjectType is used to identify an APIResponse containing a
	// contact revision object.
	ContactRevisionObjectType string = "contactrevisionobject"

	// APIUserObjectType is used to identify an APIResponse containing an
	// api user object.
	APIUserObjectType string = "apiuserobject"

	// APIUserRevisionObjectType is used to identify an APIResponse containing an
	// api user revision object.
	APIUserRevisionObjectType string = "apiuserrevisionobject"

	// ApproverObjectType is used to identify an APIResponse containing an
	// approver object.
	ApproverObjectType string = "approverobject"

	// ApproverRevisionObjectType is used to identify an APIResponse containing an
	// approver revision object.
	ApproverRevisionObjectType string = "approverrevisionobject"

	// ApproverSetObjectType is used to identify an APIResponse containing an
	// approver set object.
	ApproverSetObjectType string = "approversetobject"

	// ApproverSetRevisionObjectType is used to identify an APIResponse containing an
	// approver set revision object.
	ApproverSetRevisionObjectType string = "approversetrevisionobject"

	// ChangeRequestObjectType is used to identify an APIResponse containing an
	// change request object.
	ChangeRequestObjectType string = "changerequestobject"

	// ApprovalObjectType is used to identify an APIResponse containing an
	// approval object.
	ApprovalObjectType string = "approvalobject"
)

// APIResponse is an object that is populated when responding to an API
// request.
type APIResponse struct {
	DomainObject          *DomainExport          `json:",omitempty"`
	DomainRevisionObject  *DomainRevisionExport  `json:",omitempty"`
	HostObject            *HostExport            `json:",omitempty"`
	HostRevisionObject    *HostRevisionExport    `json:",omitempty"`
	ContactObject         *ContactExport         `json:",omitempty"`
	ContactRevisionObject *ContactRevisionExport `json:",omitempty"`

	APIUserObject             *APIUserExportFull         `json:",omitempty"`
	APIUserRevisionObject     *APIUserRevisionExport     `json:",omitempty"`
	ApproverObject            *ApproverExportFull        `json:",omitempty"`
	ApproverRevisionObject    *ApproverRevisionExport    `json:",omitempty"`
	ApproverSetObject         *ApproverSetExportFull     `json:",omitempty"`
	ApproverSetRevisionObject *ApproverSetRevisionExport `json:",omitempty"`
	ChangeRequestObject       *ChangeRequestExport       `json:",omitempty"`
	ApprovalObject            *ApprovalExport            `json:",omitempty"`

	HostIPAllowList     *[]string `json:",omitempty"`
	ProtectedDomainList *[]string `json:",omitempty"`

	DomainIDList  *[]int64 `json:"DomainIDList,omitempty"`
	HostIDList    *[]int64 `json:"HostIDList,omitempty"`
	ContactIDList *[]int64 `json:"ContactIDList,omitempty"`

	DomainHintList  *[]APIRevisionHint `json:",omitempty"`
	HostHintList    *[]APIRevisionHint `json:",omitempty"`
	ContactHintList *[]APIRevisionHint `json:",omitempty"`

	HostnamesMap *map[string]int64 `json:",omitempty"`

	Signature *SignatureResponse `json:",omitempty"`
	Approval  *ApprovalDownload  `json:",omitempty"`
	Token     *TokenResponse     `json:",omitempty"`

	EPPRunID      *int64  `json:",omitempty"`
	EPPPassphrase *string `json:",omitempty"`

	MessageType string   `json:"MessageType"`
	Errors      []string `json:"Errors"`
}

// APIRevisionHint is used to provide a hint to the clients of which object
// revision is current for each current object.
type APIRevisionHint struct {
	ObjectID   int64
	RevisionID int64
	LastUpdate time.Time
}

// APIRequest is an object that is populated when a client is making a
// request to the api entry point. No body is needed when all data is
// passed in as get parameters.
type APIRequest struct {
	Signature   *SignatureUpload `json:"omitempty"`
	MessageType string           `json:"MessageType"`
	Token       string           `json:"Token"`
}

// GenerateErrorResponse will take a list of errors and create an
// APIReponse object indicating an error with the provided errors
// converted and set as the error list.
func GenerateErrorResponse(errs []error) APIResponse {
	apiResponse := APIResponse{MessageType: ErrorResponseType, Errors: ErrsToStrings(errs)}

	return apiResponse
}

// TokenResponse is the object that is populated when a request is sent
// for a CSRF token over the api.
type TokenResponse struct {
	Token string `json:"Token"`
	Time  int64  `json:"Time"`
}

// GenerateTokenResponse will create a new APIReponse object with the
// TokenResponse sub-object set.
func GenerateTokenResponse(token string) APIResponse {
	tokenResponse := TokenResponse{
		Token: token,
		Time:  time.Now().Unix(),
	}

	apiResponse := APIResponse{
		MessageType: TokenResponseType,
		Token:       &tokenResponse,
	}

	return apiResponse
}

// SignatureUploadType is used to identify an APIRequest containing a
// SignatureRequest object.
const SignatureUploadType string = "uploadsig"

// SignatureDownloadType is used to identify an APIResponse containing a
// SignatureRequest object.
const SignatureDownloadType string = "downloadsig"

// SignatureResponse is used to transmit a signed approval object from the
// server to the client.
type SignatureResponse struct {
	Signature []byte `json:"Signature" xml:"Signature"`
}

// SignatureUpload is used to transmit a signed approval object from a
// client to the server.
type SignatureUpload SignatureResponse

// GenerateSignatureResponse will create a new APIResponse object with
// the SignatureResponse sub-object set.
func GenerateSignatureResponse(data []byte) APIResponse {
	sr := SignatureResponse{Signature: data}
	apiResponse := APIResponse{
		MessageType: SignatureDownloadType,
		Signature:   &sr,
	}

	return apiResponse
}

// GenerateSignatureUpload will create a new APIRequest object with
// the SignatureResponse sub-object set.
func GenerateSignatureUpload(data []byte) APIRequest {
	su := SignatureUpload{Signature: data}
	apiRequest := APIRequest{
		MessageType: SignatureUploadType,
		Signature:   &su,
	}

	return apiRequest
}

// ApprovalDownloadType is used to identify an APIResponse containing a
// ApprovalDownload object.
const ApprovalDownloadType string = "downloadapproval"

// ApprovalDownload is used to transmit a unsigned approval object over
// an APIResponse.
type ApprovalDownload struct {
	Approval []byte `json:"Approval"`
}

// GenerateApprovalDownload is used to create an APIReponse object that
// has the apporval data and errors populated to be sent to the client.
func GenerateApprovalDownload(data []byte, errs []error) APIResponse {
	ad := ApprovalDownload{Approval: data}
	apiResponse := APIResponse{
		MessageType: ApprovalDownloadType,
		Approval:    &ad,
		Errors:      ErrsToStrings(errs),
	}

	return apiResponse
}

// GenerateIDList will take an object type and a list of IDs and create
// the correct IDList APIResponse object to return to the client.
func GenerateIDList(objectType string, ids []int64, revisions []APIRevisionHint) APIResponse {
	apiResponse := APIResponse{}

	switch objectType {
	case DomainType:
		apiResponse.MessageType = DomainIDListType
		apiResponse.DomainIDList = &ids
		apiResponse.DomainHintList = &revisions
	case HostType:
		apiResponse.MessageType = HostIDListType
		apiResponse.HostIDList = &ids
		apiResponse.HostHintList = &revisions
	case ContactType:
		apiResponse.MessageType = ContactIDListType
		apiResponse.ContactIDList = &ids
		apiResponse.ContactHintList = &revisions
	default:
		apiResponse.MessageType = ErrorResponseType
		apiResponse.Errors = append(apiResponse.Errors, fmt.Sprintf("Unknown object type %s", objectType))
	}

	return apiResponse
}

// Serialize returns a form of the response to be written to
// the wire.
func (response APIResponse) Serialize() ([]byte, error) {
	resp, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return resp, fmt.Errorf("error marshaling apiResponse: %w", err)
	}

	return resp, nil
}

// Write writes the response to a writer, returning
// number of bytes written.
func (response APIResponse) Write(writer io.Writer) (int, error) {
	raw, err := response.Serialize()
	if err != nil {
		return 0, err
	}

	byteCount, writeErr := writer.Write(raw)
	if writeErr != nil {
		return byteCount, fmt.Errorf("error writing message: %w", writeErr)
	}

	return byteCount, nil
}

// APIRespond sends the given response, ignoring possible errors.
// For error handling, use APIResponse.Write instead.
func APIRespond(httpResponse http.ResponseWriter, response APIResponse) {
	_, err := response.Write(httpResponse)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusInternalServerError)
	}
}

// GenerateObjectResponse is used to prepare an APIResponse object that
// contains the object presented.
func GenerateObjectResponse(object RegistrarObjectExport) APIResponse {
	apiResponse := APIResponse{}

	switch typedObject := object.(type) {
	case DomainExport:
		apiResponse.MessageType = DomainObjectType
		domainTyped := typedObject
		apiResponse.DomainObject = &domainTyped
	case DomainRevisionExport:
		apiResponse.MessageType = DomainRevisionObjectType
		domainRevisionTyped := typedObject
		apiResponse.DomainRevisionObject = &domainRevisionTyped
	case ContactExport:
		apiResponse.MessageType = ContactObjectType
		contactTyped := typedObject
		apiResponse.ContactObject = &contactTyped
	case ContactRevisionExport:
		apiResponse.MessageType = ContactRevisionObjectType
		contactRevisionTyped := typedObject
		apiResponse.ContactRevisionObject = &contactRevisionTyped
	case HostExport:
		apiResponse.MessageType = HostObjectType
		hostTyped := typedObject
		apiResponse.HostObject = &hostTyped
	case HostRevisionExport:
		apiResponse.MessageType = HostRevisionObjectType
		hostRevisionTyped := typedObject
		apiResponse.HostRevisionObject = &hostRevisionTyped
	case APIUserExportFull:
		apiResponse.MessageType = APIUserObjectType
		apiUserTyped := typedObject
		apiResponse.APIUserObject = &apiUserTyped
	case APIUserRevisionExport:
		apiResponse.MessageType = APIUserRevisionObjectType
		apiUserRevisionTyped := typedObject
		apiResponse.APIUserRevisionObject = &apiUserRevisionTyped
	case ApproverExportFull:
		apiResponse.MessageType = ApproverObjectType
		approverTyped := typedObject
		apiResponse.ApproverObject = &approverTyped
	case ApproverRevisionExport:
		apiResponse.MessageType = ApproverRevisionObjectType
		approverRevisionTyped := typedObject
		apiResponse.ApproverRevisionObject = &approverRevisionTyped
	case ApproverSetExportFull:
		apiResponse.MessageType = ApproverSetObjectType
		approverSetTyped := typedObject
		apiResponse.ApproverSetObject = &approverSetTyped
	case ApproverSetRevisionExport:
		apiResponse.MessageType = ApproverSetRevisionObjectType
		approverSetRevisionTyped := typedObject
		apiResponse.ApproverSetRevisionObject = &approverSetRevisionTyped
	case ChangeRequestExport:
		apiResponse.MessageType = ChangeRequestObjectType
		changeRequestTyped := typedObject
		apiResponse.ChangeRequestObject = &changeRequestTyped
	case ApprovalExport:
		apiResponse.MessageType = ApprovalObjectType
		approvalTyped := typedObject
		apiResponse.ApprovalObject = &approvalTyped
	default:
		apiResponse.MessageType = ErrorResponseType
		apiResponse.Errors = append(apiResponse.Errors, fmt.Sprintf("unsupported object type %s", reflect.TypeOf(object).Name()))
	}

	return apiResponse
}

// GetRegistrarObject will return the RegistrarObjectExport
// contained in the APIResponse object if it exists or the errors
// if the APIResponse is an error response.
func (response *APIResponse) GetRegistrarObject() (outObj RegistrarObjectExport, errs []error) {
	switch response.MessageType {
	case DomainObjectType:
		return response.DomainObject, errs
	case DomainRevisionObjectType:
		return response.DomainRevisionObject, errs
	case HostObjectType:
		return response.HostObject, errs
	case HostRevisionObjectType:
		return response.HostRevisionObject, errs
	case ContactObjectType:
		return response.ContactObject, errs
	case ContactRevisionObjectType:
		return response.ContactRevisionObject, errs
	case APIUserObjectType:
		return response.APIUserObject, errs
	case APIUserRevisionObjectType:
		return response.APIUserRevisionObject, errs
	case ApproverObjectType:
		return response.ApproverObject, errs
	case ApproverRevisionObjectType:
		return response.ApproverRevisionObject, errs
	case ApproverSetObjectType:
		return response.ApproverSetObject, errs
	case ApproverSetRevisionObjectType:
		return response.ApproverSetRevisionObject, errs
	case ChangeRequestObjectType:
		return response.ChangeRequestObject, errs
	case ApprovalObjectType:
		return response.ApprovalObject, errs
	case ErrorResponseType:
		return nil, StringsToErrs(response.Errors)
	}

	return outObj, errs
}

// ErrsToStrings takes a list of error objects and converts them into a
// list of strings.
func ErrsToStrings(errs []error) (strs []string) {
	for _, err := range errs {
		if err != nil {
			strs = append(strs, err.Error())
		}
	}

	return
}

// StringsToErrs takes a list of strings and converts them into a
// list of errors.
func StringsToErrs(strs []string) (errs []error) {
	for _, str := range strs {
		errs = append(errs, errors.New(str))
	}

	return
}
