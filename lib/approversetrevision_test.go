package lib

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestApproverSetRevisionExportGetDiff(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverSetRevisionExport", t, func() {
		asre := ApproverSetRevisionExport{}
		diff, err := asre.GetDiff()
		// GetDiff should return an empty diff and an error
		So(len(diff), ShouldEqual, 0)
		So(err, ShouldNotBeNil)
	})
}

func TestApproverSetRevisionGetState(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision object", t, func() {
		asr := ApproverSetRevision{}
		// calling GetState with the \"active\" string
		So(asr.GetState(StateActive), ShouldEqual, StateActive)
		// calling GetState with the \"inactive\" string
		So(asr.GetState(StateInactive), ShouldEqual, StateInactive)
		// calling GetState with the \"foo\" string
		So(asr.GetState("foo"), ShouldEqual, StateActive)
	})
}

// func TestApproverSetRevisionPostUpdate(t *testing.T) {
// 	Convey("Given an approver set revision", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateEmpty)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		asr := ApproverSetRevision{}
// 		Convey("Calling PostUpdate should not panic", func() {
// 			So(func() { asr.PostUpdate(db) }, ShouldNotPanic)
// 		})
// 	})
// }

func TestApproverSetRevisionUpdateState(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrapped database", t, func() {
		conf := mustGetTestConf()
		conf.Logging.DatabaseDebugging = true

		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapDoneApproverSetApproval)
		if err != nil {
			return
		}

		apps := ApproverSet{}
		err = apps.SetID(1)
		So(err, ShouldBeNil)
		err = apps.Prepare(dbCache)
		So(err, ShouldBeNil)
		err = apps.CurrentRevision.Prepare(dbCache)
		So(err, ShouldBeNil)

		newAppr := ApproverSetRevision{}
		newAppr.ApproverSetID = apps.CurrentRevision.ApproverSetID
		newAppr.DesiredState = apps.CurrentRevision.DesiredState
		newAppr.RevisionState = StateNew

		newAppr.CreatedAt = TimeNow()
		newAppr.CreatedBy = TestUser1Username
		newAppr.UpdatedAt = TimeNow()
		newAppr.UpdatedBy = TestUser1Username
		newAppr.Approvers = append(newAppr.Approvers, apps.CurrentRevision.Approvers...)
		newAppr.RequiredApproverSets = append(newAppr.RequiredApproverSets, apps.CurrentRevision.RequiredApproverSets...)
		newAppr.RequiredApproverSets = append(newAppr.RequiredApproverSets, apps.CurrentRevision.RequiredApproverSets...)

		err = dbCache.Save(&newAppr)
		So(err, ShouldBeNil)

		app2 := ApproverSet{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		err = app2.Prepare(dbCache)
		So(err, ShouldBeNil)
		changeMade, errs := app2.UpdateState(dbCache, conf)
		// Update state should not return any errors
		So(len(errs), ShouldEqual, 0)
		So(changeMade, ShouldBeFalse)

		revision := ApproverSetRevision{}
		err = revision.SetID(newAppr.ID)
		So(err, ShouldBeNil)
		err = revision.Prepare(dbCache)
		So(err, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		startAppErr := revision.StartApprovalProcess(r, dbCache, conf)

		revision = ApproverSetRevision{}
		err = revision.SetID(newAppr.ID)
		So(err, ShouldBeNil)
		err = revision.Prepare(dbCache)
		So(err, ShouldBeNil)

		// StartApprovalProcess on the new revision should work and a CRID should be set
		So(startAppErr, ShouldBeNil)
		So(revision.CRID.Valid, ShouldBeTrue)

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(revision.CRID.Int64)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)
		for _, app := range changeRequest.Approvals {
			err = DeclineApproval(dbCache, TestUser1Username, app.ID, conf)
			So(err, ShouldBeNil)
		}

		app3 := Approver{}
		err = app3.SetID(1)
		So(err, ShouldBeNil)
		err = app3.Prepare(dbCache)
		So(err, ShouldBeNil)
	})
}

func TestApproverSetRevisionGetID(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision with a valid ID", t, func() {
		asr := ApproverSetRevision{}
		var testingID int64 = 1
		err := asr.SetID(testingID)
		So(err, ShouldBeNil)
		returnedID := asr.GetID()
		// GetID should return the value of the ID for the object
		So(returnedID, ShouldEqual, testingID)
	})
}

