package lib

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestApproverRevisionExportToJSON(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Revision Export object with an ID of 0", t, func() {
		a := ApproverRevisionExport{ID: 0}
		_, err := a.ToJSON()
		Convey("Calling ToJSON should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
	Convey("Given an Approver Revision Export object with an ID of -1", t, func() {
		a := ApproverRevisionExport{ID: -1}
		_, err := a.ToJSON()
		Convey("Calling ToJSON should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
}

func TestApproverRevisionExportGetDiff(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Revision Export object with an invalid ID", t, func() {
		a := ApproverRevisionExport{ID: -1}
		_, err := a.GetDiff()
		Convey("Calling GetDiff on the object should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
}

// Test to make sure an error is thrown when no remote user is set.
func TestApproverRevisionStartApprovalProcess_NoUserSet(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap database, calling start approval process with no user set", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("No error should be thrown when getting the database", func() {
			So(dberr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision", func() {
			So(prepareErr, ShouldBeNil)
		})
		saErr := app.StartApprovalProcess(request, dbCache, conf)
		Convey("Calling StartApprovalProcess with no remote user set should throw an error", func() {
			So(saErr, ShouldNotBeNil)
		})
	})
}

// Test to make sure that a non-new revision will not start an approval
// and throw an error.
func TestApproverRevisionStartApprovalProcess_NonNewRevision(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap database, calling start approval process with pending revision", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("No error should be thrown when getting the database", func() {
			So(dberr, ShouldBeNil)
		})

		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr1 := app.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision", func() {
			So(prepareErr1, ShouldBeNil)
		})
		app.RevisionState = StateCancelled
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app = ApproverRevision{}
		err = app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision after an update", func() {
			So(prepareErr2, ShouldBeNil)
		})
		saErr := app.StartApprovalProcess(request, dbCache, conf)
		Convey("Calling StartApprovalProcess with no pending revision set should throw an error", func() {
			So(saErr, ShouldNotBeNil)
		})
	})
}

// Test to make sure the approver state will be pending approval.
func TestApproverRevisionStartApprovalProcess_ExistingRevision(t *testing.T) {
	t.Parallel()
	Convey("Given the bootstrap database", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("No error should be thrown when getting the database", func() {
			So(dberr, ShouldBeNil)
		})

		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr1 := app.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision (pending revision)", func() {
			So(prepareErr1, ShouldBeNil)
		})

		approver := Approver{}
		err = approver.SetID(app.ApproverID)
		So(err, ShouldBeNil)
		prepareErr2 := approver.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver", func() {
			So(prepareErr2, ShouldBeNil)
		})

		apprCurrent := ApproverRevision{}
		err = apprCurrent.SetID(approver.CurrentRevisionID.Int64)
		So(err, ShouldBeNil)
		prepareErr3 := apprCurrent.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision (current revision)", func() {
			So(prepareErr3, ShouldBeNil)
		})

		apprCurrent.RevisionState = StateActive
		apprCurrent.DesiredState = StateActive
		err = dbCache.Save(&apprCurrent)
		So(err, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		startAppErr := app.StartApprovalProcess(request, dbCache, conf)
		Convey("No error should be thrown when starting the approval process", func() {
			So(startAppErr, ShouldBeNil)
		})

		approver = Approver{}
		err = approver.SetID(app.ApproverID)
		So(err, ShouldBeNil)
		prepareErr4 := approver.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision (pending revision) after the update", func() {
			So(prepareErr4, ShouldBeNil)
		})
		Convey("The approver's state should be Pending Approval after the process is completed", func() {
			So(approver.State, ShouldEqual, StateActivePendingApproval)
		})
	})

	Convey("Given the bootstrap database - DesiredState: inactive", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("No error should be thrown when getting the database", func() {
			So(dberr, ShouldBeNil)
		})

		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr1 := app.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision (pending revision)", func() {
			So(prepareErr1, ShouldBeNil)
		})

		approver := Approver{}
		err = approver.SetID(app.ApproverID)
		So(err, ShouldBeNil)
		prepareErr2 := approver.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver", func() {
			So(prepareErr2, ShouldBeNil)
		})

		apprCurrent := ApproverRevision{}
		err = apprCurrent.SetID(approver.CurrentRevisionID.Int64)
		So(err, ShouldBeNil)
		prepareErr3 := apprCurrent.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision (current revision)", func() {
			So(prepareErr3, ShouldBeNil)
		})

		apprCurrent.RevisionState = StateInactive
		apprCurrent.DesiredState = StateInactive
		err = dbCache.Save(&apprCurrent)
		So(err, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		startAppErr := app.StartApprovalProcess(request, dbCache, conf)
		Convey("No error should be thrown when starting the approval process", func() {
			So(startAppErr, ShouldBeNil)
		})

		approver = Approver{}
		err = approver.SetID(app.ApproverID)
		So(err, ShouldBeNil)
		prepareErr4 := approver.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision (pending revision) after the update", func() {
			So(prepareErr4, ShouldBeNil)
		})
		Convey("The approver's state should be Pending Approval after the process is completed", func() {
			So(approver.State, ShouldEqual, StateInactivePendingApproval)
		})
	})
}

