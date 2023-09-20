package client

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/clearsign"

	"github.com/timapril/go-registrar/epp"
	"github.com/timapril/go-registrar/keychain"
	"github.com/timapril/go-registrar/lib"
	"github.com/timapril/go-registrar/whois/objects"

	"github.com/howeyc/gopass"
	logging "github.com/op/go-logging"
)

var format = logging.MustStringFormatter(
	"%{color}%{level:.4s} %{time:15:04:05.000000} %{shortfile} %{longfunc} %{callpath} %{id:03x}%{color:reset} %{message}",
)

// DiskCacheConfig is used to include in configruations files to handle the
// initialization of Disk Caches
type DiskCacheConfig struct {
	CacheDirectory string
	Enabled        bool
	UseHints       bool
}

// DiskCache is used to handle the storage and reterival of objects stored on
// the local disk
type DiskCache struct {
	Enabled  bool
	BasePath string
	UseHints bool

	log *logging.Logger
}

// NewDiskCache will generate an initialize a disk cache using the
// DiskCacheConfig passed
func NewDiskCache(dcc DiskCacheConfig, log *logging.Logger) (DiskCache, error) {
	cache := DiskCache{
		BasePath: dcc.CacheDirectory,
		Enabled:  dcc.Enabled,
		UseHints: dcc.UseHints,
		log:      log,
	}

	createRootErr := createDirIfNotExist(dcc.CacheDirectory, log)
	if createRootErr != nil {
		return cache, createRootErr
	}
	createCrErr := createDirIfNotExist(path.Join(dcc.CacheDirectory, lib.ChangeRequestType), log)
	if createCrErr != nil {
		return cache, createCrErr
	}
	createObjectErr := createDirIfNotExist(path.Join(dcc.CacheDirectory, "object"), log)
	if createObjectErr != nil {
		return cache, createObjectErr
	}
	createRevisionErr := createDirIfNotExist(path.Join(dcc.CacheDirectory, "snapshots"), log)
	if createRevisionErr != nil {
		return cache, createRevisionErr
	}

	return cache, nil
}

// GetObject will attempt to reterive the object of the type passed with the id
// pass from disk otherwise an error will be returned
func (d *DiskCache) GetObject(objType string, id int64, lastUpdate *time.Time) (outObj lib.RegistrarObjectExport, errs []error) {
	d.log.Debugf("Checking for %s with the ID of %d", objType, id)
	var filePath string
	if objType == lib.ChangeRequestType {
		filePath = d.changeRequestPath(id)
	} else {
		filePath = d.objectPath(objType, id)
	}
	fc, err := os.ReadFile(filePath)
	if err != nil {
		d.log.Error(err)
		errs = append(errs, err)
		return
	}
	var oc ObjectCache
	err = json.Unmarshal(fc, &oc)
	if err != nil {
		d.log.Error(err)
		errs = append(errs, err)
		return
	}

	obj, objErr := oc.Response.GetRegistrarObject()

	needsRefresh := false
	if lastUpdate != nil {
		switch t := obj.(type) {
		case lib.ContactExport:
			if (*lastUpdate).After(t.UpdatedAt) {
				needsRefresh = true
			}
		case lib.HostExport:
			if (*lastUpdate).After(t.UpdatedAt) {
				needsRefresh = true
			}
		case lib.DomainExport:
			if (*lastUpdate).After(t.UpdatedAt) {
				needsRefresh = true
			}
		}
	}
	if needsRefresh {
		errs = append(errs, errors.New("Objects needs to be refreshed"))
		return outObj, errs
	}

	return obj, objErr
}

// GetObjectAt will attempt to retrieve the object type with the id passed at
// the timestamp given. If no object is caches for that timestamp, an error
// is returned
func (d *DiskCache) GetObjectAt(objectType string, id int64, timestamp int64, od ObjectDirectory) (outObj lib.RegistrarObjectExport, errs []error) {
	d.log.Debugf("Checking for %s with the ID of %d @ %d", objectType, id, timestamp)

	if d.UseHints {
		var hint int64
		var lastUpdate time.Time
		canHandle := false
		switch objectType {
		case lib.DomainType:
			if val, ok := od.DomainHints[id]; ok {
				hint = val.RevisionID
				lastUpdate = val.LastUpdate
				canHandle = true
			}
		case lib.ContactType:
			if val, ok := od.ContactHints[id]; ok {
				hint = val.RevisionID
				lastUpdate = val.LastUpdate
				canHandle = true
			}
		case lib.HostType:
			if val, ok := od.HostHints[id]; ok {
				hint = val.RevisionID
				lastUpdate = val.LastUpdate
				canHandle = true
			}
		default:
			canHandle = false
		}

		if canHandle {
			outObj, errs = d.getSnapshotObject(objectType, hint, &lastUpdate)
			if len(errs) == 0 {
				return
			}
		}
	}
	return d.getRevisionObject(objectType, id, timestamp, nil)
}

// ObjectCache is used to serialize and object into the disk cache
type ObjectCache struct {
	Response lib.APIResponse
}

// ErrUnexpectedObject represents an unexpected object type received
var ErrUnexpectedObject = errors.New("unexpected object")

// SaveObject will take the given object and save it to disk in the cache
func (d *DiskCache) SaveObject(resp lib.APIResponse) error {
	oc := ObjectCache{
		Response: resp,
	}
	var objType string
	var objID int64
	typedObj, errs := resp.GetRegistrarObject()
	if len(errs) != 0 {
		var lastErr error
		for _, err := range errs {
			d.log.Error(err)
			lastErr = err
		}
		return fmt.Errorf("error retrieving object type: %w", lastErr)
	}
	switch t := typedObj.(type) {
	case *lib.ChangeRequestExport:
		objType = lib.ChangeRequestType
		objID = t.ID
	default:
		d.log.Errorf("Unhandled type %T", t)
		return ErrUnexpectedObject
	}
	d.log.Debugf("Saving object type %s", objType)
	var filePath string
	if objType == lib.ChangeRequestType {
		filePath = d.changeRequestPath(objID)
	} else {
		filePath = d.objectPath(objType, objID)
	}
	return d.saveObject(oc, filePath)
}

// SaveObjectAt will save the given object to disk for the given ID and add the
// timestamp to the date rage for the object revision table
func (d *DiskCache) SaveObjectAt(resp lib.APIResponse, id int64, timestamp int64) error {
	oc := ObjectCache{
		Response: resp,
	}

	var objType string
	var objID int64
	var currentRev int64

	typedObj, errs := resp.GetRegistrarObject()
	if len(errs) != 0 {
		var lastErr error
		for _, err := range errs {
			d.log.Error(err)
			lastErr = err
		}
		return fmt.Errorf("error retrieving object type: %w", lastErr)
	}
	switch t := typedObj.(type) {
	case *lib.ChangeRequestExport:
		objType = lib.ChangeRequestType
		objID = t.ID
		currentRev = t.ID
	case *lib.HostExport:
		objType = "host"
		objID = t.ID
		currentRev = t.CurrentRevision.ID
	case *lib.DomainExport:
		objType = "domain"
		objID = t.ID
		currentRev = t.CurrentRevision.ID
	case *lib.ApproverExportFull:
		objType = "approver"
		objID = t.ID
		currentRev = t.CurrentRevision.ID
	case *lib.ApproverSetExportFull:
		objType = "approverset"
		objID = t.ID
		currentRev = t.CurrentRevision.ID
	case *lib.ContactExport:
		objType = "contact"
		objID = t.ID
		currentRev = t.CurrentRevision.ID
	default:
		d.log.Errorf("Unhandled type %T", t)
		return ErrUnexpectedObject
	}
	d.log.Noticef("%s id %d -- Current Revision ID %d", objType, objID, currentRev)
	path := d.snapshotPath(objType, currentRev)
	err := d.saveObject(oc, path)
	if err != nil {
		return err
	}
	return d.UpdateRevisionList(objType, objID, currentRev, timestamp)
}

// RevisionValidity is used to store information about which revisions were
// active for given times
type RevisionValidity struct {
	RevisionID int64
	MinTime    int64
	MaxTime    int64
}

// ObjectInfo is used to store multiple revision validity windows for objects
// of the same ID. Object info is what will be serialized to disk
type ObjectInfo struct {
	ObjectID  int64
	Revisions []RevisionValidity
}

