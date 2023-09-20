package lib

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp/packet"
	. "github.com/smartystreets/goconvey/convey"
)

func TestApproverSetExportFullGetDiff(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverSetExportFull object with an empty pending revision", t, func() {
		approverSetExport := ApproverSetExportFull{}
		approverSetExport.CurrentRevision = ApproverSetRevisionExport{}
		approverSetExport.PendingRevision = ApproverSetRevisionExport{ID: 0}

		_, err := approverSetExport.GetDiff()

		// GetDiff should return an error
		So(err, ShouldNotBeNil)
	})

	Convey("Given an ApproverSetExportFull object with valid revisions", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr1 := approverSet.Prepare(dbCache)
		// No error should be thrown when preparing the approver set export full
		So(prepareErr1, ShouldBeNil)

		exportver := approverSet.GetExportVersion()
		diff, err := exportver.GetDiff()
		// There should be a JSON string returned and no error
		So(len(diff), ShouldBeGreaterThan, 0)
		So(err, ShouldBeNil)
	})
}

func TestApproverSetExportFullToJSON(t *testing.T) {
	t.Parallel()

	Convey("Given an initial bootstrap database, test the approver set export full to json method", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		approverSet := ApproverSet{}
		err = approverSet.SetID(1)
		So(err, ShouldBeNil)
		err = approverSet.Prepare(dbCache)
		So(err, ShouldBeNil)

		export := approverSet.GetExportVersion()

		Convey("Given a valid ApproverSetExportFull", func() {
			exportStr1, exportErr1 := export.ToJSON()
			Convey("The error returned should be nil", func() {
				So(exportErr1, ShouldBeNil)
			})

			Convey("The length of the string returned should be greater than 0", func() {
				So(len(exportStr1), ShouldBeGreaterThan, 0)
			})
		})

		Convey("Given a valid ApproverSetExportFull with its ID changed to 0", func() {
			typedExport := export.(ApproverSetExportFull)
			typedExport.ID = 0
			_, exportErr2 := typedExport.ToJSON()
			Convey("The error returned should not be nil", func() {
				So(exportErr2, ShouldNotBeNil)
			})
		})

		Convey("Given a valid ApproverSetExportFull with its ID change to -1", func() {
			typedExport := export.(ApproverSetExportFull)
			typedExport.ID = -1
			_, exportErr2 := typedExport.ToJSON()
			Convey("The error returned should not be nil", func() {
				So(exportErr2, ShouldNotBeNil)
			})
		})
	})
}

func TestApproverSetHasPendingRevision(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Set that does not have a revision", t, func() {
		approverSet := ApproverSet{}
		// HasPendingRevision should return false
		So(approverSet.HasPendingRevision(), ShouldBeFalse)
	})

	Convey("Given an Approver Set that has a revision", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		So(err, ShouldBeNil)

		approverSet := ApproverSet{}
		err = approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr1 := approverSet.Prepare(dbCache)
		// No error should be thrown when preparing the approver set
		So(prepareErr1, ShouldBeNil)

		// HasPendingRevision should return false
		So(approverSet.HasPendingRevision(), ShouldBeTrue)
	})
}

func TestApproverSetHasRevision(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Set that does not have a revision", t, func() {
		approverSet := ApproverSet{}
		// HasRevision should return false
		So(approverSet.HasRevision(), ShouldBeFalse)
	})

	Convey("Given an Approver Set that has a revision", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrap)
		if err != nil {
			return
		}
		approverSet := ApproverSet{}
		err = approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr1 := approverSet.Prepare(dbCache)
		// No error should be thrown when preparing the approver set
		So(prepareErr1, ShouldBeNil)

		// HasRevision should return false
		So(approverSet.HasRevision(), ShouldBeTrue)
	})
}

func TestApproverSetParseFromForm(t *testing.T) {
	t.Parallel()
	Convey("Given a HTTP request with no user set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		app := ApproverSet{}
		parseError := app.ParseFromForm(r, dbCache)
		// An error should be returned
		So(parseError, ShouldNotBeNil)
	})

	Convey("Given a HTTP request with a valid username and all fields filled in", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		r.Form = make(url.Values)
		apps := ApproverSet{}
		perr := apps.ParseFromForm(r, dbCache)
		// ParseFromForm should return an error
		So(perr, ShouldBeNil)

		// The resulting approver should have the expected values
		So(apps.CreatedBy, ShouldEqual, TestUser1Username)
		So(apps.UpdatedBy, ShouldEqual, TestUser1Username)
		So(apps.State, ShouldEqual, StateNew)
	})
}

func TestApproverSetParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	Convey("Given ParseFromFormUpdate can run", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateEmpty)
		if err != nil {
			return
		}
		app := ApproverSet{}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

		err2 := app.ParseFromFormUpdate(request, dbCache, mustGetTestConf())

		// An error should be returned
		So(err2, ShouldNotBeNil)
	})
}

// func TestApproverSetPostUpdate(t *testing.T) {
// 	Convey("Given an approver set object", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateEmpty)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
//
// 		as := ApproverSet{}
// 		Convey("Calling PostUpdate should not panic", func() {
// 			So(func() { as.PostUpdate(db) }, ShouldNotPanic)
// 		})
// 	})
// }

func TestApproverSetPrepare(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set without a valid ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		approverSet.ID = 0
		prepareErr := approverSet.Prepare(dbCache)
		// Prepare should return an error
		So(prepareErr, ShouldNotBeNil)
	})
}

func TestApproverSetPrepareShallow(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set without a valid ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		approverSet.ID = 0
		prepareErr := approverSet.PrepareShallow(dbCache)
		// PrepareShallow should return an error
		So(prepareErr, ShouldNotBeNil)
	})

	Convey("Given an approver set with a valid ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		Convey("Getting the database should not throw an error", func() {
			So(dberr, ShouldBeNil)
		})
		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.PrepareShallow(dbCache)

		// PrepareShallow should not return an error
		So(prepareErr, ShouldBeNil)
	})

	Convey("Given an approver set with a non-zero invalid ID", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(2)
		So(err, ShouldBeNil)
		prepareErr := approverSet.PrepareShallow(dbCache)

		// PrepareShallow should return an error
		So(prepareErr, ShouldNotBeNil)
	})

	Convey("Given an approver set with a valid ID that has been prepared", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr1 := approverSet.PrepareShallow(dbCache)
		// PrepareShallow should not return an error the first tim
		So(prepareErr1, ShouldBeNil)

		prepareErr2 := approverSet.PrepareShallow(dbCache)
		// PrepareShallow should not return an error the second time
		So(prepareErr2, ShouldBeNil)
	})
}

func TestApproverSetSetID(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Set", t, func() {
		approverSet := ApproverSet{}
		setIDerr1 := approverSet.SetID(0)
		// SetID of 0 should return an error
		So(setIDerr1, ShouldNotBeNil)
	})

	Convey("Given an Approver Set", t, func() {
		approverSet := ApproverSet{}
		setIDerr1 := approverSet.SetID(-1)
		// SetID of -1 should return an error
		So(setIDerr1, ShouldNotBeNil)
	})

	Convey("Given an Approver Set", t, func() {
		approverSet := ApproverSet{}
		setIDerr1 := approverSet.SetID(1)

		// SetID of 1 should not return an erro
		So(setIDerr1, ShouldBeNil)

		setIDerr2 := approverSet.SetID(1)
		// SetID of 1 again should return an error
		So(setIDerr2, ShouldNotBeNil)
	})
}

func TestApproverSetIsEditable(t *testing.T) {
	t.Parallel()
	Convey("Given an Approver Set in the 'new' state", t, func() {
		approverSet := ApproverSet{}
		approverSet.State = StateNew
		editable := approverSet.IsEditable()

		// IsEditable should return true
		So(editable, ShouldBeTrue)
	})

	Convey("Given an Approver Set in the 'pendingapproval' state", t, func() {
		approverSet := ApproverSet{}
		approverSet.State = StatePendingApproval
		editable := approverSet.IsEditable()
		// IsEditable should return false
		So(editable, ShouldBeFalse)
	})

	Convey("Given an Approver Set in the 'bogus' state", t, func() {
		approverSet := ApproverSet{}
		approverSet.State = bogusState
		editable := approverSet.IsEditable()
		// IsEditable should return false
		So(editable, ShouldBeFalse)
	})
}

