package lib

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestApproverGetID(t *testing.T) {
	t.Parallel()
	Convey("Given an approver of ID 1", t, func() {
		app := Approver{Model: Model{ID: 1}}
		// The ID should be GetID should return 1
		So(app.GetID(), ShouldEqual, 1)
	})

	Convey("Given an approver of ID 2", t, func() {
		app2 := Approver{Model: Model{ID: 2}}
		// The ID should be GetID should return 2
		So(app2.GetID(), ShouldEqual, 2)
	})
}

func TestApproverSetID(t *testing.T) {
	t.Parallel()
	Convey("Given SetID of 1", t, func() {
		app := Approver{}
		err := app.SetID(1)
		// The ID should be 1
		So(app.GetID(), ShouldEqual, 1)

		// No error should be thrown
		So(err, ShouldBeNil)
	})

	Convey("Given SetID of 2 should work", t, func() {
		app := Approver{}
		err := app.SetID(2)
		// The ID should be 2
		So(app.GetID(), ShouldEqual, 2)

		// No error should be thrown
		So(err, ShouldBeNil)
	})

	Convey("Given SetID of 0", t, func() {
		app := Approver{}
		err := app.SetID(0)
		// An error should be thrown
		So(err, ShouldNotBeNil)
	})

	Convey("Given SetID of -1", t, func() {
		app := Approver{}
		err := app.SetID(-1)
		// An error should be thrown
		So(err, ShouldNotBeNil)
	})

	Convey("Given SetID on an Approver that has an ID set", t, func() {
		app := Approver{}
		err := app.SetID(1)
		// The first set should not throw an error
		So(err, ShouldBeNil)

		// The first set should have an ID of 1
		So(app.GetID(), ShouldEqual, 1)

		err2 := app.SetID(2)

		// An error should be thrown
		So(err2, ShouldNotBeNil)

		// The ID of the approver should be 1
		So(app.GetID(), ShouldEqual, 1)
	})
}

func TestApproverGetExportShortVersion(t *testing.T) {
	t.Parallel()
	Convey("Given an existing Approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		export := app.GetExportShortVersion()
		// The ID should be greater than 0
		So(export.ID, ShouldEqual, 1)

		// The State should not be empty
		So(len(export.State), ShouldBeGreaterThan, 0)
	})

	Convey("Given a new approver", t, func() {
		app := Approver{}
		export := app.GetExportShortVersion()
		// The ID should be 0
		So(export.ID, ShouldEqual, 0)

		// The State should be empty
		So(len(export.State), ShouldEqual, 0)
	})
}

func TestApproverHasRevision(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap version with a pending new revision", t, func() {
		// dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		// HasRevision should be true
		So(app.HasRevision(), ShouldBeTrue)
	})

	Convey("Given an non-prepared approver", t, func() {
		app := Approver{}
		// Has Revision should be false
		So(app.HasRevision(), ShouldBeFalse)
	})
}

func TestApproverHasPendingRevision(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrap version with a pending new revision", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		// HasPendingRevision should be true
		So(app.HasPendingRevision(), ShouldBeTrue)
	})

	Convey("Given a bootstrapped version with no new revision", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		// HasPendingRevision should be false
		So(app.HasPendingRevision(), ShouldBeFalse)
	})

	Convey("Given an non-prepared approver", t, func() {
		app := Approver{}
		// HasPendingRevision should be false
		So(app.HasPendingRevision(), ShouldBeFalse)
	})
}

func TestApproverSuggestedRevisionValue(t *testing.T) {
	t.Parallel()
	Convey("Given the default approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		curRev := app.CurrentRevision
		// The Name field should be the same as the current revisions Name Field", func() {
		So(app.SuggestedRevisionValue(ApproverFieldName), ShouldEqual, curRev.Name)

		// The Username field should be the same as the current revisions Username Field", func() {
		So(app.SuggestedRevisionValue(ApproverFieldUsername), ShouldEqual, curRev.Username)

		// The EmployeeID field should be the same as the current revisions EmployeeID Field (As a string)", func() {
		So(app.SuggestedRevisionValue(ApproverFieldEmployeeID), ShouldEqual, fmt.Sprintf("%d", curRev.EmployeeID))

		// The Department field should be the same as the current revisions Department Field", func() {
		So(app.SuggestedRevisionValue(ApproverFieldDepartment), ShouldEqual, curRev.Department)

		// The Fingerprint field should be the same as the current revisions Fingerprint Field", func() {
		So(app.SuggestedRevisionValue(ApproverFieldFingerprint), ShouldEqual, curRev.Fingerprint)

		// The PublicKey field should be the same as the current revisions PublicKey Field", func() {
		So(app.SuggestedRevisionValue(ApproverFieldPublicKey), ShouldEqual, curRev.PublicKey)

		// The EmailAddres field should be the same as the current revisions EmailAddres Field", func() {
		So(app.SuggestedRevisionValue(ApproverFieldEmailAddres), ShouldEqual, curRev.EmailAddress)

		// The Role field should be the same as the current revisions Role Field", func() {
		So(app.SuggestedRevisionValue(ApproverFieldRole), ShouldEqual, curRev.Role)

		// Asking for the suggestedRevisionValue of an unknown field should be an empty string", func() {
		So(app.SuggestedRevisionValue("BogusField"), ShouldEqual, "")
	})

	Convey("Given an empty approver", t, func() {
		app := Approver{}
		// The Name field should be an empty string", func() {
		So(app.SuggestedRevisionValue(ApproverFieldName), ShouldEqual, "")

		// The Username field should be an empty string", func() {
		So(app.SuggestedRevisionValue(ApproverFieldUsername), ShouldEqual, "")

		// The EmployeeID field should be an empty string", func() {
		So(app.SuggestedRevisionValue(ApproverFieldEmployeeID), ShouldEqual, "")

		// The Department field should be an empty string", func() {
		So(app.SuggestedRevisionValue(ApproverFieldDepartment), ShouldEqual, "")

		// The Fingerprint field should be an empty string", func() {
		So(app.SuggestedRevisionValue(ApproverFieldFingerprint), ShouldEqual, "")

		// The PublicKey field should be an empty string", func() {
		So(app.SuggestedRevisionValue(ApproverFieldPublicKey), ShouldEqual, "")

		// The EmailAddres field should be an empty string", func() {
		So(app.SuggestedRevisionValue(ApproverFieldEmailAddres), ShouldEqual, "")

		// The Role field should be an empty string", func() {
		So(app.SuggestedRevisionValue(ApproverFieldRole), ShouldEqual, "")

		// Asking for the suggestedRevisionValue of an unknown field should be an empty string", func() {
		So(app.SuggestedRevisionValue("BogusField"), ShouldEqual, "")
	})
}

