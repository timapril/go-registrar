package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type ContactBaseObj struct {
	rev ContactRevision
	obj Contact
}
type ContactBaseObjs struct {
	Approved, Pending, Declined ContactBaseObj
}

func TestContactHasRevision(t *testing.T) {
	t.Parallel()
	Convey("Given an Contact that does not have a revision", t, func() {
		as := Contact{}
		Convey("HasRevision should return false", func() {
			So(as.HasRevision(), ShouldBeFalse)
		})
	})

	Convey("Given an Approver Set that has a revision", t, func() {
		DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
			Convey("HasRevision should return false", func() {
				So(objs.Approved.obj.HasRevision(), ShouldBeTrue)
			})
		})
	})
}

func TestContactExportFullGetDiff(t *testing.T) {
	t.Parallel()
	Convey("Given an ContactExportFull object with an empty pending revision", t, func() {
		c := ContactExport{}
		c.CurrentRevision = ContactRevisionExport{}
		c.PendingRevision = ContactRevisionExport{ID: 0}

		_, err := c.GetDiff()

		Convey("GetDiff should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Given an ContactExportFull object with valid revisions", t, func() {
		DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
			exportver := objs.Pending.obj.GetExportVersion()
			diff, err := exportver.GetDiff()
			Convey("There should be a JSON string returned and no error", func() {
				So(len(diff), ShouldBeGreaterThan, 0)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestContactExportVersionAt(t *testing.T) {
	t.Parallel()
	Convey("Given an ContactExportFull object with valid revisions", t, func() {
		DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
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

func TestContactParseFromForm(t *testing.T) {
	t.Parallel()
	DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
		Convey("Given a HTTP request with no user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			contact := Contact{}
			parseError := contact.ParseFromForm(request, dbCache)
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})

		Convey("Given a HTTP request with a valid user set", t, func() {
			timeBefore := TimeNow()
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			request.Header.Add("REMOTE_USER", TestUser1Username)
			request.Form = make(url.Values)
			contact := Contact{}
			err := dbCache.Save(&contact)
			So(err, ShouldBeNil)
			parseError := contact.ParseFromForm(request, dbCache)
			timeAfter := TimeNow()
			Convey("No error should be returned", func() {
				So(parseError, ShouldBeNil)
				Convey("The resulting approver should have the expected values", func() {
					So(contact.CreatedBy, ShouldEqual, TestUser1Username)
					So(contact.UpdatedBy, ShouldEqual, TestUser1Username)
					So(contact.CreatedAt, ShouldHappenOnOrBetween, timeBefore, timeAfter)
					So(contact.UpdatedAt, ShouldHappenOnOrBetween, timeBefore, timeAfter)
					So(contact.State, ShouldEqual, StateNew)
				})
			})
		})
	})
}

func TestContactParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
		Convey("Given a HTTP request with no user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			contact := Contact{}
			parseError := contact.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})

		Convey("Given a HTTP request with a valid user set", t, func() {
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			request.Header.Add("REMOTE_USER", TestUser1Username)
			request.Form = make(url.Values)
			contact := Contact{}
			err := dbCache.Save(&contact)
			So(err, ShouldBeNil)
			parseError := contact.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})
	})
}

func TestContactExportToJSON(t *testing.T) {
	t.Parallel()
	DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
		export := objs.Approved.obj.GetExportVersion()
		Convey("Given a valid ContactExportFull", t, func() {
			exportStr1, exportErr1 := export.ToJSON()
			Convey("The error returned should be nil", func() {
				So(exportErr1, ShouldBeNil)
			})

			Convey("The length of the string returned should be greater than 0", func() {
				So(len(exportStr1), ShouldBeGreaterThan, 0)
			})
		})

		Convey("Given a valid ContactExportFull with its ID changed to 0", t, func() {
			typedExport := export.(ContactExport)
			typedExport.ID = 0
			_, exportErr2 := typedExport.ToJSON()
			Convey("The error returned should not be nil", func() {
				So(exportErr2, ShouldNotBeNil)
			})
		})

		Convey("Given a valid ContactExportFull with its ID change to -1", t, func() {
			typedExport := export.(ContactExport)
			typedExport.ID = -1
			_, exportErr2 := typedExport.ToJSON()
			Convey("The error returned should not be nil", func() {
				So(exportErr2, ShouldNotBeNil)
			})
		})
	})
}

func TestContactExportShortToJSON(t *testing.T) {
	t.Parallel()
	DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
		export := objs.Approved.obj.GetExportVersionShort()
		Convey("Given a valid ContactExportShort", t, func() {
			exportStr1, exportErr1 := json.MarshalIndent(export, "", "  ")
			Convey("The error returned should be nil", func() {
				So(exportErr1, ShouldBeNil)
			})

			Convey("The length of the string returned should be greater than 0", func() {
				So(len(exportStr1), ShouldBeGreaterThan, 0)
			})
		})
	})
}

