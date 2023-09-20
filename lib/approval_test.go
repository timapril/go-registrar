package lib

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_Approval_GetExportVersion(t *testing.T) {
	t.Parallel()

	app := Approval{}
	ret := app.GetExportVersion()

	if reflect.TypeOf(ret) != reflect.TypeOf(ApprovalExport{}) {
		t.Error("Expected a ApprovalExport Object to be returned")
	}
}

func Test_Approval_ParseFromForm(t *testing.T) {
	t.Parallel()

	dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
	if err != nil {
		return
	}

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	request.Header.Add("REMOTE_USER", TestUser1Username)

	app := Approval{}
	parseError := app.ParseFromForm(request, dbCache)

	if parseError == nil {
		t.Error("Expected an error when calling ParseFromForm on an Approval")
	}
}

func Test_Approval_SetID_DoubleSet(t *testing.T) {
	t.Parallel()

	app := Approval{}

	err := app.SetID(1)
	if err != nil {
		t.Errorf("Unexpected error when calling SetID on an approval %s", err.Error())
	}

	if app.GetID() != 1 {
		t.Errorf("Expected ID to be 1, Got: %d", app.GetID())
	}

	err2 := app.SetID(2)
	if err2 == nil {
		t.Errorf("Expected an error when calling SetID on an approval that had its ID set")
	}

	if app.GetID() != 1 {
		t.Errorf("Expected ID to be 1, Got: %d", app.GetID())
	}
}

func Test_Approval_SetID_NonPositive(t *testing.T) {
	t.Parallel()

	app := Approval{}

	err := app.SetID(-1)

	if err == nil {
		t.Errorf("Expected an error when calling SetID(-1) on an approval ")
	}

	if app.GetID() != 0 {
		t.Errorf("Expected ID to be 0, Got: %d", app.GetID())
	}
}

func Test_Approval_ParseFromFormUpdate_NoRemoteUser(t *testing.T) {
	t.Parallel()

	dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
	if err != nil {
		return
	}

	logger.Warningf("db %+V", dbCache)

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)

	appr := Approval{}

	err = appr.SetID(1)
	if err != nil {
		t.Error(err)
	}

	err = appr.Prepare(dbCache)
	if err != nil {
		t.Error(err)
	}

	message := appr.GetDownload(dbCache, "test", ActionApproved)

	signed, err := ClearsignMessage(message, TestUser1Username)
	if err != nil {
		t.Errorf("Unable to sign message %s", err.Error())
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("sig", "./sig")
	if err != nil {
		t.Errorf("unable to create multipart for upload: %s", err.Error())
	}

	if _, partWriteErr := part.Write([]byte(signed)); err != nil {
		t.Errorf("Error writing sig into form: %s", partWriteErr.Error())
	}

	if writerCloseErr := writer.Close(); err != nil {
		t.Errorf("Error closing writer: %s", writerCloseErr.Error())
	}

	ctx := context.Background()
	request, _ = http.NewRequestWithContext(ctx, http.MethodPost, "/", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	err = appr.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
	if err == nil {
		t.Errorf("Expected an error when not providing a remote user")
	}
}

func Test_Approval_ParseFromFormUpdate_IncorrectSigner(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		appr := Approval{}
		err = appr.SetID(1)

		So(err, ShouldBeNil)

		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)

		message := appr.GetDownload(dbCache, "test", ActionApproved)

		path, err := getOrGenerateTestingGPGKey(TestUser2Username)
		So(err, ShouldBeNil)
		t.Log(path)

		signed, err := ClearsignMessage(message, TestUser2Username)
		So(err, ShouldBeNil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("sig", "./sig")
		So(err, ShouldBeNil)

		_, partWriteErr := part.Write([]byte(signed))
		So(partWriteErr, ShouldBeNil)

		writerCloseErr := writer.Close()
		So(writerCloseErr, ShouldBeNil)

		request, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)

		err = appr.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		So(err, ShouldNotBeNil)
	})
}