// func TestApproverRevisionStartApprovalProcess_JSONCreateError(t *testing.T) {
// 	Convey("Given the bootstrap database", t, func() {
// 		dbCache, dberr  := DBFactory.GetDB(t, TestStateBootstrap)
// 		Convey("No error should be thrown when getting the database", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		approver := Approver{}
// 		approver.SetID(1)
// 		approver.Prepare(dbCache)
// 		approver.CurrentRevisionID.Int64 = approver.PendingRevision.ID
// 		dbCache.Save(&approver)
//
// 		app := ApproverRevision{}
// 		app.SetID(2)
// 		prepareErr := app.Prepare(dbCache)
// 		Convey("No error should be thrown when preparing the approver revision", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		reqeuest, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		request.Header.Add("REMOTE_USER", TestUser1Username)
//
// 		startAppErr := app.StartApprovalProcess(request, dbCache)
// 		Convey("No error should be thrown when starting the approval process for the approver revision", func() {
// 			So(startAppErr, ShouldNotBeNil)
// 		})
//
// 		approver2 := Approver{}
// 		approver2.SetID(app.ApproverID)
// 		approver2.Prepare(dbCache)
// 		Convey("The parent approver's state should be set to \"active\"", func() {
// 			So(approver2.State, ShouldEqual, StateActive)
// 		})
// 	})
// }

// func TestApproverRevisionStartApprovalProcess_NewApprover(t *testing.T) {
// 	Convey("Given a bootstrap database", t, func() {
// 		dbCache, dberr  := DBFactory.GetDB(t, TestStateBootstrap)
// 		Convey("No error should be thrown when getting the database", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		app := Approver{}
// 		app.SetID(1)
// 		prepareErr := app.Prepare(dbCache)
// 		Convey("No error should be thrown when preparing the approver", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
// 		app.CurrentRevisionID.Valid = false
// 		app.State = StateNew
// 		dbCache.Save(&app)
//
// 		appr := ApproverRevision{}
// 		appr.SetID(2)
// 		prepareErr2 := appr.Prepare(dbCache)
// 		Convey("No error should be thrown when preparing the approver revision", func() {
// 			So(prepareErr2, ShouldBeNil)
// 		})
//
// 		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		request.Header.Add("REMOTE_USER", TestUser1Username)
//
// 		startAppErr := appr.StartApprovalProcess(r, dbCache)
// 		Convey("No error should be thrown when starting the approval process for the approver revision", func() {
// 			So(startAppErr, ShouldBeNil)
// 		})
//
// 		approver := Approver{}
// 		approver.SetID(appr.ApproverID)
// 		approver.Prepare(dbCache)
// 		Convey("The parent approver's state should be set to \"new\"", func() {
// 			So(approver.State, ShouldEqual, StatePendingNew)
// 		})
// 	})
// }

// Test to make sure the approver state will be pending approval.
func TestApproverRevisionStartApprovalProcess_BootstrapRevision(t *testing.T) {
	t.Parallel()
	Convey("Given the bootstrap database", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("No error should be thrown when getting the database", func() {
			So(dberr, ShouldBeNil)
		})

		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision", func() {
			So(prepareErr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		startAppErr := app.StartApprovalProcess(request, dbCache, conf)
		Convey("No error should be thrown when starting the approval process for the approver revision", func() {
			So(startAppErr, ShouldBeNil)
		})

		approver := Approver{}
		err = approver.SetID(app.ApproverID)
		So(err, ShouldBeNil)
		err = approver.Prepare(dbCache)
		So(err, ShouldBeNil)
		Convey("The parent approver's state should be set to \"pendingbootstrap\"", func() {
			So(approver.State, ShouldEqual, StatePendingBootstrap)
		})
	})
}

