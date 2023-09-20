package handler

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/op/go-logging"
	"github.com/timapril/go-registrar/lib"
)

// NoAuthWebHandlerFunc is a function definition that web functions that do not
// require authentication implement to be handled by the no auth web handler
// factory
type NoAuthWebHandlerFunc func(ResponseWriter, *http.Request, noAuthWebContext) error

// newNoAuthWebContext creates a no auth web context for the given app request
func newNoAuthWebContext(request *http.Request, conf lib.Config, db *lib.DBCache, logger *logging.Logger, liveness *LivenessCheckFunc) (*noAuthWebContext, error) {
	ctx := &noAuthWebContext{
		routeVars: mux.Vars(request),
		db:        db,
		conf:      conf,
		logger:    logger,
		startTime: time.Now(),
		liveness:  liveness,
	}

	return ctx, nil
}
