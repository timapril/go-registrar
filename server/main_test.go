package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/timapril/go-registrar/handler"
	"github.com/timapril/go-registrar/lib"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

var TestUser1Username = "test1"
var TestUser2Username = "test2"
var InvalidTestUsername = "invaliduser"
var ExampleUserOrg = "@example.com"
var BaseTestingKey = "test1"
var TestConfigPath = "test.cfg"

func Test_HomeHandlerCheck(t *testing.T) {
	Convey("Given a configured route and loaded templates", t, func() {
		r, _ := http.NewRequest(http.MethodGet, "/", nil)
		router := getTestRouter()

		Convey("HomeHandler should check for username", func() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			body, err := getBody(w)
			So(body, ShouldContainSubstring, handler.NoUsernameError)
			So(err, ShouldBeNil)
		})

		Convey("HomeHandler should load with a valid request", func() {
			// TODO
		})
	})
}

func TestMainNewHandler(t *testing.T) {
	Convey("Given a configured route and loaded templates", t, func() {
		router := getTestRouter()

		Convey("Calling NewHandler without a username set should return an error", func() {
			r, _ := http.NewRequest(http.MethodGet, "/new/bogus", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			body, err := getBody(w)
			So(body, ShouldContainSubstring, handler.NoUsernameError)
			So(err, ShouldBeNil)
		})

		Convey("Calling NewHandler with unknown object type should return an error", func() {
			r, _ := http.NewRequest(http.MethodGet, "/new/bogus", nil)
			r.Header.Add("REMOTE_USER", TestUser1Username)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)
			body, err := getBody(w)
			So(body, ShouldContainSubstring, lib.UnknownObjectTypeError)
			So(err, ShouldBeNil)
		})

		Convey("Calling NewHandler for an approver should not return an error", func() {
			r, _ := http.NewRequest(http.MethodGet, "/new/approver", nil)
			r.Header.Add("REMOTE_USER", TestUser1Username)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)
			body, err := getBody(w)
			So(body, ShouldNotContainSubstring, "An Error Has Occured")
			So(body, ShouldNotContainSubstring, lib.UnknownObjectTypeError)
			So(err, ShouldBeNil)
		})
	})
}