func TestApproverRevisionStartApprovalProcessNoApproverSets(t *testing.T) {
	t.Parallel()
	Convey("Given the bootstrap database", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("No error should be thrown when getting the database", func() {
			So(dberr, ShouldBeNil)
		})

		app := ApproverRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr1 := app.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision - Current Revision", func() {
			So(prepareErr1, ShouldBeNil)
		})
		err = UpdateApproverSets(&app, dbCache, "RequiredApproverSets", []ApproverSet{})
		So(err, ShouldBeNil)

		app = ApproverRevision{}
		err = app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app.Prepare(dbCache)
		Convey("No error should be thrown when preparing the approver revision - Pending Revision", func() {
			So(prepareErr2, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		startAppErr := app.StartApprovalProcess(request, dbCache, conf)
		Convey("No error should be thrown when starting the approval process for the approver revision", func() {
			So(startAppErr, ShouldBeNil)
		})

		approver := Approver{}
		err = approver.SetID(app.ApproverID)
		So(err, ShouldBeNil)
		err = approver.Prepare(dbCache)
		So(err, ShouldBeNil)
		Convey("The parent approver's state should be set to \"pendingbootstrap\"", func() {
			So(approver.State, ShouldEqual, StatePendingBootstrap)
		})
	})
}

func TestApproverRevisionGetState(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision object", t, func() {
		approverRevision := ApproverRevision{}
		Convey("calling GetState with the \"active\" string", func() {
			So(approverRevision.GetState(StateActive), ShouldEqual, StateActive)
		})
		Convey("calling GetState with the \"inactive\" string", func() {
			So(approverRevision.GetState(StateInactive), ShouldEqual, StateInactive)
		})
		Convey("calling GetState with the \"foo\" string", func() {
			So(approverRevision.GetState("foo"), ShouldEqual, StateActive)
		})
	})
}

func TestApproverRevisionGetID(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Revision with an ID set to 1", t, func() {
		approverRevision := ApproverRevision{}
		err := approverRevision.SetID(1)
		So(err, ShouldBeNil)
		id := approverRevision.GetID()
		Convey("GetID should return 1", func() {
			So(id, ShouldEqual, 1)
		})
	})
}

func TestApproverRevisionSetID(t *testing.T) {
	t.Parallel()
	Convey("Given a new approver revision", t, func() {
		approverRevision := ApproverRevision{}
		err := approverRevision.SetID(1)
		Convey("The first SetID should not return an error", func() {
			So(err, ShouldBeNil)
		})

		err2 := approverRevision.SetID(2)
		Convey("The second SetID should return an error", func() {
			So(err2, ShouldNotBeNil)
		})
	})

	Convey("Given a new approver revision - less than zero test", t, func() {
		approverRevision := ApproverRevision{}
		err1 := approverRevision.SetID(-1)
		Convey("SetID should return an error when the ID is less than or equal to 0 - -1", func() {
			So(err1, ShouldNotBeNil)
		})

		approverRevision2 := ApproverRevision{}
		err2 := approverRevision2.SetID(0)
		Convey("SetID should return an error when the ID is less than or equal to 0 - 0", func() {
			So(err2, ShouldNotBeNil)
		})
	})
}

func TestApproverRevisionPrepareInvalidID(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap database, trying to prepare an approver revision with an invalid ID", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		if err != nil {
			t.Error(err)
		}

		app := ApproverRevision{Model: Model{ID: -1}}
		prepareErr := app.Prepare(dbCache)
		Convey("Prepare should throw an error", func() {
			So(prepareErr, ShouldNotBeNil)
		})
	})
}

func TestApproverRevisionIsDesiredState(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision with the desired state of \"active\"", t, func() {
		approverRevision := ApproverRevision{DesiredState: StateActive}
		Convey("calling IsDesiredState with the argument of 'active' should return true", func() {
			So(approverRevision.IsDesiredState(StateActive), ShouldBeTrue)
		})
		Convey("calling IsDesiredState with the argument of 'inactive' should return false", func() {
			So(approverRevision.IsDesiredState(StateInactive), ShouldBeFalse)
		})
	})

	Convey("Given an approver revision with the desired state of \"inactive\"", t, func() {
		approverRevision := ApproverRevision{DesiredState: StateInactive}
		Convey("calling IsDesiredState with the argument of 'active' should return false", func() {
			So(approverRevision.IsDesiredState(StateActive), ShouldBeFalse)
		})
		Convey("calling IsDesiredState with the argument of 'inactive' should return true", func() {
			So(approverRevision.IsDesiredState(StateInactive), ShouldBeTrue)
		})
	})
}