func TestApproverSetRevisionSetID(t *testing.T) {
	t.Parallel()
	Convey("Given a new approver set revision", t, func() {
		approverSetRevision := ApproverSetRevision{}
		err := approverSetRevision.SetID(1)
		// The first SetID should not return an error
		So(err, ShouldBeNil)

		err2 := approverSetRevision.SetID(2)
		// The second SetID should return an error
		So(err2, ShouldNotBeNil)
	})

	Convey("Given a new approver set revision", t, func() {
		a := ApproverSetRevision{}
		err := a.SetID(-1)
		// SetID with a value less than 0 should return an error
		So(err, ShouldNotBeNil)
	})
}

func TestApproverSetRevisionIsDesiredState(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision with the desired state of \"active\"", t, func() {
		a := ApproverSetRevision{DesiredState: StateActive}
		// calling IsDesiredState with the argument of 'active' should return true"
		So(a.IsDesiredState(StateActive), ShouldBeTrue)

		// calling IsDesiredState with the argument of 'inactive' should return false
		So(a.IsDesiredState(StateInactive), ShouldBeFalse)
	})

	Convey("Given an approver set revision with the desired state of \"inactive\"", t, func() {
		a := ApproverSetRevision{DesiredState: StateInactive}
		// calling IsDesiredState with the argument of 'active' should return false
		So(a.IsDesiredState(StateActive), ShouldBeFalse)

		// calling IsDesiredState with the argument of 'inactive' should return true
		So(a.IsDesiredState(StateInactive), ShouldBeTrue)
	})
}

func TestApproverSetRevisionIsEditable(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision in the \"new\" state", t, func() {
		a := ApproverSetRevision{RevisionState: StateNew}
		// it should be editable
		So(a.IsEditable(), ShouldBeTrue)
	})

	Convey("Given an approver set revision not in the \"new\" state", t, func() {
		a := ApproverSetRevision{RevisionState: StateCancelled}
		// it should not be editable
		So(a.IsEditable(), ShouldBeFalse)
	})
}

func TestApproverRevisionPrepare(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap database, trying to prepare a approver set revision with a bogus ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		asr := ApproverSetRevision{Model: Model{ID: 1000}}
		prepareErr := asr.Prepare(dbCache)
		// Prepare should throw an error
		So(prepareErr, ShouldNotBeNil)
	})

	Convey("Given a bootstrap database, trying to prepare an approver set revision with an invalid ID", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		asr := ApproverSetRevision{Model: Model{ID: -1}}
		prepareErr := asr.Prepare(dbCache)
		// Prepare should throw an error
		So(prepareErr, ShouldNotBeNil)
	})

	Convey("Given a bootstrap database, trying to prepare an approver set revision that has already been prepared", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		asr := ApproverSetRevision{}
		err = asr.SetID(1)
		So(err, ShouldBeNil)
		prepareErr1 := asr.Prepare(dbCache)
		// Prepare should not throw an error the first time
		So(prepareErr1, ShouldBeNil)

		prepareErr2 := asr.Prepare(dbCache)
		// Prepare should not throw an error the second time
		So(prepareErr2, ShouldBeNil)
	})
}

func TestApproverSetRevisionPrepareShallow(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision without a valid ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		asr := ApproverSetRevision{}
		asr.ID = 0
		prepareErr := asr.PrepareShallow(dbCache)
		// PrepareShallow should return an error
		So(prepareErr, ShouldNotBeNil)
	})

	Convey("Given an approver set with a valid ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		asr := ApproverSetRevision{}
		err := asr.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := asr.PrepareShallow(dbCache)
		// PrepareShallow should not return an error
		So(prepareErr, ShouldBeNil)
	})

	Convey("Given an approver set with a non-zero invalid ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		asr := ApproverSetRevision{}
		err := asr.SetID(200)
		So(err, ShouldBeNil)
		prepareErr := asr.PrepareShallow(dbCache)
		// PrepareShallow should return an error
		So(prepareErr, ShouldNotBeNil)
	})

	Convey("Given an approver set with a valid ID that has been prepared", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		asr := ApproverSetRevision{}
		err := asr.SetID(1)
		So(err, ShouldBeNil)
		prepareErr1 := asr.PrepareShallow(dbCache)
		// PrepareShallow should not return an error the first time
		So(prepareErr1, ShouldBeNil)

		prepareErr2 := asr.PrepareShallow(dbCache)
		// PrepareShallow should not return an error the second time
		So(prepareErr2, ShouldBeNil)
	})
}

