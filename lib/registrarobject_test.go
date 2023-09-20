package lib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/op/go-logging"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	TestUser1Username   = "test"
	TestUser2Username   = "test2"
	InvalidTestUsername = "invaliduser"
	ExampleUserOrg      = "@example.com"
)

type RegistrarObjectTestBlocks []struct {
	objTypeStr string
	obj        RegistrarObject
}

var (
	registrarObjectTestBlocks           RegistrarObjectTestBlocks
	registrarObjectTestBlocksParentOnly RegistrarObjectTestBlocks
)

func TestNewRegistrarObject(t *testing.T) {
	t.Parallel()

	for _, group := range registrarObjectTestBlocks {
		newObj, err := NewRegistrarObject(group.objTypeStr)
		if err != nil {
			t.Errorf("NewRegistrarObject returned an error ('%s') on valid type '%s'", err.Error(), group.objTypeStr)

			continue
		}

		if !reflect.DeepEqual(newObj, group.obj) {
			t.Errorf("NewRegistrarObject returned an object %+v that doesn't match %+v", newObj, group.obj)
		}
	}

	badObjTypes := []string{
		"",
		bogusState,
		ApproverType + " ",
		" " + ApproverType,
	}

	for _, badObjType := range badObjTypes {
		newObj, err := NewRegistrarObject(badObjType)
		if err == nil {
			t.Errorf("NewRegistrarObject did not return an error on invalid type '%s' and returned obj %+v",
				badObjType, newObj)
		}
	}
}

func TestNewRegistrarObjectPage(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrapped database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapped)
		So(err, ShouldBeNil)

		// roParentTestBlocks := RegistrarObjectTestBlocks{
		// 	{APIUserType, &APIUser{Model: Model{ID: 1}}},
		// 	{ApprovalType, &Approval{Model: Model{ID: 1}}},
		// 	{ApproverType, &Approver{Model: Model{ID: 1}}},
		// 	{ApproverSetType, &ApproverSet{Model: Model{ID: 1}}},
		// 	{ChangeRequestType, &ChangeRequest{Model: Model{ID: 1}}},
		// 	{ContactType, &Contact{Model: Model{ID: 1}}},
		// 	{HostType, &Host{Model: Model{ID: 1}}},
		// 	{DomainType, &Domain{Model: Model{ID: 1}}},
		// }

		for _, group := range registrarObjectTestBlocksParentOnly {
			page1, err1 := group.obj.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
			if err1 != nil {
				t.Fatal(err1)
			}

			page2, err2 := group.obj.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
			if err2 != nil {
				t.Error(err2)
			}
			pages := []RegistrarObjectPage{
				page1,
				page2,
			}
			testTokenString := testingCSRFToken
			for _, page := range pages {
				page.SetCSRFToken(testTokenString)
				getValue := page.GetCSRFToken()
				if getValue != testTokenString {
					t.Errorf("%T.GetCSRFToken of %T returned '%s' rather than '%s'", page, group.obj, getValue, testTokenString)
				}
			}
		}
	})
}

func TestRevisionObjectExportVersionAt(t *testing.T) {
	t.Parallel()

	registrarApprovalableObjectRevs := []RegistrarObject{
		&APIUserRevision{},
		&ApproverRevision{},
		&ApproverSetRevision{},
		&ContactRevision{},
		&DomainRevision{},
		&HostRevision{},
	}

	for _, obj := range registrarApprovalableObjectRevs {
		_, err := obj.GetExportVersionAt(nil, TimeNow().Unix())
		if err == nil {
			t.Errorf("Revision type %T.GetExportVersionAt did not return an error", obj)
		}
	}
}

func TestEmptyRegistrarApprovalableObjects(t *testing.T) {
	t.Parallel()

	objs := []RegistrarApprovalable{
		&APIUser{},
		&Approver{},
		&ApproverSet{},
		&Contact{},
		&Domain{},
		&Host{},
	}

	for _, obj := range objs {
		if obj.HasPendingRevision() {
			t.Errorf("Empty %T should not return true for HasPendingRevision", obj)
		}

		if obj.IsCancelled() {
			t.Errorf("Empty %T should not return true for IsCancelled", obj)
		}
	}
}

func Test_NotExportableObject_GetDiff(t *testing.T) {
	t.Parallel()

	ne := NotExportableObject{}
	_, err := ne.GetDiff()

	if err == nil {
		t.Errorf("Expected an error when calling GetDiff on NotExportableObject")
	}
}

func Test_NotExportableObject_ToJSON(t *testing.T) {
	t.Parallel()

	ne := NotExportableObject{}
	_, err := ne.ToJSON()

	if err == nil {
		t.Errorf("Expected an error when calling GetDiff on NotExportableObject")
	}
}