func TestApproverRevisionGetAllPage(t *testing.T) {
	t.Parallel()
	Convey("Given an empty database", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		objectPageRaw, err := app.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		objectPage := objectPageRaw.(*ApproverRevisionsPage)
		Convey("There should be no approver revisions returned", func() {
			So(len(objectPage.ApproverRevisions), ShouldEqual, 0)
		})
	})

	Convey("Given an empty database", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		objectPageRaw, err := app.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		objectPage := objectPageRaw.(*ApproverRevisionsPage)
		Convey("There should be 2 approver revisions returned", func() {
			So(len(objectPage.ApproverRevisions), ShouldEqual, 2)
		})
	})
}

func TestApproverRevisionHasHappened(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap database", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		appr := ApproverRevision{}
		err := appr.SetID(2)
		So(err, ShouldBeNil)
		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)
		Convey("HasHappened should be true for 'Updated', 'ApprovalStarted', 'Promoted' and not the others", func() {
			So(appr.HasHappened("Updated"), ShouldBeTrue)
			So(appr.HasHappened("ApprovalStarted"), ShouldBeTrue)
			So(appr.HasHappened("Promoted"), ShouldBeTrue)

			So(appr.HasHappened("ApprovalFailed"), ShouldBeFalse)
			So(appr.HasHappened("Superseded"), ShouldBeFalse)
			So(appr.HasHappened("Bogus"), ShouldBeFalse)
		})
	})
}

func TestApproverRevisionGetPage(t *testing.T) {
	t.Parallel()
	Convey("Given an bootstrap database and an approver revision without an ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapped)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		app.ApproverID = 1
		objectPageRaw, err := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		objectPage := objectPageRaw.(*ApproverRevisionPage)
		Convey("The approver page object should be populated properly", func() {
			So(objectPage.IsNew, ShouldBeTrue)
			So(objectPage.IsEditable, ShouldBeFalse)
			So(len(objectPage.PendingActions), ShouldEqual, 0)
			So(len(objectPage.ValidApproverSets), ShouldBeGreaterThan, 0)
		})
	})

	Convey("Given an bootstrap database and an approver revision with an ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		objectPageRaw, err := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		objectPage := objectPageRaw.(*ApproverRevisionPage)
		Convey("There should be 2 approver revisions returned", func() {
			So(objectPage.IsNew, ShouldBeFalse)
			So(objectPage.IsEditable, ShouldBeFalse)
			So(len(objectPage.PendingActions), ShouldEqual, 0)
			So(len(objectPage.ValidApproverSets), ShouldBeGreaterThan, 0)
		})
	})
}

func TestApproverRevisionIsEditable(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision in the \"new\" state", t, func() {
		a := ApproverRevision{RevisionState: StateNew}
		Convey("it should be editable", func() {
			So(a.IsEditable(), ShouldBeTrue)
		})
	})

	Convey("Given an approver revision not in the \"new\" state", t, func() {
		a := ApproverRevision{RevisionState: StateCancelled}
		Convey("it should not be editable", func() {
			So(a.IsEditable(), ShouldBeFalse)
		})
	})
}

func TestApproverRevisionPrepareBogusObjectID(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap database, trying to prepare a approver revision with ", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{Model: Model{ID: 1000}}
		prepareErr := app.Prepare(dbCache)
		Convey("Prepare should throw an error", func() {
			So(prepareErr, ShouldNotBeNil)
		})
	})
}

