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

type DomainBaseObj struct {
	rev DomainRevision
	obj Domain
}
type DomainBaseObjs struct {
	Approved, Pending, Declined DomainBaseObj
}

func TestDomainHasRevision(t *testing.T) {
	t.Parallel()
	Convey("Given an Domain that does not have a revision", t, func() {
		domain := Domain{}
		Convey("HasRevision should return false", func() {
			So(domain.HasRevision(), ShouldBeFalse)
		})
	})

	Convey("Given an Approver Set that has a revision", t, func() {
		DoDomainDBTests(t, func(dbCache *DBCache, objs *DomainBaseObjs) {
			Convey("HasRevision should return false", func() {
				So(objs.Approved.obj.HasRevision(), ShouldBeTrue)
			})
		})
	})
}

func TestDomainExportFullGetDiff(t *testing.T) {
	t.Parallel()
	Convey("Given an DomainExportFull object with an empty pending revision", t, func() {
		domainExport := DomainExport{}
		domainExport.CurrentRevision = DomainRevisionExport{}
		domainExport.PendingRevision = DomainRevisionExport{ID: 0}

		_, err := domainExport.GetDiff()

		Convey("GetDiff should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Given an DomainExportFull object with valid revisions", t, func() {
		DoDomainDBTests(t, func(dbCache *DBCache, objs *DomainBaseObjs) {
			exportver := objs.Pending.obj.GetExportVersion()
			diff, err := exportver.GetDiff()
			Convey("There should be a JSON string returned and no error", func() {
				So(len(diff), ShouldBeGreaterThan, 0)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestDomainExportToJSON(t *testing.T) {
	t.Parallel()
	DoDomainDBTests(t, func(dbCache *DBCache, objs *DomainBaseObjs) {
		export := objs.Approved.obj.GetExportVersion()
		Convey("Given a valid DomainExportFull", t, func() {
			exportStr1, exportErr1 := export.ToJSON()
			Convey("The error returned should be nil", func() {
				So(exportErr1, ShouldBeNil)
			})

			Convey("The length of the string returned should be greater than 0", func() {
				So(len(exportStr1), ShouldBeGreaterThan, 0)
			})
		})

		Convey("Given a valid DomainExportFull with its ID changed to 0", t, func() {
			typedExport := export.(DomainExport)
			typedExport.ID = 0
			_, exportErr2 := typedExport.ToJSON()
			Convey("The error returned should not be nil", func() {
				So(exportErr2, ShouldNotBeNil)
			})
		})

		Convey("Given a valid DomainExportFull with its ID change to -1", t, func() {
			typedExport := export.(DomainExport)
			typedExport.ID = -1
			_, exportErr2 := typedExport.ToJSON()
			Convey("The error returned should not be nil", func() {
				So(exportErr2, ShouldNotBeNil)
			})
		})
	})
}

func TestDomainExportVersionAt(t *testing.T) {
	t.Parallel()
	Convey("Given an DomainExportFull object with valid revisions", t, func() {
		DoDomainDBTests(t, func(dbCache *DBCache, objs *DomainBaseObjs) {
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
				revTS := rev.PromotedTime.Unix()
				baseJSON, err := export.ToJSON()
				Convey("export should product valid JSON", func() {
					So(err, ShouldBeNil)
					exportAt, err := obj.GetExportVersionAt(dbCache, revTS)
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

func TestDomainParseFromForm(t *testing.T) {
	t.Parallel()
	DoDomainDBTests(t, func(dbCache *DBCache, objs *DomainBaseObjs) {
		Convey("Given a HTTP request with no user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			domain := Domain{}
			parseError := domain.ParseFromForm(request, dbCache)
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})
	})
}

func TestDomainParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	DoDomainDBTests(t, func(dbCache *DBCache, objs *DomainBaseObjs) {
		Convey("Given a HTTP request with no user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			domain := Domain{}
			parseError := domain.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})

		Convey("Given a HTTP request with a valid user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			request.Header.Add("REMOTE_USER", TestUser1Username)
			request.Form = make(url.Values)
			domain := Domain{}
			err := dbCache.Save(&domain)
			So(err, ShouldBeNil)
			parseError := domain.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})
	})
}

func TestDomainVerifyCR(t *testing.T) {
	t.Parallel()
	Convey("Given an bootstrap database", t, func() {
		DoDomainDBTests(t, func(dbCache *DBCache, objs *DomainBaseObjs) {
			obj := objs.Approved.obj
			obj.PendingRevision.ID = 0
			checksOut, errs := obj.VerifyCR(dbCache)
			// VerifyCR with no pending revision should return an error
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

func TestDomainSuggestedFields(t *testing.T) {
	t.Parallel()
	Convey("Given the default Domain", t, func() {
		DoDomainDBTests(t, func(dbCache *DBCache, objs *DomainBaseObjs) {
			obj := objs.Approved.obj
			tests := []struct{ name, fieldName, revFieldName, value string }{
				{"Name", DomainFieldName, "", obj.DomainName},
			}
			emptyDomain := Domain{}
			for _, test := range tests {
				Convey(fmt.Sprintf("SuggestedRevisionValue(%s) should return an empty string when there is no current revision", test.name),
					func() {
						So(emptyDomain.SuggestedRevisionValue(test.fieldName), ShouldBeBlank)
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
						So(emptyDomain.GetCurrentValue(test.fieldName), ShouldEqual, UnPreparedDomainError)
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
				{"ClientRenewProhibitedStatus", ClientRenewFlag, curRev.ClientRenewProhibitedStatus},
				{"ServerRenewProhibitedStatus", ServerRenewFlag, curRev.ServerRenewProhibitedStatus},
				{"ClientHoldStatus", ClientHoldFlag, curRev.ClientHoldStatus},
				{"ServerHoldStatus", ServerHoldFlag, curRev.ServerHoldStatus},
				{"ClientTransferProhibitedStatus", ClientTransferFlag, curRev.ClientTransferProhibitedStatus},
				{"ServerTransferProhibitedStatus", ServerTransferFlag, curRev.ServerTransferProhibitedStatus},
				{"ClientUpdateProhibitedStatus", ClientUpdateFlag, curRev.ClientUpdateProhibitedStatus},
				{"ServerUpdateProhibitedStatus", ServerUpdateFlag, curRev.ServerUpdateProhibitedStatus},
				{"DesiredState == StateActive", DesiredStateActive, curRev.DesiredState == StateActive},
				{"DesiredStateActive", DesiredStateActive, curRev.DesiredState == StateActive},
				{"DesiredStateInactive", DesiredStateInactive, curRev.DesiredState == StateInactive},
				{"DesiredStateExternal", DesiredStateExternal, curRev.DesiredState == StateExternal},
			}
			for _, test := range boolTests {
				if test.value {
					t.Logf("SuggestedRevisionBool(%s) should return true", test.name)
					So(emptyDomain.SuggestedRevisionBool(test.fieldName), ShouldBeTrue)
				} else {
					t.Logf("SuggestedRevisionBool(%s) should return false", test.name)
					So(emptyDomain.SuggestedRevisionBool(test.fieldName), ShouldBeFalse)
				}
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

func GetDomainName(name string) string {
	return fmt.Sprintf("%s.org", strings.ToUpper(name))
}

func StartDomainApproval(dbCache *DBCache, conf Config, name string) (rev DomainRevision, err error) {
	dbCache.DB = dbCache.DB.Debug()
	contactObj := ContactBaseObj{}

	if err = contactObj.Load(dbCache, "Approved"); err != nil {
		logger.Errorf("StartDomainApproval %s failed to fetch contactObj: %s", name, err.Error())

		return rev, err
	}

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)

	domain := Domain{}
	domain.DomainName = GetDomainName(name)

	if err = dbCache.Save(&domain); err != nil {
		logger.Errorf("StartDomainApproval %s failed to save domain: %s", name, err.Error())

		return rev, err
	}

	rev.DomainID = domain.ID
	rev.DomainAdminContactID = contactObj.obj.ID
	rev.DomainBillingContactID = contactObj.obj.ID
	rev.DomainTechContactID = contactObj.obj.ID
	rev.DomainRegistrantID = contactObj.obj.ID
	rev.DesiredState = StateActive
	rev.RevisionState = StateNew

	appSet, prepErr := GetDefaultApproverSet(dbCache)
	if prepErr != nil {
		err = prepErr
		logger.Errorf("StartDomainApproval %s failed to get default approver: %s", name, err.Error())

		return rev, err
	}

	rev.RequiredApproverSets = append(rev.RequiredApproverSets, appSet)

	if err = dbCache.Save(&rev); err != nil {
		logger.Errorf("StartDomainApproval %s failed save approver rev: %s", name, err.Error())

		return rev, err
	}

	if err = rev.StartApprovalProcess(request, dbCache, conf); err != nil {
		logger.Errorf("StartDomainApproval %s failed to start approval process rev: %s", name, err.Error())

		return rev, err
	}

	return rev, err
}

func SetupDomainDB(dbCache *DBCache, _ Config) error {
	logger.Info("calling SetupDomainDB")

	conf, err := getTestConf()
	if err != nil {
		return err
	}

	if err = BootstrapRegistrar(dbCache, conf); err != nil {
		return err
	}

	appRev, err := StartDomainApproval(dbCache, conf, "Approved")
	if err != nil {
		return fmt.Errorf("Failed to start Approved user: %s", err.Error())
	}

	if err = ApproveApproval(dbCache, TestUser1Username, appRev.CR.Approvals[0].ID, conf); err != nil {
		return fmt.Errorf("Failed to approve Approved user: %s", err.Error())
	}

	appRev, err = StartDomainApproval(dbCache, conf, "Declined")

	if err != nil {
		return fmt.Errorf("Failed to start Declined user: %s", err.Error())
	}

	if err = DeclineApproval(dbCache, TestUser1Username, appRev.CR.Approvals[0].ID, conf); err != nil {
		return fmt.Errorf("Failed to approve Declined user: %s", err.Error())
	}

	_, err = StartDomainApproval(dbCache, conf, "Pending")

	if err != nil {
		return fmt.Errorf("Failed to start Pending user: %s", err.Error())
	}

	return nil
}

func GetDomainDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, "DomainDB", GetContactDB, SetupDomainDB, roFunc, mustGetTestConf())
}

func (obj *DomainBaseObj) Load(t *testing.T, dbCache *DBCache, name string) (err error) {
	t.Helper()

	if err = dbCache.DB.Where("domain_name = ?", GetDomainName(name)).First(&obj.obj).Error; err != nil {
		return
	}

	if err = obj.obj.Prepare(dbCache); err != nil {
		return
	}

	if err = dbCache.DB.Where("domain_id = ?", obj.obj.ID).First(&obj.rev).Error; err != nil {
		return
	}

	if err = obj.rev.Prepare(dbCache); err != nil {
		return
	}

	return nil
}

func DoDomainDBTests(t *testing.T, testBlock func(*DBCache, *DomainBaseObjs)) {
	t.Helper()

	dbCache, err := DBFactory.GetDB(t, TestStateBootstrappedWithDomain)
	if err != nil {
		panic(err)
	}

	objs := DomainBaseObjs{}

	if err := objs.Approved.Load(t, dbCache, "Approved"); err != nil {
		t.Fatalf("fetching a Domain records threw an error: %s", err.Error())
	}

	if err := objs.Declined.Load(t, dbCache, "Declined"); err != nil {
		t.Fatalf("fetching a Domain records threw an error: %s", err.Error())
	}

	if err := objs.Pending.Load(t, dbCache, "Pending"); err != nil {
		t.Fatalf("fetching a Domain records threw an error: %s", err.Error())
	}

	testBlock(dbCache, &objs)
}
