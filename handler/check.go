package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/lib"
)

// CheckHandlerWeb will check that the object is allowed to be moved
// into the check required state and will display the form to allow
// the user to set the check required bit
func CheckHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	if request.Method == http.MethodGet {
		obj, err := objFromRoute(ctx)
		if err != nil {
			return err
		}
		template := ""
		if obj.GetType() != lib.DomainType && obj.GetType() != lib.HostType && obj.GetType() != lib.ContactType {
			return fmt.Errorf("Hold is not supported for %s objects", obj.GetType())
		}
		template = fmt.Sprintf("%scheck", obj.GetType())

		ctx.LogRequest(logging.INFO, request.URL.String(), "CheckHandlerWeb", ctx.db.GetCacheStatsLog())

		return displayObjectByTemplate(w, ctx, obj, template)
	}
	ctx.LogRequest(logging.ERROR, request.URL.String(), "CheckHandlerWeb", fmt.Sprintf("%s %s", ctx.db.GetCacheStatsLog(), "Unsupported HTTP Method"))

	return errors.New("Unsupported HTTP Method")
}

// CheckUpdateHandlerWeb handles the check required update request for
// and object. The function will verify that the user is allowed to
// make the change (is an admin) and then will set the check required
// bit on the object
func CheckUpdateHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	if request.Method == http.MethodPost {
		isAdmin, err := lib.IsAdminUser(ctx.GetUsername(), ctx.GetDB())
		if err != nil {
			return err
		}
		if !isAdmin {
			return errors.New("This action may only be taken by an admin")
		}

		obj, err := objFromRoute(ctx)
		if err != nil {
			return err
		}
		redirectTo := ""

		checkStatusRaw := request.FormValue("check_status")
		if checkStatusRaw != "true" {
			return errors.New("Check can only be set through this interface")
		}

		redirectTo = fmt.Sprintf("/check/%s/%d", obj.GetType(), obj.GetID())
		switch obj.GetType() {
		case lib.DomainType:
			dom := obj.(*lib.Domain)
			dom.CheckRequired = true
		case lib.HostType:
			hos := obj.(*lib.Host)
			hos.CheckRequired = true
		case lib.ContactType:
			con := obj.(*lib.Contact)
			con.CheckRequired = true
		default:
			return fmt.Errorf("Check is not supported for %s objects", obj.GetType())
		}

		saveErr := ctx.GetDB().Save(obj)
		if saveErr != nil {
			return saveErr
		}

		ctx.LogRequest(logging.INFO, request.URL.String(), "CheckUpdateHandlerWeb", ctx.db.GetCacheStatsLog())

		return w.Redirect(request, redirectTo, 302)
	}

	ctx.LogRequest(logging.ERROR, request.URL.String(), "CheckUpdateHandlerWeb", fmt.Sprintf("%s %s", ctx.db.GetCacheStatsLog(), "Unsupported HTTP Method"))

	return errors.New("Unsupported HTTP Method")
}