func Test_Approval_ParseFromFormUpdate_CorrectSigner(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started attempt to sign with a correct signer", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		So(err, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		message := app.GetDownload(dbCache, "test", ActionApproved)

		signed, err := ClearsignMessage(message, TestUser1Username)
		if err != nil {
			t.Errorf("Unable to sign message %s", err.Error())
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("sig", "./sig")
		if err != nil {
			t.Errorf("unable to create multipart for upload: %s", err.Error())
		}

		if _, err := part.Write([]byte(signed)); err != nil {
			t.Errorf("Error writing sig into form: %s", err.Error())
		}

		if err := writer.Close(); err != nil {
			t.Errorf("Error closing writer: %s", err.Error())
		}

		request, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)
		parseErr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())

		if parseErr != nil {
			t.Errorf("No error was expected when uploading a valid signature, got %s", parseErr.Error())
		}

		dbCache.DB.Model(app).UpdateColumns(app)

		dbCache.WipeCache()

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		err = app.PostUpdate(dbCache, mustGetTestConf())
		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StateApproved {
			t.Errorf("Expected Approval %d to be in %s rather than %s", app.ID, StateApproved, app.State)
		}

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(1)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)

		if changeRequest.State != StateApproved {
			t.Errorf("Expected Change Request %d to be in %s rather than %s", changeRequest.ID, StateApproved, changeRequest.State)
		}

		appr := ApproverRevision{}
		err = appr.SetID(2)
		So(err, ShouldBeNil)
		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)

		if appr.RevisionState != StateActive {
			t.Errorf("Expected Approver Revision %d to be in %s rather than %s", appr.ID, StateActive, appr.RevisionState)
		}
	})
}

// func Test_Approval_ParseFromFormUpdate_CorrectSigner_MissingCR(t *testing.T) {
// 	db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 	if err != nil {
// 		return
// 	}
// 	r, _ := http.NewRequestWithContext(context.Background(),http.MethodGet, "/", nil)
// 	r.Header.Add("REMOTE_USER", TestUser1Username)
//
// 	app := Approval{}
// 	app.SetID(1)
// 	app.Prepare(db)
//
// 	if app.State != StatePendingApproval {
// 		t.Errorf("Expected approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State))
// 	}
//
// 	message := app.GetDownload(db, "test", ActionApproved)
//
// 	signed, err := ClearsignMessage(message, TestUser1Username)
// 	if err != nil {
// 		t.Errorf("Unable to sign message %s", err.Error())
// 	}
//
// 	tempCR := ChangeRequest{}
// 	tempCR.SetID(1)
// 	tempCR.Prepare(db)
// 	db.Delete(&tempCR)
//
// 	update := Approval{}
// 	update.Signature = []byte(signed)
// 	update.IsSigned = true
// 	db.Model(app).UpdateColumns(update)
//
// 	app = Approval{}
// 	app.SetID(1)
// 	app.Prepare(db)
//
// 	app.PostUpdate(db)
//
// 	app = Approval{}
// 	app.SetID(1)
// 	app.Prepare(db)
//
// 	t.Log(app.ChangeRequestID)
// 	tempCR2 := ChangeRequest{}
// 	tempCR2.SetID(1)
// 	tempCR2.Prepare(db)
// 	fmt.Println(tempCR2)
//
// 	if app.State != StateApproved {
// 		t.Errorf("Expected Approval %d to be in %s rather than %s", app.ID, StateApproved, app.State))
// 	}
//
// 	appr := ApproverRevision{}
// 	appr.SetID(2)
// 	appr.Prepare(db)
//
// 	if appr.RevisionState != StatePendingApproval {
// 		t.Errorf("Expected Approver Revision %d to be in %s rather than %s", appr.ID, StatePendingApproval, appr.RevisionState))
// 	}
// }

func Test_Approval_ParseFromFormUpdate_CorrectSigner_Declined(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started attempt to sign with a correct signer declining the request", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		message := app.GetDownload(dbCache, "test", ActionDeclined)

		signed, err := ClearsignMessage(message, TestUser1Username)
		if err != nil {
			t.Errorf("Unable to sign message %s", err.Error())
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("sig", "./sig")
		if err != nil {
			t.Errorf("unable to create multipart for upload: %s", err.Error())
		}

		if _, err := part.Write([]byte(signed)); err != nil {
			t.Errorf("Error writing sig into form: %s", err.Error())
		}

		if err := writer.Close(); err != nil {
			t.Errorf("Error closing writer: %s", err.Error())
		}

		request, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)

		parseErr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		if parseErr != nil {
			t.Errorf("No error was expected when uploading a valid signature, got %s", parseErr.Error())
		}

		dbCache.DB.Model(app).UpdateColumns(app)

		dbCache.WipeCache()

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		err = app.PostUpdate(dbCache, mustGetTestConf())
		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StateDeclined {
			t.Errorf("Expected Approval %d to be in %s rather than %s", app.ID, StateDeclined, app.State)
		}

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(1)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)

		if changeRequest.State != StateDeclined {
			t.Errorf("Expected Change Request %d to be in %s rather than %s", changeRequest.ID, StateDeclined, changeRequest.State)
		}

		appr := ApproverRevision{}
		err = appr.SetID(2)
		So(err, ShouldBeNil)
		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)

		if appr.RevisionState != StateApprovalFailed {
			t.Errorf("Expected Approver Revision %d to be in %s rather than %s", appr.ID, StateApprovalFailed, appr.RevisionState)
		}

		approver := Approver{}
		err = approver.SetID(1)
		So(err, ShouldBeNil)
		err = approver.Prepare(dbCache)
		So(err, ShouldBeNil)

		if approver.State != StateActive {
			t.Errorf("Expected Approver %d to be in state %s rather than state %s", approver.ID, StateActive, approver.State)
		}
	})
}