/*
func TestMainActionHandler(t *testing.T) {
	Convey("Given a http request and a response writer", t, func() {
		r, _ := http.NewRequest(http.MethodGet, "/action/bogus/1/action", nil)
		w := httptest.NewRecorder()
		conf = lib.Config{}
		router := getTestRouter(conf)
		router.ServeHTTP(w, r)
		Convey("Calling ActionHandler without a username set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, NoUsernameSetError)
		})

		r, _ = http.NewRequest(http.MethodGet, "/action/bogus/1/action", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)
		Convey("Calling ActionHandler with a username but no config set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "Configuration not loaded")
		})
		var err error
		conf, err = GetTestConfig()
		db, err := GetDB()
		router = getTestRouter(conf)
		lib.BootstrapRegistrar(db, conf)
		So(err, ShouldBeNil)

		r, _ = http.NewRequest(http.MethodGet, "/action/bogus/1/action", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling ActionHandler with a username and a valid config set but no object type should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "An Error Has Occured")
		})

		r, _ = http.NewRequest(http.MethodGet, "/action/approverrevision/1/action", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling ActionHandler with a username, a valid config set, object type should not return an error", func() {
			So(getBody(w), ShouldContainSubstring, "Unknown action action for approverrevision")
		})
	})
}

func TestMainViewAllHandler(t *testing.T) {
	Convey("Given a http request and a response writer", t, func() {
		r, _ := http.NewRequest(http.MethodGet, "/viewall/bogus", nil)
		w := httptest.NewRecorder()
		conf = lib.Config{}
		router := getTestRouter(conf)
		router.ServeHTTP(w, r)
		Convey("Calling ViewAllHandler without a username set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, NoUsernameSetError)
		})

		r, _ = http.NewRequest(http.MethodGet, "/viewall/bogus", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)
		Convey("Calling ViewAllHandler with a username but no config set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "Configuration not loaded")
		})
		var err error
		conf, err = GetTestConfig()
		db, err := GetDB()
		router = getTestRouter(conf)
		lib.BootstrapRegistrar(db, conf)
		So(err, ShouldBeNil)

		r, _ = http.NewRequest(http.MethodGet, "/viewall/bogus", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling ViewAllHandler with a username and a valid config set but no object type should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "An Error Has Occured")
		})

		r, _ = http.NewRequest(http.MethodGet, "/viewall/approverrevision", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling ViewAllHandler with a username, a valid config set, object type should not return an error", func() {
			So(getBody(w), ShouldContainSubstring, "is undefined")
		})
	})
}

func TestMainDBCheck(t *testing.T) {

	Convey("Given a http request and a response writer", t, func() {
		conf, _ = GetTestConfig()
		r, _ := http.NewRequest(http.MethodGet, "/dbcheck", nil)
		w := httptest.NewRecorder()
		router := getTestRouter(conf)
		router.ServeHTTP(w, r)
		Convey("Calling DBCheck with a valid config", func() {
			So(getBody(w), ShouldContainSubstring, "Found")
			So(w.Code, ShouldEqual, http.StatusFound)
		})
	})
	Convey("Given a http request and a response writer", t, func() {
		conf = lib.Config{}
		conf.Loaded = false
		r, _ := http.NewRequest(http.MethodGet, "/dbcheck", nil)
		w := httptest.NewRecorder()
		router := getTestRouter(conf)
		router.ServeHTTP(w, r)
		Convey("Calling DBCheck with an invalid config", func() {
			So(getBody(w), ShouldContainSubstring, "Configuration not loaded")
		})
	})
}

func TestMainViewHandler(t *testing.T) {
	Convey("Given a http request and a response writer", t, func() {
		r, _ := http.NewRequest(http.MethodGet, "/view/bogus/1", nil)
		w := httptest.NewRecorder()
		conf = lib.Config{}
		router := getTestRouter(conf)
		router.ServeHTTP(w, r)
		Convey("Calling ViewHandler without a username set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, NoUsernameSetError)
		})

		r, _ = http.NewRequest(http.MethodGet, "/view/bogus/1", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)
		Convey("Calling ViewHandler with a username but no config set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "Configuration not loaded")
		})
		var err error
		conf, err = GetTestConfig()
		db, err := GetDB()
		router = getTestRouter(conf)
		lib.BootstrapRegistrar(db, conf)
		So(err, ShouldBeNil)

		r, _ = http.NewRequest(http.MethodGet, "/view/bogus/1", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling ViewHandler with a username and a valid config set but no object type should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "An Error Has Occured")
		})

		r, _ = http.NewRequest(http.MethodGet, "/view/approverrevision/1", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling ViewHandler with a username, a valid config set, object type should not return an error", func() {
			So(getBody(w), ShouldContainSubstring, "<h1>Approver Revision</h1>")
		})

		templates = template.Must(template.ParseGlob("testing_template/*"))
		r, _ = http.NewRequest(http.MethodGet, "/view/approverrevision/1", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling ViewHandler with a username, a valid config set, object type should not return an error - bad template", func() {
			So(getBody(w), ShouldContainSubstring, "is undefined")
		})
		loadTemplates(conf)
	})
}

func GetTestCSRF(conf lib.Config) string {
	csrf := csrf.GenerateCSRF(TestUser1Username, conf)
	return csrf
}

func TestMainUpdateHandler(t *testing.T) {
	Convey("Given a http request and a response writer", t, func() {
		r, _ := http.NewRequest(http.MethodGet, "/update/bogus", nil)
		w := httptest.NewRecorder()
		conf = lib.Config{}
		router := getTestRouter(conf)
		router.ServeHTTP(w, r)
		Convey("Calling UpdateHandler without a username set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, NoUsernameSetError)
		})

		r, _ = http.NewRequest(http.MethodGet, "/update/bogus", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)
		Convey("Calling UpdateHandler with a username but no config set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "Configuration not loaded")
		})
		var err error
		conf, err = GetTestConfig()
		db, err := GetDB()
		router = getTestRouter(conf)
		lib.BootstrapRegistrar(db, conf)
		So(err, ShouldBeNil)

		r, _ = http.NewRequest(http.MethodGet, "/update/bogus", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling UpdateHandler with a username and a valid config set but no object type should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "An Error Has Occured")
		})

		r, _ = http.NewRequest(http.MethodGet, "/update/approverrevision", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		r.Form = make(url.Values)
		r.Form.Add("id", "2")
		r.Form.Add("revision_approver_id", "1")
		r.Form.Add("revision_empid", "2")
		r.Form.Add("revision_desiredstate", "inactive")

		r.Form.Add("approver_set_informed_id", "1")
		r.Form.Add("revision_name", "5")
		r.Form.Add("revision_email", "6")
		r.Form.Add("revision_role", "7")
		r.Form.Add("revision_username", "8")
		r.Form.Add("revision_dept", "9")
		r.Form.Add("revision_fingerprint", "10")
		r.Form.Add("revision_pubkey", "11")
		r.Form.Add("csrf_token", GetTestCSRF(conf))
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling UpdateHandler with a username, a valid config set and a valid object type should not return an error - all required fields found", func() {
			So(getBody(w), ShouldNotContainSubstring, "An Error Has Occured")
			So(w.Code, ShouldEqual, http.StatusFound)
			So(w.HeaderMap.Get("Location"), ShouldContainSubstring, "/view/approverrevision/")
		})

		r, _ = http.NewRequest(http.MethodGet, "/update/approverrevision", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		r.Form = make(url.Values)
		r.Form.Add("id", "2")
		r.Form.Add("revision_approver_id", "1")
		r.Form.Add("revision_empid", "2")
		r.Form.Add("revision_desiredstate", "inactive")

		r.Form.Add("approver_set_informed_id", "1")
		r.Form.Add("revision_name", "5")
		r.Form.Add("revision_email", "6")
		r.Form.Add("revision_role", "7")
		r.Form.Add("revision_username", "8")
		r.Form.Add("revision_dept", "9")
		r.Form.Add("revision_fingerprint", "10")
		r.Form.Add("revision_pubkey", "11")
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling UpdateHandler with a username, a valid config set and a valid object type but no CSRF token should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "An Error Has Occured")
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		r, _ = http.NewRequest(http.MethodGet, "/update/approverrevision", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		r.Form = make(url.Values)
		r.Form.Add("id", "2")
		r.Form.Add("csrf_token", GetTestCSRF(conf))
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling UpdateHandler with a username, a valid config set and a valid object type should not return an error - missing fields", func() {
			So(getBody(w), ShouldContainSubstring, "Error")
		})
	})
}

func TestMainMain(t *testing.T) {
	Convey("With another program listening on port 5000", t, func() {

		_, err := CreateAndLoadConfig()

		ln, err := net.Listen("tcp", ":6000")
		defer ln.Close()
		Convey("Opening the port should not result in an error", func() {
			So(err, ShouldBeNil)
			So(func() { main() }, ShouldPanic)
		})
	})

	Convey("With an invlid config log file name", t, func() {
		loc := "./server.badlogfilename.cfg"
		configLoc = &loc
		Convey("Main should panic when its not able to create the log file", func() {
			So(func() { main() }, ShouldPanic)
		})
	})

	Convey("With an invalid config file", t, func() {
		loc := "./bogusconfig.cfg"
		configLoc = &loc
		Convey("Main should panic when its not able to find the config file", func() {
			So(func() { main() }, ShouldPanic)
		})
	})

}

func PrepareTemplates(conf lib.Config) {
	loadTemplates(conf)
}
*/