func TestApproverRevisionParseFromForm(t *testing.T) {
	t.Parallel()
	Convey("Given a HTTP request without a valid username set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Form = make(url.Values)
		app := ApproverRevision{}
		perr := app.ParseFromForm(request, dbCache)
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request with a valid username set but no revision_approver_id set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		app := ApproverRevision{}
		perr := app.ParseFromForm(request, dbCache)
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request with a valid username set but no revision_empid set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		app := ApproverRevision{}
		perr := app.ParseFromForm(request, dbCache)
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request with a valid username set but a bogus approver_set_required_id is set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		request.Form.Add("revision_empid", "1")
		request.Form.Add("approver_set_required_id", "asdas")
		app := ApproverRevision{}
		perr := app.ParseFromForm(request, dbCache)
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request with a valid username set but a bogus approver_set_informed_id is set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		request.Form.Add("revision_empid", "1")
		request.Form.Add("approver_set_informed_id", "asdas")
		app := ApproverRevision{}
		perr := app.ParseFromForm(request, dbCache)
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request with a valid username and all fields filled in", t, func() {
		// Used for testing fingerprint generation.
		testPubKey, err := getTestPubKey()
		So(err, ShouldBeNil)

		testPubKeyFP, err := keyToFingerprint(testPubKey)
		So(err, ShouldBeNil)

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		ras := ApproverSet{}
		err = dbCache.Save(&ras)
		So(err, ShouldBeNil)
		ias := ApproverSet{}
		err = dbCache.Save(&ias)
		So(err, ShouldBeNil)
		// Getting the database should not throw an error
		So(dberr, ShouldBeNil)
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		request.Form.Add("revision_empid", "2")
		request.Form.Add("revision_desiredstate", "active")
		request.Form.Add("approver_set_required_id", strconv.FormatInt(ras.ID, 10))
		request.Form.Add("approver_set_informed_id", strconv.FormatInt(ias.ID, 10))
		request.Form.Add("revision_name", "5")
		request.Form.Add("revision_email", "6")
		request.Form.Add("revision_role", "7")
		request.Form.Add("revision_username", "8")
		request.Form.Add("revision_dept", "9")
		request.Form.Add("revision_fingerprint", testPubKeyFP)
		request.Form.Add("revision_pubkey", testPubKey)
		app := ApproverRevision{}
		perr := app.ParseFromForm(request, dbCache)
		// ParseFromForm should return an error
		So(perr, ShouldBeNil)

		Convey("The resulting approver should have the expected values", func() {
			So(app.CreatedBy, ShouldEqual, TestUser1Username)
			So(app.UpdatedBy, ShouldEqual, TestUser1Username)
			So(app.ApproverID, ShouldEqual, 1)
			So(app.RevisionState, ShouldEqual, StateNew)
			So(app.DesiredState, ShouldEqual, StateActive)
			So(app.Name, ShouldEqual, "5")
			So(app.EmailAddress, ShouldEqual, "6")
			So(app.Role, ShouldEqual, "7")
			So(app.Username, ShouldEqual, "8")
			So(app.EmployeeID, ShouldEqual, 2)
			So(app.Department, ShouldEqual, "9")
			So(app.PublicKey, ShouldEqual, testPubKey)
			So(app.Fingerprint, ShouldEqual, testPubKeyFP)

			// The length of the RequiredApproverSets should be 1 and the value should be 3
			t.Log(app.RequiredApproverSets)
			So(len(app.RequiredApproverSets), ShouldEqual, 2)
			So(app.RequiredApproverSets[0].ID, ShouldEqual, 1)
			// Default approver set should be added
			So(app.RequiredApproverSets[1].ID, ShouldEqual, ras.ID)

			// The length of the InformedApproverSets should be 1 and the value should be 4
			So(len(app.InformedApproverSets), ShouldEqual, 1)
			So(app.InformedApproverSets[0].ID, ShouldEqual, ias.ID)
		})
	})
}

func TestApproverRevisionParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	Convey("Given a HTTP request without a valid ID set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Form = make(url.Values)
		app := ApproverRevision{}
		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		Convey("ParseFromFormUpdate should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request without a valid username set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Form = make(url.Values)
		app := ApproverRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		Convey("ParseFromFormUpdate should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request with a valid username set but no other fields set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		app := ApproverRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		Convey("ParseFromFormUpdate should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request with a valid username and all fields filled in", t, func() {
		// Used for testing fingerprint generation.
		testPubKey, err := getTestPubKey()
		So(err, ShouldBeNil)

		testPubKeyFP, err := keyToFingerprint(testPubKey)
		So(err, ShouldBeNil)

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		request.Form.Add("revision_empid", "2")
		request.Form.Add("revision_desiredstate", "inactive")

		request.Form.Add("approver_set_informed_id", "1")
		request.Form.Add("revision_name", "5")
		request.Form.Add("revision_email", "6")
		request.Form.Add("revision_role", "7")
		request.Form.Add("revision_username", "8")
		request.Form.Add("revision_dept", "9")
		request.Form.Add("revision_pubkey", testPubKey)
		app := ApproverRevision{}
		err = app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})
		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		ret := app
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldBeNil)
		})
		Convey("The resulting approver should have the expected values", func() {
			So(ret.DesiredState, ShouldEqual, StateInactive)
			So(ret.UpdatedBy, ShouldEqual, TestUser1Username)
			So(ret.Name, ShouldEqual, "5")
			So(ret.EmailAddress, ShouldEqual, "6")
			So(ret.Role, ShouldEqual, "7")
			So(ret.Username, ShouldEqual, "8")
			So(ret.EmployeeID, ShouldEqual, 2)
			So(ret.Department, ShouldEqual, "9")
			So(ret.PublicKey, ShouldEqual, testPubKey)
			So(ret.Fingerprint, ShouldEqual, testPubKeyFP)
		})
		dbCache.DB.Model(&app).UpdateColumns(ret)

		err = dbCache.Purge(&app)
		So(err, ShouldBeNil)

		err = dbCache.Save(&app)
		So(err, ShouldBeNil)

		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - Post Update", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The length of the RequiredApproverSets should be 1", func() {
			So(len(app2.RequiredApproverSets), ShouldEqual, 1)
			So(app2.RequiredApproverSets[0].ID, ShouldEqual, 1)
		})
		Convey("The length of the InformedApproverSets should be 1 and the value should be 1", func() {
			So(len(app2.InformedApproverSets), ShouldEqual, 1)
			So(app2.InformedApproverSets[0].ID, ShouldEqual, 1)
		})
	})

	Convey("Given a HTTP request with a valid username an invalid employee id", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser2Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		request.Form.Add("revision_empid", "abc")
		request.Form.Add("revision_desiredstate", "inactive")

		request.Form.Add("approver_set_informed_id", "1")
		request.Form.Add("revision_name", "5")
		request.Form.Add("revision_email", "6")
		request.Form.Add("revision_role", "7")
		request.Form.Add("revision_username", "8")
		request.Form.Add("revision_dept", "9")
		request.Form.Add("revision_fingerprint", "10")
		request.Form.Add("revision_pubkey", "11")
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		So(app.UpdatedBy, ShouldNotEqual, TestUser2Username)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		ret := app
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
		Convey("The resulting approver should have the expected values", func() {
			So(ret.DesiredState, ShouldNotEqual, StateInactive)
			So(ret.UpdatedBy, ShouldNotEqual, TestUser2Username)
			So(ret.Name, ShouldNotEqual, "5")
			So(ret.EmailAddress, ShouldNotEqual, "6")
			So(ret.Role, ShouldNotEqual, "7")
			So(ret.Username, ShouldNotEqual, "8")
			So(ret.EmployeeID, ShouldNotEqual, 2)
			So(ret.Department, ShouldNotEqual, "9")
			So(ret.Fingerprint, ShouldNotEqual, "10")
			So(ret.PublicKey, ShouldNotEqual, "11")
		})
	})

	Convey("Given a HTTP request with a valid username an invalid state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser2Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		request.Form.Add("revision_empid", "abc")
		request.Form.Add("revision_desiredstate", "inactive")

		request.Form.Add("approver_set_informed_id", "1")
		request.Form.Add("revision_name", "5")
		request.Form.Add("revision_email", "6")
		request.Form.Add("revision_role", "7")
		request.Form.Add("revision_username", "8")
		request.Form.Add("revision_dept", "9")
		request.Form.Add("revision_fingerprint", "10")
		request.Form.Add("revision_pubkey", "11")

		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		app.RevisionState = StateActive
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})
		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		ret := app
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
		Convey("The resulting approver should have the expected values", func() {
			So(ret.DesiredState, ShouldNotEqual, StateInactive)
			So(ret.UpdatedBy, ShouldNotEqual, TestUser2Username)
			So(ret.Name, ShouldNotEqual, "5")
			So(ret.EmailAddress, ShouldNotEqual, "6")
			So(ret.Role, ShouldNotEqual, "7")
			So(ret.Username, ShouldNotEqual, "8")
			So(ret.EmployeeID, ShouldNotEqual, 2)
			So(ret.Department, ShouldNotEqual, "9")
			So(ret.Fingerprint, ShouldNotEqual, "10")
			So(ret.PublicKey, ShouldNotEqual, "11")
		})
	})

	Convey("Given a HTTP request with a valid username an invalid informed approver set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		request.Form.Add("revision_empid", "1")
		request.Form.Add("revision_desiredstate", "inactive")

		request.Form.Add("approver_set_informed_id", "a")
		request.Form.Add("revision_name", "5")
		request.Form.Add("revision_email", "6")
		request.Form.Add("revision_role", "7")
		request.Form.Add("revision_username", "8")
		request.Form.Add("revision_dept", "9")
		request.Form.Add("revision_fingerprint", "10")
		request.Form.Add("revision_pubkey", "11")
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})
		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})

	Convey("Given a HTTP request with a valid username an required approver set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_id", "1")
		request.Form.Add("revision_empid", "1")
		request.Form.Add("revision_desiredstate", "inactive")

		request.Form.Add("approver_set_required_id", "a")
		request.Form.Add("revision_name", "5")
		request.Form.Add("revision_email", "6")
		request.Form.Add("revision_role", "7")
		request.Form.Add("revision_username", "8")
		request.Form.Add("revision_dept", "9")
		request.Form.Add("revision_fingerprint", "10")
		request.Form.Add("revision_pubkey", "11")
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})
		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		Convey("ParseFromForm should return an error", func() {
			So(perr, ShouldNotBeNil)
		})
	})
}

