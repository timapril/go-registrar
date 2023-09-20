package lib

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestChangeRequestPageSplitJSON(t *testing.T) {
	t.Parallel()
	Convey("Given a Change Request Page", t, func() {
		crp := ChangeRequestPage{}
		sliptstring := crp.SplitJSON("testing1\ntesting2\ntesting3")
		// SplitJSON of a string with three new lines should be an array of length 3
		So(len(sliptstring), ShouldEqual, 3)
	})
}

func TestChangeRequestGetExportVersion(t *testing.T) {
	t.Parallel()
	Convey("Given a Change Request", t, func() {
		changeRequest := ChangeRequest{}
		exportVersion := changeRequest.GetExportVersion()
		// GetExportVersion should return an object of the type NotExportableObject with the type set to 'changerequest'
		So(exportVersion, ShouldHaveSameTypeAs, ChangeRequestExport{})
	})
}

func TestChangeRequestParseFromForm(t *testing.T) {
	t.Parallel()
	Convey("Given a Change Request", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Form = make(url.Values)

		changeRequest := ChangeRequest{}
		err := changeRequest.ParseFromForm(r, dbCache)
		// ParseFromForm should return an error
		So(err, ShouldNotBeNil)
	})
}

func TestChangeRequestParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	Convey("Given a Change Request", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Form = make(url.Values)
		changeRequest := ChangeRequest{}
		err := changeRequest.ParseFromFormUpdate(r, dbCache, mustGetTestConf())
		Convey("ParseFromFormUpdate should return an empty object and an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
}

func TestChangeRequestTakeAction(t *testing.T) {
	t.Parallel()
	Convey("Given a Change Request", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		changeRequest := ChangeRequest{}
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w := httptest.NewRecorder()

		// Calling TakeAction should not panic
		So(func() { changeRequest.TakeAction(w, r, dbCache, "", false, RemoteUserAuthType, conf) }, ShouldNotPanic)
	})
}

// func TestChangeRequestPostUpdate(t *testing.T) {
// 	Convey("Given a Change Request", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateEmpty)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		cr := ChangeRequest{}
// 		Convey("Calling PostUpdate should not panic", func() {
// 			So(func() { cr.PostUpdate(db) }, ShouldNotPanic)
// 		})
// 	})
// }

func TestChangeRequestSetID(t *testing.T) {
	t.Parallel()
	Convey("Given an empty change request", t, func() {
		changeRequest := ChangeRequest{}
		err := changeRequest.SetID(1)
		// SetID with an ID of 1 should not return an error
		So(err, ShouldBeNil)
		So(changeRequest.ID, ShouldEqual, 1)

		err2 := changeRequest.SetID(2)
		// SetID with an ID of 2 on a CR that has had its ID set should return an error
		So(err2, ShouldNotBeNil)
		So(changeRequest.ID, ShouldEqual, 1)
	})

	Convey("Given an empty change request", t, func() {
		changeRequest := ChangeRequest{}
		err := changeRequest.SetID(-1)
		// SetID with an ID of -1 should not return an error
		So(err, ShouldNotBeNil)
	})
}

func TestChangeRequestIsEditable(t *testing.T) {
	t.Parallel()
	Convey("Given a change request", t, func() {
		changeRequest := ChangeRequest{}
		editable := changeRequest.IsEditable()
		// CRs should not be editable
		So(editable, ShouldBeFalse)
	})
}

func TestChangeRequestGetPage(t *testing.T) {
	t.Parallel()
	Convey("Given a valid change request", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		So(dberr, ShouldBeNil)

		changeRequest := ChangeRequest{}
		err := changeRequest.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := changeRequest.Prepare(dbCache)
		// Preparing the change request should not return an error
		So(prepareErr, ShouldBeNil)

		rawpage, err := changeRequest.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedPage := rawpage.(*ChangeRequestPage)
		// GetPage should return a valid ChangeRequestPage with the CR as a member
		So(typedPage.CR.ID, ShouldEqual, 1)
		So(typedPage.CR.prepared, ShouldBeTrue)
	})
}

func TestChangeRequestGetAllPage(t *testing.T) {
	t.Parallel()
	Convey("Given a valid change request", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		So(dberr, ShouldBeNil)

		changeRequest := ChangeRequest{}
		rawpage, err := changeRequest.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedPage := rawpage.(*ChangeRequestsPage)
		// GetAllPage should return a valid ChangeRequestsPage with a list of CRs with a length of 1
		So(len(typedPage.CRs), ShouldEqual, 1)
	})
}

func TestChangeRequestPrepare(t *testing.T) {
	t.Parallel()
	Convey("Given a change request with an ID of 0", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		changeRequest := ChangeRequest{}
		changeRequest.ID = 0
		err := changeRequest.Prepare(dbCache)
		/// Prepare called on a Change Request with an ID of 0 should return an error
		So(err, ShouldNotBeNil)
	})
}