func TestApproverSetGetPage(t *testing.T) {
	t.Parallel()
	Convey("Given an existing approver set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		So(prepareErr, ShouldBeNil)

		untypedObj, err := approverSet.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedObj := untypedObj.(*ApproverSetPage)
		// The approver set page returned should be populated correctly
		So(typedObj.Editable, ShouldBeFalse)
		So(typedObj.IsNew, ShouldBeFalse)
		So(typedObj.AppS, ShouldNotBeNil)
		So(typedObj.AppS.prepared, ShouldBeTrue)
		So(typedObj.AppS.ID, ShouldEqual, 1)
		So(len(typedObj.PendingActions), ShouldEqual, 4)
	})

	Convey("Given an existing approver set with no current revision", t, func() {
		ddbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(ddbCache)
		So(prepareErr, ShouldBeNil)

		approverSet.CurrentRevisionID.Valid = false
		approverSet.CurrentRevision.ID = 0

		untypedObj, err := approverSet.GetPage(ddbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedObj := untypedObj.(*ApproverSetPage)
		// The approver set page returned should be populated correctly
		So(typedObj.AppS.CurrentRevisionID.Valid, ShouldBeFalse)
		So(typedObj.Editable, ShouldBeFalse)
		So(typedObj.IsNew, ShouldBeFalse)
		So(typedObj.AppS, ShouldNotBeNil)
		So(typedObj.AppS.prepared, ShouldBeTrue)
		So(typedObj.AppS.ID, ShouldEqual, 1)
		So(len(typedObj.PendingActions), ShouldEqual, 4)
	})

	Convey("Given an existing approver set with a current revision with both informed and required approver sets", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		So(prepareErr, ShouldBeNil)

		approverSet.CurrentRevision.InformedApproverSets = append(approverSet.CurrentRevision.InformedApproverSets, approverSet)

		err = UpdateApproverSets(&approverSet.CurrentRevision, dbCache, "InformedApproverSets", approverSet.CurrentRevision.InformedApproverSets)
		So(err, ShouldBeNil)

		err = dbCache.Save(&approverSet)
		So(err, ShouldBeNil)

		approverSet2 := ApproverSet{}
		err = approverSet2.SetID(1)
		So(err, ShouldBeNil)
		prepareErr2 := approverSet2.Prepare(dbCache)
		So(prepareErr2, ShouldBeNil)

		untypedObj, err := approverSet2.GetPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedObj := untypedObj.(*ApproverSetPage)
		// The approver set page returned should be populated correctly
		So(typedObj.AppS.CurrentRevisionID.Valid, ShouldBeTrue)
		So(typedObj.Editable, ShouldBeFalse)
		So(typedObj.IsNew, ShouldBeFalse)
		So(typedObj.AppS, ShouldNotBeNil)
		So(typedObj.AppS.prepared, ShouldBeTrue)
		So(typedObj.AppS.ID, ShouldEqual, 1)
		So(len(typedObj.PendingActions), ShouldEqual, 4)
	})
}

func TestApproverSetGetAllPage(t *testing.T) {
	t.Parallel()
	Convey("Given an empty approver set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		untypedPage, err := approverSet.GetAllPage(dbCache, TestUser1Username, TestUser1Username+ExampleUserOrg)
		So(err, ShouldBeNil)

		typedPage := untypedPage.(*ApproverSetsPage)
		// The page object returned should have at least one approver set
		So(len(typedPage.ApproverSets), ShouldBeGreaterThan, 0)
	})
}

func TestApproverSetGetRequiredApproverSets(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set with a required approver set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		So(prepareErr, ShouldBeNil)
		asList, err := approverSet.GetRequiredApproverSets(dbCache)
		// GetRequiredApproverSets should return a list of approver sets and no error
		So(err, ShouldBeNil)
		So(len(asList), ShouldBeGreaterThan, 0)
	})

	Convey("Given an approver set without a required approver set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapped)
		So(dberr, ShouldBeNil)

		err := dbCache.DB.Exec("delete from required_approverset_to_approversetrevision where approver_set_revision_id = 2").Error
		So(err, ShouldBeNil)

		approverSet := ApproverSet{}
		err = approverSet.SetID(1)
		So(err, ShouldBeNil)

		asList, err := approverSet.GetRequiredApproverSets(dbCache)
		// GetRequiredApproverSets should return a list of approver sets with a length of 1 and an error
		So(err, ShouldNotBeNil)
		So(len(asList), ShouldEqual, 1)
	})
}