// func TestApproverRevisionPostUpdate(t *testing.T) {
// 	Convey("Given an approver revision", t, func() {
// 		dbCache, dberr  := DBFactory.GetDB(t, TestStateEmpty)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		appr := ApproverRevision{}
// 		Convey("Calling PostUpdate should not panic", func() {
// 			So(func() { appr.PostUpdate(dbCache) }, ShouldNotPanic)
// 		})
// 	})
// }

func TestApproverRevisionCancel(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision in the 'new' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		cancelErr := app.Cancel(dbCache, conf)
		Convey("Cancel should not throw an error", func() {
			So(cancelErr, ShouldBeNil)
		})
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after cancel", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The approver revision should have the state 'cancelled'", func() {
			So(app.RevisionState, ShouldEqual, StateCancelled)
		})
	})

	Convey("Given an approver revision in the 'bootstrap' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		cancelErr := app.Cancel(dbCache, conf)
		Convey("Cancel should not throw an error", func() {
			So(cancelErr, ShouldNotBeNil)
		})
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after cancel", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The approver revision should have the state 'cancelled'", func() {
			So(app.RevisionState, ShouldNotEqual, StateCancelled)
		})
	})
}

func TestApproverRevisionGetActions(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision in the 'new' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		actions := app.GetActions(true)
		Convey("The number of actions returned should be 3 when self", func() {
			So(len(actions), ShouldEqual, 3)
		})
		actions2 := app.GetActions(false)
		Convey("The number of actions returned should be 3 when not self", func() {
			So(len(actions2), ShouldEqual, 3)
		})
	})

	Convey("Given an approver revision in the 'pendingapproval' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		actions := app.GetActions(true)
		Convey("The number of actions returned should be 3 when self", func() {
			So(len(actions), ShouldEqual, 3)
		})
		actions2 := app.GetActions(false)
		Convey("The number of actions returned should be 2 when not self", func() {
			So(len(actions2), ShouldEqual, 2)
		})
	})

	Convey("Given an approver revision in the 'cancelled' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		app.RevisionState = StateCancelled
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		actions := app.GetActions(true)
		Convey("The number of actions returned should be 2 when self", func() {
			So(len(actions), ShouldEqual, 2)
		})
		actions2 := app.GetActions(false)
		Convey("The number of actions returned should be 1 when not self", func() {
			So(len(actions2), ShouldEqual, 1)
		})
	})
}

