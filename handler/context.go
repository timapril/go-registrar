package handler

import (
	"fmt"
	"time"

	"github.com/op/go-logging"
	"github.com/timapril/go-registrar/lib"
)

// LivenessCheckFunc is the type of the liveness check function that can be
// implemented
type LivenessCheckFunc func(*lib.DBCache) (float64, error)


// Context represents state surrounding a request.
type Context interface {
	// GetRouteVars returns variables gorilla mux collected from the route.
	GetRouteVars() map[string]string
	// GetUsername returns the username of the requester.
	GetUsername() string
	// GetDB returns a DB handle.
	GetDB() *lib.DBCache
	// GetConf returns the application's configuration
	GetConf() lib.Config
	// GetLogger returns the application's logger facility
	GetLogger() *logging.Logger
	// LogRequest is used to log a request from teh current context
	LogRequest(level logging.Level, url, function, message string)

	// GetLivenessCheckFunc is used to retrieve the liveness check function
	// for the server. If no function is present, nil is returned
	GetLivenessCheckFunc() *LivenessCheckFunc
}

// noAuthWebContext represents the state of a Web UI request that does not
// require an authenticated user
type noAuthWebContext struct {
	routeVars map[string]string
	db        *lib.DBCache
	conf      lib.Config
	logger    *logging.Logger
	startTime time.Time
	liveness *LivenessCheckFunc
}

func (ctx noAuthWebContext) GetRouteVars() map[string]string {
	return ctx.routeVars
}

func (ctx noAuthWebContext) GetDB() *lib.DBCache {
	return ctx.db
}

func (ctx noAuthWebContext) GetConf() lib.Config {
	return ctx.conf
}

func (ctx noAuthWebContext) GetLogger() *logging.Logger {
	return ctx.logger
}

func (ctx noAuthWebContext) RequestTime() float64 {
	return time.Since(ctx.startTime).Seconds() * 1000
}

func (ctx noAuthWebContext) LogRequest(level logging.Level, url, function, message string) {
	logMessage(ctx.logger, level, fmt.Sprintf("%.02f noauth:%s:%s  %s", ctx.RequestTime(), function, url, message))
}

func (ctx noAuthWebContext) GetLivenessCheckFunc() *LivenessCheckFunc {
	return ctx.liveness
}

// webContext represents the state of a Web UI request that requires an
// authenticated user
type webContext struct {
	username  string
	email     string
	routeVars map[string]string
	db        *lib.DBCache
	conf      lib.Config
	logger    *logging.Logger
	startTime time.Time
	liveness *LivenessCheckFunc
}

func (ctx webContext) GetRouteVars() map[string]string {
	return ctx.routeVars
}

func (ctx webContext) GetUsername() string {
	return ctx.username
}

func (ctx webContext) GetEmail() string {
	return ctx.email
}

func (ctx webContext) GetDB() *lib.DBCache {
	return ctx.db
}

func (ctx webContext) GetConf() lib.Config {
	return ctx.conf
}

func (ctx webContext) GetLogger() *logging.Logger {
	return ctx.logger
}

func (ctx webContext) RequestTime() float64 {
	return time.Since(ctx.startTime).Seconds() * 1000
}

func (ctx webContext) LogRequest(level logging.Level, url, function, message string) {
	logMessage(ctx.logger, level, fmt.Sprintf("%.02f web:%s:%s %s %s", ctx.RequestTime(), function, url, ctx.username, message))
}

func (ctx webContext) GetLivenessCheckFunc() *LivenessCheckFunc {
	return ctx.liveness
}

// apiContext represents the state of an API request.
type apiContext struct {
	user      *lib.APIUser
	username  string
	routeVars map[string]string
	db        *lib.DBCache
	conf      lib.Config
	logger    *logging.Logger
	startTime time.Time
	liveness *LivenessCheckFunc
}

func (ctx apiContext) GetAPIUser() *lib.APIUser {
	return ctx.user
}

func (ctx apiContext) GetRouteVars() map[string]string {
	return ctx.routeVars
}

func (ctx apiContext) GetUsername() string {
	return ctx.username
}

func (ctx apiContext) GetDB() *lib.DBCache {
	return ctx.db
}

func (ctx apiContext) GetConf() lib.Config {
	return ctx.conf
}

func (ctx apiContext) GetLogger() *logging.Logger {
	return ctx.logger
}

func (ctx apiContext) RequestTime() float64 {
	return time.Since(ctx.startTime).Seconds() * 1000
}

func (ctx apiContext) LogRequest(level logging.Level, url, function, message string) {
	logMessage(ctx.logger, level, fmt.Sprintf("%.02f api:%s:%s %s %s", ctx.RequestTime(), function, url, ctx.username, message))
}

func (ctx apiContext) GetLivenessCheckFunc() *LivenessCheckFunc {
	return ctx.liveness
}

func logMessage(logger *logging.Logger, level logging.Level, message string) {
	switch level {
	case logging.CRITICAL:
		logger.Critical(message)
	case logging.DEBUG:
		logger.Debug(message)
	case logging.ERROR:
		logger.Error(message)
	case logging.INFO:
		logger.Info(message)
	case logging.NOTICE:
		logger.Notice(message)
	case logging.WARNING:
		logger.Warning(message)
	default:
		logger.Error("Unhandled logging type")
	}
}