func TestContactVerifyCR(t *testing.T) {
	t.Parallel()
	Convey("Given an bootstrap database", t, func() {
		DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
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

func TestContactSuggestedFields(t *testing.T) {
	t.Parallel()
	Convey("Given the default Contact", t, func() {
		DoContactDBTests(t, func(dbCache *DBCache, objs *ContactBaseObjs) {
			curRev := objs.Approved.rev
			tests := []struct{ name, fieldName, revFieldName, value string }{
				{"Name", ContactFieldName, "Name", curRev.Name},
				{"Org", ContactFieldOrg, "Org", curRev.Org},
				{"EmailAddress", ContactFieldEmail, "EmailAddress", curRev.EmailAddress},
				{"AddressStreet1", ContactFieldAddressStreet1, "AddressStreet1", curRev.AddressStreet1},
				{"AddressStreet2", ContactFieldAddressStreet2, "AddressStreet2", curRev.AddressStreet2},
				{"AddressStreet3", ContactFieldAddressStreet3, "AddressStreet3", curRev.AddressStreet3},
				{"AddressCity", ContactFieldAddressCity, "AddressCity", curRev.AddressCity},
				{"AddressState", ContactFieldAddressState, "AddressState", curRev.AddressState},
				{"AddressPostalCode", ContactFieldAddressPostalCode, "AddressPostalCode", curRev.AddressPostalCode},
				{"AddressCountry", ContactFieldAddressCountry, "AddressCountry", curRev.AddressCountry},
				{"VoicePhoneNumber", ContactFieldVoiceNumber, "VoicePhoneNumber", curRev.VoicePhoneNumber},
				{"VoicePhoneExtension", ContactFieldVoiceExtension, "VoicePhoneExtension", curRev.VoicePhoneExtension},
				{"FaxPhoneNumber", ContactFieldFaxNumber, "FaxPhoneNumber", curRev.FaxPhoneNumber},
				{"FaxPhoneExtension", ContactFieldFaxExtension, "FaxPhoneExtension", curRev.FaxPhoneExtension},
				{"FullAddress()", ContactFieldFullAddress, "FullAddress()", curRev.FullAddress()},
			}
			emptyContact := Contact{}
			for _, test := range tests {
				Convey(fmt.Sprintf("SuggestedRevisionValue(%s) should return an empty string when there is no current revision", test.name),
					func() {
						So(emptyContact.SuggestedRevisionValue(test.fieldName), ShouldBeBlank)
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
						So(emptyContact.GetCurrentValue(test.fieldName), ShouldEqual, UnPreparedContactError)
					})
			}
			for _, test := range tests {
				Convey(fmt.Sprintf("GetCurrentValue(%s) should be the same as the current revision's %s Field",
					test.name, test.revFieldName),
					func() {
						So(objs.Approved.obj.GetCurrentValue(test.fieldName), ShouldEqual, test.value)
					})
			}
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
				{"DesiredStateActive", DesiredStateActive, curRev.DesiredState == StateActive},
				{"DesiredStateInactive", DesiredStateInactive, curRev.DesiredState == StateInactive},
			}
			for _, test := range boolTests {
				Convey(fmt.Sprintf("SuggestedRevisionBool(%s) should return false", test.name),
					func() {
						So(emptyContact.SuggestedRevisionBool(test.fieldName), ShouldBeFalse)
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

func StartContactApproval(dbCache *DBCache, conf Config, name string) (rev ContactRevision, err error) {
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)

	Cont := Contact{}

	if err = dbCache.Save(&Cont); err != nil {
		return rev, err
	}

	rev.ContactID = Cont.ID
	rev.Name = name
	rev.Org = fmt.Sprintf("%s Org", name)

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

func SetupContactDB(dbCache *DBCache, _ Config) error {
	logger.Info("calling SetupContactDB")

	conf, err := getTestConf()
	if err != nil {
		return err
	}

	if err = BootstrapRegistrar(dbCache, conf); err != nil {
		return err
	}

	appRev, err := StartContactApproval(dbCache, conf, "Approved")
	if err != nil {
		return fmt.Errorf("Failed to start Approved user: %s", err.Error())
	}

	if err = ApproveApproval(dbCache, TestUser1Username, appRev.CR.Approvals[0].ID, conf); err != nil {
		return fmt.Errorf("Failed to approve Approved user: %s", err.Error())
	}

	appRev, err = StartContactApproval(dbCache, conf, "Declined")

	if err != nil {
		return fmt.Errorf("Failed to start Declined user: %s", err.Error())
	}

	if err = DeclineApproval(dbCache, TestUser1Username, appRev.CR.Approvals[0].ID, conf); err != nil {
		return fmt.Errorf("Failed to approve Declined user: %s", err.Error())
	}

	_, err = StartContactApproval(dbCache, conf, "Pending")

	if err != nil {
		return fmt.Errorf("Failed to start Pending user: %s", err.Error())
	}

	return nil
}

func GetContactDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()
	GetDBFunc(t, "ContactDB", GetBootstrappedDB, SetupContactDB, roFunc, mustGetTestConf())
}

func (obj *ContactBaseObj) Load(dbCache *DBCache, name string) (err error) {
	dbCache.DB = dbCache.DB.Debug()
	if err = dbCache.DB.Where("name = ?", name).First(&obj.rev).Error; err != nil {
		return
	}

	if err = obj.rev.Prepare(dbCache); err != nil {
		return
	}

	if err = dbCache.DB.First(&obj.obj, obj.rev.ContactID).Error; err != nil {
		return
	}

	if err = obj.obj.Prepare(dbCache); err != nil {
		return
	}

	return nil
}

func DoContactDBTests(t *testing.T, testBlock func(*DBCache, *ContactBaseObjs)) {
	t.Helper()

	dbCache, err := DBFactory.GetDB(t, TestStateBootstrappedWithContact)
	if err != nil {
		panic(err)
	}

	objs := ContactBaseObjs{}

	if err := objs.Approved.Load(dbCache, "Approved"); err != nil {
		t.Fatalf("fetching a API User records threw an error: %s", err.Error())
	}

	if err := objs.Declined.Load(dbCache, "Declined"); err != nil {
		t.Fatalf("fetching a API User records threw an error: %s", err.Error())
	}

	if err := objs.Pending.Load(dbCache, "Pending"); err != nil {
		t.Fatalf("fetching a API User records threw an error: %s", err.Error())
	}

	testBlock(dbCache, &objs)
}