func Test_Approval_ParseFromFormUpdate_CorrectSigner_CorruptedSigAfterUpload(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started attempt to sign with a correct signer with a corrupted upload after the fact", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		message := app.GetDownload(dbCache, "test", ActionDeclined)

		signed, err := ClearsignMessage(message, TestUser1Username)
		if err != nil {
			t.Errorf("Unable to sign message %s", err.Error())
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("sig", "./sig")
		if err != nil {
			t.Errorf("unable to create multipart for upload: %s", err.Error())
		}

		if _, err := part.Write([]byte(signed)); err != nil {
			t.Errorf("Error writing sig into form: %s", err.Error())
		}

		if err := writer.Close(); err != nil {
			t.Errorf("Error closing writer: %s", err.Error())
		}

		request, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)

		parseErr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		if parseErr != nil {
			t.Errorf("No error was expected when uploading a valid signature, got %s", parseErr.Error())
		}

		dbCache.DB.Model(app).UpdateColumns(app)

		dbCache.WipeCache()

		app = Approval{}
		err = app.SetID(1)

		So(err, ShouldBeNil)

		err = app.Prepare(dbCache)

		So(err, ShouldBeNil)

		message = strings.Replace(message, "\"Action\": \"approve\"", "\"Action\": \"bogus\"", 1)
		app.Signature = []byte(message)
		err = dbCache.Save(&app)

		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)

		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		err = app.PostUpdate(dbCache, mustGetTestConf())
		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected Approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(1)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)

		if changeRequest.State != StatePendingApproval {
			t.Errorf("Expected Change Request %d to be in %s rather than %s", changeRequest.ID, StatePendingApproval, changeRequest.State)
		}

		approvalRevision := ApproverRevision{}
		err = approvalRevision.SetID(2)
		So(err, ShouldBeNil)
		err = approvalRevision.Prepare(dbCache)
		So(err, ShouldBeNil)

		if approvalRevision.RevisionState != StatePendingApproval {
			t.Errorf("Expected Approver Revision %d to be in %s rather than %s", approvalRevision.ID, StatePendingApproval, approvalRevision.RevisionState)
		}

		approver := Approver{}
		err = approver.SetID(1)
		So(err, ShouldBeNil)
		err = approver.Prepare(dbCache)
		So(err, ShouldBeNil)

		if approver.State != StatePendingBootstrap {
			t.Errorf("Expected Approver %d to be in state %s rather than state %s", approver.ID, StatePendingBootstrap, approver.State)
		}
	})
}

func Test_Approval_ParseFromFormUpdate_BogusAction(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started attempt to sign with a bogus action", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		message := app.GetDownload(dbCache, "test", bogusState)
		if len(message) != 0 {
			t.Error("Expected GetDownload to return an empty string with passed an unknown action")
		}

		// Now lets create an action that does not parse correctly
		message = app.GetDownload(dbCache, "test", ActionApproved)
		message = strings.Replace(message, "\"Action\": \"approve\"", "\"Action\": \"bogus\"", 1)

		signed, err := ClearsignMessage(message, TestUser1Username)
		if err != nil {
			t.Errorf("Unable to sign message %s", err.Error())
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("sig", "./sig")
		if err != nil {
			t.Errorf("unable to create multipart for upload: %s", err.Error())
		}

		if _, err := part.Write([]byte(signed)); err != nil {
			t.Errorf("Error writing sig into form: %s", err.Error())
		}

		if err := writer.Close(); err != nil {
			t.Errorf("Error closing writer: %s", err.Error())
		}

		request, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)

		parseErr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		if parseErr != nil {
			t.Errorf("No error was expected when uploading a valid signature, got %s", parseErr.Error())
		}

		dbCache.DB.Model(app).UpdateColumns(app)

		dbCache.WipeCache()

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		err = app.PostUpdate(dbCache, mustGetTestConf())
		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected Approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(1)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)

		if changeRequest.State != StatePendingApproval {
			t.Errorf("Expected Change Request %d to be in %s rather than %s", changeRequest.ID, StatePendingApproval, changeRequest.State)
		}

		appr := ApproverRevision{}
		err = appr.SetID(2)
		So(err, ShouldBeNil)
		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)

		if appr.RevisionState != StatePendingApproval {
			t.Errorf("Expected Approver Revision %d to be in %s rather than %s", appr.ID, StatePendingApproval, appr.RevisionState)
		}

		approver := Approver{}
		err = approver.SetID(1)
		So(err, ShouldBeNil)
		err = approver.Prepare(dbCache)
		So(err, ShouldBeNil)

		if approver.State != StatePendingBootstrap {
			t.Errorf("Expected Approver %d to be in state %s rather than state %s", approver.ID, StatePendingBootstrap, approver.State)
		}
	})
}