func TestApproverRevisionTakeAction(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision in the 'new' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		cancelErr := app.TakeAction(response, request, dbCache, "cancel", true, RemoteUserAuthType, conf)
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after cancel", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The cancel action should not return an error and should redirect with a 302", func() {
			So(cancelErr, ShouldBeNil)
			So(app2.RevisionState, ShouldEqual, StateCancelled)
			So(response.Code, ShouldEqual, http.StatusFound)
		})
	})

	Convey("Given an approver revision in the 'bootstrap' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		cancelErr := app.TakeAction(response, request, dbCache, "cancel", false, RemoteUserAuthType, conf)
		app2 := ApproverRevision{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after cancel", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The cancel action should not return an error and should not redirect with a 302", func() {
			So(cancelErr, ShouldNotBeNil)
			So(app2.RevisionState, ShouldEqual, StateBootstrap)
			So(response.Code, ShouldNotEqual, http.StatusFound)
		})
	})

	Convey("Given an approver revision in the 'new' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		startapprovalErr := app.TakeAction(response, request, dbCache, "startapproval", true, RemoteUserAuthType, conf)
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after startapproval", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The startapproval action should not return an error and should redirect with a 302", func() {
			So(startapprovalErr, ShouldBeNil)
			So(app2.RevisionState, ShouldEqual, StatePendingApproval)
			So(response.Code, ShouldEqual, http.StatusFound)
		})
	})

	Convey("Given an approver revision in the 'bootstrap' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		startapprovalErr := app.TakeAction(response, request, dbCache, "startapproval", false, RemoteUserAuthType, conf)
		app2 := ApproverRevision{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after startapproval", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The startapproval action should return an error and should not redirect with a 302", func() {
			So(startapprovalErr, ShouldNotBeNil)
			So(app2.RevisionState, ShouldEqual, StateBootstrap)
			So(response.Code, ShouldNotEqual, http.StatusFound)
		})
	})

	Convey("Given an approver revision in the 'new' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		gotocrErr := app.TakeAction(response, request, dbCache, "gotochangerequest", false, RemoteUserAuthType, conf)
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after gotochangerequest", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The gotochangerequest action should return an error and should not redirect with a 302", func() {
			So(gotocrErr, ShouldNotBeNil)
			So(app2.RevisionState, ShouldEqual, StateNew)
			So(response.Code, ShouldNotEqual, http.StatusFound)
		})
	})

	Convey("Given an approver revision in the 'pendingapproval' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		gotocrErr := app.TakeAction(response, request, dbCache, "gotochangerequest", false, RemoteUserAuthType, conf)
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after gotochangerequest", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The gotochangerequest action should not return an error and should redirect with a 302", func() {
			So(gotocrErr, ShouldBeNil)
			So(response.Code, ShouldEqual, http.StatusFound)
		})
	})

	Convey("Given an approver revision in the 'new' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		bogusErr := app.TakeAction(response, request, dbCache, bogusState, false, RemoteUserAuthType, conf)
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after bogus", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The bogus action should return an error and should not redirect with a 302", func() {
			So(bogusErr, ShouldNotBeNil)
			So(response.Code, ShouldNotEqual, http.StatusFound)
		})
	})
}

func TestApproverRevisionPromote(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision in the 'pendingapproval' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		promoteErr := app.Promote(dbCache)
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after Promote", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The approver revision promotion should have failed and should be in the pendingapproval state", func() {
			So(promoteErr, ShouldNotBeNil)
			So(app2.RevisionState, ShouldEqual, StatePendingApproval)
		})
	})

	Convey("Given an approver revision in the 'new' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		promoteErr := app.Promote(dbCache)
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after Promote", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The approver revision promotion should have failed and should be in the pendingapproval state", func() {
			So(promoteErr, ShouldNotBeNil)
			So(app2.RevisionState, ShouldEqual, StateNew)
		})
	})
}

func TestApproverRevisionDecline(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision in the 'new' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		declineErr := app.Decline(dbCache)
		app2 := ApproverRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		Convey("Preparing the approver revision should not return an error - after Decline", func() {
			So(prepareErr2, ShouldBeNil)
		})
		Convey("The approver revision declined should have failed and should be in the pendingapproval state", func() {
			So(declineErr, ShouldNotBeNil)
			So(app2.RevisionState, ShouldEqual, StateNew)
		})
	})
}

func TestApproverRevisionIsCanclled(t *testing.T) {
	t.Parallel()
	Convey("Given an approver revision in the 'new' state", t, func() {
		app := ApproverRevision{}
		app.RevisionState = StateNew
		Convey("IsCancelled should return false", func() {
			So(app.IsCancelled(), ShouldBeFalse)
		})
	})

	Convey("Given an approver revision in the 'cancelled' state", t, func() {
		app := ApproverRevision{}
		app.RevisionState = StateCancelled
		Convey("IsCancelled should return true", func() {
			So(app.IsCancelled(), ShouldBeTrue)
		})
	})
}

func TestApproverRevisionPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverRevisionPage", t, func() {
		approverRevisionPage := ApproverRevisionPage{}
		Convey("Calling SetCSRFToken should not panic", func() {
			So(func() { approverRevisionPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)
		})

		testTokenString := testingCSRFToken
		approverRevisionPage.CSRFToken = testTokenString
		Convey("Calling GetCSRFToken should return the testing token string", func() {
			So(approverRevisionPage.GetCSRFToken(), ShouldEqual, testTokenString)
		})
	})
}

func TestApproverRevisionsPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverRevisionsPage", t, func() {
		approverRevisionPage := ApproverRevisionsPage{}
		Convey("Calling SetCSRFToken should not panic", func() {
			So(func() { approverRevisionPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)
		})

		testTokenString := testingCSRFToken
		approverRevisionPage.CSRFToken = testTokenString
		Convey("Calling GetCSRFToken should return the testing token string", func() {
			So(approverRevisionPage.GetCSRFToken(), ShouldEqual, testTokenString)
		})
	})
}