// getRevisionObject wil attempt to retrieve the object type with the id passed
// at the given timestamp from disk. If the object cannot be found on disk
// or there is an error reading the object off disk, an error will be returned
func (d *DiskCache) getRevisionObject(objectType string, objectID int64, ts int64, lastUpdate *time.Time) (outObj lib.RegistrarObjectExport, errs []error) {
	var oi ObjectInfo

	path := d.objectPath(objectType, objectID)

	fc, err := os.ReadFile(path)
	if err != nil {
		errs = append(errs, err)
		d.log.Error(err)
		return
	}
	err = json.Unmarshal(fc, &oi)
	if err != nil {
		errs = append(errs, err)
		d.log.Error(err)
		return
	}

	for _, rev := range oi.Revisions {
		if rev.MaxTime >= ts && rev.MinTime <= ts {
			d.log.Noticef("Use revision %d", rev.RevisionID)
			return d.getSnapshotObject(objectType, rev.RevisionID, lastUpdate)
		}
	}

	errs = append(errs, errors.New("Unable to located a cached version of the object"))
	return
}

// getSnapshotObject will attempt to read the object at a specific revision from
// disk
func (d *DiskCache) getSnapshotObject(objectType string, revision int64, lastUpdate *time.Time) (outObj lib.RegistrarObjectExport, errs []error) {
	filePath := d.snapshotPath(objectType, revision)

	fc, err := os.ReadFile(filePath)
	if err != nil {
		d.log.Error(err)
		errs = append(errs, err)
		return
	}
	var oc ObjectCache
	err = json.Unmarshal(fc, &oc)
	if err != nil {
		d.log.Error(err)
		errs = append(errs, err)
		return
	}

	obj, objErr := oc.Response.GetRegistrarObject()

	needsRefresh := false
	if lastUpdate != nil {
		switch t := obj.(type) {
		case *lib.ContactExport:
			if (*lastUpdate).After(t.UpdatedAt) {
				needsRefresh = true
			}
		case *lib.HostExport:
			if (*lastUpdate).After(t.UpdatedAt) {
				needsRefresh = true
			}
		case *lib.DomainExport:
			if (*lastUpdate).After(t.UpdatedAt) {
				needsRefresh = true
			}
		}
	}
	if needsRefresh {
		errs = append(errs, errors.New("Objects needs to be refreshed"))
		return outObj, errs
	}

	return obj, objErr
}

// UpdateRevisionList will attempt to read in a object from cache and add a
// revision at a timestamp to the object infomration and then write the object
// back to disk
func (d *DiskCache) UpdateRevisionList(objectType string, objectID int64, revisionID int64, ts int64) error {
	var oi ObjectInfo

	path := d.objectPath(objectType, objectID)

	fc, err := os.ReadFile(path)
	if err != nil {
		oi.ObjectID = objectID
	} else {
		err = json.Unmarshal(fc, &oi)
		if err != nil {
			d.log.Error(err)
			return err
		}
	}

	foundRevision := false
	for idx, rev := range oi.Revisions {
		if rev.RevisionID == revisionID {
			foundRevision = true
			if ts > rev.MaxTime {
				oi.Revisions[idx].MaxTime = ts
			}
			if ts < rev.MinTime {
				oi.Revisions[idx].MinTime = ts
			}
		}
	}
	if !foundRevision {
		rv := RevisionValidity{
			RevisionID: revisionID,
			MinTime:    ts,
			MaxTime:    ts,
		}
		oi.Revisions = append(oi.Revisions, rv)
	}

	return d.saveObject(oi, path)

}

// saveObject will serialize and save the given object to the provided path
func (d *DiskCache) saveObject(obj interface{}, path string) error {
	d.log.Debugf("Saving to %s", path)
	fc, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		d.log.Error(err)
		return err
	}
	writeErr := os.WriteFile(path, fc, 0644)
	if writeErr != nil {
		d.log.Error(writeErr)
		return writeErr
	}
	return nil
}

// objectPath will generate and return the path for the object based on the
// object type and id passed
func (d *DiskCache) objectPath(objType string, id int64) string {
	return path.Join(d.BasePath, "object", fmt.Sprintf("%s.%d.json", objType, id))
}

// snapshotPath will generate and return the path for the object based on the
// object type and id passed
func (d *DiskCache) snapshotPath(objType string, id int64) string {
	return path.Join(d.BasePath, "snapshots", fmt.Sprintf("%s.%d.json", objType, id))
}

// changeRequestPath will generate and return the path for a change request with
// the provided id
func (d *DiskCache) changeRequestPath(id int64) string {
	return path.Join(d.BasePath, lib.ChangeRequestType, fmt.Sprintf("%d.json", id))
}

// createDirIfNotExist will check if the given directory exists and then create
// it if it does not exit
func createDirIfNotExist(directory string, log *logging.Logger) error {
	log.Debugf("Checking to see if %s exists", directory)
	stat, err := os.Stat(directory)
	if err != nil || stat == nil {
		log.Debug("Fresh Cache Directory Created")
		err := os.Mkdir(directory, 0777)
		if err != nil {
			return err
		}
		return nil
	}
	if !stat.IsDir() {
		return errors.New("path provided is not a directory")
	}
	return nil
}

// ObjectDirectory handles a list of IDs and revision hints for the client to
// allow quick lookups in the disk cache if used
type ObjectDirectory struct {
	DomainIDs   []int64
	DomainHints map[int64]lib.APIRevisionHint

	HostIDs   []int64
	HostHints map[int64]lib.APIRevisionHint

	ContactIDs   []int64
	ContactHints map[int64]lib.APIRevisionHint
}

// NewObjectDirectory will initialize a new object directory object and return
// it
func NewObjectDirectory() ObjectDirectory {
	od := ObjectDirectory{}
	od.DomainHints = make(map[int64]lib.APIRevisionHint)
	od.ContactHints = make(map[int64]lib.APIRevisionHint)
	od.HostHints = make(map[int64]lib.APIRevisionHint)
	return od
}

// LoadObjectDirectory will attempt to load the list of active objects and the
// current hints for all objects. If any errors are returned when trying to
// load the objects, the errors will be returned
func (od *ObjectDirectory) LoadObjectDirectory(cli *Client) (errs []error) {
	var domErrs, conErrs, hosErrs []error
	od.DomainIDs, od.DomainHints, domErrs = cli.GetAll(lib.DomainType)
	if len(domErrs) != 0 {
		return domErrs
	}
	od.ContactIDs, od.ContactHints, conErrs = cli.GetAll(lib.ContactType)
	if len(conErrs) != 0 {
		return conErrs
	}
	od.HostIDs, od.HostHints, hosErrs = cli.GetAll(lib.HostType)
	if len(hosErrs) != 0 {
		return hosErrs
	}
	return errs
}

// Client is an object that will hold the required state to
// communicate with the registrar server
type Client struct {
	httpsClient *http.Client
	basePath    string

	spoofedClientCert string
	spoofedHeaderName string

	log *logging.Logger

	currentRunID int64

	TrustAnchor TrustAnchors

	cache DiskCache

	ObjectDir ObjectDirectory
}

// Get will send a request to the server using the current client
// and return a respose and error, similar to the response from
// http.Client.Get
func (a *Client) Get(path string) (data []byte, err error) {
	a.logDebug(fmt.Sprintf("Get %s%s", a.basePath, path))
	req, reqErr := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", a.basePath, path), nil)
	if reqErr != nil {
		a.logDebug("Error creating request")
		err = reqErr
		return
	}
	if len(a.spoofedClientCert) != 0 {
		a.logDebug("Adding spoofed header")
		req.Header.Add(a.spoofedHeaderName, strings.Replace(a.spoofedClientCert, "\n", " ", -1))
	}

	resp, getErr := a.httpsClient.Do(req)
	if getErr != nil {
		err = getErr
		return
	}
	defer resp.Body.Close()

	var readErr error
	data, readErr = io.ReadAll(resp.Body)
	if readErr != nil {
		err = readErr
		return
	}

	return
}

// Post will send a request to the server using the current client
// and return a respose and error, similar to the response from
// http.Client.Post
func (a *Client) Post(path string, bodyType string, body io.Reader) (*http.Response, error) {
	req, reqErr := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", a.basePath, path), body)
	if reqErr != nil {
		return nil, reqErr
	}
	if len(a.spoofedClientCert) != 0 {
		a.logDebug("Adding spoofed header")
		req.Header.Add(a.spoofedHeaderName, strings.Replace(a.spoofedClientCert, "\n", " ", -1))
	}

	return a.httpsClient.Do(req)
}