func TestApproverGetCurrentValue(t *testing.T) {
	t.Parallel()
	Convey("Given the default approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		curRev := app.CurrentRevision
		// The Name field should be the same as the current revisions Name Field", func() {
		So(app.GetCurrentValue(ApproverFieldName), ShouldEqual, curRev.Name)

		// The Username field should be the same as the current revisions Username Field", func() {
		So(app.GetCurrentValue(ApproverFieldUsername), ShouldEqual, curRev.Username)

		// The EmployeeID field should be the same as the current revisions EmployeeID Field (As a string)", func() {
		So(app.GetCurrentValue(ApproverFieldEmployeeID), ShouldEqual, fmt.Sprintf("%d", curRev.EmployeeID))

		// The Department field should be the same as the current revisions Department Field", func() {
		So(app.GetCurrentValue(ApproverFieldDepartment), ShouldEqual, curRev.Department)

		// The Fingerprint field should be the same as the current revisions Fingerprint Field", func() {
		So(app.GetCurrentValue(ApproverFieldFingerprint), ShouldEqual, curRev.Fingerprint)

		// The PublicKey field should be the same as the current revisions PublicKey Field", func() {
		So(app.GetCurrentValue(ApproverFieldPublicKey), ShouldEqual, curRev.PublicKey)

		// The EmailAddres field should be the same as the current revisions EmailAddres Field", func() {
		So(app.GetCurrentValue(ApproverFieldEmailAddres), ShouldEqual, curRev.EmailAddress)

		// The Role field should be the same as the current revisions Role Field", func() {
		So(app.GetCurrentValue(ApproverFieldRole), ShouldEqual, curRev.Role)

		// Asking for the suggestedRevisionValue of an unknown field should be an empty string", func() {
		So(app.GetCurrentValue("BogusField"), ShouldEqual, "")
	})

	Convey("Given an empty approver", t, func() {
		app := Approver{}
		// The Name field should be an empty string", func() {
		So(app.GetCurrentValue(ApproverFieldName), ShouldEqual, UnPreparedApproverError)

		// The Username field should be an empty string", func() {
		So(app.GetCurrentValue(ApproverFieldUsername), ShouldEqual, UnPreparedApproverError)

		// The EmployeeID field should be an empty string", func() {
		So(app.GetCurrentValue(ApproverFieldEmployeeID), ShouldEqual, UnPreparedApproverError)

		// The Department field should be an empty string", func() {
		So(app.GetCurrentValue(ApproverFieldDepartment), ShouldEqual, UnPreparedApproverError)

		// The Fingerprint field should be an empty string", func() {
		So(app.GetCurrentValue(ApproverFieldFingerprint), ShouldEqual, UnPreparedApproverError)

		// The PublicKey field should be an empty string", func() {
		So(app.GetCurrentValue(ApproverFieldPublicKey), ShouldEqual, UnPreparedApproverError)

		// The EmailAddres field should be an empty string", func() {
		So(app.GetCurrentValue(ApproverFieldEmailAddres), ShouldEqual, UnPreparedApproverError)

		// The Role field should be an empty string", func() {
		So(app.GetCurrentValue(ApproverFieldRole), ShouldEqual, UnPreparedApproverError)

		// Asking for the suggestedRevisionValue of an unknown field should be an empty string", func() {
		So(app.GetCurrentValue("BogusField"), ShouldEqual, UnPreparedApproverError)
	})

	Convey("Given a new", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{}
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)
		testID := app.ID
		app = Approver{Model: Model{ID: testID}}
		err = app.Prepare(dbCache)
		if err != nil {
			t.Error(err.Error())
		}

		Convey("The Name field should be an empty string", func() {
			So(app.GetCurrentValue(ApproverFieldName), ShouldEqual, "")
		})
		Convey("The Username field should be an empty string", func() {
			So(app.GetCurrentValue(ApproverFieldUsername), ShouldEqual, "")
		})
		Convey("The EmployeeID field should be an empty string", func() {
			So(app.GetCurrentValue(ApproverFieldEmployeeID), ShouldEqual, "")
		})
		Convey("The Department field should be an empty string", func() {
			So(app.GetCurrentValue(ApproverFieldDepartment), ShouldEqual, "")
		})
		Convey("The Fingerprint field should be an empty string", func() {
			So(app.GetCurrentValue(ApproverFieldFingerprint), ShouldEqual, "")
		})
		Convey("The PublicKey field should be an empty string", func() {
			So(app.GetCurrentValue(ApproverFieldPublicKey), ShouldEqual, "")
		})
		Convey("The EmailAddres field should be an empty string", func() {
			So(app.GetCurrentValue(ApproverFieldEmailAddres), ShouldEqual, "")
		})
		Convey("The Role field should be an empty string", func() {
			So(app.GetCurrentValue(ApproverFieldRole), ShouldEqual, "")
		})
		Convey("Asking for the suggestedRevisionValue of an unknown field should be an empty string", func() {
			So(app.GetCurrentValue("BogusField"), ShouldEqual, "")
		})
	})
}

