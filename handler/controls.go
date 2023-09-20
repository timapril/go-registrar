package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/lib"
)

// LogEPPAction is used to log an epp action that was taken. If an error occurs
// while processing the request, it will be returned.
func LogEPPAction(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	ctx.LogRequest(logging.INFO, request.URL.String(), "LogEPPAction", ctx.db.GetCacheStatsLog())
	return lib.AppendEPPActionLog(ctx.db, request)
}

// StartEPPRun is used to create a new run ID for an EPP run and return the
// id to the client so it can be used as a unique key for the client transaction
// ids that are sent to the registry
func StartEPPRun(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	resp := lib.APIResponse{}
	id, createErr := lib.CreateEPPRunRecord(ctx.GetDB(), ctx.GetUsername())
	ctx.LogRequest(logging.INFO, request.URL.String(), "StartEPPRun", fmt.Sprintf("%s - id %d", ctx.db.GetCacheStatsLog(), id))
	if createErr == nil {
		resp.EPPRunID = &id
		return w.SendAPIResponse(resp)
	}
	return createErr
}

// EndEPPRun is used to mark an EPP run as completed in the database
func EndEPPRun(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	id, parseErr := strconv.ParseInt(ctx.routeVars["id"], 10, 64)

	ctx.LogRequest(logging.INFO, request.URL.String(), "EndEPPRun", fmt.Sprintf("%s - id %d", ctx.db.GetCacheStatsLog(), id))

	if parseErr != nil {
		return parseErr
	}

	return lib.EndEPPRunRecord(ctx.GetDB(), id, ctx.GetUsername())
}

// EPPPassphrase is used to handle requests to get or save the encrypted EPP
// user passphrase.
func EPPPassphrase(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	username := ctx.routeVars["username"]
	if request.Method == http.MethodGet {
		ctx.LogRequest(logging.INFO, request.URL.String(), "EPPPassphrase", fmt.Sprintf("%s - get - username %s", ctx.db.GetCacheStatsLog(), username))
		encPassphrase, err := lib.GetEPPEncryptedPassphrase(ctx.GetDB(), username)
		if err != nil {
			return err
		}
		resp := lib.APIResponse{}
		resp.EPPPassphrase = &encPassphrase
		return w.SendAPIResponse(resp)
	}
	if request.Method == http.MethodPost {
		ctx.LogRequest(logging.INFO, request.URL.String(), "EPPPassphrase", fmt.Sprintf("%s - set - username %s", ctx.db.GetCacheStatsLog(), username))
		defer request.Body.Close()
		data, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			return readErr
		}
		saveErr := lib.SetEPPEncryptedPassphrase(ctx.GetDB(), username, string(data))
		if saveErr != nil {
			return saveErr
		}
		return nil
	}
	return fmt.Errorf("Unsupported HTTP Method %s", request.Method)
}

// GetHostIPAllowList is used to retrieve the list of IPs that are contained in
// the host ip allow list if one is set. If an error occurs while trying to
// obtain the list of IPs, it will be returned
func GetHostIPAllowList(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	return getControlList(w, request, ctx, lib.HostIPAllowList)
}

// SetHostIPAllowList is used to set the HostIPAllowList in the registrar
// application. If an error occurs while trying to set the IP allow list it will
// be returned
func SetHostIPAllowList(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	return setControlList(w, request, ctx, lib.HostIPAllowList)
}

// GetProtectedDomainList is used to retrieve the list of proteced domains that
// in the database. If an error occurs trying to obtain the list, the error will
// be returned
func GetProtectedDomainList(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	return getControlList(w, request, ctx, lib.ProtectedDomainList)
}

// SetProtectedDomainList is used to set the Protected Domain list in the
// registrar application. If an error occurs while trying to set the IP
// allowlist it will be returned
func SetProtectedDomainList(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	return setControlList(w, request, ctx, lib.ProtectedDomainList)
}

// GetControlList combines the logic related to returning a control list to the
// the user. This function works for IPs and Hosts
func getControlList(w ResponseWriter, request *http.Request, ctx apiContext, listType string) (err error) {
	resp := lib.APIResponse{}
	resp.MessageType = listType

	switch listType {
	case lib.ProtectedDomainList:
		domains, getErr := lib.GetProtectedDomainList(ctx.db)
		if getErr != nil {
			return getErr
		}
		resp.ProtectedDomainList = &domains
	case lib.HostIPAllowList:
		ips, getErr := lib.GetIPAllowList(ctx.db)
		if getErr != nil {
			return getErr
		}
		resp.HostIPAllowList = &ips
	default:
		return fmt.Errorf("Unknown control type %s", listType)
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "getControlList", fmt.Sprintf("%s %s", ctx.db.GetCacheStatsLog(), listType))

	return w.SendAPIResponse(resp)
}

func setControlList(w ResponseWriter, request *http.Request, ctx apiContext, listType string) (err error) {
	data, dataErr := io.ReadAll(request.Body)
	if dataErr != nil {
		return dataErr
	}

	values := []string{}
	unMarshalErr := json.Unmarshal(data, &values)
	if unMarshalErr != nil {
		return unMarshalErr
	}

	switch listType {
	case lib.ProtectedDomainList:
		err = lib.SetProtectedDomainList(ctx.db, values, ctx.username)
	case lib.HostIPAllowList:
		err = lib.SetIPAllowList(ctx.db, values, ctx.username)
	default:
		return fmt.Errorf("Unknown control type %s", listType)
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "setControlList", fmt.Sprintf("%s %s", ctx.db.GetCacheStatsLog(), listType))

	return err
}

// DomainNameToID attempt to convert a domain name into a domain ID. If an error
// occurs, the error will be returned
func DomainNameToID(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	resp := lib.APIResponse{}
	resp.MessageType = lib.DomainIDListType

	domainID, err := lib.GetDomainIDFromDomainName(ctx.db, strings.ToUpper(ctx.routeVars["domainname"]))
	if err != nil {
		return err
	}

	resp.DomainIDList = &[]int64{}
	*resp.DomainIDList = append(*resp.DomainIDList, domainID)

	return w.SendAPIResponse(resp)
}