// Prepare will take the required information for the client and prepare
// it to send queries to the server
func (a *Client) Prepare(base string, log *logging.Logger, dcc DiskCacheConfig) {
	a.ObjectDir = NewObjectDirectory()

	a.basePath = base

	a.httpsClient = &http.Client{}

	if log != nil {
		a.log = log
	} else {
		backend := logging.NewLogBackend(os.Stdout, "", 0)
		backendFormatter := logging.NewBackendFormatter(backend, format)
		backendLevel := logging.AddModuleLevel(backendFormatter)
		backendLevel.SetLevel(logging.CRITICAL, "")
		logging.SetBackend(backendLevel)

		a.log = logging.MustGetLogger("registrar")
	}

	var cacheCreateErr error
	a.cache, cacheCreateErr = NewDiskCache(dcc, log)
	if cacheCreateErr != nil {
		log.Error(cacheCreateErr)
	}
}

// SpoofCertificateForTesting is used to spoof a client certificate that
// is used to authenticatea client to the testing server
func (a *Client) SpoofCertificateForTesting(cert string, headername string) {
	a.spoofedClientCert = cert
	a.spoofedHeaderName = headername
}

// PrepareSSL will take the required infomration for the client and
// prepare it to send queries to the server using TLS
func (a *Client) PrepareSSL(base, certFile, keyFile, caFile string, keychainConf keychain.Conf, log *logging.Logger, dcc DiskCacheConfig) {
	a.ObjectDir = NewObjectDirectory()

	a.basePath = base

	if log != nil {
		a.log = log
	} else {
		backend := logging.NewLogBackend(os.Stdout, "", 0)
		backendFormatter := logging.NewBackendFormatter(backend, format)
		backendLevel := logging.AddModuleLevel(backendFormatter)
		backendLevel.SetLevel(logging.CRITICAL, "")
		logging.SetBackend(backendLevel)

		a.log = logging.MustGetLogger("registrar")
	}

	var cacheCreateErr error
	a.cache, cacheCreateErr = NewDiskCache(dcc, log)
	if cacheCreateErr != nil {
		log.Error(cacheCreateErr)
	}

	a.logDebugf("Cert File Path: %s", certFile)
	a.logDebugf("Key File Path: %s", keyFile)
	a.logDebugf("CA File Path: %s", caFile)

	// Load client cert
	cert, err := loadCert(certFile, keyFile, keychainConf) //tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		a.logError(err.Error())
	}

	// Load CA cert
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		a.logError(err.Error())
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	log.Debug("checking for HTTP_PROXY")
	var proxyUrl *url.URL = nil
	httpProxyEnv := os.Getenv("HTTP_PROXY")
	if httpProxyEnv != "" {
		proxyUrl, err = url.Parse(httpProxyEnv)
		if err != nil {
			panic(err)
		}
		log.Debug("using HTTP proxy ", httpProxyEnv)
	} else {
		log.Debug("HTTP_PROXY is not set, not using a proxy")
	}

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()

	var transport *http.Transport
	if proxyUrl == nil {
		transport = &http.Transport{TLSClientConfig: tlsConfig}
	} else {
		transport = &http.Transport{TLSClientConfig: tlsConfig, Proxy: http.ProxyURL(proxyUrl)}
	}

	a.httpsClient = &http.Client{Transport: transport}
}

// PrepareObjectDirectory will attempt to prepare the object directory for use.
// If any errors are encountered, they will be returned
func (a *Client) PrepareObjectDirectory() (errs []error) {
	return a.ObjectDir.LoadObjectDirectory(a)
}

// GetToken will retrieve a CSRF token from the server for the user that
// is currently logged in
func (a *Client) GetToken() (token string, errs []error) {
	a.logDebug("Getting an api token")
	data, getErr := a.Get("/api/gettoken")
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	if respObj.MessageType == lib.TokenResponseType && respObj.Token != nil {
		token = respObj.Token.Token
	} else if respObj.MessageType == lib.ErrorResponseType {
		for _, err := range respObj.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		errs = append(errs, fmt.Errorf("Expected a message type of %s, got %s", lib.TokenResponseType, respObj.MessageType))
	}

	return
}

// GetHostIPAllowList will retrieve the list of Host IPs that are allowlisted
// as they are trusted nameservers
func (a *Client) GetHostIPAllowList() (ips []string, errs []error) {
	data, getErr := a.Get("/api/gethostipallowlist")
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	if respObj.MessageType == lib.HostIPAllowList && respObj.HostIPAllowList != nil {
		ips = *respObj.HostIPAllowList
	} else if respObj.MessageType == lib.ErrorResponseType {
		for _, err := range respObj.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		errs = append(errs, fmt.Errorf("Expected a message type of %s, got %s", lib.HostIPAllowList, respObj.MessageType))
	}

	return
}

// GetProtectedDomainList will retrieve the list of Protected domains that are
// stored in the registrar system. If an error is encountered, it will be
// returned
func (a *Client) GetProtectedDomainList() (domains []string, errs []error) {
	data, getErr := a.Get("/api/getprotecteddomainlist")
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	if respObj.MessageType == lib.ProtectedDomainList && respObj.ProtectedDomainList != nil {
		domains = *respObj.ProtectedDomainList
	} else if respObj.MessageType == lib.ErrorResponseType {
		for _, err := range respObj.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		errs = append(errs, fmt.Errorf("Expected a message type of %s, got %s", lib.ProtectedDomainList, respObj.MessageType))
	}

	return
}

// GetApproval will try to retrieve an approval from the registrar
// server given the approval ID and the desired approver id
func (a *Client) GetApproval(approvalID int64, approverID int64, action string) (approvalObject []byte, errs []error) {
	token, tokenErrs := a.GetToken()
	if tokenErrs != nil {
		errs = tokenErrs
		return
	}
	data, getErr := a.Get(fmt.Sprintf("/api/approval/%d/download?approverid=%d&action=%s&csrf_token=%s", approvalID, approverID, action, token))
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	if respObj.MessageType == lib.ApprovalDownloadType && respObj.Approval != nil {
		approvalObject = respObj.Approval.Approval
	} else if respObj.MessageType == lib.ErrorResponseType {
		for _, err := range respObj.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		errs = append(errs, fmt.Errorf("Expected a message type of %s, got %s", lib.ApprovalDownloadType, respObj.MessageType))
	}

	return
}