func TestApproverParseFromForm(t *testing.T) {
	t.Parallel()

	dbCache, err := DBFactory.GetDB(t, TestStateEmpty)
	if err != nil {
		return
	}

	Convey("Given a HTTP request with no user set", t, func() {
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		app := Approver{}
		parseError := app.ParseFromForm(request, dbCache)
		// An error should be returned
		So(parseError, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid user set", t, func() {
		currentTime := TimeNow()
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		request.Header.Add("REMOTE_USER", TestUser1Username)
		app := Approver{}
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)
		parseError := app.ParseFromForm(request, dbCache)

		// The approver's state should be new", func() {
		So(app.State, ShouldEqual, StateNew)

		// The created by field should be the name of the valid user", func() {
		So(app.CreatedBy, ShouldEqual, TestUser1Username)

		// The updated by field should be the name of the valid user", func() {
		So(app.UpdatedBy, ShouldEqual, TestUser1Username)

		// The created at field should be greater than the time the test started", func() {
		So(app.CreatedAt, ShouldHappenOnOrAfter, currentTime)

		// The updated at field should be greater than the time the test started", func() {
		So(app.UpdatedAt, ShouldHappenOnOrAfter, currentTime)

		// No error should be returned", func() {
		So(parseError, ShouldBeNil)
	})
}

func TestApproverExportVersionAt(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverExportFull object with valid revisions", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapped)
		So(err, ShouldBeNil)

		t.Log(t)
		t.Log(dbCache)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		rev := app.CurrentRevision
		So(rev, ShouldNotBeNil)
		export := app.GetExportVersion()
		Convey("At invalidly old time", func() {
			_, err := app.GetExportVersionAt(dbCache, 0)

			// Should return error
			So(err, ShouldNotBeNil)
		})
		Convey("At Time of Current Revision", func() {
			So(rev, ShouldNotBeNil)
			So(rev.PromotedTime, ShouldNotBeNil)
			revTS := rev.PromotedTime.Unix()
			baseJSON, err := export.ToJSON()
			// export should product valid JSON
			So(err, ShouldBeNil)
			exportAt, err := app.GetExportVersionAt(dbCache, revTS)

			// Should return valid export
			So(err, ShouldBeNil)

			// exportAt should product valid JSON
			atJSON, err := exportAt.ToJSON()
			So(err, ShouldBeNil)

			// Json should match
			So(atJSON, ShouldEqual, baseJSON)
		})
	})
}

func TestApproverIsEditable(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver in state new", t, func() {
		app := Approver{}
		app.State = StateNew

		// The Approver should be Editable
		So(app.IsEditable(), ShouldBeTrue)
	})

	Convey("Given an Approver in state pendingnew", t, func() {
		app := Approver{}
		app.State = StatePendingNew

		// The Approver should not be Editable
		So(app.IsEditable(), ShouldBeFalse)
	})

	Convey("Given an Approver in state active", t, func() {
		app := Approver{}
		app.State = StateActive
		// The Approver should not be Editable
		So(app.IsEditable(), ShouldBeFalse)
	})

	Convey("Given an Approver in state inactive", t, func() {
		app := Approver{}
		app.State = StateInactive
		// The Approver should not be Editable
		So(app.IsEditable(), ShouldBeFalse)
	})

	Convey("Given an Approver in state activependingapproval", t, func() {
		app := Approver{}
		app.State = StateActivePendingApproval
		// The Approver should not be Editable
		So(app.IsEditable(), ShouldBeFalse)
	})

	Convey("Given an Approver in state inactivependingapproval", t, func() {
		app := Approver{}
		app.State = StateInactivePendingApproval
		// The Approver should not be Editable
		So(app.IsEditable(), ShouldBeFalse)
	})

	Convey("Given an Approver in state bootstrap", t, func() {
		app := Approver{}
		app.State = StateBootstrap
		// The Approver should not be Editable
		So(app.IsEditable(), ShouldBeFalse)
	})

	Convey("Given an Approver in state pendingbootstrap", t, func() {
		app := Approver{}
		app.State = StatePendingBootstrap
		// The Approver should not be Editable
		So(app.IsEditable(), ShouldBeFalse)
	})
}

func TestApproverGetAllPage(t *testing.T) {
	t.Parallel()
	Convey("Given an empty database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateEmpty)
		So(err, ShouldBeNil)

		app := Approver{}
		page, err := app.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)
		typedApprovers := page.(*ApproversPage)

		// There should be no approvers
		So(len(typedApprovers.Approvers), ShouldEqual, 0)
	})

	Convey("Given a bootstrap database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{}
		page, err := app.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)
		typedApprovers := page.(*ApproversPage)
		// There should be one approve
		So(len(typedApprovers.Approvers), ShouldEqual, 1)

		// The approver should be prepared
		So(typedApprovers.Approvers[0].prepared, ShouldBeTrue)
	})
}

