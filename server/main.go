package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/op/go-logging"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"

	"github.com/timapril/go-registrar/handler"
	"github.com/timapril/go-registrar/lib"
)

var (
	configLoc          = flag.String("conf", "./server.cfg", "The location of the configuration file")
	limitConfigCheck   = flag.Int("confcheck", -1, "the number of times to check for a config before an exit (-1 is off)")
	configCheckTimeout = flag.Int("conftimeout", 5, "the number of seconds to wait between config checks 0<x<=60")

	export = flag.Bool("export", false, "set if the db export should be updated")
)

func waitForConfig(configLocation string, configWaitCountMax int, configWaitTimeout time.Duration, logger *logging.Logger) (conf lib.Config, err error) {
	waitCounter := configWaitCountMax
	for {
		conf, err = lib.LoadConfig(configLocation)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load config: %s", err.Error()))
		} else {
			return conf, err
		}
		if waitCounter > 0 {
			waitCounter = waitCounter - 1
		}
		if waitCounter <= 0 {
			return conf, errors.New("Unable to get configuration")
		}
		time.Sleep(configWaitTimeout)
	}
}

func main() {
	flag.Parse()

	// Create the logger we'll use
	logger := lib.MustGetLogger("registrar")

	if *configCheckTimeout <= 0 || *configCheckTimeout > 60 {
		logger.Fatal("Config Check Timeout is invalid, must be between 0 and 60 seconds")
	}

	conf, confErr := waitForConfig(*configLoc, *limitConfigCheck, time.Duration(*configCheckTimeout)*time.Second, logger)
	if confErr != nil {
		logger.Fatalf("Configuration Loading Error: %s", confErr.Error())
	}

	configureLoggingErr := lib.ConfigureLogging(conf)
	if configureLoggingErr != nil {
		logger.Fatalf("Failed to setup logging: %s", configureLoggingErr)
	}

	db, loadDBErr := lib.LoadDB(conf, logger)
	if loadDBErr != nil {
		logger.Fatalf("Failed to load DB: %s", loadDBErr)
	}

	if *export {
		// TODO: Make data escrow write to file
		return
	}

	cacheFactory := lib.NewDBCacheFactory(db)

	// bootstrapErr := lib.BootstrapAkaRegistrar(cacheFactory.GetNewDBCache(), conf)
	// if bootstrapErr != nil {
	// 	logger.Criticalf("Error bootstrapping the database: %s", bootstrapErr)
	// 	return
	// }

	templates := lib.LoadTemplates(conf.Server.TemplatePath)

	factory := handler.NewFactory(conf, cacheFactory, templates, logger, nil)
	http.Handle("/", getRouter(conf, factory))

	// FIXME, only listen on localhost
	host := fmt.Sprintf(":%d", conf.Server.Port)
	logger.Info(fmt.Sprintf("Starting Server on %s", host))

	serverStartErr := http.ListenAndServe(host, nil)
	if serverStartErr != nil {
		logger.Fatalf("Unable to start server: %s", serverStartErr)
	}
}