// GetDomain will try and retrieve a domain object from the server
func (a *Client) GetDomain(id int64) (outobj *lib.DomainExport, errs []error) {
	obj, errs := a.GetObject(lib.DomainType, id, nil)
	outObj, ok := obj.(*lib.DomainExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetDomainIDFromName will attempt to find the domain ID from the domain name
// provided. If a domain object is found, its id will be returned, otherwise
// an error will be returned
func (a *Client) GetDomainIDFromName(domainName string) (id int64, errs []error) {
	data, getErr := a.Get(fmt.Sprintf("/api/domainnametoid/%s", domainName))
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	fmt.Println(string(data))

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	if respObj.MessageType == lib.DomainIDListType {
		if len(*respObj.DomainIDList) == 1 {
			id = (*respObj.DomainIDList)[0]
			return
		}
		errs = append(errs, errors.New("Only expected one domain ID to be returned"))
		return
	}
	if len(respObj.Errors) != 0 {
		for _, err := range respObj.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		errs = append(errs, fmt.Errorf("Unexpected message type of %s", respObj.MessageType))
	}
	return
}

// GetVerifiedDomain will attempt to download and verify a Domain
// with the given ID at the given time. A bool of if the Domain was
// verified or not, a list of errors and the resulting
// DomainExport object will be returned. An empty object is
// returned if the object did not verify.
func (a *Client) GetVerifiedDomain(domainID int64, timestamp int64) (verified bool, errs []error, obj *lib.DomainExport) {
	verified = false
	dom, domGetErrs := a.GetDomainAt(domainID, timestamp)
	a.log.Debugf("Getting Domain %d", domainID)
	if len(domGetErrs) != 0 {
		errs = append(errs, domGetErrs...)
		return
	}

	if dom.CurrentRevision.ID <= 0 {
		errMsg := fmt.Sprintf("No current revision found for Domain %d", dom.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}
	crid := dom.CurrentRevision.ChangeRequestID
	if crid <= 0 {
		errMsg := fmt.Sprintf("Unable to find a change request for Domain Revision %d", dom.CurrentRevision.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}

	a.log.Debugf("Found Change Request %d for Domain %d", crid, dom.ID)
	verified, errs, signedData := a.VerifyChangeRequest(crid, dom.CurrentRevision)
	if !verified {
		return false, errs, obj
	}

	var de lib.DomainExport
	err := json.Unmarshal(signedData, &de)
	if err != nil {
		errs = append(errs, err)
		return false, errs, obj
	}

	// Compare the export version to the current working export
	// object
	isSame, errList := dom.CurrentRevision.CompareExport(de.PendingRevision)

	// If they are the same, the object is verified
	if isSame {
		a.log.Debugf("Domain %d has been transativly signed by one of the trust anchors", de.ID)
		return true, errs, dom
	}
	errs = append(errs, errList...)
	return false, errs, obj
}

// GetDomainRevision will try and retrieve a domain object from the server
func (a *Client) GetDomainRevision(id int64) (outobj *lib.DomainRevisionExport, errs []error) {
	obj, errs := a.GetObject(lib.DomainRevisionType, id, nil)
	outObj, ok := obj.(*lib.DomainRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetContact will try and retrieve a domain object from the server
func (a *Client) GetContact(id int64) (outobj *lib.ContactExport, errs []error) {
	obj, errs := a.GetObject(lib.ContactType, id, nil)
	outObj, ok := obj.(*lib.ContactExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetVerifiedContact will attempt to download and verify a Contact
// with the given ID at the given time. A bool of if the Contact was
// verified or not, a list of errors and the resulting
// ContactExport object will be returned. An empty object is
// returned if the object did not verify.
func (a *Client) GetVerifiedContact(contactID int64, timestamp int64) (verified bool, errs []error, obj *lib.ContactExport) {
	verified = false
	con, conGetErrs := a.GetContactAt(contactID, timestamp)
	a.log.Debugf("Getting Contact %d", contactID)
	if len(conGetErrs) != 0 {
		errs = append(errs, conGetErrs...)
		return
	}

	if con.CurrentRevision.ID <= 0 {
		errMsg := fmt.Sprintf("No current revision found for Contact %d", con.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}
	crid := con.CurrentRevision.ChangeRequestID
	if crid <= 0 {
		errMsg := fmt.Sprintf("Unable to find a change request for Contact Revision %d", con.CurrentRevision.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}

	a.log.Debugf("Found Change Request %d for Contact %d", crid, con.ID)
	verified, errs, signedData := a.VerifyChangeRequest(crid, con.CurrentRevision)
	if !verified {
		return false, errs, obj
	}

	var ce lib.ContactExport
	err := json.Unmarshal(signedData, &ce)
	if err != nil {
		errs = append(errs, err)
		return false, errs, obj
	}

	// Compare the export version to the current working export
	// object
	isSame, errList := con.CurrentRevision.CompareExport(ce.PendingRevision)

	// If they are the same, the object is verified
	if isSame {
		a.log.Debugf("Contact %d has been transativly signed by one of the trust anchors", ce.ID)
		return true, errs, con
	}
	errs = append(errs, errList...)
	return false, errs, obj
}

// GetContactRevision will try and retrieve a domain object from the server
func (a *Client) GetContactRevision(id int64) (outobj *lib.ContactRevisionExport, errs []error) {
	obj, errs := a.GetObject(lib.ContactRevisionType, id, nil)
	outObj, ok := obj.(*lib.ContactRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetHost will try and retrieve a domain object from the server
func (a *Client) GetHost(id int64) (outobj *lib.HostExport, errs []error) {
	obj, errs := a.GetObject(lib.HostType, id, nil)
	outObj, ok := obj.(*lib.HostExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetVerifiedHost will attempt to download and verify a Host
// with the given ID at the given time. A bool of if the Host was
// verified or not, a list of errors and the resulting
// HostExport object will be returned. An empty object is
// returned if the object did not verify.
func (a *Client) GetVerifiedHost(hostID int64, timestamp int64) (verified bool, errs []error, obj *lib.HostExport) {
	verified = false
	hos, hosGetErrs := a.GetHostAt(hostID, timestamp)
	a.log.Debugf("Getting Host %d", hostID)
	if len(hosGetErrs) != 0 {
		errs = append(errs, hosGetErrs...)
		return
	}

	if hos.CurrentRevision.ID <= 0 {
		errMsg := fmt.Sprintf("No current revision found for Host %d", hos.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}
	crid := hos.CurrentRevision.ChangeRequestID
	if crid <= 0 {
		errMsg := fmt.Sprintf("Unable to find a change request for Host Revision %d", hos.CurrentRevision.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}

	a.log.Debugf("Found Change Request %d for Host %d", crid, hos.ID)
	verified, errs, signedData := a.VerifyChangeRequest(crid, hos.CurrentRevision)
	if !verified {
		return false, errs, obj
	}

	var he lib.HostExport
	err := json.Unmarshal(signedData, &he)
	if err != nil {
		errs = append(errs, err)
		return false, errs, obj
	}

	// Compare the export version to the current working export
	// object
	isSame, errList := hos.CurrentRevision.CompareExport(he.PendingRevision)

	// If they are the same, the object is verified
	if isSame {
		a.log.Debugf("Host %d has been transativly signed by one of the trust anchors", he.ID)
		return true, errs, hos
	}
	errs = append(errs, errList...)
	return false, errs, obj
}

// GetHostRevision will try and retrieve a domain object from the server
func (a *Client) GetHostRevision(id int64) (outobj *lib.HostRevisionExport, errs []error) {
	obj, errs := a.GetObject(lib.HostRevisionType, id, nil)
	outObj, ok := obj.(*lib.HostRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetAPIUser will try and retrieve a domain object from the server
func (a *Client) GetAPIUser(id int64) (outobj *lib.APIUserExportFull, errs []error) {
	obj, errs := a.GetObject(lib.APIUserType, id, nil)
	outObj, ok := obj.(*lib.APIUserExportFull)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetAPIUserRevision will try and retrieve a domain object from the server
func (a *Client) GetAPIUserRevision(id int64) (outobj *lib.APIUserRevisionExport, errs []error) {
	obj, errs := a.GetObject(lib.APIUserRevisionType, id, nil)
	outObj, ok := obj.(*lib.APIUserRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetApprover will try and retrieve a domain object from the server
func (a *Client) GetApprover(id int64) (outobj *lib.ApproverExportFull, errs []error) {
	obj, errs := a.GetObject(lib.ApproverType, id, nil)
	outObj, ok := obj.(*lib.ApproverExportFull)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetVerifiedApprover will attempt to download and verify an approver
// with the given ID at the given time. A bool of if the Approver was
// verified or not, a list of errors and the resulting
// ApproverExportFull object will be returned. An empty object is
// returned if the object did not verify.
func (a *Client) GetVerifiedApprover(approverID int64, timestamp int64) (verified bool, errs []error, obj *lib.ApproverExportFull) {
	verified = false
	app, appGetErrs := a.GetApproverAt(approverID, timestamp)
	a.log.Debugf("Getting Approver %d", approverID)
	if len(appGetErrs) != 0 {
		errs = append(errs, appGetErrs...)
		return
	}

	if app.CurrentRevision.ID <= 0 {
		errMsg := fmt.Sprintf("No current revision found for Approver %d", app.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}
	crid := app.CurrentRevision.ChangeRequestID
	if crid <= 0 {
		errMsg := fmt.Sprintf("Unable to find a change request for Approver Revision %d", app.CurrentRevision.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}

	a.log.Debugf("Found Change Request %d for Approver %d", crid, app.ID)
	verified, errs, signedData := a.VerifyChangeRequest(crid, app.CurrentRevision)
	if !verified {
		return false, errs, obj
	}

	var ae lib.ApproverExportFull
	err := json.Unmarshal(signedData, &ae)
	if err != nil {
		errs = append(errs, err)
		return false, errs, obj
	}

	// Compare the export version to the current working export
	// object
	isSame, errList := app.CurrentRevision.CompareExport(ae.PendingRevision)

	// If they are the same, the object is verified
	if isSame {
		a.log.Debugf("Approver %d has been transativly signed by one of the trust anchors", ae.ID)
		return true, errs, app
	}
	errs = append(errs, errList...)
	return false, errs, obj
}

// GetApproverRevision will try and retrieve a domain object from the server
func (a *Client) GetApproverRevision(id int64) (outobj *lib.ApproverRevisionExport, errs []error) {
	obj, errs := a.GetObject(lib.ApproverRevisionType, id, nil)
	outObj, ok := obj.(*lib.ApproverRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetApproverSet will try and retrieve a domain object from the server
func (a *Client) GetApproverSet(id int64) (outobj *lib.ApproverSetExportFull, errs []error) {
	obj, errs := a.GetObject(lib.ApproverSetType, id, nil)
	outObj, ok := obj.(*lib.ApproverSetExportFull)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetVerifiedApproverSet will attempt to download and verify an
// approver set with the given ID at the given time. A bool of if the
// Approver Set was verified or not, a list of errors and the resulting
// ApproverSetExportFull object will be returned. An empty object is
// returned if the object did not verify
func (a *Client) GetVerifiedApproverSet(approverSetID int64, timestamp int64) (verified bool, errs []error, obj *lib.ApproverSetExportFull) {
	verified = false
	apps, appSGetErrs := a.GetApproverSetAt(approverSetID, timestamp)
	a.log.Debugf("Getting Approver Set %d", approverSetID)
	if len(appSGetErrs) != 0 {
		errs = append(errs, appSGetErrs...)
		return
	}

	a.log.Debugf("Verifying Change Request %d for Approver Set %d", apps.CurrentRevision.ChangeRequestID, apps.ID)
	verified, errs, signedData := a.VerifyChangeRequest(apps.CurrentRevision.ChangeRequestID, apps.CurrentRevision)
	if !verified {
		return false, errs, obj
	}

	var ase lib.ApproverSetExportFull
	err := json.Unmarshal(signedData, &ase)
	if err != nil {
		errs = append(errs, err)
		return false, errs, obj
	}

	// Compare the export version to the current working export
	// object
	isSame, errList := apps.CurrentRevision.CompareExport(ase.PendingRevision)

	// If they are the same, the object is verified
	if isSame {
		a.log.Debugf("Approver Set %d has been transativly signed by one of the trust anchors", ase.ID)
		return true, errs, apps
	}
	errs = append(errs, errList...)
	return false, errs, obj
}

// VerifyApproverSet will attempt to verify that the approver set
// provided has been signed by one of the trust anchors or has a chain
// of signatures and revisions back to a trust anchor. A bool indicating
// if the object was verified, a list of errors and a
// ApproverSetRevisionExport object are returned. If the Approver set
// was not verified an empty object is returned. If only some of the
// approvers of the current approver set were verified then only the
// verifable approvers are returned. If no valid approvers were found
// verified will be set to false, an error will be added and the object
// will be just the Approver set.
func (a *Client) VerifyApproverSet(as *lib.ApproverSetExportFull) (verified bool, obj lib.ApproverSetRevisionExport, errs []error) {
	a.log.Debugf("Starting verify for Approver Set %d", as.ID)

	verified = false

	// Check the approver set to make sure that it has a valid ID
	if as.CurrentRevision.ID <= 0 {
		errMsg := fmt.Sprintf("No current revision found for Approver Set %d", as.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}

	// Check to make sure that a change request has been created for the
	// approver set
	crid := as.CurrentRevision.ChangeRequestID
	if crid <= 0 {
		errMsg := fmt.Sprintf("Unable to find a change request for Approver Set Revision %d", as.CurrentRevision.ID)
		a.log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
		return
	}

	a.log.Debugf("Found Change Request %d for Approver Set %d", crid, as.ID)

	// Get the change request
	cr, crErrs := a.GetChangeRequest(crid)
	if len(crErrs) != 0 {
		for _, err := range crErrs {
			a.log.Error(err.Error())
		}
		errs = append(errs, crErrs...)
		return
	}

	// Cycle through all of the approvals to try and find the final
	// approval
	for _, app := range cr.Approvals {
		if app.IsFinalApproval {
			// We've found the final approval, now lets check to see how it
			// was signed.
			var trustedSignedData []byte

			// If it was signed by one of the trust anchors, lets get the
			// signed message
			signedByAnchor, trustedSignedData := a.TrustAnchor.IsSignedBy(app.Signature)

			// If the data was not signed by one of the trust anchors lets go
			// dig backwards to find where we can anchor it
			if !signedByAnchor {
				nextEarilerTS := app.CreatedAt.Unix()
				a.log.Debugf("Approver Set %d at Change Request %d was not signed by a trust anchor", as.ID, crid)
				a.log.Debugf("Tring to verify the Approver Set at t=%d", nextEarilerTS)
				asNew, errList := a.GetApproverSetAt(app.ApproverSetID, nextEarilerTS)
				if len(errList) != 0 {
					for _, err := range errList {
						a.log.Error(err.Error())
					}
					errs = append(errs, errList...)
					return
				}
				asVerified, _, asErrs := a.VerifyApproverSet(asNew)

				a.log.Infof("Approver Set Verified: %t", asVerified)
				errs = append(errs, asErrs...)

				// now we need to extract the signed data
				block, _ := clearsign.Decode(app.Signature)
				if block == nil {
					errs = append(errs, errors.New("no signature found"))
				} else {
					trustedSignedData = block.Bytes
				}
			}
			// Unpack the approval assertion
			var aa lib.ApprovalAttestationUnmarshal
			err := json.Unmarshal(trustedSignedData, &aa)
			if err != nil {
				errs = append(errs, err)
				return false, obj, errs
			}

			// If the approval is an approval
			if aa.Action == lib.ActionApproved {
				// and the approval is for an approver set
				if aa.ObjectType == lib.ApproverSetType {

					// unpack the signed approver set export object
					var ase lib.ApproverSetExportFull
					err = json.Unmarshal(aa.ExportRev, &ase)
					if err != nil {
						errs = append(errs, err)
						return false, obj, errs
					}

					// Compare the export version to the current working export
					// object
					isSame, errList := as.CurrentRevision.CompareExport(ase.PendingRevision)

					// If they are the same, the object is verified
					if isSame {
						a.log.Debugf("Approver Set %d was signed by one of the trust anchors, grabbing the verified approvers", as.ID)
						//return true, as.CurrentRevision, errs
						a.log.Debugf("Grabbing all verified approvers for the approver set")
						for _, revApprover := range as.CurrentRevision.Approvers {
							a.log.Debugf("Grabbing Approver %d", revApprover.ID)
							verified, grabErrs, app := a.GetVerifiedApprover(revApprover.ID, cr.CreatedAt.Unix())
							if !verified {
								newErr := fmt.Sprintf("Unable to find the verified approver %d at time %d", revApprover.ID, cr.CreatedAt.Unix())
								errs = append(errs, errors.New(newErr))
								errs = append(errs, grabErrs...)
							} else {
								err = as.CurrentRevision.AddVerifiedApprover(*app)
								if err != nil {
									errs = append(errs, err)
								}
							}
							a.log.Debugf("Approver %d at time %d was verified?: %t", revApprover.ID, cr.CreatedAt.Unix(), verified)
						}
						if !as.CurrentRevision.HasVerifiedApprovers() {
							newErr := fmt.Sprintf("No verified approvers found for approver set %d at time %d", as.ID, cr.CreatedAt.Unix())
							errs = append(errs, errors.New(newErr))
							return false, obj, errs
						}
						return true, as.CurrentRevision, errs
					}
					errs = append(errs, errList...)
					return false, obj, errs
				}
			} else {
				errs = append(errs, errors.New("Change request was not approved"))
			}
		}
	}

	return
}

// GetApproverSetRevision will try and retrieve a domain object from the server
func (a *Client) GetApproverSetRevision(id int64) (outobj *lib.ApproverSetRevisionExport, errs []error) {
	obj, errs := a.GetObject(lib.ApproverSetRevisionType, id, nil)
	outObj, ok := obj.(*lib.ApproverSetRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetChangeRequest will try and retrieve a change request object from
// the server
func (a *Client) GetChangeRequest(id int64) (outobj *lib.ChangeRequestExport, errs []error) {
	obj, errs := a.GetObject(lib.ChangeRequestType, id, nil)
	outObj, ok := obj.(*lib.ChangeRequestExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// VerifyChangeRequest will attempt to verify that the change request
// was signed by one of the trust anchors an approver that has a chain
// of approvals leading back to a trust anchor. A bool indicating if the
// verification succeeded, a list of errors and the signed data are
// returned. If the object was not signed no data is returned.
func (a *Client) VerifyChangeRequest(id int64, revision lib.RegistrarObjectExport) (verified bool, errs []error, signedData []byte) {
	verified = false

	if id <= 0 {
		errMsg := fmt.Sprintf("Invalid Change Request ID %d", id)
		errs = append(errs, errors.New(errMsg))
		return
	}

	cr, crErrs := a.GetChangeRequest(id)
	a.log.Debugf("Getting Change Request %d", id)
	if len(crErrs) != 0 {
		errs = append(errs, crErrs...)
		return
	}

	for _, app := range cr.Approvals {
		if app.IsFinalApproval {
			signedByAnchor, data := a.TrustAnchor.IsSignedBy(app.Signature)

			if signedByAnchor {
				a.log.Debugf("CR %d was signed by trust anchor", id)
				var aa lib.ApprovalAttestationUnmarshal
				err := json.Unmarshal(data, &aa)
				if err != nil {
					errs = append(errs, err)
					return false, errs, data
				}

				// If the approval is an approval
				if aa.Action != lib.ActionApproved {
					errMsg := fmt.Sprintf("Change request %d was not approved", id)
					errs = append(errs, errors.New(errMsg))
					return false, errs, data
				}

				return true, errs, aa.ExportRev
			}
			as, getAppSetErrs := a.GetApproverSetAt(app.ApproverSetID, app.CreatedAt.Unix())
			if len(getAppSetErrs) != 0 {
				errs = append(errs, getAppSetErrs...)
				return
			}

			//asVerified, asE, asErrs := verifyApproverSet(cli, ta, as)
			asVerified, ase, asErrs := a.VerifyApproverSet(as)
			if len(asErrs) != 0 {
				errs = append(errs, asErrs...)
				return
			}
			if !asVerified {
				errMsg := fmt.Sprintf("Approver Set %d was not verified", app.ApproverSetID)
				errs = append(errs, errors.New(errMsg))
				return false, errs, data
			}
			signedByAnchor, data = ase.IsSignedBy(app.Signature)
			if !signedByAnchor {
				errs = append(errs, errors.New("Not signed by anchor"))
				return false, errs, data
			}

			var aa lib.ApprovalAttestationUnmarshal
			unmarshalErr := json.Unmarshal(data, &aa)
			if unmarshalErr != nil {
				errs = append(errs, unmarshalErr)
				return false, errs, data
			}

			// If the approval is an approval
			if aa.Action != lib.ActionApproved {
				errMsg := fmt.Sprintf("Change request %d was not approved", id)
				errs = append(errs, errors.New(errMsg))
				return false, errs, data
			}

			return true, errs, aa.ExportRev
		}
	}

	return
}

// GetApprovalObject will try and retrieve a domain object from the server
func (a *Client) GetApprovalObject(id int64) (outobj *lib.ApprovalExport, errs []error) {
	obj, errs := a.GetObject(lib.ApprovalType, id, nil)
	outObj, ok := obj.(*lib.ApprovalExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetObject will try to retrieve an object from the server
func (a *Client) GetObject(objectType string, id int64, lastUpdate *time.Time) (outObj lib.RegistrarObjectExport, errs []error) {
	var cacheLookupErr []error
	outObj, cacheLookupErr = a.cache.GetObject(objectType, id, lastUpdate)
	if cacheLookupErr == nil {
		a.log.Debugf("%s with id of %d fetched from cache", objectType, id)
		return
	}

	data, getErr := a.Get(fmt.Sprintf("/api/view/%s/%d", objectType, id))
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	err := a.cache.SaveObject(respObj)
	if err != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	return respObj.GetRegistrarObject()
}

// GetDomainAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetDomainAt(id int64, ts int64) (outobj *lib.DomainExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.DomainType, id, ts)
	outObj, ok := obj.(*lib.DomainExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetDomainRevisionAt will try and retrieve a domain object from the
// server at the given timestamp
func (a *Client) GetDomainRevisionAt(id int64, ts int64) (outobj *lib.DomainRevisionExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.DomainRevisionType, id, ts)
	outObj, ok := obj.(*lib.DomainRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetContactAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetContactAt(id int64, ts int64) (outobj *lib.ContactExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.ContactType, id, ts)
	outObj, ok := obj.(*lib.ContactExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetContactRevisionAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetContactRevisionAt(id int64, ts int64) (outobj *lib.ContactRevisionExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.ContactRevisionType, id, ts)
	outObj, ok := obj.(*lib.ContactRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetHostAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetHostAt(id int64, ts int64) (outobj *lib.HostExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.HostType, id, ts)
	outObj, ok := obj.(*lib.HostExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetHostRevisionAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetHostRevisionAt(id int64, ts int64) (outobj *lib.HostRevisionExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.HostRevisionType, id, ts)
	outObj, ok := obj.(*lib.HostRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetAPIUserAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetAPIUserAt(id int64, ts int64) (outobj *lib.APIUserExportFull, errs []error) {
	obj, errs := a.GetObjectAt(lib.APIUserType, id, ts)
	outObj, ok := obj.(*lib.APIUserExportFull)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetAPIUserRevisionAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetAPIUserRevisionAt(id int64, ts int64) (outobj *lib.APIUserRevisionExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.APIUserRevisionType, id, ts)
	outObj, ok := obj.(*lib.APIUserRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetApproverAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetApproverAt(id int64, ts int64) (outobj *lib.ApproverExportFull, errs []error) {
	obj, errs := a.GetObjectAt(lib.ApproverType, id, ts)
	outObj, ok := obj.(*lib.ApproverExportFull)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetApproverRevisionAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetApproverRevisionAt(id int64, ts int64) (outobj *lib.ApproverRevisionExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.ApproverRevisionType, id, ts)
	outObj, ok := obj.(*lib.ApproverRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetApproverSetAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetApproverSetAt(id int64, ts int64) (outobj *lib.ApproverSetExportFull, errs []error) {
	obj, errs := a.GetObjectAt(lib.ApproverSetType, id, ts)
	outObj, ok := obj.(*lib.ApproverSetExportFull)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetApproverSetRevisionAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetApproverSetRevisionAt(id int64, ts int64) (outobj *lib.ApproverSetRevisionExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.ApproverSetRevisionType, id, ts)
	outObj, ok := obj.(*lib.ApproverSetRevisionExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetChangeRequestAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetChangeRequestAt(id int64, ts int64) (outobj *lib.ChangeRequestExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.ChangeRequestType, id, ts)
	outObj, ok := obj.(*lib.ChangeRequestExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetApprovalObjectAt will try and retrieve a domain object from the server at
// the given timestamp
func (a *Client) GetApprovalObjectAt(id int64, ts int64) (outobj *lib.ApprovalExport, errs []error) {
	obj, errs := a.GetObjectAt(lib.ApprovalType, id, ts)
	outObj, ok := obj.(*lib.ApprovalExport)
	if ok {
		return outObj, errs
	}
	errs = append(errs, errors.New("Unable to parse object"))
	return
}

// GetObjectAt will try to retrieve an object from the server at a given
// unix timestamp
func (a *Client) GetObjectAt(objectType string, id int64, ts int64) (outObj lib.RegistrarObjectExport, errs []error) {
	var cacheLookupErr []error
	outObj, cacheLookupErr = a.cache.GetObjectAt(objectType, id, ts, a.ObjectDir)
	if cacheLookupErr == nil {
		a.log.Debugf("%s with id of %d fetched from cache", objectType, id)
		return
	}

	data, getErr := a.Get(fmt.Sprintf("/api/viewat/%s/%d/%d", objectType, id, ts))
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	err := a.cache.SaveObjectAt(respObj, id, ts)
	if err != nil {
		errs = append(errs, err)
		return
	}

	return respObj.GetRegistrarObject()
}

// GetAll will try to retrieve a list of all IDs for objects that
// are in an active state or require work to be done
func (a *Client) GetAll(objectType string) (ids []int64, hints map[int64]lib.APIRevisionHint, errs []error) {
	return a.GetIDList(fmt.Sprintf("/api/viewall/%s", objectType))
}

// GetIDList will try to retrieve a list of IDs for the provided object
// type using the url provided
func (a *Client) GetIDList(url string) (ids []int64, hints map[int64]lib.APIRevisionHint, errs []error) {
	hints = make(map[int64]lib.APIRevisionHint)

	data, getErr := a.Get(url)
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	switch respObj.MessageType {
	case lib.DomainIDListType:
		if respObj.DomainIDList != nil {
			if respObj.DomainHintList != nil {
				return *respObj.DomainIDList, buildHintMap(*respObj.DomainHintList), []error{}
			}
			return *respObj.DomainIDList, hints, []error{}
		}
	case lib.HostIDListType:
		if respObj.HostIDList != nil {
			if respObj.HostHintList != nil {
				return *respObj.HostIDList, buildHintMap(*respObj.HostHintList), []error{}
			}
			return *respObj.HostIDList, hints, []error{}
		}
	case lib.ContactIDListType:
		if respObj.ContactIDList != nil {
			if respObj.ContactHintList != nil {
				return *respObj.ContactIDList, buildHintMap(*respObj.ContactHintList), []error{}
			}
			return *respObj.ContactIDList, hints, []error{}
		}
	case lib.ErrorResponseType:
		errs = lib.StringsToErrs(respObj.Errors)
		return
	default:
		errs = append(errs, fmt.Errorf("Unknown object type %s", lib.ContactIDListType))
	}

	return
}

// buildHintMap will take a list of APIRevisionHints and generate a map of hints
// for that timestamp
func buildHintMap(hintsin []lib.APIRevisionHint) map[int64]lib.APIRevisionHint {
	retval := make(map[int64]lib.APIRevisionHint)
	for _, hint := range hintsin {
		retval[hint.ObjectID] = hint
	}
	return retval
}

// GetWork will try to retrieve a list of IDs for the provided object
// type that require work to be done
func (a *Client) GetWork(objectType string) (ids []int64, hints map[int64]lib.APIRevisionHint, errs []error) {
	token, tokenErrs := a.GetToken()
	if len(tokenErrs) != 0 {
		errs = append(errs, tokenErrs...)
		return
	}
	return a.GetIDList(fmt.Sprintf("/api/%s/getwork?csrf_token=%s", objectType, token))
}

// GetHostsWork will try to retrieve a list of host IDs that have work
// to be done
func (a *Client) GetHostsWork() (ids []int64, hints map[int64]lib.APIRevisionHint, errs []error) {
	return a.GetWork(lib.HostType)
}

// GetContactsWork will try to retrieve a list of contacts IDs that
// have work to be done
func (a *Client) GetContactsWork() (ids []int64, hints map[int64]lib.APIRevisionHint, errs []error) {
	return a.GetWork(lib.ContactType)
}

// GetDomainsWork will try to retrieve a list of domains IDs that
// have work to be done
func (a *Client) GetDomainsWork() (ids []int64, hints map[int64]lib.APIRevisionHint, errs []error) {
	return a.GetWork(lib.DomainType)
}

// GetSig will try and retireve the signature associated with an
// approval given an approval ID. If the approval is not signed or
// another error occurs an error will be retured.
func (a *Client) GetSig(approvalID int64) (sigBytes []byte, errs []error) {
	data, getErr := a.Get(fmt.Sprintf("/api/approval/%d/downloadsig", approvalID))
	if getErr != nil {
		errs = []error{getErr}
		return
	}

	respObj := lib.APIResponse{}
	unmarshalErr := json.Unmarshal(data, &respObj)
	if unmarshalErr != nil {
		errs = append(errs, unmarshalErr)
		return
	}

	if respObj.MessageType == lib.SignatureDownloadType && respObj.Signature != nil {
		sigBytes = respObj.Signature.Signature
	} else if respObj.MessageType == lib.ErrorResponseType {
		for _, err := range respObj.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		errs = append(errs, fmt.Errorf("Expected a message type of %s, got %s", lib.SignatureDownloadType, respObj.MessageType))
	}

	return
}

// GetWHOIS creates an objects.WHOIS object that can be serialized and
// installed on a WHOIS server
func (a *Client) GetWHOIS(defaultContactID int64) (objects.WHOIS, []error) {
	data := objects.NewWHOIS()
	var errs []error
	requiredContacts := make(map[int64]int64)
	requiredHosts := make(map[int64]int64)
	requiredContacts[defaultContactID] = 1

	workDomains, _, domainsErrs := a.GetAll(lib.DomainType)
	errs = append(errs, domainsErrs...)
	for _, workDomain := range workDomains {
		domain, err := a.GetDomain(workDomain)
		if err == nil {
			domAddErr := data.AddDomain(domain)
			if domAddErr != nil {
				errs = append(errs, domAddErr)
			} else {
				for _, host := range domain.CurrentRevision.Hostnames {
					if val, ok := requiredHosts[host.ID]; ok {
						requiredHosts[host.ID] = val + 1
					} else {
						requiredHosts[host.ID] = 1
					}
				}

				for _, contact := range []lib.ContactExportShort{domain.CurrentRevision.DomainAdminContact, domain.CurrentRevision.DomainBillingContact, domain.CurrentRevision.DomainRegistrant, domain.CurrentRevision.DomainTechContact} {
					if val, ok := requiredContacts[contact.ID]; ok {
						requiredContacts[contact.ID] = val + 1
					} else {
						requiredContacts[contact.ID] = 1
					}
				}
			}
		}
	}

	workContacts, _, contactsErrs := a.GetAll(lib.ContactType)
	errs = append(errs, contactsErrs...)
	for _, contactID := range workContacts {
		if val, ok := requiredContacts[contactID]; ok {
			requiredContacts[contactID] = val + 1
		} else {
			requiredContacts[contactID] = 1
		}
	}
	for contactID := range requiredContacts {
		contact, err := a.GetContact(contactID)
		if err == nil {
			conAddErr := data.AddContact(contact)
			if conAddErr != nil {
				errs = append(errs, conAddErr)
			}
			if contactID == defaultContactID {
				defConErr := data.SetDefaultContact(contact)
				if defConErr != nil {
					errs = append(errs, defConErr)
				}
			}
		}
	}

	workHosts, _, hostsErrs := a.GetAll(lib.HostType)
	errs = append(errs, hostsErrs...)
	for _, hostID := range workHosts {
		if val, ok := requiredHosts[hostID]; ok {
			requiredHosts[hostID] = val + 1
		} else {
			requiredHosts[hostID] = 1
		}
	}
	for hostID := range requiredHosts {
		host, err := a.GetHost(hostID)
		if err == nil {
			hosAddErr := data.AddHost(host)
			if hosAddErr != nil {
				errs = append(errs, hosAddErr)
			}
		}
	}

	data.LastUpdate = time.Now()

	return data, errs
}

// PushEPPActionLog will attempt to push an epp action to the server and if it
// fails, the errors will be returned
func (a *Client) PushEPPActionLog(action lib.EPPAction) (errs []error) {
	action.RunID = a.currentRunID

	token, errs := a.GetToken()
	if len(errs) != 0 {
		return errs
	}
	dataBuffer, marshalErr := json.MarshalIndent(action, "", "  ")
	if marshalErr != nil {
		errs = append(errs, marshalErr)
		return
	}

	reader := bytes.NewReader(dataBuffer)

	url := fmt.Sprintf("/api/appendepplog?csrf_token=%s", token)
	resp, postErr := a.Post(url, "application/json", reader)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}

	return
}

// PushHostIPAllowList will attempt to uplaod the IP allow list provided to
// the registrar server. If an error occurs, it will be returned
func (a *Client) PushHostIPAllowList(ips []string, token string) (errs []error) {
	dataBuffer, marshalErr := json.MarshalIndent(ips, "", "  ")
	if marshalErr != nil {
		errs = append(errs, marshalErr)
		return
	}

	reader := bytes.NewReader(dataBuffer)

	url := fmt.Sprintf("/api/sethostipallowlist?csrf_token=%s", token)
	resp, postErr := a.Post(url, "application/json", reader)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}

	return
}

// PushProtectedDomainList will attempt to uplaod the protected domain list
// provided to the registrar server. If an error occurs, it will be returned
func (a *Client) PushProtectedDomainList(domains []string, token string) (errs []error) {
	dataBuffer, marshalErr := json.MarshalIndent(domains, "", "  ")
	if marshalErr != nil {
		errs = append(errs, marshalErr)
		return
	}

	reader := bytes.NewReader(dataBuffer)

	url := fmt.Sprintf("/api/setprotecteddomainlist?csrf_token=%s", token)
	resp, postErr := a.Post(url, "application/json", reader)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}

	return
}

// PushSig will try to push a signature associated with an approval to
// the server.
func (a *Client) PushSig(approvalID int64, sigData []byte, token string) (errs []error) {
	dataBuffer, marshalErr := json.MarshalIndent(lib.GenerateSignatureUpload(sigData), "", "  ")
	if marshalErr != nil {
		errs = append(errs, marshalErr)
		return
	}

	reader := bytes.NewReader(dataBuffer)

	url := fmt.Sprintf("/api/approval/%d/%s?csrf_token=%s", approvalID, lib.SignatureUploadType, token)
	resp, postErr := a.Post(url, "application/json", reader)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}

	return
}

// PushInfoEPP will try to push the EPP Info response associated with a
// registry object
func (a *Client) PushInfoEPP(objectType string, objectID int64, info *epp.Response) (errs []error) {
	dataBuffer, marshalErr := json.MarshalIndent(info, "", "  ")
	if marshalErr != nil {
		errs = append(errs, marshalErr)
		return
	}

	token, tokenErrs := a.GetToken()
	if len(tokenErrs) != 0 {
		errs = append(errs, tokenErrs...)
		return
	}

	reader := bytes.NewReader(dataBuffer)

	url := fmt.Sprintf("/api/%s/%d/%s?csrf_token=%s", objectType, objectID, lib.ActionUpdateEPPInfo, token)
	resp, postErr := a.Post(url, "application/json", reader)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}

	return nil
}

// UnsetEPPCheck will try to unset the check_required field for the registry
// object
func (a *Client) UnsetEPPCheck(objectType string, objectID int64) (errs []error) {
	token, tokenErrs := a.GetToken()
	if len(tokenErrs) != 0 {
		errs = append(errs, tokenErrs...)
		return
	}

	url := fmt.Sprintf("/api/%s/%d/%s?csrf_token=%s", objectType, objectID, lib.ActionUpdateEPPCheckRequired, token)
	resp, postErr := a.Post(url, "", nil)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}

	return nil
}

// RequestNewEPPRunID will request a new EPP Run ID from he registrar server
// and return it. If an error occurs getting a new ID, it will be returned
func (a *Client) RequestNewEPPRunID() (id int64, errs []error) {
	a.logDebug("Requesting NewEPPRunID")
	token, tokenErrs := a.GetToken()
	if len(tokenErrs) != 0 {
		errs = append(errs, tokenErrs...)
		return
	}

	url := fmt.Sprintf("/api/startepprun?csrf_token=%s", token)
	resp, postErr := a.Post(url, "", nil)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}

	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}

		if respObj.EPPRunID != nil {
			a.currentRunID = *respObj.EPPRunID
			return *respObj.EPPRunID, nil
		}
	}
	errs = append(errs, errors.New("No response from server"))
	return
}

// EndEPPRun is used to end an epp run. If the process of ending the run fails
// an error will be returned.
func (a *Client) EndEPPRun(id int64) (errs []error) {
	token, tokenErrs := a.GetToken()
	if len(tokenErrs) != 0 {
		errs = append(errs, tokenErrs...)
		return
	}

	url := fmt.Sprintf("/api/endepprun/%d?csrf_token=%s", id, token)
	resp, postErr := a.Post(url, "", nil)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}

	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}
	return

}

// GetEncryptedPassphrase will attempt to get the encrypted passphrase from the
// server based on the username provided. If errors occur when trying to locate
// the passphrase, they will be returned.
func (a *Client) GetEncryptedPassphrase(username string) (encryptedPassphrase string, errs []error) {
	token, tokenErrs := a.GetToken()
	if len(tokenErrs) != 0 {
		errs = append(errs, tokenErrs...)
		return
	}

	url := fmt.Sprintf("/api/epppassphrase/%s?csrf_token=%s", username, token)
	data, getErr := a.Get(url)
	if getErr != nil {
		errs = append(errs, getErr)
		return
	}

	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
		if respObj.EPPPassphrase != nil {
			encryptedPassphrase = *respObj.EPPPassphrase
			return
		}
	}
	errs = append(errs, errors.New("No encrypted passphrase found"))
	return
}

// SetEncryptedPassphrase will attempt to se the encrypted passphrase for the
// username provided. If the process results in errors, they will be returned
func (a *Client) SetEncryptedPassphrase(username string, encryptedPassphrase string) (errs []error) {
	token, tokenErrs := a.GetToken()
	if len(tokenErrs) != 0 {
		errs = append(errs, tokenErrs...)
		return
	}

	reader := bytes.NewReader([]byte(encryptedPassphrase))

	url := fmt.Sprintf("/api/epppassphrase/%s?csrf_token=%s", username, token)
	resp, postErr := a.Post(url, "application/base64", reader)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}

	return nil
}

// PushContactRegistryID will try to push the RegistryID selected for
// the contact to the server.
func (a *Client) PushContactRegistryID(objectID int64, token string, registryID string) (errs []error) {
	dataBuffer, marshalErr := json.MarshalIndent(registryID, "", "  ")
	if marshalErr != nil {
		errs = append(errs, marshalErr)
		return
	}

	reader := bytes.NewReader(dataBuffer)

	url := fmt.Sprintf("/api/contact/%d/setContactID?csrf_token=%s", objectID, token)
	resp, postErr := a.Post(url, "application/json", reader)
	if postErr != nil {
		errs = append(errs, postErr)
		return
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		errs = append(errs, readErr)
		return
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		respObj := lib.APIResponse{}
		unmarshalErr := json.Unmarshal(data, &respObj)
		if unmarshalErr != nil {
			errs = append(errs, unmarshalErr)
			return
		}

		if respObj.MessageType == lib.ErrorResponseType {
			for _, err := range respObj.Errors {
				errs = append(errs, errors.New(err))
			}
			return
		}
	}
	return
}

// logError is an internal function to the Client object that will
// attempt to log an error if the log attribute is set
func (a *Client) logError(message string) {
	if a.log != nil {
		a.log.Error(message)
	}
}

// logDebug is an internal function to the Client object that will
// attempt to log a debug message if the log attribute is set
func (a *Client) logDebug(message string) {
	if a.log != nil {
		a.log.Debug(message)
	}
}

// logErrorf is an internal function to the Client object that will
// attempt to log an error if the log attribute is set using the format
// and args passed
// func (a *Client) logErrorf(format string, args ...interface{}) {
// 	if a.log != nil {
// 		a.log.Errorf(format, args)
// 	}
// }

// logDebugf is an internal function to the Client object that will
// attempt to log a debug message if the log attribute is set using the format
// and args passed
func (a *Client) logDebugf(format string, args ...interface{}) {
	if a.log != nil {
		a.log.Debugf(format, args)
	}
}

// SetLogger is used to set or reset the logger that the Client object
// will call
func (a *Client) SetLogger(logger *logging.Logger) {
	a.log = logger
}

// loadCert will attempt to load a client certificate using the values
// passed
func loadCert(clientCertPath string, clientKeyPath string, keychainConf keychain.Conf) (cert tls.Certificate, err error) {
	clientCertBytes, clientCertErr := os.ReadFile(clientCertPath)
	if clientCertErr != nil {
		err = clientCertErr
		return
	}

	clientKeyBytes, clientKeyErr := os.ReadFile(clientKeyPath)
	if clientKeyErr != nil {
		err = clientKeyErr
		return
	}

	clientKeyBlock, _ := pem.Decode(clientKeyBytes)
	if isBlockEncrypted(clientKeyBlock) {

		var passBytes []byte

		if len(keychainConf.MacKeychainAccount) != 0 && keychainConf.MacKeychainEnabled {
			var keychainPassErr error
			passBytes, keychainPassErr = keychain.GetKeyChainPassphrase(keychainConf)
			if keychainPassErr != nil {
				return cert, keychainPassErr
			}
		}
		if len(passBytes) == 0 {
			passBytes, err = getPasswordFromTerminal("Client Cert Key Password: ")
			if err != nil {
				return
			}
		}

		var decryptErr error
		clientKeyBytes, decryptErr = x509.DecryptPEMBlock(clientKeyBlock, passBytes)
		if decryptErr != nil {
			err = decryptErr
			return
		}

		key, parseErr := x509.ParsePKCS1PrivateKey(clientKeyBytes)
		if parseErr != nil {
			err = parseErr
			return
		}

		cert, err = x509KeyPair(clientCertBytes, key)
	} else {
		cert, err = tls.X509KeyPair(clientCertBytes, clientKeyBytes)
	}
	return
}

// Mostly taken from the crypto/tls X509KeyPair function but removed the
// check for key and cert matching
func x509KeyPair(certPEMBlock []byte, privKey *rsa.PrivateKey) (cert tls.Certificate, err error) {
	var certDERBlock *pem.Block
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		}
	}

	if len(cert.Certificate) == 0 {
		err = errors.New("crypto/tls: failed to parse certificate PEM data")
		return
	}

	cert.PrivateKey = privKey

	return
}

// isBlockEncrypted is used to determine if a pem block is an encrypted
// block, mostly used to determine if a decryption operation has to
// occur before using the data inside.
func isBlockEncrypted(data *pem.Block) bool {
	for _, v := range data.Headers {
		if strings.Contains(v, "ENCRYPTED") {
			return true
		}
	}
	return false
}

// getPasswordFromTerminal will prompt an end user for a password
// and return a byte string of the entered password
func getPasswordFromTerminal(prompt string) (pass []byte, err error) {
	fmt.Print(prompt)
	pass, err = gopass.GetPasswd()
	return
}