func TestApproverSetRevisionGetPage(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision with a valid ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// reparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		rawpage, err := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedPage := rawpage.(*ApproverSetRevisionPage)
		// The approver set revision page object should be populated properly
		So(typedPage.IsEditable, ShouldBeTrue)
		So(typedPage.IsNew, ShouldBeFalse)
		So(len(typedPage.PendingActions), ShouldEqual, 4)
		So(typedPage.Revision.ID, ShouldEqual, 2)
		So(len(typedPage.ValidApprovers), ShouldEqual, 1)
		So(len(typedPage.ValidApproverSets), ShouldEqual, 1)
	})
}

func TestApproverSetRevisionGetAllPage(t *testing.T) {
	t.Parallel()
	Convey("Given an aprover set revision", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		rawpage, err := app.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedPage := rawpage.(*ApproverSetRevisionsPage)
		// No items should be retuned in the list of approver sets (its not implemented yet)
		So(len(typedPage.ApproverSetRevisions), ShouldEqual, 0)
	})
}

func TestApproverSetRevisionCancel(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision that is in the 'new' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSetRevision := ApproverSetRevision{}
		err := approverSetRevision.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := approverSetRevision.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		cancelErr := approverSetRevision.Cancel(dbCache, conf)

		approverSetRevision2 := ApproverSetRevision{}
		err = approverSetRevision2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := approverSetRevision2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after cancel
		So(prepareErr2, ShouldBeNil)

		// Calling cancel should not return an error and the state should be cancelled
		So(cancelErr, ShouldBeNil)
		So(approverSetRevision2.RevisionState, ShouldEqual, StateCancelled)
	})

	Convey("Given an approver set revision that is in the 'bootstrap' state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		cancelErr := app.Cancel(dbCache, conf)

		app2 := ApproverSetRevision{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after cancel
		So(prepareErr2, ShouldBeNil)

		// Calling cancel should not return an error and the state should be cancelled
		So(cancelErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StateBootstrap)
	})
}

func TestApproverSetRevisionTakeAction(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision in the 'new' state, TakeAction with the action of 'cancel'", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		cancelErr := app.TakeAction(response, request, dbCache, ApproverSetRevisionActionCancel, true, RemoteUserAuthType, conf)
		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after TakeAction
		So(prepareErr2, ShouldBeNil)

		// calling TakeAction with the 'cancel' action
		So(cancelErr, ShouldBeNil)
		So(app2.RevisionState, ShouldEqual, StateCancelled)
		So(response.Code, ShouldEqual, http.StatusFound)
	})

	Convey("Given an approver set revision in the 'new' state, TakeAction with the action of 'startapproval'", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSetRevision := ApproverSetRevision{}
		err := approverSetRevision.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := approverSetRevision.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()

		cancelErr := approverSetRevision.TakeAction(response, request, dbCache, ApproverSetRevisionActionStartApproval, true, RemoteUserAuthType, conf)
		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after TakeAction
		So(prepareErr2, ShouldBeNil)

		// calling TakeAction with the 'startapproval' action
		So(cancelErr, ShouldBeNil)
		So(app2.RevisionState, ShouldEqual, StatePendingApproval)
		So(response.Code, ShouldEqual, http.StatusFound)

		r2, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		w2 := httptest.NewRecorder()
		cancelErr2 := app2.TakeAction(w2, r2, dbCache, ApproverSetRevisionActionStartApproval, false, RemoteUserAuthType, conf)
		app3 := ApproverSetRevision{}
		err = app3.SetID(2)
		So(err, ShouldBeNil)
		prepareErr3 := app3.Prepare(dbCache)
		// Preparing the approver set revision should return an error - after approval and second TakeAction
		So(prepareErr3, ShouldBeNil)

		// calling TakeAction with the 'startapproval' action the second time
		So(cancelErr2, ShouldNotBeNil)
		So(app3.RevisionState, ShouldEqual, StatePendingApproval)
	})

	Convey("Given an approver set revision in the 'bootstrap' state, TakeAction with the action of 'cancel'", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w := httptest.NewRecorder()

		cancelErr := app.TakeAction(w, r, dbCache, ApproverSetRevisionActionCancel, false, RemoteUserAuthType, conf)
		app2 := ApproverSetRevision{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after TakeAction
		So(prepareErr2, ShouldBeNil)

		// calling TakeAction with the 'cancel' action
		So(cancelErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StateBootstrap)
	})

	Convey("Given an approver set revision in the 'bootstrap' state, TakeAction with the action of 'bogusaction'", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w := httptest.NewRecorder()

		cancelErr := app.TakeAction(w, r, dbCache, "bogusaction", false, RemoteUserAuthType, conf)
		app2 := ApproverSetRevision{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after TakeAction
		So(prepareErr2, ShouldBeNil)

		// calling TakeAction with the 'cancel' action
		So(cancelErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StateBootstrap)
	})

	Convey("Given an approver set revision in the 'bootstrap' state, TakeAction with the action of 'gotochangerequest'", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		// Getting the database should not throw an error
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w := httptest.NewRecorder()
		cancelErr := app.TakeAction(w, r, dbCache, ApproverSetRevisionActionGOTOChangeRequest, false, RemoteUserAuthType, conf)
		app2 := ApproverSetRevision{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after TakeAction
		So(prepareErr2, ShouldBeNil)

		// calling TakeAction with the 'gotochangerequest' action
		So(cancelErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StateBootstrap)
	})

	Convey("Given an approver set revision in the 'pendingapproval' state, TakeAction with the action of 'gotochangerequest'", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		response := httptest.NewRecorder()
		cancelErr := app.TakeAction(response, request, dbCache, ApproverSetRevisionActionGOTOChangeRequest, false, RemoteUserAuthType, conf)
		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after TakeAction
		So(prepareErr2, ShouldBeNil)

		// calling TakeAction with the 'gotochangerequest' action
		So(cancelErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StatePendingApproval)
		So(response.Code, ShouldEqual, http.StatusFound)
		So(response.Header()["Location"][0], ShouldStartWith, "/view/changerequest/")
	})
}

