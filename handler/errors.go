package handler

const (
	// NoUsernameError is the error message that will be displayed if
	// no username can be determined from the HTTP request
	NoUsernameError = "Unable to find username"

	// NoEmailError is the error message that will be displayed if
	// no email can be determined from the HTTP request
	NoEmailError = "Unable to find email"

	// ParsingError parsing failed on a field.
	ParsingError = "Unable to parse value"

	// InvalidOrExpiredCSRF is the error message that will be
	// displayed if a csrf token presented is invalid or expired
	InvalidOrExpiredCSRF = "Invalid or expired CSRF token"

	// RouteVarMissing indicates a bad route/handler match.
	RouteVarMissing = "Invalid route; missing variable"

	// UnspecifiedError indicates that an error message could not be specified.
	UnspecifiedError = "Unspecified error encountered"

	// FailedToLoadUserError indicates a user could not be loaded
	FailedToLoadUserError = "User could not be loaded"

	// ResponseMissingError indicates an action name was not specified
	ResponseMissingError = "Response not specified"

	// ObjectNotSupportedError indicates that the object type specified
	// is inappropriate.
	ObjectNotSupportedError = "Requested object type is not supported"

	// ValueMissingError indicates a required value (such as an argument)
	// is not present.
	ValueMissingError = "Value is missing"
)

const (
	// ErrorTemplateName is the name of our error template.
	ErrorTemplateName = "error"

	// ErrorsTemplateName is the name of our multiple error template.
	ErrorsTemplateName = "errors"
)

// ErrorPage is used whe rendering the Error Page HTML Template
type ErrorPage struct {
	ErrorMessage   string
	TargetLink     string
	TargetLinkText string
}

// MakeErrorPage generates a page for use with ErrorTemplateName
// passing the text of the error to be displayed in.
func MakeErrorPage(referer string, err error) ErrorPage {
	ep := ErrorPage{}
	ep.ErrorMessage = err.Error()
	ep.TargetLink = referer
	if len(ep.TargetLink) > 0 {
		ep.TargetLinkText = "Go Back to " + referer
	}

	return ep
}

// ErrorsPage is used when rendering the Errors (list of errors) Page
// HTML Template
type ErrorsPage struct {
	ErrorMessages  []string
	TargetLink     string
	TargetLinkText string
}

// MakeErrorsPage is used to generate an error reponse page for an
// action that returns multiple errors
func MakeErrorsPage(referer string, errs []error) ErrorsPage {
	ep := ErrorsPage{}

	for _, err := range errs {
		ep.ErrorMessages = append(ep.ErrorMessages, err.Error())
	}

	ep.TargetLink = referer
	if len(ep.TargetLink) > 0 {
		ep.TargetLinkText = "Go Back to " + referer
	}

	return ep
}