func Test_Approval_ParseFromFormUpdate_JSONError(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started attempt to sign with a JSON error", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		So(err, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		message := app.GetDownload(dbCache, "test", bogusState)
		if len(message) != 0 {
			t.Error("Expected GetDownload to return an empty string with passed an unknown action")
		}

		// Now lets create an action that does not parse correctly
		message = app.GetDownload(dbCache, "test", ActionApproved)
		message = strings.Replace(message, "\"Action\": \"approve\"", "\"Action\": \"bogus\"}", 1)

		signed, err := ClearsignMessage(message, TestUser1Username)
		So(err, ShouldBeNil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("sig", "./sig")
		So(err, ShouldBeNil)

		_, err = part.Write([]byte(signed))
		So(err, ShouldBeNil)

		err = writer.Close()
		So(err, ShouldBeNil)

		request, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)
		err = app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())

		// Expected an error when posting a badly formed JSON object
		So(err, ShouldNotBeNil)
		dbCache.DB.Model(app).UpdateColumns(app)

		dbCache.WipeCache()

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		err = app.PostUpdate(dbCache, mustGetTestConf())
		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)

		So(err, ShouldBeNil)

		err = app.Prepare(dbCache)

		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected Approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(1)
		So(err, ShouldBeNil)

		err = changeRequest.Prepare(dbCache)

		So(err, ShouldBeNil)

		if changeRequest.State != StatePendingApproval {
			t.Errorf("Expected Change Request %d to be in %s rather than %s", changeRequest.ID, StatePendingApproval, changeRequest.State)
		}

		appr := ApproverRevision{}
		err = appr.SetID(2)

		So(err, ShouldBeNil)

		err = appr.Prepare(dbCache)

		So(err, ShouldBeNil)

		if appr.RevisionState != StatePendingApproval {
			t.Errorf("Expected Approver Revision %d to be in %s rather than %s", appr.ID, StatePendingApproval, appr.RevisionState)
		}

		approver := Approver{}
		err = approver.SetID(1)

		So(err, ShouldBeNil)

		err = approver.Prepare(dbCache)

		So(err, ShouldBeNil)

		if approver.State != StatePendingBootstrap {
			t.Errorf("Expected Approver %d to be in state %s rather than state %s", approver.ID, StatePendingBootstrap, approver.State)
		}
	})
}

func Test_Approval_ParseFromFormUpdate_FormError(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started attempt to sign with a form error", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		err = app.SetID(1)

		So(err, ShouldBeNil)

		err = app.Prepare(dbCache)

		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		message := app.GetDownload(dbCache, "test", bogusState)
		if len(message) != 0 {
			t.Error("Expected GetDownload to return an empty string with passed an unknown action")
		}

		// Now lets create an action that does not parse correctly
		message = app.GetDownload(dbCache, "test", ActionApproved)
		message = strings.Replace(message, "\"Action\": \"approve\"", "\"Action\": \"bogus\"", 1)

		signed, err := ClearsignMessage(message, TestUser1Username)
		if err != nil {
			t.Errorf("Unable to sign message %s", err.Error())
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("", "./sig")
		if err != nil {
			t.Errorf("unable to create multipart for upload: %s", err.Error())
		}

		if _, err := part.Write([]byte(signed)); err != nil {
			t.Errorf("Error writing sig into form: %s", err.Error())
		}

		if err := writer.Close(); err != nil {
			t.Errorf("Error closing writer: %s", err.Error())
		}

		request, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)

		parseErr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		So(parseErr, ShouldNotBeNil)

		dbCache.DB.Model(app).UpdateColumns(app)

		dbCache.WipeCache()

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		err = app.PostUpdate(dbCache, mustGetTestConf())
		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		if app.State != StatePendingApproval {
			t.Errorf("Expected Approval %d to be in %s rather than %s", app.ID, StatePendingApproval, app.State)
		}

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(1)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)

		if changeRequest.State != StatePendingApproval {
			t.Errorf("Expected Change Request %d to be in %s rather than %s", changeRequest.ID, StatePendingApproval, changeRequest.State)
		}

		appr := ApproverRevision{}
		err = appr.SetID(2)
		So(err, ShouldBeNil)
		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)

		if appr.RevisionState != StatePendingApproval {
			t.Errorf("Expected Approver Revision %d to be in %s rather than %s", appr.ID, StatePendingApproval, appr.RevisionState)
		}

		approver := Approver{}
		err = approver.SetID(1)
		So(err, ShouldBeNil)
		err = approver.Prepare(dbCache)
		So(err, ShouldBeNil)

		if approver.State != StatePendingBootstrap {
			t.Errorf("Expected Approver %d to be in state %s rather than state %s", approver.ID, StatePendingBootstrap, approver.State)
		}
	})
}

