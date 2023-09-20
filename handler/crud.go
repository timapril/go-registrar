package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/lib"
)

// ViewHandlerWeb is used to display an object's information using the
// templates provided to the factory
func ViewHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	obj, err := objFromRoute(ctx)
	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "ViewHandlerWeb", ctx.db.GetCacheStatsLog())

	return displayObject(w, request.URL.String(), ctx, obj)
}

// ViewHandlerAPI is used to return an API request to get an object's
// information in a json serialized format
func ViewHandlerAPI(w ResponseWriter, request *http.Request, ctx apiContext) error {
	obj, err := objFromRoute(ctx)
	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "ViewHandlerAPI", ctx.db.GetCacheStatsLog())

	return w.SendAPIResponse(lib.GenerateObjectResponse(obj.GetExportVersion()))
}

// ViewAtHandlerAPI is used to serialize a JSON form of the object for
// api requests with a version of the object from a specific point in
// time
func ViewAtHandlerAPI(w ResponseWriter, request *http.Request, ctx apiContext) error {
	db := ctx.GetDB()

	obj, err := objFromRoute(ctx)
	if err != nil {
		return err
	}

	objTSRaw, err := getRouteVar(ctx, ObjTimeStampParam)
	if err != nil {
		return err
	}

	objTS, err := strconv.ParseInt(objTSRaw, 10, 64)
	if err != nil {
		return err
	}

	retObj, err := obj.GetExportVersionAt(db, objTS)
	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "ViewAtHandlerAPI", ctx.db.GetCacheStatsLog())

	return w.SendAPIResponse(lib.GenerateObjectResponse(retObj))
}

// UpdateHandlerWeb handles an update request from the web interface
// and will redirect the user to the object's page when the action is
// completed assuming no error was encountered
func UpdateHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	db := ctx.GetDB()

	objType, err := getRouteVar(ctx, ObjTypeParam)
	if err != nil {
		return err
	}

	form := url.Values{
		ObjTypeParam: {objType},
		ObjIDParam:   {request.FormValue(ObjIDParam)},
	}

	obj, err := objFromForm(db, form)
	if err != nil {
		return err
	}

	err = obj.ParseFromFormUpdate(request, db, ctx.GetConf())
	if err != nil {
		return err
	}

	err = db.Save(obj)
	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "UpdateHandlerWeb", ctx.db.GetCacheStatsLog())

	return w.Redirect(request, viewLink(obj), http.StatusFound)
}

// SaveHandlerWeb handles the save action for all objects that implement
// the RegistrarObject interface.
func SaveHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	db := ctx.GetDB()

	obj, err := blankObjFromRoute(ctx)
	if err != nil {
		return err
	}

	err = obj.ParseFromForm(request, db)
	if err != nil {
		return err
	}

	err = db.Save(obj)
	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "SaveHandlerWeb", ctx.db.GetCacheStatsLog())

	return w.Redirect(request, viewLink(obj), http.StatusFound)
}

// NewHandlerWeb handles the new action for all objects that
// implement the RegistrarObject interface.
func NewHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	obj, err := blankObjFromRoute(ctx)
	if err != nil {
		return err
	}

	return displayObject(w, request.URL.String(), ctx, obj)
}

// ViewAllHandlerWeb will generate a page listing all objects for an
// object type
func ViewAllHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	obj, err := blankObjFromRoute(ctx)
	if err != nil {
		return err
	}

	username := ctx.GetUsername()
	templateName := fmt.Sprintf("%ss", obj.GetType())
	page, err := obj.GetAllPage(ctx.GetDB(), username, ctx.GetEmail())
	if err != nil {
		return err
	}
	ctx.LogRequest(logging.INFO, request.URL.String(), "ViewAllHandlerWeb", ctx.db.GetCacheStatsLog())

	return w.DisplayTemplate(templateName, page, username)
}

// ViewAllHandlerAPI will generate a JSON respons listing all of the
// object IDs for a given object type
func ViewAllHandlerAPI(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	db := ctx.GetDB()

	objType, err := getRouteVar(ctx, ObjTypeParam)
	if err != nil {
		return err
	}

	var idList []int64
	var revisions []lib.APIRevisionHint
	switch objType {
	case lib.DomainType:
		idList, revisions, err = lib.GetAllDomains(db)
	case lib.HostType:
		idList, revisions, err = lib.GetAllHosts(db)
	case lib.ContactType:
		idList, revisions, err = lib.GetAllContacts(db)
	}

	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "ViewAllHandlerAPI", ctx.db.GetCacheStatsLog())

	return w.SendAPIResponse(lib.GenerateIDList(objType, idList, revisions))
}

// GetHostNames will generate an api response with a map of hostnames to ids
func GetHostNames(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	db := ctx.GetDB()

	hostnames, err := lib.GetAllHostNamess(db)
	if err != nil {
		return err
	}

	resp := lib.APIResponse{}
	resp.MessageType = lib.HostnameListType
	resp.HostnamesMap = &hostnames

	ctx.LogRequest(logging.INFO, request.URL.String(), "GetHostNamess", ctx.db.GetCacheStatsLog())

	return w.SendAPIResponse(resp)
}

func displayObject(w ResponseWriter, url string, ctx webContext, obj lib.RegistrarObject) error {
	username := ctx.GetUsername()
	page, err := obj.GetPage(ctx.GetDB(), username, ctx.GetEmail())
	if err != nil {
		return err
	}
	templateName := obj.GetType()

	ctx.LogRequest(logging.INFO, url, "displayObject", ctx.db.GetCacheStatsLog())

	return w.DisplayTemplate(templateName, page, username)
}

func viewLink(obj lib.RegistrarObject) string {
	return fmt.Sprintf("/view/%s/%d", obj.GetType(), obj.GetID())
}