// getRouter will create and return the mux Router that is used to route
// reqeusts to the correct function
func getRouter(conf lib.Config, factory *handler.Factory) *mux.Router {
	r := mux.NewRouter()

	r.Handle("/", factory.ForWeb(handler.HomeHandlerWeb))

	saveWeb := factory.ForWeb(handler.CheckCSRFWeb, handler.SaveHandlerWeb)
	r.Handle("/save/{objecttype}", saveWeb)

	holdWeb := factory.ForWeb(handler.CheckHandlerWeb)
	r.Handle("/hold/{objecttype}/{id:[0-9]+}", holdWeb)

	holdUpdateWeb := factory.ForWeb(handler.CheckCSRFWeb, handler.HoldUpdateHandlerWeb)
	r.Handle("/hold/update/{objecttype}/{id:[0-9]+}", holdUpdateWeb)

	checkWeb := factory.ForWeb(handler.CheckHandlerWeb)
	r.Handle("/check/{objecttype}/{id:[0-9]+}", checkWeb)

	checkUpdateWeb := factory.ForWeb(handler.CheckCSRFWeb, handler.CheckUpdateHandlerWeb)
	r.Handle("/check/update/{objecttype}/{id:[0-9]+}", checkUpdateWeb)

	updateWeb := factory.ForWeb(handler.CheckCSRFWeb, handler.UpdateHandlerWeb)
	r.Handle("/update/{objecttype}", updateWeb)

	viewWeb := factory.ForWeb(handler.ViewHandlerWeb)
	r.Handle("/view/{objecttype}/{id:[0-9]+}", viewWeb)

	viewAPI := factory.ForAPI(handler.ViewHandlerAPI)
	r.Handle("/api/view/{objecttype}/{id:[0-9]+}", viewAPI)

	viewAtAPI := factory.ForAPI(handler.ViewAtHandlerAPI)
	r.Handle("/api/viewat/{objecttype}/{id:[0-9]+}/{ts:[0-9]+}", viewAtAPI)

	viewAllWeb := factory.ForWeb(handler.ViewAllHandlerWeb)
	r.Handle("/viewall/{objecttype}", viewAllWeb)

	viewAllAPI := factory.ForAPI(handler.ViewAllHandlerAPI)
	r.Handle("/api/viewall/{objecttype}", viewAllAPI)

	newWeb := factory.ForWeb(handler.NewHandlerWeb)
	r.Handle("/new/{objecttype}", newWeb)

	actionWeb := factory.ForWeb(handler.CheckCSRFWeb, handler.TakeActionHandlerWeb)
	r.Handle("/action/{objecttype}/{id:[0-9]+}/{action}", actionWeb)

	actionAPI := factory.ForAPI(handler.CheckCSRFAPI, handler.TakeActionHandlerAPI)
	r.Handle("/api/{objecttype}/{id:[0-9]+}/{action}", actionAPI)

	r.Handle("/api/gethostnames", factory.ForAPI(handler.GetHostNames))

	r.Handle("/api/gettoken", factory.ForAPI(handler.GetTokenHandlerAPI))

	r.Handle("/api/gethostipallowlist", factory.ForAPI(handler.GetHostIPAllowList))
	r.Handle("/api/sethostipwallowlist", factory.ForAPI(handler.CheckCSRFAPI, handler.RequireAdminAPIUser, handler.SetHostIPAllowList))

	r.Handle("/api/getprotecteddomainlist", factory.ForAPI(handler.GetProtectedDomainList))
	r.Handle("/api/setprotecteddomainlist", factory.ForAPI(handler.CheckCSRFAPI, handler.RequireAdminAPIUser, handler.SetProtectedDomainList))

	getWorkAPI := factory.ForAPI(handler.CheckCSRFAPI, handler.GetWorkHandlerAPI)
	r.Handle("/api/{objecttype}/getwork", getWorkAPI)

	r.Handle("/api/appendepplog", factory.ForAPI(handler.CheckCSRFAPI, handler.LogEPPAction))

	r.Handle("/api/domainnametoid/{domainname}", factory.ForAPI(handler.DomainNameToID))

	// TODO: restrict to EPP Clients
	r.Handle("/api/startepprun", factory.ForAPI(handler.CheckCSRFAPI, handler.StartEPPRun))
	r.Handle("/api/endepprun/{id:[0-9]+}", factory.ForAPI(handler.CheckCSRFAPI, handler.EndEPPRun))
	r.Handle("/api/epppassphrase/{username}", factory.ForAPI(handler.CheckCSRFAPI, handler.EPPPassphrase))

	r.Handle("/locks", factory.ForWeb(handler.GetServerLocksWorkWeb))

	r.Handle("/liveness", factory.ForNoAuthWeb(handler.LivenessCheck))
	r.Handle("/renewalCheck", factory.ForNoAuthWeb(handler.RenewCheck))
	r.Handle("/whoisconfirmemail", factory.ForNoAuthWeb(handler.WHOISConfirmEmail))

	// FIXME: Delete me when in production.
	r.Handle("/dbcheck", factory.ForWeb(handler.DBCheckWeb))

	r.PathPrefix("/static/").Handler(http.FileServer(http.Dir(conf.Server.BasePath)))

	return r
}