func TestChangeRequestUpdateState(t *testing.T) {
	t.Parallel()
	// Convey("Given a change request who's object does not have any current revision", t, func() {
	// 	db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
	// 	Convey("Getting the database should not throw an error", func() {
	// 		So(dberr, ShouldBeNil)
	// 	})
	// 	cr := ChangeRequest{}
	// 	cr.SetID(1)
	// 	prepareErr := cr.Prepare(db)
	// 	Convey("Preparing the approver set revision should not return an error - before update", func() {
	// 		So(prepareErr, ShouldBeNil)
	// 	})
	// 	cr.State = StateNew
	// 	db.Save(&cr)
	//
	// 	app := (cr.Object).(*Approver)
	// 	app.CurrentRevisionID.Valid = false
	// 	db.Save(app)
	//
	// 	cr2 := ChangeRequest{}
	// 	cr2.SetID(1)
	// 	prepareErr2 := cr2.Prepare(db)
	// 	Convey("Preparing the approver set revision should not return an error - after update", func() {
	// 		So(prepareErr2, ShouldBeNil)
	// 	})
	// 	errs := cr2.UpdateState(db,conf)
	// 	Convey("UpdateState with an object that cant get the ApproverSet should return an error", func() {
	// 		So(len(errs), ShouldEqual, 1)
	// 	})
	// })

	Convey("Given a change request who's object that has a cancelled pending revision", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		So(dberr, ShouldBeNil)

		changeRequest := ChangeRequest{}
		err := changeRequest.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := changeRequest.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - before update
		So(prepareErr, ShouldBeNil)

		changeRequest.State = StateNew
		err = dbCache.Save(&changeRequest)
		So(err, ShouldBeNil)

		app := (changeRequest.Object).(*Approver)
		appr := ApproverRevision{}
		err = appr.SetID(app.PendingRevision.ID)
		So(err, ShouldBeNil)
		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)
		appr.RevisionState = StateCancelled
		err = dbCache.Save(&appr)
		So(err, ShouldBeNil)

		approval := changeRequest.Approvals[0]
		approval.State = StateNew
		err = dbCache.Save(&approval)
		So(err, ShouldBeNil)

		cr2 := ChangeRequest{}
		err = cr2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := cr2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after update
		So(prepareErr2, ShouldBeNil)

		changeMade, errs := cr2.UpdateState(dbCache, conf)
		// UpdateState with an object that can't get the ApproverSet should not return an error
		So(errs, ShouldBeEmpty)
		So(changeMade, ShouldBeTrue)
	})

	Convey("Given a change request who's object does not have any current revision", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		// Getting the database should not throw an error
		So(dberr, ShouldBeNil)

		changeRequest := ChangeRequest{}
		err := changeRequest.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := changeRequest.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - before update
		So(prepareErr, ShouldBeNil)

		changeRequest.State = StateNew
		err = dbCache.Save(&changeRequest)
		So(err, ShouldBeNil)

		app := (changeRequest.Object).(*Approver)

		appsets, _ := app.GetRequiredApproverSets(dbCache)
		NewAppSet := ApproverSet{}
		err = dbCache.Save(&NewAppSet)
		So(err, ShouldBeNil)
		appsets = append(appsets, NewAppSet)

		err = UpdateApproverSets(&app.CurrentRevision, dbCache, "RequiredApproverSets", appsets)
		So(err, ShouldBeNil)

		cr2 := ChangeRequest{}
		err = cr2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := cr2.Prepare(dbCache)
		// Preparing the approver set revision should not return an error - after update
		So(prepareErr2, ShouldBeNil)

		changesMade, errs := cr2.UpdateState(dbCache, conf)
		// UpdateState with an object that cant get the ApproverSet should return an error
		So(len(errs), ShouldEqual, 1)
		So(changesMade, ShouldBeFalse)
	})

	Convey("Given a change request with an approvals in the No Valid Approvers State and StateInactiveApproverSet state", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		// Getting the database should not throw an error
		So(dberr, ShouldBeNil)

		nvaApproval := Approval{}
		nvaApproval.ChangeRequestID = 1
		nvaApproval.ApproverSetID = 1
		nvaApproval.State = StateNoValidApprovers
		err := dbCache.Save(&nvaApproval)
		So(err, ShouldBeNil)

		iasApproval := Approval{}
		iasApproval.ChangeRequestID = 1
		iasApproval.ApproverSetID = 1
		iasApproval.State = StateInactiveApproverSet
		err = dbCache.Save(&iasApproval)
		So(err, ShouldBeNil)

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := changeRequest.Prepare(dbCache)
		/// Preparing the approver set revision should not return an error - before update
		So(prepareErr, ShouldBeNil)

		changesMade, errs := changeRequest.UpdateState(dbCache, conf)
		// UpdateState with an object that cant get the ApproverSet should return an error
		So(len(errs), ShouldEqual, 0)
		So(changesMade, ShouldBeFalse)
	})
}

func TestChangeRequestPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ChangeRequestPage", t, func() {
		changeRequestPage := ChangeRequestPage{}
		// Calling SetCSRFToken should not panic
		So(func() { changeRequestPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)

		testTokenString := testingCSRFToken
		changeRequestPage.CSRFToken = testTokenString
		// Calling GetCSRFToken should return the testing token string
		So(changeRequestPage.GetCSRFToken(), ShouldEqual, testTokenString)
	})
}

func TestChangeRequestsPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ChangeRequestsPage", t, func() {
		changeRequestPage := ChangeRequestsPage{}
		// Calling SetCSRFToken should not panic
		So(func() { changeRequestPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)

		testTokenString := testingCSRFToken
		changeRequestPage.CSRFToken = testTokenString
		// Calling GetCSRFToken should return the testing token string
		So(changeRequestPage.GetCSRFToken(), ShouldEqual, testTokenString)
	})
}
