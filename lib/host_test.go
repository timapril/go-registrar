package lib

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type HostBaseObj struct {
	rev HostRevision
	obj Host
}
type HostBaseObjs struct {
	Approved, Pending, Declined HostBaseObj
}

func TestHostExportFullGetDiff(t *testing.T) {
	t.Parallel()
	Convey("Given an HostExportFull object with an empty pending revision", t, func() {
		host := HostExport{}
		host.CurrentRevision = HostRevisionExport{}
		host.PendingRevision = HostRevisionExport{ID: 0}

		_, err := host.GetDiff()

		Convey("GetDiff should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
}

//func TestHostExportFullToJSON(t *testing.T) {
//	db, err  := DBFactory.GetDB(t, TestStateBootstrap)
//	if err != nil {
//		return
//	}
//	h := Host{}
//	h.SetID(1)
//	h.Prepare(db)
//
//	export := h.GetExportVersion()
//
//	Convey("Given a valid HostExportFull", t, func() {
//		exportStr1, exportErr1 := export.ToJSON()
//		Convey("The error returned should be nil", func() {
//			So(exportErr1, ShouldBeNil)
//		})
//
//		Convey("The length of the string returned should be greater than 0", func() {
//			So(len(exportStr1), ShouldBeGreaterThan, 0)
//		})
//
//	})
//
//	Convey("Given a valid HostExportFull with its ID changed to 0", t, func() {
//		typedExport := export.(HostExport)
//		typedExport.ID = 0
//		_, exportErr2 := typedExport.ToJSON()
//		Convey("The error returned should not be nil", func() {
//			So(exportErr2, ShouldNotBeNil)
//		})
//
//	})
//
//	Convey("Given a valid HostExportFull with its ID change to -1", t, func() {
//		typedExport := export.(HostExport)
//		typedExport.ID = -1
//		_, exportErr2 := typedExport.ToJSON()
//		Convey("The error returned should not be nil", func() {
//			So(exportErr2, ShouldNotBeNil)
//		})
//	})
//}

func TestHostExportVersionAt(t *testing.T) {
	t.Parallel()
	Convey("Given an HostExportFull object with valid revisions", t, func() {
		DoHostDBTests(t, func(dbCache *DBCache, objs *HostBaseObjs) {
			dbCache.DB = dbCache.DB.Debug()
			obj := objs.Approved.obj
			rev := objs.Approved.rev
			export := obj.GetExportVersion()
			Convey("At invalidly old time", func() {
				_, err := obj.GetExportVersionAt(dbCache, 0)
				Convey("Should return error", func() {
					So(err, ShouldNotBeNil)
				})
			})
			Convey("At Time of Current Revision", func() {
				rervTS := rev.PromotedTime.Unix()
				baseJSON, err := export.ToJSON()
				Convey("export should product valid JSON", func() {
					So(err, ShouldBeNil)
					exportAt, err := obj.GetExportVersionAt(dbCache, rervTS)
					Convey("Should return valid export", func() {
						So(err, ShouldBeNil)
						Convey("exportAt should product valid JSON", func() {
							atJSON, err := exportAt.ToJSON()
							So(err, ShouldBeNil)
							Convey("Json should match", func() {
								So(atJSON, ShouldEqual, baseJSON)
							})
						})
					})
				})
			})
		})
	})
}

func TestHostParseFromForm(t *testing.T) {
	t.Parallel()
	DoHostDBTests(t, func(dbCache *DBCache, objs *HostBaseObjs) {
		Convey("Given a HTTP request with no user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			host := Host{}
			parseError := host.ParseFromForm(request, dbCache)
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})
	})
}

func TestHostParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	DoHostDBTests(t, func(dbCache *DBCache, objs *HostBaseObjs) {
		Convey("Given a HTTP request with no user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			host := Host{}
			parseError := host.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})

		Convey("Given a HTTP request with a valid user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			request.Header.Add("REMOTE_USER", TestUser1Username)
			request.Form = make(url.Values)
			host := Host{}
			err := dbCache.Save(&host)
			So(err, ShouldBeNil)
			parseError := host.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})
	})
}

