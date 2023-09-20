package handler

import (
	"net/http"

	"github.com/timapril/go-registrar/lib"

	"github.com/op/go-logging"
)

const (
	// IndexPageTemplateName is the name of the template that is used
	// to represent the index of the application
	IndexPageTemplateName = "indexPage"
)

// IndexPage is used when rendering the HTML Template for the default
// index page
type IndexPage struct {
	WorkDomains []lib.Domain
	WorkHosts   []lib.Host
}

// HomeHandlerWeb handles request for the home page of the application
func HomeHandlerWeb(w ResponseWriter, request *http.Request, ctx webContext) error {
	ip := IndexPage{}
	var err error

	ctx.LogRequest(logging.INFO, request.URL.String(), "HomeHandlerWeb", ctx.db.GetCacheStatsLog())

	db := ctx.GetDB()

	ip.WorkDomains, err = lib.GetWorkDomainsFull(db)
	if err != nil {
		return err
	}

	ip.WorkHosts, err = lib.GetWorkHostsFull(db)
	if err != nil {
		return err
	}

	return w.DisplayTemplate(IndexPageTemplateName, ip, ctx.GetUsername())
}
