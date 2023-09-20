package handler

import (
	"net/http"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/csrf"
	"github.com/timapril/go-registrar/lib"
)

// GetTokenHandlerAPI is used to generate a token for an API request
// and will respond with the token in a JSON serialized format
func GetTokenHandlerAPI(w ResponseWriter, request *http.Request, ctx apiContext) error {
	newCSRF, err := csrf.GenerateCSRF(ctx.GetUsername(), ctx.GetConf())
	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "GetTokenHandlerAPI", ctx.db.GetCacheStatsLog())

	return w.SendAPIResponse(lib.GenerateTokenResponse(newCSRF))
}