func TestApproverSetGetInformedApproverSets(t *testing.T) {
	t.Parallel()
	Convey("Given an existing approver set that has an current revision", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		// Prepare should not return an error
		So(prepareErr, ShouldBeNil)

		asList, err := approverSet.GetInformedApproverSets(dbCache)
		// GetInformedApproverSets should return an empty list of approver sets and no error
		So(err, ShouldBeNil)
		So(asList, ShouldBeEmpty)
	})

	Convey("Given an approver set without a current revision", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(2)
		So(err, ShouldBeNil)

		asList, err := approverSet.GetInformedApproverSets(dbCache)
		// GetInformedApproverSets should return an empty list of approver sets and an erro
		So(err, ShouldNotBeNil)
		So(len(asList), ShouldEqual, 0)
	})
}

func TestApproverSetTakeAction(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set revision", t, func() {
		conf := mustGetTestConf()

		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		r.Header.Add("REMOTE_USER", TestUser1Username)
		w := httptest.NewRecorder()
		// Calling TakeAction should not panic
		So(func() { approverSet.TakeAction(w, r, dbCache, "", false, RemoteUserAuthType, conf) }, ShouldNotPanic)
	})
}

func TestApproverSetApproverFromIdentityName(t *testing.T) {
	t.Parallel()
	Convey("Given an empty approver set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateEmpty)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		app, err := approverSet.ApproverFromIdentityName("test", dbCache)
		// ApproverFromIdentityName called on a new Approver Set should return an error and no approver
		So(err, ShouldNotBeNil)
		So(app.ID, ShouldEqual, 0)
	})
}

func TestApproverSetPrepareGPGKeys(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set with an approver with an inproperly formatted public key", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrap)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		err = approverSet.Prepare(dbCache)
		So(err, ShouldBeNil)
		approverSet.CurrentRevision.Approvers[0].CurrentRevisionID.Valid = false
		err = approverSet.PrepareGPGKeys(dbCache)
		// No error should be thrown when calling PrepareGPGKeys
		So(err, ShouldBeNil)
	})
}

func TestApproverSetDecryptionKeys(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set", t, func() {
		approverSet := ApproverSet{}
		keys := approverSet.DecryptionKeys()
		// Should return an empty list of keys
		So(len(keys), ShouldEqual, 0)
	})
}

func TestApproverSetKeysById(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set with a valid approver", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		// Prepare should not return an error
		So(prepareErr, ShouldBeNil)

		gpgPrepErr := approverSet.PrepareGPGKeys(dbCache)
		// PrepareGPGKeys should not return an error
		So(gpgPrepErr, ShouldBeNil)

		var lastKey uint64
		var lastKeyIndex int
		for idx, key := range approverSet.Keys {
			lastKey = key.PrimaryKey.KeyId
			lastKeyIndex = idx
		}
		keys := approverSet.KeysById(lastKey)
		// KeysById should return 1 key - test 1
		So(len(keys), ShouldEqual, 1)

		var keyName string
		for name := range approverSet.Keys[lastKeyIndex].Identities {
			keyName = name
		}

		trueVal := true

		approverSet.Keys[lastKeyIndex].Identities["test"] = approverSet.Keys[lastKeyIndex].Identities[keyName]
		approverSet.Keys[lastKeyIndex].Identities["test"].SelfSignature.IsPrimaryId = &trueVal
		keys2 := approverSet.KeysById(lastKey)
		// KeysById should return 1 key - test 2
		So(len(keys2), ShouldEqual, 1)

		approverSet.Keys[lastKeyIndex].Subkeys[0].PublicKey.KeyId = lastKey
		keys3 := approverSet.KeysById(lastKey)
		// KeysById should return 1 key - test 3
		So(len(keys3), ShouldEqual, 2)
	})
}