func Test_Approval_Prepare(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started attempt to sign (prepare)", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}

		err = app.SetID(1)
		So(err, ShouldBeNil)

		if err := app.Prepare(dbCache); err != nil {
			t.Error("No Error Expected when calling prepare on an existing Approval")
		}

		app = Approval{}
		app.ID = -1

		if err := app.Prepare(dbCache); err == nil {
			t.Error("Error Expected when calling prepare on an Approval with an ID of -1")
		}

		app = Approval{}
		app.ID = 2

		if err := app.Prepare(dbCache); err == nil {
			t.Error("Error Expected when calling prepare on an Approval that does not exist")
		}
	})
}

func Test_Approval_IsEditable(t *testing.T) {
	t.Parallel()

	app := Approval{}
	app.State = StateApproved

	if app.IsEditable() {
		t.Errorf("Approvals should not be editable when in state %s", StateApproved)
	}

	app.State = StateDeclined

	if app.IsEditable() {
		t.Errorf("Approvals should not be editable when in state %s", StateDeclined)
	}

	app.State = StateDeclined

	if app.IsEditable() {
		t.Errorf("Approvals should not be editable when in state %s", StateDeclined)
	}

	app.State = StateCancelled

	if app.IsEditable() {
		t.Errorf("Approvals should not be editable when in state %s", StateCancelled)
	}

	app.State = StateSkippedNoValidApprovers

	if app.IsEditable() {
		t.Errorf("Approvals should not be editable when in state %s", StateSkippedNoValidApprovers)
	}

	app.State = StateSkippedInactiveApproverSet

	if app.IsEditable() {
		t.Errorf("Approvals should not be editable when in state %s", StateSkippedInactiveApproverSet)
	}

	app.State = StateNew

	if !app.IsEditable() {
		t.Errorf("Approvals should be editable when in state %s", StateNew)
	}

	app.State = StatePendingApproval

	if !app.IsEditable() {
		t.Errorf("Approvals should be editable when in state %s", StatePendingApproval)
	}

	app.State = StateNoValidApprovers

	if !app.IsEditable() {
		t.Errorf("Approvals should be editable when in state %s", StateNoValidApprovers)
	}

	app.State = StateInactiveApproverSet

	if !app.IsEditable() {
		t.Errorf("Approvals should be editable when in state %s", StateInactiveApproverSet)
	}
}

func Test_ApprovalAttestation_ToJSON(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval started attempt to sign with a correct signer, test ToJSON", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)

		if prepErr := app.Prepare(dbCache); prepErr != nil {
			t.Error("No Error Expected when calling prepare on an existing Approval")
		}

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(app.ChangeRequestID)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)

		approvalAttestation := ApprovalAttestation{
			ApprovalID: app.ID,
			ExportRev:  changeRequest.Object.GetExportVersion(),
			Username:   TestUser1Username,
			Action:     "approve",
		}

		_, err = approvalAttestation.ToJSON()
		if err != nil {
			t.Errorf("Did not expect an error when calling ToJSON on a valid ApprovalAttestation")
		}

		approvalAttestation.ApprovalID = -1

		_, err = approvalAttestation.ToJSON()
		if err == nil {
			t.Errorf("Expected an error when calling ToJSON on an ApprovalAttestation with an approval id of -1")
		}

		approvalAttestation.ApprovalID = 0

		_, err = approvalAttestation.ToJSON()
		if err == nil {
			t.Errorf("Expected an error when calling ToJSON on an ApprovalAttestation with an approval id of 0")
		}
	})
}

func Test_Approval_GetSigner(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval finished attempt to sign with a get the signer", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		if err != nil {
			return
		}

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		appSet2 := ApproverSet{}
		err = dbCache.Save(&appSet2)
		So(err, ShouldBeNil)

		app.ApproverSetID = appSet2.ID
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		_, err = app.GetSigner(dbCache)
		if err == nil {
			t.Error("Expected an error when calling GetSigner on a invalid approval")
		}
	})
}