func TestHostVerifyCR(t *testing.T) {
	t.Parallel()
	Convey("Given an bootstrap database", t, func() {
		DoHostDBTests(t, func(dbCache *DBCache, objs *HostBaseObjs) {
			obj := objs.Approved.obj
			obj.PendingRevision.ID = 0
			checksOut, errs := obj.VerifyCR(dbCache)
			/// VerifyCR with no pending revision should return an error
			So(checksOut, ShouldBeFalse)
			So(len(errs), ShouldEqual, 1)

			obj = objs.Approved.obj
			obj.PendingRevision.CRID.Valid = false
			checksOut, errs = obj.VerifyCR(dbCache)
			// VerifyCR with no CR should return an error
			So(checksOut, ShouldBeFalse)
			So(len(errs), ShouldEqual, 1)

			checksOut, errs = objs.Pending.obj.VerifyCR(dbCache)
			// VerifyCR on a Pending User should fail
			So(checksOut, ShouldBeFalse)
			So(len(errs), ShouldEqual, 1)

			obj = objs.Pending.obj
			dbCache.DB = dbCache.DB.Debug()
			AlmostApprove(t, dbCache, &obj, &obj.PendingRevision, &obj.PendingRevision.CR, mustGetTestConf())
			checksOut, errs = obj.VerifyCR(dbCache)
			// VerifyCR on a Pending User should not return an error
			So(checksOut, ShouldBeTrue)
			So(errs, ShouldBeEmpty)
		})
	})
}

func TestHostSuggestedFields(t *testing.T) {
	t.Parallel()
	Convey("Given the default Host", t, func() {
		DoHostDBTests(t, func(dbCache *DBCache, objs *HostBaseObjs) {
			obj := objs.Approved.obj
			tests := []struct{ name, fieldName, revFieldName, value string }{
				{"Name", HostFieldName, "", obj.HostName},
			}
			emptyHost := Host{}
			for _, test := range tests {
				Convey(fmt.Sprintf("SuggestedRevisionValue(%s) should return an empty string when there is no current revision", test.name),
					func() {
						So(emptyHost.SuggestedRevisionValue(test.fieldName), ShouldBeBlank)
					})
			}
			// API is loaded but not prepared, suggested revision values should work fine
			for _, test := range tests {
				Convey(fmt.Sprintf("SuggestedRevisionValue(%s) should be the same as the current revision's %s Field",
					test.name, test.revFieldName),
					func() {
						So(objs.Approved.obj.SuggestedRevisionValue(test.fieldName), ShouldEqual, test.value)
					})
			}
			for _, test := range tests {
				Convey(fmt.Sprintf("GetCurrentValue(%s) should return an empty string when there is no current revision",
					test.name),
					func() {
						So(emptyHost.GetCurrentValue(test.fieldName), ShouldEqual, UnPreparedHostError)
					})
			}
			for _, test := range tests {
				Convey(fmt.Sprintf("GetCurrentValue(%s) should be the same as the current revision's %s Field",
					test.name, test.revFieldName),
					func() {
						So(objs.Approved.obj.GetCurrentValue(test.fieldName), ShouldEqual, test.value)
					})
			}
			curRev := objs.Approved.rev
			boolTests := []struct {
				name, fieldName string
				value           bool
			}{
				{"ClientDeleteProhibitedStatus", ClientDeleteFlag, curRev.ClientDeleteProhibitedStatus},
				{"ServerDeleteProhibitedStatus", ServerDeleteFlag, curRev.ServerDeleteProhibitedStatus},
				{"ClientTransferProhibitedStatus", ClientTransferFlag, curRev.ClientTransferProhibitedStatus},
				{"ServerTransferProhibitedStatus", ServerTransferFlag, curRev.ServerTransferProhibitedStatus},
				{"ClientUpdateProhibitedStatus", ClientUpdateFlag, curRev.ClientUpdateProhibitedStatus},
				{"ServerUpdateProhibitedStatus", ServerUpdateFlag, curRev.ServerUpdateProhibitedStatus},
				{"DesiredState == StateActive", DesiredStateActive, curRev.DesiredState == StateActive},
				{"DesiredStateActive", DesiredStateActive, curRev.DesiredState == StateActive},
				{"DesiredStateInactive", DesiredStateInactive, curRev.DesiredState == StateInactive},
			}
			for _, test := range boolTests {
				Convey(fmt.Sprintf("SuggestedRevisionBool(%s) should return false", test.name),
					func() {
						So(emptyHost.SuggestedRevisionBool(test.fieldName), ShouldBeFalse)
					})
			}
			for _, test := range boolTests {
				Convey(fmt.Sprintf("SuggestedRevisionBool(%s) should be equal to value", test.name),
					func() {
						So(objs.Approved.obj.SuggestedRevisionBool(test.fieldName), ShouldEqual, test.value)
					})
			}
		})
	})
}