func TestApproverSetRevisionSupersed(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set in the 'active' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		supersedErr := app.Supersed(dbCache)
		app2 := ApproverSetRevision{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after Supersed
		So(prepareErr2, ShouldBeNil)

		// Supersed should work and all of the correct values should be set
		So(supersedErr, ShouldBeNil)
		So(app2.RevisionState, ShouldEqual, StateSuperseded)
		So(app2.SupersededTime.UnixNano(), ShouldBeGreaterThan, 0)
		So(*app2.SupersededTime, ShouldHappenOnOrBefore, TimeNow())
	})

	Convey("Given an approver set revision in the 'pending' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		// Getting the database should not throw an error
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		supersedErr := app.Supersed(dbCache)
		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after Supersed
		So(prepareErr2, ShouldBeNil)

		// Supersed should not work and the declined timestamp should not be set
		So(supersedErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StatePendingApproval)
		So(app.SupersededTime, ShouldBeNil)
	})
}

func TestApproverSetRevisionDecline(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set in the 'new' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		supersedErr := app.Decline(dbCache)
		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after Supersed
		So(prepareErr2, ShouldBeNil)

		// Decline should not work and the approval failed timestamp should not be set
		So(supersedErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StateNew)
		So(app.SupersededTime, ShouldBeNil)
	})

	Convey("Given an approver set in the 'new' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		supersedErr := app.Decline(dbCache)
		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after Supersed
		So(prepareErr2, ShouldBeNil)

		// "Decline should work and the approval failed timestamp should be set
		So(supersedErr, ShouldBeNil)
		So(app2.RevisionState, ShouldEqual, StateApprovalFailed)
		So(app2.ApprovalFailedTime.UnixNano(), ShouldBeGreaterThan, 0)
		So(*app2.ApprovalFailedTime, ShouldHappenOnOrBefore, TimeNow())
	})
}

func TestApproverSetRevisionGetActions(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Set Revision in the 'pendingapproval' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		Convey("Preparing the approver set revision should not return an error", func() {
			So(prepareErr, ShouldBeNil)
		})

		actionMap := app.GetActions(true)
		Convey("GetActions where self is true should return a map of actions", func() {
			So(len(actionMap), ShouldEqual, 4)
		})
		actionMap2 := app.GetActions(false)
		Convey("GetActions where self is false should return a map of actions", func() {
			So(len(actionMap2), ShouldEqual, 4)
		})
	})
}