func AlmostApprove(t *testing.T, dbCache *DBCache, obj, rev RegistrarObject, changeRequest *ChangeRequest, conf Config) {
	t.Helper()

	if err := dbCache.DB.Where("registrar_object_type = ? and proposed_revision_id = ?", obj.GetType(), rev.GetID()).Find(&changeRequest).Error; err != nil {
		t.Fatalf("No Error Expected when finding CR: %s", err.Error())
	}

	if err := changeRequest.Prepare(dbCache); err != nil {
		t.Fatalf("No Error Expected when calling prepare: %s", err.Error())
	}

	logger.Errorf("AlmostApprove start obj id: %d, rev id %d, cr id %d, approval len %d",
		obj.GetID(), rev.GetID(), changeRequest.GetID(), len(changeRequest.Approvals))

	for i := range changeRequest.Approvals {
		app := &changeRequest.Approvals[i]

		if err := app.Prepare(dbCache); err != nil {
			t.Fatalf("No Error Expected when calling prepare on an existing Approval")
		}

		message := app.GetDownload(dbCache, TestUser1Username, ActionApproved)

		signed, err := ClearsignMessage(message, TestUser1Username)
		if err != nil {
			t.Fatalf("Unable to sign message %s", err.Error())
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("sig", "./sig")
		if err != nil {
			t.Fatalf("unable to create multipart for upload: %s", err.Error())
		}

		if _, err := part.Write([]byte(signed)); err != nil {
			t.Fatalf("Error writing sig into form: %s", err.Error())
		}

		if err := writer.Close(); err != nil {
			t.Fatalf("Error closing writer: %s", err.Error())
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)

		if parseErr := app.ParseFromFormUpdate(request, dbCache, conf); parseErr != nil {
			t.Fatalf("No error was expected when uploading a valid signature, got %s", parseErr.Error())
		}

		if err := dbCache.DB.Model(&app).UpdateColumns(app).Error; err != nil {
			t.Fatalf("No error was expected when uploading a valid signature, got %s", err.Error())
		}

		dbCache.WipeCache()
	}
}

// func Test_GetRegistrarObject(t *testing.T) {
// 	//db, err := GetTestDB(t)
// 	db, err  := DBFactory.GetDB(t, TestStateEmpty)
// 	if err != nil {
// 		t.Fatalf("Unable to open DB: %s", err.Error())
// 	}
//
// 	Test known objects
// 	runGetEmptyObjectTest(t, ApproverType, db)
// 	runGetEmptyObjectTest(t, ApproverRevisionType, db)
// 	runGetEmptyObjectTest(t, ApproverSetType, db)
// 	runGetEmptyObjectTest(t, ApproverSetRevisionType, db)
// 	runGetEmptyObjectTest(t, ChangeRequestType, db)
// 	runGetEmptyObjectTest(t, ApprovalType, db)
// 	runGetEmptyObjectTest(t, ContactType, db)
// 	runGetEmptyObjectTest(t, ContactRevisionType, db)
// 	runGetEmptyObjectTest(t, HostType, db)
// 	runGetEmptyObjectTest(t, HostRevisionType, db)
// 	runGetEmptyObjectTest(t, DomainType, db)
// 	runGetEmptyObjectTest(t, DomainRevisionType, db)
//
// 	// Test and unknown object
// 	bogusType := "bogustype"
// 	obj, err := GetRegistrarObject(bogusType, false, "", db)
// 	if err == nil {
// 		t.Errorf("Expected an error trying to create an unknown type, got %s", obj.GetType())
// 	}
//
// 	runGetExistingObjectTest(t, ApproverType, "1")
// 	runGetExistingObjectTest(t, ApproverRevisionType, "1")
// 	runGetExistingObjectTest(t, ApproverSetType, "1")
// 	runGetExistingObjectTest(t, ApproverSetRevisionType, "1")
// 	runGetExistingObjectTest(t, ChangeRequestType, "1")
// 	runGetExistingObjectTest(t, ApprovalType, "1")
//
// }

// func Test_GetRegistrarObjectIncorrectInt(t *testing.T) {
// 	//db, err := GetTestDB(t)
// 	db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 	if err != nil {
// 		t.Fatalf("Unable to open DB: %s", err.Error())
// 	}
// 	if err != nil {
// 		t.Fatalf("Error preparing the database: %s", err.Error())
// 	}
//
// 	_, err = GetRegistrarObject(ApproverType, true, "a", db)
// 	if err == nil {
// 		t.Errorf("Expected error when creating object with non-int ID")
// 	}
// 	_, err = GetRegistrarObject(ApproverType, true, "-1", db)
// 	if err == nil {
// 		t.Errorf("Expected error when creating object with a negitive ID")
// 	}
// 	_, err = GetRegistrarObject(ApproverType, true, "0", db)
// 	if err == nil {
// 		t.Errorf("Expected error when creating object with an ID of 0")
// 	}
// }

// func runGetExistingObjectTest(t *testing.T, objType string, id string) {
//
// 	db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 	if err != nil {
// 		t.Fatalf("Error preparing the database: %s", err.Error())
// 	}
//
// 	obj, err := GetRegistrarObject(objType, true, id, db)
// 	if err != nil {
// 		t.Errorf("Creating an existing %s failed: %s", objType, err.Error())
// 	}
// 	if obj.GetType() != objType {
// 		t.Errorf("Returned object had wrong type, expected %s got %s", objType, obj.GetType())
// 	}
// }
//
// func runGetEmptyObjectTest(t *testing.T, objType string, db *lib.DBCache) {
// 	obj, err := GetRegistrarObject(objType, false, "", db)
// 	if err != nil {
// 		t.Errorf("Creating an empty %s failed", objType)
// 	}
// 	if obj.GetType() != objType {
// 		t.Errorf("Returned object had wrong type, expected %s got %s", objType, obj.GetType())
// 	}
// }

var matchTestName *regexp.Regexp

// func ReadPubKey(filename string) error {
// 	f, err := os.Open(filename)
// 	if err != nil {
// 		return err
// 	}
// 	entityList, errReadArm := openpgp.ReadArmoredKeyRing(f)
// 	fmt.Println(entityList)
// 	fmt.Println(errReadArm)
// 	return nil
// }

func ClearsignMessage(message string, keyname string) (clearsignedMessage string, err error) {
	entity, errKey := getKey(keyname)

	if errKey != nil {
		err = errKey

		return "", err
	}

	buffer := new(bytes.Buffer)

	writer, err := clearsign.Encode(buffer, entity.PrivateKey, nil)
	if err != nil {
		return "", fmt.Errorf("unable to sign the message: %w", err)
	}

	_, err = writer.Write([]byte(message))

	if err != nil {
		return "", fmt.Errorf("error writing signed message: %w", err)
	}

	writer.Close()

	clearsignedMessage = buffer.String()

	return
}

var (
	savedConfSet = false
	savedConf    Config
)

const (
	TestStateEmpty                             = "empty"
	TestStateBootstrap                         = "bootstrap"
	TestStateBootstrapStartApproverApproval    = "bootstrap-startapproverapproval"
	TestStateBootstrapDoneApproverApproval     = "bootstrap-doneapproverapproval"
	TestStateBootstrapStartApproverSetApproval = "bootstrap-startapproversetapproval"
	TestStateBootstrapDoneApproverSetApproval  = "bootstrap-doneapproversetapproval"
	TestStateBootstrapped                      = "bootstrapped"
	TestStateBootstrappedWithHost              = "bootstrapped-with-host"
	TestStateBootstrappedWithContact           = "bootstrapped-with-contact"
	TestStateBootstrappedWithDomain            = "bootstrapped-with-domain"
	TestStateBootstrapAPIUser                  = "bootstrapped-with-apiuser"
)

type TestFile struct {
	*os.File
	lock int
}

func (f *TestFile) Flock(lock int, action string) (err error) {
	fileDesc := int(f.Fd())

	logger.Errorf("Attempting to %s %s(fd %d) with lock %d", action, f.Name(), fileDesc, f.lock)

	err = syscall.Flock(fileDesc, lock|syscall.LOCK_NB)

	if err != nil {
		return fmt.Errorf("error locking file: %w", err)
	}

	if errors.Is(err, syscall.EWOULDBLOCK) {
		logger.Errorf("%s %s(fd %d) with lock %d blocked, trying blocking", action, f.Name(), fileDesc, f.lock)

		err = syscall.Flock(fileDesc, lock|syscall.LOCK_NB)
		if err != nil {
			return fmt.Errorf("error flocking file: %w", err)
		}
	}

	if err == nil {
		logger.Errorf("%s %s(fd %d) with lock %d succeeded!", action, f.Name(), fileDesc, f.lock)
		f.lock = lock
	} else {
		logger.Errorf("%s %s(fd %d) with lock %d failed: %s", action, f.Name(), fileDesc, f.lock, err.Error())
	}

	return
}

func (f *TestFile) Lock() error {
	return f.Flock(syscall.LOCK_EX, "get exclusive lock for")
}

func (f *TestFile) RLock() error {
	return f.Flock(syscall.LOCK_SH, "get shared lock for")
}

func (f *TestFile) Unlock() error {
	return f.Flock(syscall.LOCK_UN, "unlock")
}

// RenameExclusive renames file to the new name (and changes access to read only.
func (f *TestFile) RenameExclusive(newName string) (err error) {
	oldName := f.Name()

	if err = os.Link(oldName, newName); err != nil {
		// logger.Infof("os.Link(%s, %s) failed: %s", oldName, newName, err.Error())
		return
	}

	if err = f.Close(); err != nil && !errors.Is(err, syscall.EINVAL) {
		return
	}

	if err = os.Remove(oldName); err != nil {
		// logger.Infof("os.Remove(%s) failed: %s", oldName, err.Error())
		return
	}

	f.File, err = os.Open(newName)

	return
}

func OpenTestFile(name string) (file *TestFile, err error) {
	file = &TestFile{}
	file.File, err = os.Open(name)

	return
}

func OpenRWTestFile(name string) (file *TestFile, err error) {
	file = &TestFile{}
	file.File, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0o644)

	return
}