func GetHostName(name string) string {
	return fmt.Sprintf("NS1.%s.org", strings.ToUpper(name))
}

func StartHostApproval(dbCache *DBCache, conf Config, name string) (rev HostRevision, err error) {
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)

	host := Host{}
	host.HostName = GetHostName(name)

	if err = dbCache.Save(&host); err != nil {
		return rev, err
	}

	rev.HostAddresses = []HostAddress{{IPAddress: "::1", Protocol: 6}, {IPAddress: "127.0.0.1", Protocol: 4}}
	rev.HostID = host.ID
	rev.DesiredState = StateActive
	rev.RevisionState = StateNew

	appSet, prepErr := GetDefaultApproverSet(dbCache)
	if prepErr != nil {
		err = prepErr

		return rev, err
	}

	rev.RequiredApproverSets = append(rev.RequiredApproverSets, appSet)

	if err = dbCache.Save(&rev); err != nil {
		return rev, err
	}

	if err = rev.StartApprovalProcess(request, dbCache, conf); err != nil {
		return rev, err
	}

	return rev, err
}

func SetupHostDB(dbCache *DBCache, _ Config) error {
	logger.Info("calling SetupHostDB")

	conf, err := getTestConf()
	if err != nil {
		return err
	}

	if err = BootstrapRegistrar(dbCache, conf); err != nil {
		return err
	}

	appRev, err := StartHostApproval(dbCache, conf, "Approved")
	if err != nil {
		return fmt.Errorf("Failed to start Approved user: %s", err.Error())
	}

	if err = ApproveApproval(dbCache, TestUser1Username, appRev.CR.Approvals[0].ID, conf); err != nil {
		return fmt.Errorf("Failed to approve Approved user: %s", err.Error())
	}

	appRev, err = StartHostApproval(dbCache, conf, "Declined")

	if err != nil {
		return fmt.Errorf("Failed to start Declined user: %s", err.Error())
	}

	if err = DeclineApproval(dbCache, TestUser1Username, appRev.CR.Approvals[0].ID, conf); err != nil {
		return fmt.Errorf("Failed to approve Declined user: %s", err.Error())
	}

	_, err = StartHostApproval(dbCache, conf, "Pending")

	if err != nil {
		return fmt.Errorf("Failed to start Pending user: %s", err.Error())
	}

	return nil
}

func GetHostDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, "HostDB", GetBootstrappedDB, SetupHostDB, roFunc, mustGetTestConf())
}

func (obj *HostBaseObj) Load(t *testing.T, dbCache *DBCache, name string) (err error) {
	t.Helper()

	dbCache.DB = dbCache.DB.Debug()
	if err = dbCache.DB.Where("host_name = ?", GetHostName(name)).First(&obj.obj).Error; err != nil {
		return
	}

	if err = obj.obj.Prepare(dbCache); err != nil {
		return
	}

	if err = dbCache.DB.Where("host_id = ?", obj.obj.ID).First(&obj.rev).Error; err != nil {
		return
	}

	if err = obj.rev.Prepare(dbCache); err != nil {
		return
	}

	return nil
}

func DoHostDBTests(t *testing.T, testBlock func(*DBCache, *HostBaseObjs)) {
	t.Helper()

	dbCache, err := DBFactory.GetDB(t, TestStateBootstrappedWithHost)
	if err != nil {
		panic(err)
	}

	objs := HostBaseObjs{}

	if err = objs.Approved.Load(t, dbCache, "Approved"); err != nil {
		t.Fatalf("fetching a Host records threw an error: %s", err.Error())
	}

	if err = objs.Declined.Load(t, dbCache, "Declined"); err != nil {
		t.Fatalf("fetching a Host records threw an error: %s", err.Error())
	}

	if err = objs.Pending.Load(t, dbCache, "Pending"); err != nil {
		t.Fatalf("fetching a Host records threw an error: %s", err.Error())
	}

	testBlock(dbCache, &objs)
}