func TestApproverSetKeysByIdUsage(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set with a valid approver with a revocation", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		// Prepare should not return an error
		So(prepareErr, ShouldBeNil)

		gpgPrepErr := approverSet.PrepareGPGKeys(dbCache)
		// PrepareGPGKeys should not return an error
		So(gpgPrepErr, ShouldBeNil)

		var lastKey uint64
		var lastKeyIndex int
		for idx, key := range approverSet.Keys {
			lastKey = key.PrimaryKey.KeyId
			lastKeyIndex = idx
		}
		var test byte
		revocation := packet.Signature{}
		approverSet.Keys[lastKeyIndex].Revocations = append(approverSet.Keys[lastKeyIndex].Revocations, &revocation)
		testKeys := approverSet.KeysByIdUsage(lastKey, test)
		// KeysByIdUsage should return 0 keys
		So(len(testKeys), ShouldEqual, 0)
	})

	Convey("Given an approver set with a valid approver with a revocation reason", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		// Prepare should not return an error
		So(prepareErr, ShouldBeNil)

		gpgPrepErr := approverSet.PrepareGPGKeys(dbCache)
		// PrepareGPGKeys should not return an error
		So(gpgPrepErr, ShouldBeNil)

		var lastKey uint64
		var lastKeyIndex int
		for idx, key := range approverSet.Keys {
			lastKey = key.PrimaryKey.KeyId
			lastKeyIndex = idx
		}
		var keyName string
		for name := range approverSet.Keys[lastKeyIndex].Identities {
			keyName = name
		}

		reason := packet.KeySuperseded
		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.RevocationReason = &reason

		var test byte
		testKeys := approverSet.KeysByIdUsage(lastKey, test)
		// KeysByIdUsage should return 0 keys
		So(len(testKeys), ShouldEqual, 0)
	})

	Convey("Given an approver set with a valid approver with a key with all flags set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		// Prepare should not return an error
		So(prepareErr, ShouldBeNil)

		gpgPrepErr := approverSet.PrepareGPGKeys(dbCache)
		// PrepareGPGKeys should not return an error
		So(gpgPrepErr, ShouldBeNil)

		var lastKey uint64
		var lastKeyIndex int
		for idx, key := range approverSet.Keys {
			lastKey = key.PrimaryKey.KeyId
			lastKeyIndex = idx
		}
		var keyName string
		for name := range approverSet.Keys[lastKeyIndex].Identities {
			keyName = name
		}
		var test byte = packet.KeyFlagCertify | packet.KeyFlagSign | packet.KeyFlagEncryptCommunications | packet.KeyFlagEncryptStorage

		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.FlagCertify = true
		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.FlagSign = true
		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.FlagEncryptCommunications = true
		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.FlagEncryptStorage = true

		testKeys := approverSet.KeysByIdUsage(lastKey, test)
		// KeysByIdUsage should return 1 keys
		So(len(testKeys), ShouldEqual, 1)
	})

	Convey("Given an approver set with a valid approver with a key with all flags set and incorrect usage set", t, func() {
		dbCache, dberr := DBFactory.GetDB(t, TestStateBootstrapDoneApproverApproval)
		So(dberr, ShouldBeNil)

		approverSet := ApproverSet{}
		err := approverSet.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverSet.Prepare(dbCache)
		// Prepare should not return an error
		So(prepareErr, ShouldBeNil)

		gpgPrepErr := approverSet.PrepareGPGKeys(dbCache)
		// PrepareGPGKeys should not return an error
		So(gpgPrepErr, ShouldBeNil)

		var lastKey uint64
		var lastKeyIndex int
		for idx, key := range approverSet.Keys {
			lastKey = key.PrimaryKey.KeyId
			lastKeyIndex = idx
		}
		var keyName string
		for name := range approverSet.Keys[lastKeyIndex].Identities {
			keyName = name
		}
		var test byte = packet.KeyFlagCertify | packet.KeyFlagSign | packet.KeyFlagEncryptCommunications | packet.KeyFlagEncryptStorage

		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.FlagCertify = false
		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.FlagSign = false
		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.FlagEncryptCommunications = true
		approverSet.Keys[lastKeyIndex].Identities[keyName].SelfSignature.FlagEncryptStorage = true

		testKeys := approverSet.KeysByIdUsage(lastKey, test)
		/// KeysByIdUsage should return 0 keys
		So(len(testKeys), ShouldEqual, 0)
	})
}