func TestApproverSetREvisionPromote(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Set Revision in the 'pendingapproval' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(app.CRID.Int64)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)
		changeRequest.State = StateApproved
		err = dbCache.Save(&changeRequest)
		So(err, ShouldBeNil)

		promoteErr := app.Promote(dbCache)

		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after Promoted
		So(prepareErr2, ShouldBeNil)

		// Promote should not return an error and the approver set revision should have the correct values set
		So(promoteErr, ShouldBeNil)
		So(app2.RevisionState, ShouldEqual, app2.DesiredState)
		So(app2.PromotedTime.UnixNano(), ShouldBeGreaterThan, 0)
		So(*app2.PromotedTime, ShouldHappenOnOrBefore, TimeNow())
	})

	Convey("Given an Approver Set Revision in the 'pendingapproval' state with a CR that has not been approved", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error
		So(prepareErr, ShouldBeNil)

		promoteErr := app.Promote(dbCache)

		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after Promoted
		So(prepareErr2, ShouldBeNil)

		// Promote should return an error and the approver set revision should not change
		So(promoteErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StatePendingApproval)
		So(app2.PromotedTime, ShouldBeNil)
	})

	Convey("Given an Approver Set Revision in the 'pendingapproval' state with no valid CR", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - before edit
		So(prepareErr, ShouldBeNil)

		app.CRID.Valid = false
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)

		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - before Promoted
		So(prepareErr2, ShouldBeNil)

		promoteErr := app2.Promote(dbCache)

		app3 := ApproverSetRevision{}
		err = app3.SetID(2)
		So(err, ShouldBeNil)
		prepareErr3 := app3.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after Promoted
		So(prepareErr3, ShouldBeNil)

		// Promote should return an error and the approver set revision should not change
		So(promoteErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StatePendingApproval)
		So(app2.PromotedTime, ShouldBeNil)
	})

	Convey("Given an Approver Set Revision in the 'cancelled' state", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		So(dberr, ShouldBeNil)

		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - before edit
		So(prepareErr, ShouldBeNil)

		app.RevisionState = StateCancelled
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)

		app2 := ApproverSetRevision{}
		err = app2.SetID(2)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - before Promoted
		So(prepareErr2, ShouldBeNil)

		promoteErr := app2.Promote(dbCache)

		app3 := ApproverSetRevision{}
		err = app3.SetID(2)
		So(err, ShouldBeNil)
		prepareErr3 := app3.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after Promoted
		So(prepareErr3, ShouldBeNil)

		// Promote should return an error and the approver set revision should not change
		So(promoteErr, ShouldNotBeNil)
		So(app2.RevisionState, ShouldEqual, StateCancelled)
		So(app2.PromotedTime, ShouldBeNil)
	})
}

