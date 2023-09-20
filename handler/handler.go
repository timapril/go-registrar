package handler

import (
	"html/template"
	"net/http"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/lib"
)

// Func is a function defention that must be implemented in order to be
// similar to the standard http.HandlerFunc with the ResponseWriter
// defined in this package
type Func func(ResponseWriter, *http.Request)

// Handler ss a structure that is used to wrap a http function call with
// parameters related to the application
type Handler struct {
	templates *template.Template
	conf      lib.Config
	handler   Func
}

// ServeHTTP imitates the ServeHTTP function from the net/http package
// but wraps the ResponseWriter from net/http with the versino defined
// in the package
func (wrapper *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	wrapper.handler(WrapResponseWriter(response, wrapper.templates, wrapper.conf), request)
}

// Factory is a structure that provides the ability to wrap handler
// functions while adding application specific checks that are common
// across request types such as user verification
type Factory struct {
	db        *lib.DBCacheFactory
	templates *template.Template
	conf      lib.Config
	logger    *logging.Logger
	liveness  *LivenessCheckFunc
}

// NewFactory is used to take the application context information and
// will generate and return a handler Factory object
func NewFactory(conf lib.Config, db *lib.DBCacheFactory, templates *template.Template, logger *logging.Logger, liveness *LivenessCheckFunc) *Factory {
	return &Factory{
		db:        db,
		templates: templates,
		conf:      conf,
		logger:    logger,
		liveness:  liveness,
	}
}

// ForNoAuthWeb will take a list of NoAuthWebFuncs and return a http.Handler
// object that can be passed to a router to server http requests natively in
// the web context without an authenticated user.
func (factory *Factory) ForNoAuthWeb(handlers ...NoAuthWebHandlerFunc) http.Handler {
	handler := func(w ResponseWriter, request *http.Request) {
		referer := request.Referer()

		ctx, err := newNoAuthWebContext(request, factory.conf, factory.db.GetNewDBCache(), factory.logger, factory.liveness)
		if err != nil {
			w.MustDisplayError(referer, err)
			return
		}

		for _, h := range handlers {
			err = h(w, request, *ctx)
			if w.HasWritten() {
				return
			}

			if err != nil {
				w.MustDisplayError(referer, err)
				return
			}
		}
	}

	return &Handler{
		templates: factory.templates,
		conf:      factory.conf,
		handler:   handler,
	}
}

// ForWeb will take a list of WebFuncs and return a http.Handler object
// that can be passed to a router to serve http requests natively in
// the web context (checking for a user and display the page if no
// error occurs and an error page if any error happens)
func (factory *Factory) ForWeb(handlers ...WebHandlerFunc) http.Handler {
	handler := func(w ResponseWriter, request *http.Request) {
		referer := request.Referer()

		ctx, err := newWebContext(request, factory.conf, factory.db.GetNewDBCache(), factory.logger, factory.liveness)
		if err != nil {
			w.MustDisplayError(referer, err)
			return
		}

		for _, h := range handlers {
			err = h(w, request, *ctx)
			if w.HasWritten() {
				return
			}

			if err != nil {
				w.MustDisplayError(referer, err, ctx.GetUsername())
				return
			}
		}
	}

	return &Handler{
		templates: factory.templates,
		conf:      factory.conf,
		handler:   handler,
	}
}

// ForAPI will take a list of APIFuncs and return a http.Handler object
// that can be passed to a router to server http requests natively in
// the API context (checking for an authorized user and display the api
// response or an error(s) if one occurs)
func (factory *Factory) ForAPI(handlers ...APIHandlerFunc) http.Handler {
	handler := func(w ResponseWriter, request *http.Request) {
		ctx, err := newAPIContext(request, factory.conf, factory.db.GetNewDBCache(), factory.logger, factory.liveness)
		if err != nil {
			w.MustSendAPIError(err)
			return
		}

		for _, h := range handlers {
			err = h(w, request, *ctx)
			if w.HasWritten() {
				return
			}

			if err != nil {
				w.MustSendAPIError(err)
				return
			}
		}
	}

	return &Handler{
		templates: factory.templates,
		conf:      factory.conf,
		handler:   handler,
	}
}
