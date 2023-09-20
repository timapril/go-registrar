package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/op/go-logging"
	"github.com/timapril/go-registrar/lib"
)

// WebHandlerFunc is a function definition that web functions must
// implement to be handled by the web handler factory
type WebHandlerFunc func(ResponseWriter, *http.Request, webContext) error

// newWebContext creates a web context for the given app and request.
func newWebContext(request *http.Request, conf lib.Config, db *lib.DBCache, logger *logging.Logger, liveness *LivenessCheckFunc) (*webContext, error) {
	username, err := lib.GetRemoteUser(request)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", NoUsernameError, err)
	}

	email, err := lib.GetRemoteUserEmail(request, conf)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", NoEmailError, err)
	}

	ctx := &webContext{
		username:  username,
		email:     email,
		routeVars: mux.Vars(request),
		db:        db,
		conf:      conf,
		logger:    logger,
		startTime: time.Now(),
		liveness:  liveness,
	}

	return ctx, nil
}
