package handler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/timapril/go-registrar/csrf"
	"github.com/timapril/go-registrar/lib"
)

// ResponseWriter implements and wraps http.ResponseWriter while providing
// common rendering functions.
type ResponseWriter struct {
	writer     http.ResponseWriter
	templates  *template.Template
	conf       lib.Config
	hasWritten bool
}

// WrapResponseWriter takes a http.ResponseWriter and common application
// objects and wraps the resonse writer to allow the use to templates
// and error generation
func WrapResponseWriter(
	writer http.ResponseWriter,
	templates *template.Template,
	conf lib.Config,
) ResponseWriter {
	return ResponseWriter{writer: writer, templates: templates, conf: conf}
}

// HasWritten will return true iff the response writer has had any data
// written to it
func (w ResponseWriter) HasWritten() bool {
	return w.hasWritten
}

// Redirect redirects the request to the provided location with the
// given status code
func (w ResponseWriter) Redirect(request *http.Request, loc string, status int) error {
	http.Redirect(w, request, loc, status)

	return nil
}

// DisplayTemplate renders a template, if provided an RegistrarObjectPage and
// a username, sets CSRF token.
func (w ResponseWriter) DisplayTemplate(name string, page interface{}, usernameOpt ...string) error {
	// Support the optional username parameter.
	var username string
	if len(usernameOpt) > 0 {
		username = usernameOpt[0]
	}

	// Generate and set a CSRF token if we can.
	if page, isObjPage := page.(lib.RegistrarObjectPage); isObjPage && username != "" {
		newCSRF, err := csrf.GenerateCSRF(username, w.conf)
		if err != nil {
			return err
		}
		page.SetCSRFToken(newCSRF)
	}

	return w.templates.ExecuteTemplate(w, name, page)
}

// DisplayError will use the provided error and it will generate an
// error page using the error template
func (w ResponseWriter) DisplayError(referer string, err error, username ...string) error {
	return w.DisplayTemplate(ErrorTemplateName, MakeErrorPage(referer, err), username...)
}

// DisplayErrors will use the provided errors and it will generate an
// errors page using the errors template
func (w ResponseWriter) DisplayErrors(referer string, errs []error, username ...string) error {
	return w.DisplayTemplate(ErrorsTemplateName, MakeErrorsPage(referer, errs), username...)
}

// MustDisplayError will cause an internal server error if
// the error template fails to generate.
func (w ResponseWriter) MustDisplayError(referer string, err error, username ...string) {
	displayErr := w.DisplayError(referer, err, username...)
	if displayErr != nil {
		msg := fmt.Sprintf("encountered <%s> while displaying error for <%s>", displayErr, err)

		http.Error(w, msg, http.StatusInternalServerError)
	}
}

// SendAPIResponse will serielize the API response and send it to the
// requestor as the response
func (w ResponseWriter) SendAPIResponse(response lib.APIResponse) error {
	_, err := response.Write(w)
	return err
}

// SendAPIError will generate an API error response and serialize it to
// send to the requestor
func (w ResponseWriter) SendAPIError(err error) error {
	return w.SendAPIErrors([]error{err})
}

// SendAPIErrors will generate an API error response witht he errors
// provided and serialize it to send to the requestor
func (w ResponseWriter) SendAPIErrors(errs []error) error {
	response := lib.GenerateErrorResponse(errs)

	_, err := response.Write(w)
	return err
}

// MustSendAPIError will attempt to send the api and if an error occurs
// when generating the response, a http error will be sent
func (w ResponseWriter) MustSendAPIError(err error) {
	sendErr := w.SendAPIError(err)
	if sendErr != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Header returns the header map that will be sent by WriteHeader.
// Changing the header after a call to WriteHeader (or Write) has
// no effect.
func (w ResponseWriter) Header() http.Header {
	return w.writer.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
// If WriteHeader has not yet been called, Write calls WriteHeader(http.StatusOK)
// before writing the data.  If the Header does not contain a
// Content-Type line, Write adds a Content-Type set to the result of passing
// the initial 512 bytes of written data to DetectContentType.
func (w ResponseWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
func (w ResponseWriter) WriteHeader(status int) {
	w.writer.WriteHeader(status)
}