func TestApproverGetGPGKeyBlock(t *testing.T) {
	t.Parallel()
	Convey("Given an approver with a valid public key", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		entity, err := app.GetGPGKeyBlock()
		// No error should be returned
		So(err, ShouldBeNil)

		// The entity returned should be non nil
		So(entity, ShouldNotBeNil)
	})

	Convey("Given an approver with a poorly formatted public key", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		app.CurrentRevision.PublicKey = `sdfsdfsd		`
		err = dbCache.Save(&app.CurrentRevision)
		So(err, ShouldBeNil)
		_, err = app.GetGPGKeyBlock()

		// An error should be returned
		So(err, ShouldNotBeNil)
	})

	Convey("Given an approver with a public key that has no entity", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		app.CurrentRevision.PublicKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----
			Version: GnuPG v1

			BROKEN

			-----END PGP PUBLIC KEY BLOCK-----
			`
		err = dbCache.Save(&app.CurrentRevision)
		So(err, ShouldBeNil)
		_, err = app.GetGPGKeyBlock()

		// An error should be returned
		So(err, ShouldNotBeNil)
	})

	Convey("Given an approver with no revision", t, func() {
		app := Approver{}
		_, err := app.GetGPGKeyBlock()

		// An error should be returned
		So(err, ShouldNotBeNil)
	})
}

func TestApproverGetRequiredApproverSets(t *testing.T) {
	t.Parallel()
	Convey("Given the default approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		appSets, err := app.GetRequiredApproverSets(dbCache)
		// No Error should be returned
		So(err, ShouldBeNil)

		// The Approver set list returned should have a length of 1
		So(len(appSets), ShouldEqual, 1)

		// The Approver set in the list should have an ID of 1
		So(appSets[0].ID, ShouldEqual, 1)
	})

	Convey("Given a new approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{}
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)
		testID := app.ID
		t.Logf("Test ID: %d", testID)
		app = Approver{Model: Model{ID: testID}}
		t.Logf("App: %v", app)
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		appSets, err := app.GetRequiredApproverSets(dbCache)
		// An error should be returned
		So(err, ShouldNotBeNil)

		// The Approver set list returned should have a length of 0
		So(len(appSets), ShouldEqual, 0)
	})
}

func TestApproverGetInformedApproverSets(t *testing.T) {
	t.Parallel()
	Convey("Given the default approver", t, func() {
		dbCacheb, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		dbCacheb.DB = dbCacheb.DB.Debug()

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCacheb)

		// Prepare should not error
		So(err, ShouldBeNil)

		appSets, err := app.GetInformedApproverSets(dbCacheb)
		// No Error should be returned
		So(err, ShouldBeNil)

		// The Approver set list returned should be empty
		So(appSets, ShouldBeEmpty)
	})

	Convey("Given a new approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{}
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)
		testID := app.ID
		app = Approver{Model: Model{ID: testID}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		appSets, err := app.GetInformedApproverSets(dbCache)

		// An error should be returned
		So(err, ShouldNotBeNil)

		// The Approver set list returned should have a length of 0
		So(appSets, ShouldBeEmpty)
	})
}

func TestApproverGetValidApproverMap(t *testing.T) {
	t.Parallel()
	Convey("Given the Bootstrap database with the approver approved", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(err, ShouldBeNil)

		approverMap, err := GetValidApproverMap(dbCache)
		So(err, ShouldBeNil)

		// The length of the map should be 1
		So(len(approverMap), ShouldEqual, 1)

		val, ok := approverMap[1]
		// The key 1 in the map should be set
		So(ok, ShouldBeTrue)

		// The value for key 1 should have a length greater than 0
		So(len(val), ShouldBeGreaterThan, 0)
	})

	Convey("Given an empty database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateEmpty)
		So(err, ShouldBeNil)

		approverMap, err := GetValidApproverMap(dbCache)
		So(err, ShouldBeNil)

		// The length of the map should be 0
		So(len(approverMap), ShouldEqual, 0)
	})
}

// func TestApproverParseApprovers(t *testing.T) {
// 	db, err  := DBFactory.GetDB(t, TestStateBootstrap)
// 	if err != nil {
// 		return
// 	}
//
// 	Convey("Given a HTTP request with no approvers set", t, func() {
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser2Username)
// 		r.Form = make(url.Values)
// 		approvers, err := ParseApprovers(r, db, "approverid")
// 		Convey("The error should be nil", func() {
// 			So(err, ShouldBeNil)
// 		})
//
// 		Convey("The length of the approver list should be 0", func() {
// 			So(len(approvers), ShouldEqual, 0)
// 		})
//
// 	})
//
// 	Convey("Given a HTTP request with 1 valid approver", t, func() {
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser2Username)
// 		r.Form = make(url.Values)
// 		r.Form.Add("approverid", "1")
// 		approvers, err := ParseApprovers(r, db, "approverid")
// 		Convey("The error should be nil", func() {
// 			So(err, ShouldBeNil)
// 		})
//
// 		Convey("The length of the approver list should be 1", func() {
// 			So(len(approvers), ShouldEqual, 1)
// 		})
//
// 	})
//
// 	Convey("Given a HTTP request with 1 character set as the approver set", t, func() {
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser2Username)
// 		r.Form = make(url.Values)
// 		r.Form.Add("approverid", "a")
// 		approvers, err := ParseApprovers(r, db, "approverid")
// 		Convey("The error should not be nil", func() {
// 			So(err, ShouldNotBeNil)
// 		})
//
// 		Convey("The length of the approver list should be 0", func() {
// 			So(len(approvers), ShouldEqual, 0)
// 		})
//
// 	})
//
// 	Convey("Given a HTTP request with 1 valid and 1 invalid approver", t, func() {
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser2Username)
// 		r.Form = make(url.Values)
// 		r.Form.Add("approverid", "1")
// 		r.Form.Add("approverid", "a")
// 		approvers, err := ParseApprovers(r, db, "approverid")
// 		Convey("The error should not be nil", func() {
// 			So(err, ShouldNotBeNil)
// 		})
//
// 		Convey("The length of the approver list should be 1", func() {
// 			So(len(approvers), ShouldEqual, 1)
// 		})
//
// 	})
//
// 	Convey("Given a HTTP request with unknown approvers", t, func() {
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser2Username)
// 		r.Form = make(url.Values)
// 		r.Form.Add("approverid", "2")
// 		approvers, err := ParseApprovers(r, db, "approverid")
// 		Convey("The error should not be nil", func() {
// 			So(err, ShouldNotBeNil)
// 		})
//
// 		Convey("The length of the approver list should be 0", func() {
// 			So(len(approvers), ShouldEqual, 0)
// 		})
//
// 	})
//
// }

func TestApproverGetPage(t *testing.T) {
	t.Parallel()
	Convey("Given the initial bootstrap approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		page, err := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedApproverPage := page.(*ApproverPage)

		// There should be pending action available
		So(len(typedApproverPage.PendingActions), ShouldBeGreaterThan, 0)

		Convey("The App field should be set to the passed approver", func() {
			Convey("The ID of the approvers should match", func() {
				So(typedApproverPage.App.ID, ShouldEqual, app.GetID())
			})

			Convey("The approver should be prepared", func() {
				So(typedApproverPage.App.prepared, ShouldBeTrue)
			})
		})

		// ValidApproverSets should have 1 approver in the list
		So(len(typedApproverPage.ValidApproverSets), ShouldEqual, 1)
	})

	Convey("Given a modified bootstrap approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		app.CurrentRevision.InformedApproverSets = app.CurrentRevision.RequiredApproverSets
		app.CurrentRevision.RequiredApproverSets = make([]ApproverSet, 1)
		err = dbCache.Save(&app)
		So(err, ShouldBeNil)

		page, err := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedApproverPage := page.(*ApproverPage)
		// There should be pending action available
		So(len(typedApproverPage.PendingActions), ShouldBeGreaterThan, 0)

		Convey("The App field should be set to the passed approver", func() {
			Convey("The ID of the approvers should match", func() {
				So(typedApproverPage.App.ID, ShouldEqual, app.GetID())
			})

			Convey("The approver should be prepared", func() {
				So(typedApproverPage.App.prepared, ShouldBeTrue)
			})
		})

		// ValidApproverSets should have 1 approverset in the list
		So(len(typedApproverPage.ValidApproverSets), ShouldEqual, 1)
	})

	Convey("Given a new Approver", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapped)
		So(err, ShouldBeNil)

		app := Approver{}

		page, err := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedApproverPage := page.(*ApproverPage)
		/// There should be no pending actions
		So(len(typedApproverPage.PendingActions), ShouldEqual, 0)

		// he App field should be set to the current approver
		// The ID of the approver should be 0
		So(typedApproverPage.App.ID, ShouldEqual, 0)

		// The approver should not be prepared
		So(typedApproverPage.App.prepared, ShouldBeFalse)

		// ValidApproverSets should be empty
		So(len(typedApproverPage.ValidApproverSets), ShouldEqual, 1)
	})

	Convey("Given a new approver after a bootstrap takes place", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{}
		page, err := app.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedApproverPage := page.(*ApproverPage)
		// ValidApproverSets should have 1 approverset in the list
		So(len(typedApproverPage.ValidApproverSets), ShouldEqual, 1)
	})
}

func TestApproverTakeAction(t *testing.T) {
	t.Parallel()
	Convey("Given TakeAction can run", t, func() {
		conf := mustGetTestConf()

		dbCache, err := DBFactory.GetDB(t, TestStateEmpty)
		So(err, ShouldBeNil)

		app := Approver{}

		writer := httptest.NewRecorder()
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		err2 := app.TakeAction(writer, request, dbCache, "", false, RemoteUserAuthType, conf)
		// no error was returned
		So(err2, ShouldBeNil)
	})
}

func TestApproverParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	Convey("Given ParseFromFormUpdate can run", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateEmpty)
		So(err, ShouldBeNil)

		app := Approver{}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

		err2 := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())

		// An error should be returned
		So(err2, ShouldNotBeNil)
	})
}

// func TestApproverPostUpdate(t *testing.T) {
//
// 	Convey("Given ParseFromFormUpdate can run", t, func() {
// 		db, err  := DBFactory.GetDB(t, TestStateEmpty)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
//
// 		app.PostUpdate(db)
// 		So(true, ShouldBeTrue)
//
// 	})
//
// }

func TestApproverExportFullToJSON(t *testing.T) {
	t.Parallel()

	Convey("Given an initial bootstrap database, test the approver export full to json method", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)

		export := app.GetExportVersion()

		Convey("Given a valid ApproverExportFull", func() {
			exportStr1, exportErr1 := export.ToJSON()
			// The error returned should be nil
			So(exportErr1, ShouldBeNil)

			// The length of the string returned should be greater than 0
			So(len(exportStr1), ShouldBeGreaterThan, 0)
		})

		Convey("Given a valid ApproverExportFull with its ID changed to 0", func() {
			typedExport := export.(ApproverExportFull)
			typedExport.ID = 0
			_, exportErr2 := typedExport.ToJSON()
			// The error returned should not be nil
			So(exportErr2, ShouldNotBeNil)
		})

		Convey("Given a valid ApproverExportFull with its ID change to -1", func() {
			typedExport := export.(ApproverExportFull)
			typedExport.ID = -1
			_, exportErr2 := typedExport.ToJSON()
			// The error returned should not be nil
			So(exportErr2, ShouldNotBeNil)
		})
	})
}

func TestApproverExportFullGetDiff(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverExportFull object with an empty pending revision", t, func() {
		app := ApproverExportFull{}
		app.CurrentRevision = ApproverRevisionExport{}
		app.PendingRevision = ApproverRevisionExport{ID: 0}

		_, err := app.GetDiff()

		// GetDiff should return an error
		So(err, ShouldNotBeNil)
	})
}

func TestApproverExportShortToJSON(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverExportShort object with not initialized", t, func() {
		app := ApproverExportShort{}

		retval, err := app.ToJSON()

		// ToJSON should not return an error
		So(err, ShouldBeNil)

		// ToJSON should return a string that is not empty
		So(len(retval), ShouldBeGreaterThan, 0)
	})
}

// func TestApproverUpdateState(t *testing.T) {
// 	Convey("Given a bootstrapped database", t, func() {
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapDoneApproverSetApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.CurrentRevision.Prepare(db)
//
// 		newAppr := ApproverRevision{}
// 		newAppr.ApproverID = app.CurrentRevision.ApproverID
// 		newAppr.DesiredState = app.CurrentRevision.DesiredState
// 		newAppr.RevisionState = StateNew
// 		newAppr.Name = app.CurrentRevision.Name
// 		newAppr.EmailAddress = app.CurrentRevision.EmailAddress
// 		newAppr.Role = app.CurrentRevision.Role
// 		newAppr.Username = app.CurrentRevision.Username
// 		newAppr.EmployeeID = app.CurrentRevision.EmployeeID
// 		newAppr.Department = app.CurrentRevision.Department
// 		newAppr.Fingerprint = app.CurrentRevision.Fingerprint
// 		newAppr.PublicKey = app.CurrentRevision.PublicKey
// 		newAppr.CreatedAt = time.Now()
// 		newAppr.CreatedBy = TestUser1Username
// 		newAppr.UpdatedAt = time.Now()
// 		newAppr.UpdatedBy = TestUser1Username
// 		newAppr.RequiredApproverSets = append(newAppr.RequiredApproverSets, app.CurrentRevision.RequiredApproverSets...)
// 		newAppr.RequiredApproverSets = append(newAppr.RequiredApproverSets, app.CurrentRevision.RequiredApproverSets...)
//
// 		db.Save(&newAppr)
//
// 		app2 := Approver{}
// 		app2.SetID(1)
// 		app2.Prepare(db)
// 		errs := app2.UpdateState(db,conf)
//
// 		revision := ApproverRevision{}
// 		revision.SetID(newAppr.ID)
// 		revision.Prepare(db)
//
// 		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
// 		r.Header.Add("REMOTE_USER", TestUser1Username)
// 		startAppErr := revision.StartApprovalProcess(r, db)
//
// 		revision = ApproverRevision{}
// 		revision.SetID(newAppr.ID)
// 		revision.Prepare(db)
//
// 		Convey("StartApprovalProcess on the new revision should work and a CRID should be set", func() {
// 			So(startAppErr, ShouldBeNil)
// 			So(revision.CRID.Valid, ShouldBeTrue)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(revision.CRID.Int64)
// 		cr.Prepare(db)
// 		for _, app := range cr.Approvals {
// 			ApproveApproval(db, TestUser1Username, app.ID)
// 		}
//
// 		app3 := Approver{}
// 		app3.SetID(1)
// 		app3.Prepare(db)
// 		Convey("Update state should not return any errors", func() {
// 			So(len(errs), ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver in pending bootstrap with no pending revision (cancelled revision)", t, func() {
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		appRev := ApproverRevision{}
// 		appRev.SetID(app.PendingRevision.ID)
// 		appRev.Prepare(db)
// 		errs := appRev.Cancel(db)
// 		So(len(errs), ShouldEqual, 0)
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should transition to in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StateBootstrap)
// 			So(app.PendingRevision.ID, ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver in the active state with no pending revision (cancelled revision)", t, func() {
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = StateActivePendingApproval
// 		db.Save(&app)
//
// 		appRev1 := ApproverRevision{}
// 		appRev1.SetID(app.CurrentRevisionID.Int64)
// 		appRev1.Prepare(db)
// 		appRev1.DesiredState = StateActive
// 		db.Save(&appRev1)
//
// 		appRev2 := ApproverRevision{}
// 		appRev2.SetID(app.PendingRevision.ID)
// 		appRev2.Prepare(db)
// 		errs := appRev2.Cancel(db)
// 		So(len(errs), ShouldEqual, 0)
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should transition to in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StateActive)
// 			So(app.PendingRevision.ID, ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver in the pending new state with no pending or current revision (cancelled revision)", t, func() {
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = StatePendingNew
// 		app.CurrentRevisionID.Valid = false
// 		db.Save(&app)
//
// 		appRev := ApproverRevision{}
// 		appRev.SetID(app.PendingRevision.ID)
// 		appRev.Prepare(db)
// 		errs := appRev.Cancel(db)
// 		So(len(errs), ShouldEqual, 0)
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should transition to in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StateNew)
// 			So(app.PendingRevision.ID, ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver in the New state", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = StateNew
//
// 		errs := app.UpdateState(db,conf)
// 		Convey("No error should be returned", func() {
// 			So(errs, ShouldBeEmpty)
// 		})
// 	})
//
// 	Convey("Given an approver in an unknown state", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = "Bogus"
//
// 		errs := app.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with a CR with the wrong object type", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(app.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.RegistrarObjectType = ChangeRequestType
// 		db.Save(&cr)
//
// 		errs := app.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with a CR with the wrong object id", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(app.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.RegistrarObjectID = 100
// 		db.Save(&cr)
//
// 		errs := app.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with a CR with the wrong proposed revison id", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(app.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.ProposedRevisionID = 100
// 		db.Save(&cr)
//
// 		errs := app.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with a CR with the wrong initial revison id", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(app.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.InitialRevisionID.Int64 = 100
// 		db.Save(&cr)
//
// 		errs := app.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with a CR with no initial revison", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(app.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.InitialRevisionID.Valid = false
// 		cr.InitialRevisionID.Int64 = 0
// 		db.Save(&cr)
//
// 		errs := app.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with a CR with an initial revion set but no currentRevision in the approver", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
// 		app := Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.CurrentRevisionID.Valid = false
// 		db.Save(&app)
//
// 		errs := app.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		app = Approver{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(app.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with an approved revision that has been cancelled (should not happen)", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
//
// 		approver := Approver{}
// 		approver.SetID(1)
// 		approver.Prepare(db)
// 		workingID := approver.PendingRevision.ID
// 		appr := approver.PendingRevision
// 		appr.Prepare(db)
// 		appr.RevisionState = StateApprovalFailed
// 		db.Save(&appr)
//
// 		app := Approval{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = StateApproved
// 		db.Save(&app)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(1)
// 		if err := db.First(&cr).Error; err != nil {
// 			t.Fatalf("error %s", err.Error())
// 		}
// 		cr.State = StateApproved
// 		db.Save(&cr)
//
// 		approver2 := Approver{}
// 		approver2.SetID(1)
// 		approver2.Prepare(db)
// 		standinAppr := ApproverRevision{}
// 		standinAppr.SetID(workingID)
// 		standinAppr.Prepare(db)
// 		approver2.PendingRevision = standinAppr
// 		errs := approver2.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		approver3 := Approver{}
// 		approver3.SetID(1)
// 		approver3.Prepare(db)
//
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(approver3.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with an approved revision that has been cancelled (should not happen)", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
//
// 		approver := Approver{}
// 		approver.SetID(1)
// 		approver.Prepare(db)
// 		appr := approver.CurrentRevision
// 		appr.Prepare(db)
// 		appr.RevisionState = "bogusstate"
// 		db.Save(&appr)
//
// 		app := Approval{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = StateApproved
// 		db.Save(&app)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(1)
// 		if err := db.First(&cr).Error; err != nil {
// 			t.Fatalf("error %s", err.Error())
// 		}
// 		cr.State = StateApproved
// 		db.Save(&cr)
//
// 		approver2 := Approver{}
// 		approver2.SetID(1)
// 		approver2.Prepare(db)
// 		errs := approver2.UpdateState(db,conf)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		approver3 := Approver{}
// 		approver3.SetID(1)
// 		approver3.Prepare(db)
//
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(approver3.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with an approved revision that has an invalid state (should not happen)", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
//
// 		approver := Approver{}
// 		approver.SetID(1)
// 		approver.Prepare(db)
// 		appr := approver.PendingRevision
// 		appr.Prepare(db)
// 		appr.DesiredState = "bogusstate"
// 		db.Save(&appr)
//
// 		app := Approval{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = StateApproved
// 		db.Save(&app)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(1)
// 		if err := db.First(&cr).Error; err != nil {
// 			t.Fatalf("error %s", err.Error())
// 		}
// 		cr.State = StateApproved
// 		db.Save(&cr)
//
// 		approver2 := Approver{}
// 		approver2.SetID(1)
// 		approver2.Prepare(db)
// 		errs := approver2.UpdateState(db,conf)
// 		fmt.Println(errs)
// 		Convey("An error should be returned", func() {
// 			So(errs, ShouldNotBeEmpty)
// 		})
//
// 		approver3 := Approver{}
// 		approver3.SetID(1)
// 		approver3.Prepare(db)
//
// 		Convey("The approver should remain in the PendingBootstrap state", func() {
// 			So(approver3.State, ShouldEqual, StatePendingBootstrap)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with an declined revision and a current revision not in bootstrap (should not happen)", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
//
// 		approver := Approver{}
// 		approver.SetID(1)
// 		approver.Prepare(db)
// 		appr := approver.CurrentRevision
// 		appr.Prepare(db)
// 		appr.RevisionState = StateActive
// 		db.Save(&appr)
//
// 		app := Approval{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = StateDeclined
// 		db.Save(&app)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(1)
// 		if err := db.First(&cr).Error; err != nil {
// 			t.Fatalf("error %s", err.Error())
// 		}
// 		cr.State = StateDeclined
// 		db.Save(&cr)
//
// 		approver2 := Approver{}
// 		approver2.SetID(1)
// 		approver2.Prepare(db)
// 		fmt.Println(approver2.CurrentRevisionID)
// 		fmt.Println(approver2.CurrentRevision.RevisionState)
// 		errs := approver2.UpdateState(db,conf)
// 		fmt.Println(errs)
// 		Convey("No error should be returned", func() {
// 			So(errs, ShouldBeEmpty)
// 		})
//
// 		approver3 := Approver{}
// 		approver3.SetID(1)
// 		approver3.Prepare(db)
//
// 		Convey("The approver should return to Active state", func() {
// 			So(approver3.State, ShouldEqual, StateActive)
// 		})
//
// 	})
//
// 	Convey("Given an approver in the PendingBootstrap state with an declined revision no current revision (should not happen)", t, func() {
//
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
// 		if err != nil {
// 			return
// 		}
//
// 		approver := Approver{}
// 		approver.SetID(1)
// 		approver.Prepare(db)
// 		approver.CurrentRevisionID.Valid = false
// 		db.Save(&approver)
//
// 		app := Approval{}
// 		app.SetID(1)
// 		app.Prepare(db)
// 		app.State = StateDeclined
// 		db.Save(&app)
//
// 		cr := ChangeRequest{}
// 		cr.SetID(1)
// 		if err := db.First(&cr).Error; err != nil {
// 			t.Fatalf("error %s", err.Error())
// 		}
// 		cr.State = StateDeclined
// 		cr.InitialRevisionID.Valid = false
// 		db.Save(&cr)
//
// 		approver2 := Approver{}
// 		approver2.SetID(1)
// 		approver2.Prepare(db)
// 		fmt.Println(approver2.CurrentRevisionID)
// 		fmt.Println(approver2.CurrentRevision.RevisionState)
// 		errs := approver2.UpdateState(db,conf)
// 		fmt.Println(errs)
// 		Convey("No error should be returned", func() {
// 			So(errs, ShouldBeEmpty)
// 		})
//
// 		approver3 := Approver{}
// 		approver3.SetID(1)
// 		approver3.Prepare(db)
//
// 		Convey("The approver should return to New state", func() {
// 			So(approver3.State, ShouldEqual, StateNew)
// 		})
//
// 	})
// }

func TestApproverVerifyCR(t *testing.T) {
	t.Parallel()
	Convey("Given an bootstrap database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		So(err, ShouldBeNil)

		approver := Approver{Model: Model{ID: 1}}
		prepareErr := approver.Prepare(dbCache)
		// Preparing the approver should not throw an error
		So(prepareErr, ShouldBeNil)

		approver.PendingRevision.ID = 0
		checksOut, errs := approver.VerifyCR(dbCache)
		// VerifyCR on an apporver with no pending revision should return an error
		So(checksOut, ShouldBeFalse)
		So(len(errs), ShouldEqual, 1)
	})

	Convey("Given an bootstrap database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverApproval)
		So(err, ShouldBeNil)

		approver := Approver{Model: Model{ID: 1}}
		prepareErr := approver.Prepare(dbCache)
		// Preparing the approver should not throw an error
		So(prepareErr, ShouldBeNil)

		approver.PendingRevision.CRID.Valid = false
		checksOut, errs := approver.VerifyCR(dbCache)
		// VerifyCR on an apporver with no CR should return an error
		So(checksOut, ShouldBeFalse)
		So(len(errs), ShouldEqual, 1)
	})
}

func TestApproverIsCanclled(t *testing.T) {
	t.Parallel()
	Convey("Given an approver in the 'new' state", t, func() {
		app := Approver{}
		app.State = StateNew
		// IsCancelled should return false
		So(app.IsCancelled(), ShouldBeFalse)
	})

	Convey("Given an approver in the 'cancelled' state", t, func() {
		app := Approver{}
		app.State = StateCancelled
		// IsCancelled should return true
		So(app.IsCancelled(), ShouldBeTrue)
	})
}

func TestApproverGetApproverSets(t *testing.T) {
	t.Parallel()
	Convey("Given a bootstrapped database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapDoneApproverSetApproval)
		So(err, ShouldBeNil)

		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldBeNil)
		approversets, err := app.GetActiveApproverSets(dbCache)
		// GetActiveApproverSets should not return an error and one approver set
		So(len(approversets), ShouldEqual, 1)
		So(err, ShouldBeNil)
	})

	Convey("Given an closed database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateEmpty)
		So(err, ShouldBeNil)

		dbCache.DB.Close()
		app := Approver{Model: Model{ID: 1}}
		err = app.Prepare(dbCache)
		So(err, ShouldNotBeNil)
		approversets, err := app.GetActiveApproverSets(dbCache)
		// GetActiveApproverSets should return an error and no approver sets
		So(len(approversets), ShouldEqual, 0)
		So(err, ShouldNotBeNil)
	})
}

func TestApproverPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverPage", t, func() {
		approvalPage := ApproverPage{}
		approvalPage.PendingRevisionPage = new(ApproverRevisionPage)
		// Calling SetCSRFToken should not panic
		So(func() {
			approvalPage.SetCSRFToken(testingCSRFToken)
		}, ShouldNotPanic)

		testTokenString := testingCSRFToken
		approvalPage.CSRFToken = testTokenString
		// Calling GetCSRFToken should return the testing token string
		So(approvalPage.GetCSRFToken(), ShouldEqual, testTokenString)
	})
}

func TestApproversPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproversPage", t, func() {
		approverPage := ApproversPage{}
		// Calling SetCSRFToken should not panic
		So(func() { approverPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)

		testTokenString := testingCSRFToken
		approverPage.CSRFToken = testTokenString
		// Calling GetCSRFToken should return the testing token string
		So(approverPage.GetCSRFToken(), ShouldEqual, testTokenString)
	})
}