func Test_Approval_GetPage(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval finished attempt to sign with a get page", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)

		if prepErr := app.Prepare(dbCache); prepErr != nil {
			t.Error("No Error Expected when calling prepare on an existing Approval")
		}

		obj, err := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedApprover := obj.(*ApprovalPage)
		if typedApprover.App.ID != app.ID {
			t.Error("Expected the Approver in the returned page to have the same ID as the approver passed")
		}

		if !typedApprover.CanApprove {
			t.Error("Expected CanApprove to be true for the approver name passed")
		}

		if !typedApprover.IsEditable {
			t.Error("Expected IsEditable to be set to true for tan approver in this state")
		}

		if typedApprover.IsSigned {
			t.Error("Expected IsSigned to be false since it is not sigend")
		}

		if typedApprover.SigLen != -1 {
			t.Error("Expected SigLen to be -1 since there is no signature")
		}

		message := app.GetDownload(dbCache, "test", ActionApproved)

		signed, err := ClearsignMessage(message, TestUser1Username)
		if err != nil {
			t.Errorf("Unable to sign message %s", err.Error())
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("sig", "./sig")
		if err != nil {
			t.Errorf("unable to create multipart for upload: %s", err.Error())
		}

		if _, err := part.Write([]byte(signed)); err != nil {
			t.Errorf("Error writing sig into form: %s", err.Error())
		}

		if err := writer.Close(); err != nil {
			t.Errorf("Error closing writer: %s", err.Error())
		}

		request, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Add("REMOTE_USER", TestUser1Username)

		parseErr := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
		if parseErr != nil {
			t.Errorf("No error was expected when uploading a valid signature, got %s", parseErr.Error())
		}

		dbCache.DB.Model(app).UpdateColumns(app)

		dbCache.WipeCache()

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		err = app.PostUpdate(dbCache, mustGetTestConf())
		So(err, ShouldBeNil)

		app = Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		obj, err2 := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err2, ShouldBeNil)

		typedApprover = obj.(*ApprovalPage)
		if typedApprover.App.ID != app.ID {
			t.Error("Expected the Approver in the returned page to have the same ID as the approver passed")
		}

		if !typedApprover.CanApprove {
			t.Error("Expected CanApprove to be true for the approver name passed")
		}

		if typedApprover.IsEditable {
			t.Error("Expected IsEditable to be set to true for tan approver in this state")
		}

		if typedApprover.IsSigned {
			t.Error("Expected IsSigned to be false since it is sigend")
		}

		if typedApprover.SigLen == len(message) {
			t.Errorf("Expected SigLen to be %d since there is no signature", len(message))
		}
	})
}

func Test_Approval_GetAllPage(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap db with the approver approval finished attempt to sign with a get all page", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		t.Helper()
		appr := ApproverRevision{}
		err = appr.SetID(2)
		So(err, ShouldBeNil)
		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)

		app := Approval{}
		obj, err := app.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)
		typedApprovals := obj.(*ApprovalsPage)
		if len(typedApprovals.Approvals) != 0 {
			t.Errorf("Expected the number of approvals returned to be 0, got %d", len(typedApprovals.Approvals))
		}

		if prepErr := appr.StartApprovalProcess(request, dbCache, mustGetTestConf()); prepErr != nil {
			t.Error("no error expected when starting approval when approver in active state")
		}

		app = Approval{}

		obj, err = app.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)
		typedApprovals = obj.(*ApprovalsPage)
		if len(typedApprovals.Approvals) != 1 {
			t.Errorf("Expected the number of approvals returned to be 1, got %d", len(typedApprovals.Approvals))
		}
	})
}

func Test_Approval_TakeAction_DownloadSig(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval finished attempt to sign with a downwload the signature", t, func() {
		conf := mustGetTestConf()

		ddbCache, err := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		if err != nil {
			return
		}

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(ddbCache)
		So(err, ShouldBeNil)

		writer := httptest.NewRecorder()
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		actionErrs2 := app.TakeAction(writer, request, ddbCache, "downloadsig", false, RemoteUserAuthType, conf)

		if len(actionErrs2) != 0 {
			for _, err := range actionErrs2 {
				t.Errorf("Unexpected error when downloadsig action: %s", err.Error())
			}
		}
	})
}

func Test_Approval_TakeAction_DownloadSig_NotApproved(t *testing.T) {
	t.Parallel()
	Convey("When trying to download the signature for an unapproved approval", t, func() {
		conf := mustGetTestConf()

		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		actionErrs2 := app.TakeAction(w, request, dbCache, "downloadsig", false, RemoteUserAuthType, conf)

		Convey("An error should be returned", func() {
			So(len(actionErrs2), ShouldBeGreaterThan, 0)
		})
	})
}

