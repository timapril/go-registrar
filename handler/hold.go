package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/lib"
)

// HoldUpdateHandlerWeb handles updates to object holds. The function
// will first check to see if the user is allowed to make the change
// (admin) and then that the object is allowed to be placed on hold.
func HoldUpdateHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
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

		holdStatus := false
		holdStatusRaw := request.FormValue("hold_status")
		switch strings.ToLower(holdStatusRaw) {
		case "true":
			holdStatus = true
		case "false":
			holdStatus = false
		default:
			return fmt.Errorf("unknown hold status of %s", holdStatusRaw)
		}
		holdReason := request.FormValue("hold_reason")

		switch obj.GetType() {
		case lib.DomainType:
			redirectTo = fmt.Sprintf("/hold/%s/%d", lib.DomainType, obj.GetID())
			dom := obj.(*lib.Domain)
			domUpdateErr := dom.UpdateHoldStatus(holdStatus, holdReason, ctx.GetUsername())
			if domUpdateErr != nil {
				return domUpdateErr
			}
		case lib.HostType:
			redirectTo = fmt.Sprintf("/hold/%s/%d", lib.HostType, obj.GetID())
			hos := obj.(*lib.Host)
			hosUpdateErr := hos.UpdateHoldStatus(holdStatus, holdReason, ctx.GetUsername())
			if hosUpdateErr != nil {
				return hosUpdateErr
			}
		case lib.ContactType:
			redirectTo = fmt.Sprintf("/hold/%s/%d", lib.ContactType, obj.GetID())
			con := obj.(*lib.Contact)
			conUpdateErr := con.UpdateHoldStatus(holdStatus, holdReason, ctx.GetUsername())
			if conUpdateErr != nil {
				return conUpdateErr
			}
		default:
			return fmt.Errorf("Hold is not supported for %s objects", obj.GetType())
		}

		saveErr := ctx.GetDB().Save(obj)
		if saveErr != nil {
			return saveErr
		}
		ctx.LogRequest(logging.INFO, request.URL.String(), "HoldUpdateHandlerWeb", ctx.db.GetCacheStatsLog())

		return w.Redirect(request, redirectTo, 302)
	}

	ctx.LogRequest(logging.ERROR, request.URL.String(), "HoldUpdateHandlerWeb", fmt.Sprintf("%s %s", ctx.db.GetCacheStatsLog(), "Unsupported HTTP Method"))

	return errors.New("Unsupported HTTP Method")
}

func displayObjectByTemplate(w ResponseWriter, ctx webContext, obj lib.RegistrarObject, templateName string) error {
	username := ctx.GetUsername()
	page, err := obj.GetPage(ctx.GetDB(), username, ctx.GetEmail())
	if err != nil {
		return err
	}

	return w.DisplayTemplate(templateName, page, username)
}
