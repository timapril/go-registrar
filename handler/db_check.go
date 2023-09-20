package handler

import (
	"net/http"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/lib"
)

// DBCheckWeb is used as a status check for the database of the
// application
func DBCheckWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	err := lib.BootstrapRegistrar(ctx.GetDB(), ctx.GetConf())
	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "DBCheckWeb", ctx.db.GetCacheStatsLog())

	return w.Redirect(request, "/", http.StatusFound)
}