func OpenCahcedTestFile(cacheFileName string, cacheCallback func(*TestFile) error) (f *TestFile, err error) {
	f, err = OpenTestFile(cacheFileName)
	// logger.Infof("OpenCahcedTestFile %s open: %+v", cacheFileName, err)
	if os.IsNotExist(err) {
		if f, _, err = GetTestFilenamer().TempFile(cacheFileName); err != nil {
			return
		}

		if err = cacheCallback(f); err != nil {
			return
		}

		err = f.RenameExclusive(cacheFileName)

		if os.IsExist(err) {
			f, err = OpenTestFile(cacheFileName)
		}
	}
	// logger.Infof("OpenCahcedTestFile final %s: %+v", cacheFileName, err)
	return
}

func GetCahcedString(cacheFileName string, cacheCallback func(*TestFile) error) (str string, err error) {
	var file *TestFile

	if file, err = OpenCahcedTestFile(cacheFileName, cacheCallback); err != nil {
		return
	}

	pemBytes, err := io.ReadAll(file)

	return string(pemBytes), err
}

func getTestName() string {
	programCounter, _, _, callOK := runtime.Caller(1)

	for idx := 2; callOK; idx++ {
		functionObject := runtime.FuncForPC(programCounter)

		if matchTestName.MatchString(functionObject.Name()) {
			return matchTestName.ReplaceAllString(functionObject.Name(), "$1")
		}

		programCounter, _, _, callOK = runtime.Caller(idx)
	}

	return ""
}

var testRoot, tmpDir string

func GetTestRoot() string {
	if testRoot == "" {
		testRoot = "./_testRuns"

		if err := os.MkdirAll(testRoot, 0o755); err != nil {
			err = fmt.Errorf("Error encountered trying to set up TestRoot: %s", err.Error())
			logger.Errorf("%s", err.Error())
			panic(err)
		}
	}

	return testRoot
}