func Test_Approval_TakeAction_Download(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval start request the signature object", t, func() {
		conf := mustGetTestConf()

		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)

		if err := app.Prepare(dbCache); err != nil {
			t.Error("No Error Expected when calling prepare on an existing Approval")
		}

		response := httptest.NewRecorder()
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		actionErr1 := app.TakeAction(response, request, dbCache, "download", false, RemoteUserAuthType, conf)

		if actionErr1 == nil {
			t.Errorf("Expected an error when downloading the signature")
		}

		response = httptest.NewRecorder()
		request, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser2Username)
		request.Header.Add("REMOTE_USER_ORG", ExampleUserOrg)
		request.Form = make(url.Values)
		request.Form.Add("approverid", "1")
		actionErrs2 := app.TakeAction(response, request, dbCache, "download", false, RemoteUserAuthType, conf)

		if len(actionErrs2) != 0 {
			for _, err := range actionErrs2 {
				t.Errorf("Unexpected error when downloadsig action: %s", err.Error())
			}
		}

		response = httptest.NewRecorder()
		request, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser2Username)
		request.Header.Add("REMOTE_USER_ORG", ExampleUserOrg)
		request.Form = make(url.Values)
		request.Form.Add("approverid", "1")
		request.Form.Add("approverid", "1")
		actionErr3 := app.TakeAction(response, request, dbCache, "download", false, RemoteUserAuthType, conf)

		if actionErr3 == nil {
			t.Error("Expected an error when download action with multiple approver ids")
		}

		response = httptest.NewRecorder()
		request, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser2Username)
		request.Header.Add("REMOTE_USER_ORG", ExampleUserOrg)
		request.Form = make(url.Values)
		request.Form.Add("approverid", "a")
		actionErr4 := app.TakeAction(response, request, dbCache, "download", false, RemoteUserAuthType, conf)

		if actionErr4 == nil {
			t.Error("Expected an error when download action with a single approver with an ID of 'a'")
		}

		response = httptest.NewRecorder()
		request, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser2Username)
		request.Header.Add("REMOTE_USER_ORG", ExampleUserOrg)
		request.Form = make(url.Values)
		request.Form.Add("approverid", "2")
		actionErr5 := app.TakeAction(response, request, dbCache, "download", false, RemoteUserAuthType, conf)

		if actionErr5 == nil {
			t.Error("Expected an error when download action with a single approver and an unused ID")
		}

		approver := Approver{}
		err = approver.SetID(1)
		So(err, ShouldBeNil)
		err = approver.Prepare(dbCache)
		So(err, ShouldBeNil)

		approver.CurrentRevisionID.Valid = false
		err = dbCache.Save(&approver)
		So(err, ShouldBeNil)

		appr := ApproverRevision{}
		err = appr.SetID(2)
		So(err, ShouldBeNil)
		err = appr.Prepare(dbCache)
		So(err, ShouldBeNil)
	})
}

func Test_Approval_TakeAction_Bogus(t *testing.T) {
	t.Parallel()

	Convey("Given a bootstrap db with the approver approval start reqeust a bogus action", t, func() {
		conf := mustGetTestConf()

		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		if err != nil {
			return
		}

		app := Approval{}
		err = app.SetID(1)

		So(err, ShouldBeNil)

		if err := app.Prepare(dbCache); err != nil {
			t.Error("No Error Expected when calling prepare on an existing Approval")
		}

		w := httptest.NewRecorder()
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		actionErr1 := app.TakeAction(w, request, dbCache, bogusState, false, RemoteUserAuthType, conf)

		if actionErr1 == nil {
			t.Errorf("Expected an error when bogus action")
		}
	})
}

func TestApprovalIsCanclled(t *testing.T) {
	t.Parallel()
	Convey("Given an approval in the 'new' state", t, func() {
		app := Approval{}
		app.State = StateNew
		Convey("IsCancelled should return false", func() {
			So(app.IsCancelled(), ShouldBeFalse)
		})
	})

	Convey("Given an approval in the 'cancelled' state", t, func() {
		app := Approval{}
		app.State = StateCancelled
		Convey("IsCancelled should return true", func() {
			So(app.IsCancelled(), ShouldBeTrue)
		})
	})
}

