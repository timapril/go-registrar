package handler

import (
	"fmt"
	"net/http"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/lib"
)

// TakeActionHandlerWeb is used to handle the take action request for
// web based requests.
func TakeActionHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	errs := takeAction(w, request, ctx, lib.RemoteUserAuthType)
	if len(errs) > 0 {
		ctx.LogRequest(logging.ERROR, request.URL.String(), "TakeActionHandlerWeb", fmt.Sprintf("%s %s", ctx.db.GetCacheStatsLog(), errs[0].Error()))

		return w.DisplayErrors(request.Referer(), errs, ctx.GetUsername())
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "TakeActionHandlerWeb", ctx.db.GetCacheStatsLog())

	return nil
}

// TakeActionHandlerAPI is used to handle the take action request for
// API based requests.
func TakeActionHandlerAPI(w ResponseWriter, request *http.Request, ctx apiContext) error {
	errs := takeAction(w, request, ctx, lib.CertAuthType)
	if len(errs) > 0 {
		ctx.LogRequest(logging.ERROR, request.URL.String(), "TakeActionHandlerAPI", fmt.Sprintf("%s %s", ctx.db.GetCacheStatsLog(), errs[0].Error()))

		return w.SendAPIErrors(errs)
	}
	ctx.LogRequest(logging.INFO, request.URL.String(), "TakeActionHandlerAPI", ctx.db.GetCacheStatsLog())

	return nil
}

func takeAction(w ResponseWriter, request *http.Request, ctx Context, auth lib.AuthType) []error {
	obj, err := objFromRoute(ctx)
	if err != nil {
		return []error{err}
	}

	actionName, err := getRouteVar(ctx, "action")
	if err != nil {
		return []error{err}
	}

	return obj.TakeAction(w, request, ctx.GetDB(), actionName, true, auth, ctx.GetConf())
}