func GetTempDir() string {
	if tmpDir != "" {
		return tmpDir
	}

	var err error
	tmpDir, err = os.MkdirTemp(GetTestRoot(), fmt.Sprintf("_testRun-%s-", time.Now().Format("2006.01.02_15.04.05")))

	if err != nil {
		err = fmt.Errorf("Unable to create tmpDir: %s", err.Error())
		logger.Errorf("%s", err.Error())
		panic(err)
	}

	return tmpDir
}

var testingDBPath string

func GetTestingDBPath() string {
	if testingDBPath == "" {
		testingDBPath = fmt.Sprintf("%s/tmp_testing_file.db", GetTempDir())
	}

	return testingDBPath
}

type TestFilenamer struct {
	testName string
	names    []string
}

func GetTestFilenamer() (tfp *TestFilenamer) {
	tfp = &TestFilenamer{}
	tfp.testName = getTestName()
	tfp.names = make([]string, 0, 5)

	return
}

func (tfp *TestFilenamer) TempFile(kind string) (tf *TestFile, filename string, err error) {
	var file *os.File
	file, err = os.CreateTemp(GetTempDir(), fmt.Sprintf("tmp_%s_%s", tfp.testName, kind))

	if err != nil {
		filename = ""

		return
	}

	tf = &TestFile{File: file}
	filename = tf.Name()
	tfp.names = append(tfp.names, filename)

	return
}

// TempFilename creates a temp file and immediate closes it, returning the name.
func (tfp *TestFilenamer) TempFilename(kind string) (filename string, err error) {
	var file *TestFile

	file, filename, err = tfp.TempFile(kind)

	if err != nil {
		return
	}

	file.Close()

	return
}

// TempFilename creates a temp file, writes contents to it, and closes it, returning the name.
func (tfp *TestFilenamer) TempFilenameWith(kind, contents string) (filename string, err error) {
	var file *TestFile
	file, filename, err = tfp.TempFile(kind)

	if err != nil {
		return
	}

	defer file.Close()

	_, err = file.WriteString(contents)

	return
}

func (tfp *TestFilenamer) Cleanup() {
	for _, name := range tfp.names {
		os.Remove(name)
	}
}

type (
	TestFileFunc func(*TestFile)
	DBGetFunc    func(*testing.T, TestFileFunc)
	DBSetupFunc  func(*DBCache, Config) error
	DBFunc       func(*testing.T, *DBCache)
)

func WithDBFile(t *testing.T, target *TestFile, sourceFunc DBGetFunc, setupFunc DBSetupFunc) error {
	t.Helper()

	sourceFunc(t, func(source *TestFile) {
		// logger.Infof("starting sourceFunc")
		if source != nil {
			if _, err := io.Copy(target, source); err != nil {
				t.Fatalf("Failed to copy db %s to %s: %s", source.Name(), target.Name(), err.Error())
			}
		}
	})

	if err := target.Close(); err != nil {
		t.Fatalf("Failed to close db file %s : %s", target.Name(), err.Error())
	}

	dbraw, err := gorm.Open("sqlite3", target.Name())
	if err != nil {
		t.Fatalf("Failed to open db %s : %s", target.Name(), err.Error())

		return fmt.Errorf("error opening database: %w", err)
	}

	defer dbraw.Close()

	dbc := NewDBCache(&dbraw)

	err = setupFunc(&dbc, mustGetTestConf())
	if err != nil {
		t.Error(err)

		return err
	}

	return nil
}

func GetDBFunc(t *testing.T, state string, sourceFunc DBGetFunc, setupFunc DBSetupFunc, roFunc TestFileFunc, _ Config) {
	t.Helper()

	dbFileName := fmt.Sprintf("state.%s.db", state)
	t.Logf("Opening DB File %s", dbFileName)
	// logger.Infof("GetDBFunc loading %s", dbFileName)
	file, err := OpenCahcedTestFile(dbFileName, func(tf *TestFile) error {
		err := WithDBFile(t, tf, sourceFunc, setupFunc)
		if err != nil {
			t.Fatal(err)
		}

		if t.Failed() {
			return fmt.Errorf("%s had errors building", dbFileName)
		}

		return nil
	})
	if err != nil {
		logger.Errorf("Failed to open %s: %s", dbFileName, err.Error())
		t.Fatalf("Failed to open %s: %s", dbFileName, err.Error())
	}

	roFunc(file)
}

func GetNilDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	roFunc(nil)
}

func GetEmptyDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, TestStateEmpty, GetNilDB, CreateDBSchema, roFunc, mustGetTestConf())
}

func GetBootstrapDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, TestStateBootstrap, GetNilDB, BootstrapDB, roFunc, mustGetTestConf())
}

func GetBootstrapStartApproverApprovalDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, TestStateBootstrapStartApproverApproval, GetBootstrapDB, StartApproverApproval, roFunc, mustGetTestConf())
}

func GetBootstrapDoneApproverApprovalDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, TestStateBootstrapDoneApproverApproval, GetBootstrapStartApproverApprovalDB, ApproverBootstrapApprover, roFunc, mustGetTestConf())
}

func GetBootstrapStartApproverSetApprovalDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, TestStateBootstrapStartApproverSetApproval, GetBootstrapDoneApproverApprovalDB, StartApproverSetApproval, roFunc, mustGetTestConf())
}

func GetBootstrapDoneApproverSetApprovalDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, TestStateBootstrapDoneApproverSetApproval, GetBootstrapStartApproverSetApprovalDB, ApproverSetBootstrapApproverSet, roFunc, mustGetTestConf())
}

func GetBootstrappedDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, TestStateBootstrapped, GetBootstrapDoneApproverSetApprovalDB, FinishBootstrap, roFunc, mustGetTestConf())
}

