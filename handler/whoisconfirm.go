package handler

import (
	"net/http"

	"github.com/timapril/go-registrar/lib"
)

// WHOISConfirmEmail will review all domains and send WHOIS confirmation emails
// as required by ICANN once per year based on the registration date
func WHOISConfirmEmail(w ResponseWriter, request *http.Request, ctx noAuthWebContext) (err error) {
	return lib.WHOISConfirmEmail(ctx.GetDB(), ctx.GetConf())
}