// func TestApproverSetRevisionStartApprovalProcess(t *testing.T) {
//
// 	// Test the approver set's state when not a bootstrap and the desired
// 	// state is active (should be activependingapproval)
// 	Convey("Given the bootstrap database", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrap)
// 		Convey("No error should be thrown when getting the database", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		approversetr := ApproverSetRevision{}
// 		approversetr.SetID(1)
// 		approversetr.PrepareShallow(db)
// 		approversetr.DesiredState = StateActive
// 		db.Save(&approversetr)
//
// 		app := ApproverSetRevision{}
// 		app.SetID(2)
// 		prepareErr := app.Prepare(db)
// 		Convey("No error should be thrown when preparing the approver set revision", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser1Username)
//
// 		startAppErr := app.StartApprovalProcess(r, db)
// 		Convey("An error should be thrown when starting the approval process for the approver set revision", func() {
// 			So(startAppErr, ShouldBeNil)
// 		})
//
// 		approverSet2 := ApproverSet{}
// 		approverSet2.SetID(app.ApproverSetID)
// 		approverSet2.Prepare(db)
// 		Convey("The parent approver set's state should be set to \"active\"", func() {
// 			So(approverSet2.State, ShouldEqual, StateActivePendingApproval)
// 		})
// 	})
//
// 	// Test the approver set's state when not a bootstrap and the desired
// 	// state is inactive (should be inactivependingapproval)
// 	Convey("Given the bootstrap database", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrap)
// 		Convey("No error should be thrown when getting the database", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		approverset := ApproverSet{}
// 		approverset.SetID(1)
// 		approverset.PrepareShallow(db)
// 		approverset.State = StateInactive
// 		db.Save(&approverset)
//
// 		approversetr := ApproverSetRevision{}
// 		approversetr.SetID(1)
// 		approversetr.PrepareShallow(db)
// 		approversetr.DesiredState = StateInactive
// 		db.Save(&approversetr)
//
// 		app := ApproverSetRevision{}
// 		app.SetID(2)
// 		prepareErr := app.Prepare(db)
// 		Convey("No error should be thrown when preparing the approver set revision", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser1Username)
//
// 		startAppErr := app.StartApprovalProcess(r, db)
// 		Convey("An error should be thrown when starting the approval process for the approver set revision", func() {
// 			So(startAppErr, ShouldBeNil)
// 		})
//
// 		approverSet2 := ApproverSet{}
// 		approverSet2.SetID(app.ApproverSetID)
// 		approverSet2.Prepare(db)
// 		Convey("The parent approver set's state should be set to \"inactive\"", func() {
// 			So(approverSet2.State, ShouldEqual, StateInactivePendingApproval)
// 		})
// 	})
//
// 	// Test the approver set's state when a new revision has been
// 	// submitted for approval and no current revision exists
// 	// (should be pendingnew)
// 	Convey("Given the bootstrap database", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrap)
// 		Convey("No error should be thrown when getting the database", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		approverset := ApproverSet{}
// 		approverset.SetID(1)
// 		approverset.PrepareShallow(db)
// 		approverset.State = StateNew
// 		approverset.CurrentRevisionID.Valid = false
// 		db.Save(&approverset)
//
// 		app := ApproverSetRevision{}
// 		app.SetID(2)
// 		prepareErr := app.Prepare(db)
// 		Convey("No error should be thrown when preparing the approver set revision", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser1Username)
//
// 		startAppErr := app.StartApprovalProcess(r, db)
// 		Convey("An error should be thrown when starting the approval process for the approver set revision", func() {
// 			So(startAppErr, ShouldBeNil)
// 		})
//
// 		approverSet2 := ApproverSet{}
// 		approverSet2.SetID(app.ApproverSetID)
// 		approverSet2.Prepare(db)
// 		Convey("The parent approver set's state should be set to \"pendingnew\"", func() {
// 			So(approverSet2.State, ShouldEqual, StatePendingNew)
// 		})
// 	})
//
// 	// Test to make sure than an error is thrown when the pendingrevision
// 	// id is the same as the current revision id
// 	Convey("Given the bootstrap database", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrap)
// 		Convey("No error should be thrown when getting the database", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		approverset := ApproverSet{}
// 		approverset.SetID(1)
// 		approverset.PrepareShallow(db)
// 		approverset.CurrentRevisionID.Int64 = approverset.PendingRevision.ID
// 		db.Save(&approverset)
//
// 		app := ApproverSetRevision{}
// 		app.SetID(2)
// 		prepareErr := app.Prepare(db)
// 		Convey("No error should be thrown when preparing the approver set revision", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser1Username)
//
// 		startAppErr := app.StartApprovalProcess(r, db)
// 		Convey("An error should be thrown when starting the approval process for the approver set revision", func() {
// 			So(startAppErr, ShouldNotBeNil)
// 		})
//
// 		approverSet2 := ApproverSet{}
// 		approverSet2.SetID(app.ApproverSetID)
// 		approverSet2.Prepare(db)
// 		Convey("The parent approver set's state should be set to \"active\"", func() {
// 			So(approverSet2.State, ShouldEqual, StateActive)
// 		})
// 	})
//
// 	Convey("Given the bootstrap database", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrap)
// 		Convey("No error should be thrown when getting the database", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		app := ApproverSetRevision{}
// 		app.SetID(1)
// 		prepareErr1 := app.Prepare(db)
// 		Convey("No error should be thrown when preparing the approver revision - Current Revision", func() {
// 			So(prepareErr1, ShouldBeNil)
// 		})
// 		UpdateApproverSets(&app, db, "RequiredApproverSets", []ApproverSet{})
//
// 		app = ApproverSetRevision{}
// 		app.SetID(2)
// 		prepareErr2 := app.Prepare(db)
// 		Convey("No error should be thrown when preparing the approver revision - Pending Revision", func() {
// 			So(prepareErr2, ShouldBeNil)
// 		})
//
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser1Username)
//
// 		startAppErr := app.StartApprovalProcess(r, db)
// 		Convey("No error should be thrown when starting the approval process for the approver revision", func() {
// 			So(startAppErr, ShouldBeNil)
// 		})
//
// 		approver := ApproverSet{}
// 		approver.SetID(app.ApproverSetID)
// 		approver.Prepare(db)
// 		Convey("The parent approver's state should be set to \"pendingbootstrap\"", func() {
// 			So(approver.State, ShouldEqual, StatePendingBootstrap)
// 		})
// 	})
//
// 	Convey("Given the bootstrap database", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrap)
// 		Convey("No error should be thrown when getting the database", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		app := ApproverSetRevision{}
// 		app.SetID(1)
// 		prepareErr1 := app.Prepare(db)
// 		Convey("No error should be thrown when preparing the approver revision - Current Revision", func() {
// 			So(prepareErr1, ShouldBeNil)
// 		})
// 		app.RevisionState = StateActive
// 		db.Save(&app)
//
// 		app = ApproverSetRevision{}
// 		app.SetID(2)
// 		prepareErr2 := app.Prepare(db)
// 		Convey("No error should be thrown when preparing the approver revision - Pending Revision", func() {
// 			So(prepareErr2, ShouldBeNil)
// 		})
//
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser1Username)
//
// 		startAppErr := app.StartApprovalProcess(r, db)
// 		Convey("No error should be thrown when starting the approval process for the approver revision", func() {
// 			So(startAppErr, ShouldBeNil)
// 		})
//
// 		approverset := ApproverSet{}
// 		approverset.SetID(app.ApproverSetID)
// 		approverset.Prepare(db)
// 		Convey("The parent approver's state should be set to \"pendingbootstrap\"", func() {
// 			So(approverset.State, ShouldEqual, StatePendingBootstrap)
// 		})
// 	})
// }

