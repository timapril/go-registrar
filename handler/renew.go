package handler

import (
	"net/http"

	"github.com/timapril/go-registrar/lib"
)

// RenewCheck will check for domains that need to be renewed (domains that have
// less than 1 year before it expires)
func RenewCheck(w ResponseWriter, request *http.Request, ctx noAuthWebContext) (err error) {
	return lib.FlagDomainsRequiringRenewal(ctx.GetDB())
}