// func TestApproverSetUpdateState(t *testing.T) {
// 	Convey("Given an approver set in pending bootstrap with no pending revision (cancelled revision)", t, func() {
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		if err != nil {
// 			return
// 		}
// 		apps := ApproverSet{}
// 		apps.SetID(1)
// 		apps.PrepareShallow(db)
// 		appsRev := ApproverSetRevision{}
// 		appsRev.SetID(apps.PendingRevision.ID)
// 		appsRev.PrepareShallow(db)
// 		errs := appsRev.Cancel(db)
// 		So(len(errs), ShouldEqual, 0)
// 		if len(errs) != 0 {
// 			fmt.Println(errs)
// 		}
//
// 		apps = ApproverSet{}
// 		apps.SetID(1)
// 		apps.Prepare(db)
// 		Convey("The approver set should transition to in the PendingBootstrap state", func() {
// 			So(apps.State, ShouldEqual, StateBootstrap)
// 			So(apps.PendingRevision.ID, ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver set in the active state with no pending revision (cancelled revision)", t, func() {
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		if err != nil {
// 			return
// 		}
// 		apps := ApproverSet{}
// 		apps.SetID(1)
// 		apps.PrepareShallow(db)
// 		apps.State = StateActivePendingApproval
// 		db.Save(&apps)
//
// 		appsRev1 := ApproverSetRevision{}
// 		appsRev1.SetID(apps.CurrentRevisionID.Int64)
// 		appsRev1.PrepareShallow(db)
// 		appsRev1.DesiredState = StateActive
// 		db.Save(&appsRev1)
//
// 		appsRev2 := ApproverSetRevision{}
// 		appsRev2.SetID(apps.PendingRevision.ID)
// 		appsRev2.PrepareShallow(db)
// 		errs := appsRev2.Cancel(db)
// 		So(len(errs), ShouldEqual, 0)
//
// 		apps = ApproverSet{}
// 		apps.SetID(1)
// 		apps.Prepare(db)
// 		Convey("The approver set should transition to in the PendingBootstrap state", func() {
// 			So(apps.State, ShouldEqual, StateActive)
// 			So(apps.PendingRevision.ID, ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pending new with no pending or current revision (cancelled revision)", t, func() {
// 		db, err  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		if err != nil {
// 			return
// 		}
// 		apps := ApproverSet{}
// 		apps.SetID(1)
// 		apps.PrepareShallow(db)
// 		apps.State = StatePendingNew
// 		apps.CurrentRevisionID.Valid = false
// 		db.Save(&apps)
//
// 		appsRev := ApproverSetRevision{}
// 		appsRev.SetID(apps.PendingRevision.ID)
// 		appsRev.Prepare(db)
// 		errs := appsRev.Cancel(db)
// 		So(len(errs), ShouldEqual, 0)
//
// 		apps = ApproverSet{}
// 		apps.SetID(1)
// 		apps.Prepare(db)
// 		Convey("The approver set should transition to in the PendingBootstrap state", func() {
// 			So(apps.State, ShouldEqual, StateNew)
// 			So(apps.PendingRevision.ID, ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (wrong object id, object type, revision id, initial revision id)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(as.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.RegistrarObjectID = 10
// 		cr.RegistrarObjectType = ApprovalType
// 		cr.ProposedRevisionID = 10
// 		cr.InitialRevisionID.Int64 = 10
// 		db.Save(&cr)
//
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should return errors", func() {
// 			So(len(errs), ShouldEqual, 4)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (no initial revision id)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(as.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.InitialRevisionID.Valid = false
// 		db.Save(&cr)
//
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should return errors", func() {
// 			So(len(errs), ShouldEqual, 1)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (current revision is invalid)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
// 		as.CurrentRevisionID.Valid = false
//
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should return errors", func() {
// 			So(len(errs), ShouldEqual, 1)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (CR Approved)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(as.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.State = StateApproved
// 		db.Save(&cr)
//
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should not return errors", func() {
// 			So(len(errs), ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (CR Approved)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(as.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.State = StateApproved
// 		db.Save(&cr)
//
// 		asr := ApproverSetRevision{}
// 		asr.SetID(as.PendingRevision.ID)
// 		asr.Prepare(db)
// 		asr.RevisionState = StateCancelled
// 		db.Save(&asr)
//
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should return errors", func() {
// 			So(len(errs), ShouldEqual, 1)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (CR Approved - Can't Supersed revision)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(as.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.State = StateApproved
// 		db.Save(&cr)
//
// 		asr := ApproverSetRevision{}
// 		asr.SetID(as.CurrentRevision.ID)
// 		asr.Prepare(db)
// 		asr.RevisionState = StatePendingApproval
// 		db.Save(&asr)
//
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should return errors", func() {
// 			So(len(errs), ShouldEqual, 1)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (CR Approved - Invalid Target State)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(as.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.State = StateApproved
// 		db.Save(&cr)
//
// 		as.PendingRevision.DesiredState = bogusState
//
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should return errors", func() {
// 			So(len(errs), ShouldEqual, 1)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (CR Declined)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(as.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.State = StateDeclined
// 		db.Save(&cr)
//
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should not return errors", func() {
// 			So(len(errs), ShouldEqual, 0)
// 		})
// 	})
//
// 	Convey("Given an approver set in the pendingbootstrap state (CR Declined - No Current Revision)", t, func() {
// 		db, dberr  := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
// 		Convey("Getting the database should not throw an error", func() {
// 			So(dberr, ShouldBeNil)
// 		})
// 		as := ApproverSet{}
// 		as.SetID(1)
// 		prepareErr := as.Prepare(db)
// 		Convey("Prepare should not return an error", func() {
// 			So(prepareErr, ShouldBeNil)
// 		})
//
// 		cr := ChangeRequest{}
// 		cr.SetID(as.PendingRevision.CRID.Int64)
// 		cr.Prepare(db)
// 		cr.State = StateDeclined
// 		cr.InitialRevisionID.Valid = false
// 		db.Save(&cr)
//
// 		as.CurrentRevisionID.Valid = false
// 		errs := as.UpdateState(db,conf)
// 		t.Log(as.State)
// 		Convey("UpdateState should not return errors", func() {
// 			So(len(errs), ShouldEqual, 0)
// 		})
// 	})
// }

func TestApproverSetIsCanclled(t *testing.T) {
	t.Parallel()
	Convey("Given an approver set in the 'new' state", t, func() {
		approverSet := ApproverSet{}
		approverSet.State = StateNew
		Convey("IsCancelled should return false", func() {
			So(approverSet.IsCancelled(), ShouldBeFalse)
		})
	})

	Convey("Given an approver set in the 'cancelled' state", t, func() {
		approverSet := ApproverSet{}
		approverSet.State = StateCancelled
		Convey("IsCancelled should return true", func() {
			So(approverSet.IsCancelled(), ShouldBeTrue)
		})
	})
}

func TestApproverSetVerifyCR(t *testing.T) {
	t.Parallel()
	Convey("Given an bootstrap database", t, func() {
		dbCache, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		if err != nil {
			return
		}
		approverset := ApproverSet{}
		err = approverset.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverset.Prepare(dbCache)
		Convey("Preparing the approver set should not throw an error", func() {
			So(prepareErr, ShouldBeNil)
		})
		approverset.PendingRevision.ID = 0
		checksOut, errs := approverset.VerifyCR(dbCache)
		Convey("VerifyCR on an apporver set with no pending revision should return an error", func() {
			So(checksOut, ShouldBeFalse)
			So(len(errs), ShouldEqual, 1)
		})
	})

	Convey("Given an bootstrap database", t, func() {
		dbCacheb, err := DBFactory.GetDB(t, TestStateBootstrapStartApproverSetApproval)
		if err != nil {
			return
		}
		approverset := ApproverSet{}
		err = approverset.SetID(1)
		So(err, ShouldBeNil)
		prepareErr := approverset.Prepare(dbCacheb)
		Convey("Preparing the approver set should not throw an error", func() {
			So(prepareErr, ShouldBeNil)
		})
		approverset.PendingRevision.CRID.Valid = false
		checksOut, errs := approverset.VerifyCR(dbCacheb)
		Convey("VerifyCR on an apporver set with no CR should return an error", func() {
			So(checksOut, ShouldBeFalse)
			So(len(errs), ShouldEqual, 1)
		})
	})
}

func TestApproverSetPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverSetPage", t, func() {
		approverSetPage := ApproverSetPage{}
		approverSetPage.PendingRevisionPage = new(ApproverSetRevisionPage)
		Convey("Calling SetCSRFToken should not panic", func() {
			So(func() { approverSetPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)
		})

		testTokenString := testingCSRFToken
		approverSetPage.CSRFToken = testTokenString
		Convey("Calling GetCSRFToken should return the testing token string", func() {
			So(approverSetPage.GetCSRFToken(), ShouldEqual, testTokenString)
		})
	})
}

func TestApproverSetsPageCSRFTest(t *testing.T) {
	t.Parallel()
	Convey("Given an ApproverSetsPage", t, func() {
		approverSetPage := ApproverSetsPage{}
		Convey("Calling SetCSRFToken should not panic", func() {
			So(func() { approverSetPage.SetCSRFToken(testingCSRFToken) }, ShouldNotPanic)
		})

		testTokenString := testingCSRFToken
		approverSetPage.CSRFToken = testTokenString
		Convey("Calling GetCSRFToken should return the testing token string", func() {
			So(approverSetPage.GetCSRFToken(), ShouldEqual, testTokenString)
		})
	})
}