var GetDBMap = map[string]DBGetFunc{
	TestStateBootstrapStartApproverApproval:    GetBootstrapStartApproverApprovalDB,
	TestStateBootstrapDoneApproverApproval:     GetBootstrapDoneApproverApprovalDB,
	TestStateBootstrapStartApproverSetApproval: GetBootstrapStartApproverSetApprovalDB,
	TestStateBootstrapDoneApproverSetApproval:  GetBootstrapDoneApproverSetApprovalDB,
	TestStateBootstrapped:                      GetBootstrappedDB,
	TestStateEmpty:                             GetEmptyDB,
	TestStateBootstrap:                         GetBootstrapDB,
}

func FinishBootstrap(_ *DBCache, _ Config) error {
	return nil
}

func StartApproverApproval(dbCache *DBCache, _ Config) error {
	conf, err := getTestConf()
	if err != nil {
		return err
	}

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)

	// TODO: Consider deleting this
	// err = BootstrapRegistrar(dbCache, conf)
	// if err != nil {
	// 	return err
	// }

	app := ApproverRevision{}

	err = app.SetID(2)
	if err != nil {
		return err
	}

	if err := app.Prepare(dbCache); err != nil {
		return err
	}

	err = app.StartApprovalProcess(request, dbCache, conf)

	return err
}

func StartApproverSetApproval(dbCache *DBCache, _ Config) error {
	conf, err := getTestConf()
	if err != nil {
		return err
	}

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)

	// TODO: Consider deleting this
	// err = BootstrapRegistrar(dbCache, conf)
	// if err != nil {
	// 	return err
	// }

	app := ApproverSetRevision{}

	err = app.SetID(2)
	if err != nil {
		return err
	}

	if err := app.Prepare(dbCache); err != nil {
		return err
	}

	err = app.StartApprovalProcess(request, dbCache, conf)

	return err
}

// var (
// 	dbOpen = false
// 	dbObj  *DBCache
// 	dbFh   *TestFile
// )

// func GetDatabaseAtState(t *testing.T, state string) (dbCache *DBCache, err error) {
// 	t.Helper()

// 	if dbOpen && dbObj != nil {
// 		dbObj.DB.Close()

// 		dbOpen = false

// 		if _, err = dbFh.Seek(0, 0); err != nil {
// 			logger.Errorf("GetDatabaseAtState: Error seeking to beginning of file %s", err.Error())

// 			t.Errorf("Error truncating file %s", err.Error())

// 			return dbCache, fmt.Errorf("error seeing to start of file: %w", err)
// 		}

// 		if err = dbFh.Truncate(0); err != nil {
// 			logger.Errorf("GetDatabaseAtState: Error truncating file %s", err.Error())

// 			t.Errorf("Error truncating file %s", err.Error())

// 			return dbCache, fmt.Errorf("error truncating file: %w", err)
// 		}

// 		if err = dbFh.Unlock(); err != nil {
// 			logger.Errorf("GetDatabaseAtState: Error unlocking file %s", err.Error())

// 			t.Errorf("Error unlocking file %s", err.Error())

// 			return dbCache, fmt.Errorf("error unlockingi file: %w", err)
// 		}
// 	} else {
// 		if dbFh == nil {
// 			if dbFh, err = OpenRWTestFile(GetTestingDBPath()); err != nil {
// 				logger.Errorf("GetDatabaseAtState: Error opening file %s", err.Error())

// 				t.Errorf("Error opening file %s", err.Error())

// 				return dbCache, err
// 			}
// 		}
// 	}

// 	if err = dbFh.Lock(); err != nil {
// 		logger.Errorf("GetDatabaseAtState: Error locking file %s", err.Error())

// 		t.Errorf("Error locking file %s", err.Error())

// 		return dbCache, err
// 	}

// 	if sourceFunc, ok := GetDBMap[state]; ok {
// 		sourceFunc(t, func(source *TestFile) {
// 			if _, err = io.Copy(dbFh, source); err != nil {
// 				logger.Errorf("GetDatabaseAtState: Failed to copy db %s to %s: %s", source.Name(), dbFh.Name(), err.Error())
// 				t.Fatalf("Failed to copy db %s to %s: %s", source.Name(), dbFh.Name(), err.Error())

// 				return
// 			}
// 		})

// 		if err = dbFh.Sync(); err != nil {
// 			logger.Errorf("GetDatabaseAtState: Error syncing file %s", err.Error())

// 			t.Errorf("Error Syncing file %s", err.Error())

// 			return dbCache, fmt.Errorf("error syncing file: %w", err)
// 		}

// 		dbObjRaw := &gorm.DB{}
// 		dbObj := NewDBCache(dbObjRaw)
// 		dbCache = &dbObj

// 		if *dbObjRaw, err = gorm.Open("sqlite3", fmt.Sprintf("file:%s?nolock=true", dbFh.Name())); err == nil {
// 			dbOpen = true
// 		}
// 	} else {
// 		t.Errorf("Unable to find create call for state: %s", state)

// 		err = fmt.Errorf("Unable fo find db file for state %s", state)
// 	}

// 	return dbCache, err
// }

// TODO: remove
// func copy(dst string, src string) error {
// 	s, err := os.Open(src)
// 	if err != nil {
// 		return err
// 	}
// 	defer s.Close()
// 	d, err := os.Create(dst)
// 	if err != nil {
// 		return err
// 	}
// 	if _, err := io.Copy(d, s); err != nil {
// 		d.Close()
// 		return err
// 	}
// 	return d.Close()
// }