func TestApprovalUpdateState(t *testing.T) {
	t.Parallel()
	Convey("Given an approval in an unknown state", t, func() {
		conf := mustGetTestConf()

		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapDoneApproverSetApproval)
		if err != nil {
			return
		}
		app := Approval{}
		err = app.SetID(1)
		So(err, ShouldBeNil)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		app.State = "bogusstate"
		err = dbCache.Save(&app)
		So(err, ShouldNotBeNil)

		dbCache.WipeCache()

		changeMade, errs := app.UpdateState(dbCache, conf)

		// UpdateState should return an error and the approval should remain in the bogus state
		app2 := Approval{}
		err = app2.SetID(1)
		So(err, ShouldBeNil)
		err = app2.Prepare(dbCache)
		So(err, ShouldBeNil)
		So(changeMade, ShouldEqual, false)
		So(len(errs), ShouldEqual, 1)
	})

	Convey("Given an approval with an approver set that has become invalid", t, func() {
		conf := mustGetTestConf()

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

		dbCache.WipeCache()

		revision := ApproverSetRevision{}
		err = revision.SetID(newAppr.ID)
		So(err, ShouldBeNil)
		err = revision.Prepare(dbCache)
		So(err, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		startAppErr := revision.StartApprovalProcess(request, dbCache, conf)

		revision = ApproverSetRevision{}
		err = revision.SetID(newAppr.ID)
		So(err, ShouldBeNil)
		err = revision.Prepare(dbCache)
		So(err, ShouldBeNil)

		// StartApprovalProcess on the new revision should work and a CRID should be set
		So(startAppErr, ShouldBeNil)
		So(revision.CRID.Valid, ShouldBeTrue)

		appset := ApproverSet{}
		err = appset.SetID(1)
		So(err, ShouldBeNil)
		err = appset.PrepareShallow(dbCache)
		So(err, ShouldBeNil)
		appset.State = StateInactive
		err = dbCache.Save(&appset)
		So(err, ShouldBeNil)

		dbCache.WipeCache()

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(revision.CRID.Int64)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)
		app := changeRequest.Approvals[0]

		changeMade, errs := app.UpdateState(dbCache, conf)
		Convey("UpdateState should not return any errors and the Approval should move to the StateInactiveApproverSet state", func() {
			So(len(errs), ShouldEqual, 0)
			So(app.State, ShouldEqual, StateInactiveApproverSet)
			So(changeMade, ShouldBeTrue)
		})
	})

	Convey("Given an approval in the StateNoValidApprovers state with valid approvers and approver sets", t, func() {
		conf := mustGetTestConf()

		// dbCache, err := DBFactory.GetDB(t, TestStateBootstrapDoneApproverSetApproval)
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapped)
		So(err, ShouldBeNil)

		t.Helper()
		dbCache.DB = dbCache.DB.Debug()

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

		revision := ApproverSetRevision{}
		err = revision.SetID(newAppr.ID)
		So(err, ShouldBeNil)
		err = revision.Prepare(dbCache)
		So(err, ShouldBeNil)

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		startAppErr := revision.StartApprovalProcess(request, dbCache, conf)

		revision = ApproverSetRevision{}
		err = revision.SetID(newAppr.ID)
		So(err, ShouldBeNil)
		err = revision.Prepare(dbCache)
		So(err, ShouldBeNil)

		// StartApprovalProcess on the new revision should work and a CRID should be set
		So(startAppErr, ShouldBeNil)
		So(revision.CRID.Valid, ShouldBeTrue)

		appset := ApproverSet{}
		err = appset.SetID(1)
		So(err, ShouldBeNil)
		err = appset.Prepare(dbCache)
		So(err, ShouldBeNil)
		appset.State = StateInactive
		err = dbCache.Save(&appset)
		So(err, ShouldBeNil)

		dbCache.WipeCache()

		changeRequest := ChangeRequest{}
		err = changeRequest.SetID(revision.CRID.Int64)
		So(err, ShouldBeNil)
		err = changeRequest.Prepare(dbCache)
		So(err, ShouldBeNil)
		app := changeRequest.Approvals[0]
		app.State = StateNoValidApprovers
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)

		dbCache.WipeCache()

		app2 := Approval{}
		err = app2.SetID(app.ID)
		So(err, ShouldBeNil)
		err = app2.Prepare(dbCache)
		So(err, ShouldBeNil)

		changesMade, errs := app2.UpdateState(dbCache, conf)
		// UpdateState should not return any errors and the Approval should move to the StateInactiveApproverSet state
		So(len(errs), ShouldEqual, 0)
		So(app2.State, ShouldEqual, StateInactiveApproverSet)
		So(changesMade, ShouldBeFalse)
	})
}

func TestApprovalPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApprovalPage", t, func() {
		approvalPage := ApprovalPage{}
		Convey("Calling SetCSRFToken should not panic", func() {
			So(func() { approvalPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)
		})

		testTokenString := testingCSRFToken
		approvalPage.CSRFToken = testTokenString
		Convey("Calling GetCSRFToken should return the testing token string", func() {
			So(approvalPage.GetCSRFToken(), ShouldEqual, testTokenString)
		})
	})
}

const testingCSRFToken = "testtoken"

func TestApprovalsPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApprovalsPage", t, func() {
		approvalPage := ApprovalsPage{}
		Convey("Calling SetCSRFToken should not panic", func() {
			So(func() { approvalPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)
		})

		testTokenString := testingCSRFToken
		approvalPage.CSRFToken = testTokenString
		Convey("Calling GetCSRFToken should return the testing token string", func() {
			So(approvalPage.GetCSRFToken(), ShouldEqual, testTokenString)
		})
	})
}