func TestApproverSetRevisionParseFromForm(t *testing.T) {
	t.Parallel()
	Convey("Given a HTTP request without a valid username set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Form = make(url.Values)
		app := ApproverSetRevision{}
		perr := app.ParseFromForm(r, dbCache)
		// ParseFromForm should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but no revision_approver_id set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		app := ApproverSetRevision{}
		perr := app.ParseFromForm(request, dbCache)
		// ParseFromForm should return an error", func() {
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but an invalid approver_id", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_set_id", "1")
		request.Form.Add("approver_id", bogusState)
		app := ApproverSetRevision{}
		perr := app.ParseFromForm(request, dbCache)
		// ParseFromForm should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but an invalid required approver set id", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_set_id", "1")
		request.Form.Add("approver_id", "1")
		request.Form.Add("approver_set_required_id", bogusState)
		app := ApproverSetRevision{}
		perr := app.ParseFromForm(request, dbCache)
		// ParseFromForm should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but an invalid informed approver set id", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_set_id", "1")
		request.Form.Add("approver_id", "1")
		request.Form.Add("approver_set_required_id", "1")
		request.Form.Add("approver_set_informed_id", bogusState)
		app := ApproverSetRevision{}
		perr := app.ParseFromForm(request, dbCache)
		// ParseFromForm should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but an invalid informed approver set id", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_set_id", "1")
		request.Form.Add("approver_id", "1")
		request.Form.Add("approver_set_required_id", "1")
		request.Form.Add("approver_set_informed_id", "1")
		request.Form.Add("revision_desiredstate", StateInactive)
		app := ApproverSetRevision{}
		perr := app.ParseFromForm(request, dbCache)
		err := dbCache.Save(&app)
		So(err, ShouldBeNil)

		app2 := ApproverSetRevision{}
		err = app2.SetID(app.ID)
		So(err, ShouldBeNil)
		prepareErr2 := app2.Prepare(dbCache)
		// No error should be thrown when preparing the approver revision - Current Revision
		So(prepareErr2, ShouldBeNil)

		// ParseFromForm should return an error
		So(perr, ShouldBeNil)
		So(app2.ApproverSetID, ShouldEqual, 1)
		So(len(app2.Approvers), ShouldEqual, 1)
		So(app2.Approvers[0].ID, ShouldEqual, 1)
		So(app2.RevisionState, ShouldEqual, StateNew)
		So(app2.DesiredState, ShouldEqual, StateInactive)
		So(len(app2.RequiredApproverSets), ShouldEqual, 1)
		So(app2.RequiredApproverSets[0].ID, ShouldEqual, 1)
		So(len(app2.InformedApproverSets), ShouldEqual, 1)
		So(app2.InformedApproverSets[0].ID, ShouldEqual, 1)
	})
}

func TestApproverSetRevisionParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	Convey("Given a HTTP request without a valid ID set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Form = make(url.Values)
		app := ApproverSetRevision{}
		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		// ParseFromFormUpdate should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request without a valid ID set valid approver set revision with no username", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Form = make(url.Values)
		app := ApproverSetRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// No error should be thrown when preparing the approver revision
		So(prepareErr, ShouldBeNil)

		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		// ParseFromFormUpdate should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but no other fields set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		app := ApproverSetRevision{}
		err := app.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// No error should be thrown when preparing the approver revision
		So(prepareErr, ShouldBeNil)

		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		// ParseFromFormUpdate should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but an invalid approver id", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_set_id", "1")
		request.Form.Add("approver_id", "a")
		request.Form.Add("approver_set_required_id", "1")
		request.Form.Add("approver_set_informed_id", "1")
		request.Form.Add("revision_desiredstate", StateInactive)
		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// No error should be thrown when preparing the approver revision
		So(prepareErr, ShouldBeNil)

		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())

		// ParseFromForm should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but an invalid required approver set id", t, func() {
		ddbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_set_id", "1")
		request.Form.Add("approver_set_required_id", "a")
		request.Form.Add("revision_desiredstate", StateInactive)
		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(ddbCache)
		// No error should be thrown when preparing the approver revision
		So(prepareErr, ShouldBeNil)

		perr := app.ParseFromFormUpdate(request, ddbCache, mustGetTestConf())

		// ParseFromForm should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set but an invalid informed approver set id", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_set_id", "1")
		request.Form.Add("approver_set_informed_id", "a")
		request.Form.Add("revision_desiredstate", StateInactive)
		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// No error should be thrown when preparing the approver revision
		So(prepareErr, ShouldBeNil)

		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())

		// ParseFromForm should return an error
		So(perr, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username set and all fields populated correctly", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		request.Form = make(url.Values)
		request.Form.Add("revision_approver_set_id", "1")
		request.Form.Add("approver_id", "1")
		request.Form.Add("approver_set_required_id", "1")
		request.Form.Add("approver_set_informed_id", "1")
		request.Form.Add("revision_desiredstate", StateInactive)
		app := ApproverSetRevision{}
		err := app.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := app.Prepare(dbCache)
		// No error should be thrown when preparing the approver revision
		So(prepareErr, ShouldBeNil)

		perr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		outputTyped := app

		// ParseFromForm should not return an error and have the correct values set
		So(perr, ShouldBeNil)
		So(outputTyped.ApproverSetID, ShouldEqual, 1)
		So(len(app.Approvers), ShouldEqual, 1)
		So(app.Approvers[0].ID, ShouldEqual, 1)
		So(outputTyped.RevisionState, ShouldEqual, StateNew)
		So(outputTyped.DesiredState, ShouldEqual, StateInactive)
		So(len(app.RequiredApproverSets), ShouldEqual, 1)
		So(app.RequiredApproverSets[0].ID, ShouldEqual, 1)
		So(len(app.InformedApproverSets), ShouldEqual, 1)
		So(app.InformedApproverSets[0].ID, ShouldEqual, 1)
	})
}

func TestApproverSetRevisionIsCanclled(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision in the 'new' state", t, func() {
		app := ApproverSetRevision{}
		app.RevisionState = StateNew
		/// IsCancelled should return false
		So(app.IsCancelled(), ShouldBeFalse)
	})

	Convey("Given an approver set revision in the 'cancelled' state", t, func() {
		app := ApproverSetRevision{}
		app.RevisionState = StateCancelled
		// IsCancelled should return true
		So(app.IsCancelled(), ShouldBeTrue)
	})
}

func TestApproverSetRevisionPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverSetRevisionPage", t, func() {
		approverSetRevisionPage := ApproverSetRevisionPage{}
		// Calling SetCSRFToken should not panic
		So(func() { approverSetRevisionPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)

		testTokenString := testingCSRFToken
		approverSetRevisionPage.CSRFToken = testTokenString
		// Calling GetCSRFToken should return the testing token string
		So(approverSetRevisionPage.GetCSRFToken(), ShouldEqual, testTokenString)
	})
}

func TestApproverSetRevisionsPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverSetRevisionsPage", t, func() {
		approverSetRevisionPage := ApproverSetRevisionsPage{}
		// Calling SetCSRFToken should not panic
		So(func() { approverSetRevisionPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)

		testTokenString := testingCSRFToken
		approverSetRevisionPage.CSRFToken = testTokenString
		// Calling GetCSRFToken should return the testing token string
		So(approverSetRevisionPage.GetCSRFToken(), ShouldEqual, testTokenString)
	})
}
