package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/timapril/go-registrar/csrf"
)

const (
	// CSRFParamName is the parameter name for submitted CSRF tokens.
	CSRFParamName = "csrf_token"
)

// checkCSRF checks the validity of a CSRF token i na request.
func checkCSRF(request *http.Request, ctx Context) error {
	token := request.FormValue(CSRFParamName)
	if token == "" {
		return fmt.Errorf("CSRF token %s is missing", CSRFParamName)
	}

	isValid, err := csrf.CheckCSRF(ctx.GetUsername(), ctx.GetConf(), token)
	if err != nil {
		return err
	}

	if !isValid {
		return errors.New(InvalidOrExpiredCSRF)
	}

	return nil
}

// CheckCSRFWeb checks submitted CSRF token and returns an error
// if it is not valid or is missing.
func CheckCSRFWeb(
	w ResponseWriter,
	request *http.Request,
	ctx webContext,
) error {
	return checkCSRF(request, ctx)
}

// CheckCSRFAPI checks submitted CSRF token and returns an error
// if it is not valid or is missing.
func CheckCSRFAPI(
	w ResponseWriter,
	request *http.Request,
	ctx apiContext,
) error {
	return checkCSRF(request, ctx)
}
