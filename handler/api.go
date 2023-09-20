package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/op/go-logging"
	"github.com/timapril/go-registrar/lib"
)

// APIHandlerFunc is an function definition that API functions must
// implement to be handled by the API handler factory.
type APIHandlerFunc func(ResponseWriter, *http.Request, apiContext) error

func newAPIContext(request *http.Request, conf lib.Config, db *lib.DBCache, logger *logging.Logger, liveness *LivenessCheckFunc) (*apiContext, error) {
	user, err := lib.GetAPIUser(request, db, conf)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", FailedToLoadUserError, err)
	}

	ctx := &apiContext{
		username:  user.GetCertName(),
		user:      user,
		routeVars: mux.Vars(request),
		conf:      conf,
		db:        db,
		logger:    logger,
		startTime: time.Now(),
		liveness:  liveness,
	}

	return ctx, nil
}

// RequireAdminAPIUser will ensure that the requests are being made by an admin
// API user.
func RequireAdminAPIUser(w ResponseWriter, request *http.Request, ctx apiContext) error {
	user := ctx.GetAPIUser()
	if user == nil {
		return errors.New("Unable to locate user")
	}

	isAdmin, adminErr := user.IsAdmin(ctx.db)
	if adminErr != nil {
		return adminErr
	}

	if !isAdmin {
		return errors.New("Admin User Required")
	}

	return nil
}