func CreateDBSchema(dbCache *DBCache, _ Config) (err error) {
	MigrateDBApprover(dbCache)
	MigrateDBApproverRevision(dbCache)
	MigrateDBApproverSet(dbCache)
	MigrateDBApproverSetRevision(dbCache)
	MigrateDBChangeRequest(dbCache)
	MigrateDBApproval(dbCache)
	MigrateDBContact(dbCache)
	MigrateDBContactRevision(dbCache)
	MigrateDBHost(dbCache)
	MigrateDBHostRevision(dbCache)
	MigrateDBDomain(dbCache)
	MigrateDBDomainRevision(dbCache)
	MigrateDBAPIUser(dbCache)
	MigrateDBAPIUserRevision(dbCache)

	return nil
}

func ApproveApproval(dbCache *DBCache, username string, approvalID int64, _ Config) error {
	conf, err := getTestConf()
	if err != nil {
		return err
	}

	app := Approval{}

	err = app.SetID(approvalID)
	if err != nil {
		return err
	}

	if prepErr := app.Prepare(dbCache); prepErr != nil {
		return fmt.Errorf("no Error Expected when calling prepare on an existing Approval: %w", prepErr)
	}

	// Getting download object

	message := app.GetDownload(dbCache, username, ActionApproved)

	signed, err := ClearsignMessage(message, username)
	if err != nil {
		return fmt.Errorf("unable to sign message %s", err.Error())
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("sig", "./sig")
	if err != nil {
		return fmt.Errorf("unable to create multipart for upload: %s", err.Error())
	}

	if _, err := part.Write([]byte(signed)); err != nil {
		return fmt.Errorf("Error writing sig into form: %s", err.Error())
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("Error closing writer: %s", err.Error())
	}

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Add("REMOTE_USER", username)

	parseErr := app.ParseFromFormUpdate(r, dbCache, conf)
	if parseErr != nil {
		return fmt.Errorf("no error was expected when uploading a valid signature, got %s", parseErr.Error())
	}

	dbCache.DB.Model(app).UpdateColumns(app)

	err = dbCache.Purge(&app)
	if err != nil {
		return err
	}

	app = Approval{}

	err = app.SetID(approvalID)
	if err != nil {
		return err
	}

	err = app.Prepare(dbCache)
	if err != nil {
		return err
	}

	err = app.PostUpdate(dbCache, conf)
	if err != nil {
		return err
	}

	return nil
}

func DeclineApproval(dbCache *DBCache, username string, approvalID int64, _ Config) error {
	conf, err := getTestConf()
	if err != nil {
		return err
	}

	app := Approval{}

	err = app.SetID(approvalID)
	if err != nil {
		return err
	}

	if prepErr := app.Prepare(dbCache); prepErr != nil {
		return errors.New("no Error Expected when calling prepare on an existing Approval")
	}

	message := app.GetDownload(dbCache, username, ActionDeclined)

	signed, err := ClearsignMessage(message, username)
	if err != nil {
		return fmt.Errorf("unable to sign message %s", err.Error())
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("sig", "./sig")
	if err != nil {
		return fmt.Errorf("unable to create multipart for upload: %s", err.Error())
	}

	if _, err := part.Write([]byte(signed)); err != nil {
		return fmt.Errorf("Error writing sig into form: %s", err.Error())
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("error closing writer: %s", err.Error())
	}

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Add("REMOTE_USER", username)

	parseErr := app.ParseFromFormUpdate(request, dbCache, conf)
	if parseErr != nil {
		return fmt.Errorf("no error was expected when uploading a valid signature, got %s", parseErr.Error())
	}

	err = dbCache.DB.Model(&app).UpdateColumns(&app).Error
	if err != nil {
		return fmt.Errorf("error updating database: %w", err)
	}

	dbCache.WipeCache()

	freshApp := Approval{}

	err = freshApp.SetID(approvalID)
	if err != nil {
		return err
	}

	err = freshApp.Prepare(dbCache)
	if err != nil {
		return err
	}

	err = freshApp.PostUpdate(dbCache, conf)
	if err != nil {
		return err
	}

	return nil
}

func ApproverSetBootstrapApproverSet(dbCache *DBCache, conf Config) (err error) {
	return ApproveApproval(dbCache, TestUser1Username, 2, conf)
}

func ApproverBootstrapApprover(dbCacheb *DBCache, conf Config) (err error) {
	return ApproveApproval(dbCacheb, TestUser1Username, 1, conf)
}

func BootstrapDB(dbCache *DBCache, _ Config) error {
	conf, err := getTestConf()
	if err != nil {
		return err
	}

	return BootstrapRegistrar(dbCache, conf)
}

func getTestPubKey() (string, error) {
	contents, err := os.ReadFile(fmt.Sprint(BaseTestingKey, ".pub"))
	if err != nil {
		return "", fmt.Errorf("error reading pubkey: %w", err)
	}

	return string(contents), nil
}

func mustGetTestConf() Config {
	conf, err := getTestConf()
	if err != nil {
		panic(fmt.Errorf("Unable to load test config: %w", err))
	}

	return conf
}

func getOrGenerateTestingGPGKey(username string) (string, error) {
	pathBase := "./" + username
	pubKeyPath := fmt.Sprintf("%s.pub", pathBase)

	fileInfo, err := os.Stat(pubKeyPath)
	if err == nil {
		if fileInfo.Size() > 0 {
			return pubKeyPath, nil
		}
	}

	_, err = genTestingGPGKey("./", username, "test root", "testing key", username+ExampleUserOrg)

	return pubKeyPath, err
}

func getTestConf() (Config, error) {
	if savedConfSet {
		return savedConf, nil
	}

	tfp := GetTestFilenamer()
	confFilename, tmperr := tfp.TempFilename("conf")

	if tmperr != nil {
		return Config{}, fmt.Errorf("Error creating temporary file for config: %w", tmperr)
	}

	pubKeyPath, err := getOrGenerateTestingGPGKey(TestUser1Username)
	if err != nil {
		return Config{}, err
	}
	// fmt.Sprintf("./%s.pub", BaseTestingKey)
	CreateDummyConfig(confFilename, pubKeyPath, "", DBTypeSqlite, "")

	savedConf = Config{}

	err = savedConf.LoadConfig(confFilename)
	if err != nil {
		return savedConf, fmt.Errorf("Error loading configuration: %w", err)
	}

	savedConfSet = true

	tfp.Cleanup()

	return savedConf, nil
}

var format = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000000} %{shortfile} § %{longfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

var (
	BaseTestingKey = "test"
	Keys           map[string]openpgp.Entity
	DBFactory      *TestingDBFactory
)

func getKey(name string) (entity openpgp.Entity, err error) {
	if val, ok := Keys[name]; ok {
		return val, nil
	}

	privKey, err := os.Open(fmt.Sprintf("%s.key", name))
	if err != nil {
		return entity, fmt.Errorf("error opening private key: %w", err)
	}

	defer privKey.Close()

	entitylist, readError := openpgp.ReadArmoredKeyRing(privKey)

	if readError != nil {
		return entity, fmt.Errorf("unable to read armored key: %w", readError)
	}

	// fmt.Println(entitylist)
	if len(entitylist) > 0 {
		Keys[name] = *entitylist[0]
		entity = Keys[name]
	}

	return entity, nil
}

func TestMain(m *testing.M) {
	DBFactory = NewTestingDBFactory(true)
	DBFactory.SetupRequest <- true

	matchTestName = regexp.MustCompile(`^.*\.(Test.*)$`)
	registrarObjectTestBlocks = RegistrarObjectTestBlocks{
		{APIUserType, &APIUser{}},
		{APIUserRevisionType, &APIUserRevision{}},
		{ApprovalType, &Approval{}},
		{ApproverType, &Approver{}},
		{ApproverRevisionType, &ApproverRevision{}},
		{ApproverSetType, &ApproverSet{}},
		{ApproverSetRevisionType, &ApproverSetRevision{}},
		{ChangeRequestType, &ChangeRequest{}},
		{ContactType, &Contact{}},
		{ContactRevisionType, &ContactRevision{}},
		{HostType, &Host{}},
		{HostRevisionType, &HostRevision{}},
		{DomainType, &Domain{}},
		{DomainRevisionType, &DomainRevision{}},
	}

	registrarObjectTestBlocksParentOnly = RegistrarObjectTestBlocks{
		{APIUserType, &APIUser{}},
		{ApprovalType, &Approval{ApprovalApproverSet: ApproverSet{Model: Model{ID: 1}}}},
		{ApproverType, &Approver{}},
		{ApproverSetType, &ApproverSet{}},
		{ChangeRequestType, &ChangeRequest{}},
		{ContactType, &Contact{}},
		{HostType, &Host{}},
		{DomainType, &Domain{}},
	}

	Keys = make(map[string]openpgp.Entity)
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLevel := logging.AddModuleLevel(backendFormatter)
	backendLevel.SetLevel(logging.DEBUG, "")

	result := m.Run()

	// if result == 0 && tmpDir != "" && tmpDir != "./" {
	// 	if err := os.Remove(testingDBPath); err != nil {
	// 		logger.Errorf("Unable to clean up %s: %s", testingDBPath, err.Error())
	// 	}

	// 	if err := os.Remove(tmpDir); err != nil {
	// 		logger.Errorf("Unable to clean up %s: %s", tmpDir, err.Error())
	// 	}
	// }

	os.Exit(result)
}

type DBClientRequest struct {
	Response   chan DBClientResponse
	DBState    string
	CallSite   string
	TestingObj *testing.T
}

type DBClientResponse struct {
	Path  string
	Found bool
}

type TestingDBFactory struct {
	DBRequest    chan DBClientRequest
	SetupRequest chan interface{}
	UseDBCache   bool

	SetupComplete bool
	DBCache       map[string]string
}

func NewTestingDBFactory(useCache bool) *TestingDBFactory {
	fact := &TestingDBFactory{
		DBRequest:    make(chan DBClientRequest),
		SetupRequest: make(chan interface{}),
		UseDBCache:   useCache,

		SetupComplete: false,
		DBCache:       make(map[string]string),
	}
	go fact.MainLoop()

	return fact
}

// (dbCache *DBCache, err error).
func (factory *TestingDBFactory) GetDB(t *testing.T, status string) (dbCache *DBCache, err error) {
	t.Helper()

	retChan := make(chan DBClientResponse)

	request := DBClientRequest{
		Response:   retChan,
		DBState:    status,
		CallSite:   getTestName(),
		TestingObj: t,
	}

	factory.DBRequest <- request

	select {
	case resp := <-retChan:
		t.Logf("received %v", resp)

		if resp.Found {
			dbraw, err := gorm.Open("sqlite3", resp.Path)
			if err != nil {
				t.Fatalf("Failed to open db %s : %s", resp.Path, err.Error())

				return nil, fmt.Errorf("error opening ddatabase: %w", err)
			}

			// dbraw.LogMode(true)
			dbc := NewDBCache(&dbraw)

			return &dbc, nil
		}

		return nil, errors.New("database state not found")
	case <-time.After(time.Second):
		t.Error("Unable to get a database response")

		return nil, errors.New("Unable to get a database response")
	}
}

func (factory *TestingDBFactory) MainLoop() {
	// Preapre default states
	// Handle specific requests for db objects
	loopIdx := 0

	for {
		select {
		case <-factory.SetupRequest:
			if !factory.SetupComplete {
				var err error

				factory.DBCache, err = getOrCreateDBs(factory.UseDBCache)
				if err != nil {
					panic(err)
				}

				factory.SetupComplete = true
			}

		case dbRequest := <-factory.DBRequest:
			if !factory.SetupComplete {
				var err error

				factory.DBCache, err = getOrCreateDBs(factory.UseDBCache)
				if err != nil {
					panic(err)
				}

				factory.SetupComplete = true
			}

			dbRequest.TestingObj.Logf("Received request for %s at %s", dbRequest.DBState, dbRequest.CallSite)

			resp := DBClientResponse{}

			path, ok := factory.DBCache[dbRequest.DBState]
			if !ok {
				resp.Found = false
			} else {
				dir := GetTempDir()
				dbPath := filepath.Join(dir, fmt.Sprintf("%d.%s.%s.db", loopIdx, dbRequest.CallSite, dbRequest.DBState))

				err := copyFile(path, dbPath)
				if err != nil {
					panic(err)
				}

				resp.Found = true
				resp.Path = dbPath
			}

			dbRequest.Response <- resp
		}
		loopIdx++
	}
}

func copyFile(fromFile, toFile string) error {
	data, err := os.ReadFile(fromFile)
	if err != nil {
		return fmt.Errorf("error reading file to copy: %w", err)
	}

	err = os.WriteFile(toFile, data, 0o600)
	if err != nil {
		return fmt.Errorf("error writing to copy target: %w", err)
	}

	return nil
}

func getOrCreateDBs(tryCache bool) (retMap map[string]string, err error) {
	if tryCache {
		retMap, err := loadDBStatesFromDisk()
		if err != nil {
			panic(err)
		}

		return retMap, nil
	}

	return setupDatabaseCache()
}

func loadDBStatesFromDisk() (map[string]string, error) {
	retMap := make(map[string]string)
	dbKeys := []string{
		TestStateEmpty, TestStateBootstrap, TestStateBootstrapStartApproverApproval, TestStateBootstrapDoneApproverApproval,
		TestStateBootstrapStartApproverSetApproval, TestStateBootstrapDoneApproverSetApproval, TestStateBootstrapped, TestStateBootstrappedWithHost,
		TestStateBootstrappedWithContact, TestStateBootstrappedWithDomain, TestStateBootstrapAPIUser,
	}

	for _, key := range dbKeys {
		dbFileName := fmt.Sprintf("_dbatstate.%s.db", key)

		_, err := os.Stat(dbFileName)
		if err != nil {
			return retMap, fmt.Errorf("error statting db file: %w", err)
		}

		retMap[key] = dbFileName
	}

	return retMap, nil
}

func setupDatabaseCache() (map[string]string, error) {
	ret := make(map[string]string)

	path, err := CreateAndStore(TestStateEmpty, "", CreateDBSchema)
	if err != nil {
		return ret, err
	}

	ret[TestStateEmpty] = path

	path, err = CreateAndStore(TestStateBootstrap, TestStateEmpty, BootstrapDB)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrap] = path

	path, err = CreateAndStore(TestStateBootstrapStartApproverApproval, TestStateBootstrap, StartApproverApproval)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrapStartApproverApproval] = path

	path, err = CreateAndStore(TestStateBootstrapDoneApproverApproval, TestStateBootstrapStartApproverApproval, ApproverBootstrapApprover)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrapDoneApproverApproval] = path

	path, err = CreateAndStore(TestStateBootstrapStartApproverSetApproval, TestStateBootstrapDoneApproverApproval, StartApproverSetApproval)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrapStartApproverSetApproval] = path

	path, err = CreateAndStore(TestStateBootstrapDoneApproverSetApproval, TestStateBootstrapStartApproverSetApproval, ApproverSetBootstrapApproverSet)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrapDoneApproverSetApproval] = path

	path, err = CreateAndStore(TestStateBootstrapped, TestStateBootstrapDoneApproverSetApproval, FinishBootstrap)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrapped] = path

	path, err = CreateAndStore(TestStateBootstrappedWithHost, TestStateBootstrapped, SetupHostDB)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrappedWithHost] = path

	path, err = CreateAndStore(TestStateBootstrappedWithContact, TestStateBootstrappedWithHost, SetupContactDB)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrappedWithContact] = path

	path, err = CreateAndStore(TestStateBootstrappedWithDomain, TestStateBootstrappedWithContact, SetupDomainDB)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrappedWithDomain] = path

	path, err = CreateAndStore(TestStateBootstrapAPIUser, TestStateBootstrapped, SetupAPIUserDB)
	if err != nil {
		return ret, err
	}

	ret[TestStateBootstrapAPIUser] = path

	return ret, nil
}

func CreateAndStore(state string, source string, setupFunc DBSetupFunc) (string, error) {
	dbFileName := fmt.Sprintf("_dbatstate.%s.db", state)

	if state != TestStateEmpty {
		sourceDBFileName := fmt.Sprintf("_dbatstate.%s.db", source)

		err := copyFile(sourceDBFileName, dbFileName)
		if err != nil {
			return "", fmt.Errorf("Failed to copy db: %w", err)
		}
	}

	dbraw, err := gorm.Open("sqlite3", dbFileName)
	if err != nil {
		return "", fmt.Errorf("Failed to open DB: %w", err)
	}

	dbc := NewDBCache(&dbraw)

	err = setupFunc(&dbc, mustGetTestConf())
	if err != nil {
		return "", fmt.Errorf("Failed to setup db: %w", err)
	}

	return dbFileName, nil
}