/*
func TestMainSaveHandler(t *testing.T) {

	Convey("Given a http request and a response writer", t, func() {
		r, _ := http.NewRequest(http.MethodGet, "/save/bogus", nil)
		w := httptest.NewRecorder()
		conf = lib.Config{}
		router := getTestRouter(conf)
		router.ServeHTTP(w, r)
		Convey("Calling SaveHandler without a username set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, NoUsernameSetError)
		})

		r, _ = http.NewRequest(http.MethodGet, "/save/bogus", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)
		Convey("Calling SaveHandler with a username but no config set should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "Configuration not loaded")
		})

		var err error

		conf, err = GetTestConfig()
		So(err, ShouldBeNil)

		r, _ = http.NewRequest(http.MethodGet, "/save/bogus", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()
		router = getTestRouter(conf)
		router.ServeHTTP(w, r)
		Convey("Calling SaveHandler with a username and a valid config set but no object type should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "An Error Has Occured")
		})

		r, _ = http.NewRequest(http.MethodGet, "/save/approver", nil)
		r.Form = make(url.Values)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		r.Form.Add("csrf_token", GetTestCSRF(conf))
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling SaveHandler with a username, valid config and object type set should not return an error", func() {
			So(getBody(w), ShouldNotContainSubstring, "An Error Has Occured")
			So(w.HeaderMap.Get("Location"), ShouldContainSubstring, "/view/approver/")
		})

		r, _ = http.NewRequest(http.MethodGet, "/save/approver", nil)
		r.Form = make(url.Values)
		r.Form.Add("return_location", "/test/")
		r.Form.Add("csrf_token", GetTestCSRF(conf))
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling SaveHandler with a username, valid config and object type set should not return an error - with return location set", func() {
			So(getBody(w), ShouldNotContainSubstring, "An Error Has Occured")
			So(w.HeaderMap.Get("Location"), ShouldContainSubstring, "/test/")
		})

		r, _ = http.NewRequest(http.MethodGet, "/save/approver", nil)
		r.Form = make(url.Values)
		r.Form.Add("return_location", "/test/")
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, r)
		Convey("Calling SaveHandler with a username, valid config and object type set but not csrf token should return an error", func() {
			So(getBody(w), ShouldContainSubstring, "An Error Has Occured")
		})
	})
}
*/

func getTestRouter() *mux.Router {
	logger := lib.MustGetLogger("registrar")

	conf, err := lib.LoadConfig(TestConfigPath)
	So(err, ShouldBeNil)

	dbraw, err := lib.LoadDB(conf, logger)
	So(err, ShouldBeNil)

	db := lib.NewDBCache(dbraw)

	err = lib.BootstrapRegistrar(&db, conf)
	So(err, ShouldBeNil)

	templates := lib.LoadTemplates(conf.Server.TemplatePath)

	factory := handler.NewFactory(conf, lib.NewDBCacheFactory(dbraw), templates, logger, nil)
	return getRouter(conf, factory)
}

func getBody(w *httptest.ResponseRecorder) (string, error) {
	bodyLen := w.Body.Len()
	arr := make([]byte, bodyLen)

	_, err := w.Body.Read(arr)
	return string(arr), err
}
