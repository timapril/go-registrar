package lib

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type APIUserBaseObj struct {
	rev  APIUserRevision
	obj  APIUser
	cert string
}
type APIUserBaseObjs struct {
	Approved, Pending, Declined APIUserBaseObj
}

func TestAPIUserHasRevision(t *testing.T) {
	t.Parallel()
	Convey("Given an APIUser that does not have a revision", t, func() {
		as := APIUser{}
		Convey("HasRevision should return false", func() {
			So(as.HasRevision(), ShouldBeFalse)
		})
	})

	Convey("Given an API User that has a revision", t, func() {
		DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
			Convey("HasRevision should return false", func() {
				So(objs.Approved.obj.HasRevision(), ShouldBeTrue)
			})
		})
	})
}

func TestAPIUserExportFullGetDiff(t *testing.T) {
	t.Parallel()
	Convey("Given an APIUserExportFull object with an empty pending revision", t, func() {
		a := APIUserExportFull{}
		a.CurrentRevision = APIUserRevisionExport{}
		a.PendingRevision = APIUserRevisionExport{ID: 0}

		_, err := a.GetDiff()

		Convey("GetDiff should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Given an APIUserExportFull object with valid revisions", t, func() {
		DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
			exportver := objs.Pending.obj.GetExportVersion()
			diff, err := exportver.GetDiff()
			Convey("There should be a JSON string returned and no error", func() {
				So(len(diff), ShouldBeGreaterThan, 0)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestAPIUserExportVersionAt(t *testing.T) {
	t.Parallel()
	Convey("Given an APIUserExportFull object with valid revisions", t, func() {
		DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
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
				So(rev, ShouldNotBeNil)
				jo, _ := json.MarshalIndent(rev, "", "  ")
				t.Logf("%s", jo)
				So(rev.PromotedTime, ShouldNotBeNil)

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

func TestAPIUserExportFullToJSON(t *testing.T) {
	t.Parallel()
	DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
		export := objs.Approved.obj.GetExportVersion()
		Convey("Given a valid APIUserExportFull", t, func() {
			exportStr1, exportErr1 := export.ToJSON()
			Convey("The error returned should be nil", func() {
				So(exportErr1, ShouldBeNil)
			})

			Convey("The length of the string returned should be greater than 0", func() {
				So(len(exportStr1), ShouldBeGreaterThan, 0)
			})
		})

		Convey("Given a valid APIUserExportFull with its ID changed to 0", t, func() {
			typedExport := export.(APIUserExportFull)
			typedExport.ID = 0
			_, exportErr2 := typedExport.ToJSON()
			Convey("The error returned should not be nil", func() {
				So(exportErr2, ShouldNotBeNil)
			})
		})

		Convey("Given a valid APIUserExportFull with its ID change to -1", t, func() {
			typedExport := export.(APIUserExportFull)
			typedExport.ID = -1
			_, exportErr2 := typedExport.ToJSON()
			Convey("The error returned should not be nil", func() {
				So(exportErr2, ShouldNotBeNil)
			})
		})
	})
}

func TestAPIUserExportShortToJSON(t *testing.T) {
	t.Parallel()
	DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
		export := objs.Approved.obj.GetExportShortVersion()
		Convey("Given a valid APIUserExportShort", t, func() {
			exportStr1, exportErr1 := export.ToJSON()
			Convey("The error returned should be nil", func() {
				So(exportErr1, ShouldBeNil)
			})

			Convey("The length of the string returned should be greater than 0", func() {
				So(len(exportStr1), ShouldBeGreaterThan, 0)
			})
		})
	})
}

func TestBasicAPIUserUpdateState(t *testing.T) {
	t.Parallel()

	conf := mustGetTestConf()

	Convey("Given an approved obj", t, func() {
		DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
			obj := objs.Approved.obj
			changed, errs := obj.UpdateState(dbCache, conf)
			Convey("UpdateState with ActiveState should change nothing and have no errors", func() {
				So(changed, ShouldBeFalse)
				So(errs, ShouldBeEmpty)
			})
			obj.State = bogusState
			changed, errs = obj.UpdateState(dbCache, conf)
			Convey("UpdateState on a bogus state should make not changes and return an error", func() {
				So(changed, ShouldBeFalse)
				So(errs, ShouldNotBeEmpty)
			})
		})
	})
}

// FIXME: incomplete.
func TestPendingAPIUserUpdateState(t *testing.T) {
	t.Parallel()

	conf := mustGetTestConf()

	Convey("Given an almost approved obj", t, func() {
		DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
			obj := objs.Pending.obj

			AlmostApprove(t, dbCache, &obj, &obj.PendingRevision, &obj.PendingRevision.CR, conf)
			errs := obj.PendingRevision.Cancel(dbCache, conf)
			//_, errs := obj.UpdateState(db, conf)
			Convey("UpdateState with pending should set to New, and return no error", func() {
				// So(changed, ShouldBeTrue)
				So(obj.State, ShouldEqual, StatePendingNew)
				So(errs, ShouldBeEmpty)
			})
		})
	})
}

func TestGetAPIUserFromPEM(t *testing.T) {
	t.Parallel()
	Convey("Given an approved obj", t, func() {
		DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
			apiUser, err := GetAPIUserFromPEM(dbCache, []byte(objs.Approved.cert))
			Convey("Should Load correct user", func() {
				So(err, ShouldBeNil)
				So(apiUser, ShouldNotBeNil)
				So(apiUser.ID, ShouldEqual, objs.Approved.obj.ID)
			})
		})
	})
}

func TestAPIUserVerifyCR(t *testing.T) {
	t.Parallel()
	Convey("Given an bootstrap database", t, func() {
		DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
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

func TestAPIUserSuggestedFields(t *testing.T) {
	t.Parallel()
	Convey("Given the default APIUser", t, func() {
		DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
			curRev := objs.Approved.rev
			tests := []struct{ name, fieldName, revFieldName, value string }{
				{"Name", APIUserName, "Name", curRev.Name},
				{"Description", APIUserDescription, "Description", curRev.Description},
				{"Certificate", APIUserCertificate, "Certificate", curRev.Certificate},
				{"Serial", APIUserSerial, "Serial", curRev.Serial},
			}
			emptyAPIUser := APIUser{}
			for _, test := range tests {
				Convey(fmt.Sprintf("SuggestedRevisionValue(%s) should return an empty string when there is no current revision", test.name),
					func() {
						So(emptyAPIUser.SuggestedRevisionValue(test.fieldName), ShouldBeBlank)
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
						So(emptyAPIUser.GetCurrentValue(test.fieldName), ShouldEqual, UnPreparedAPIUserError)
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
				{"DesiredStateActive", DesiredStateActive, curRev.DesiredState == StateActive},
				{"DesiredStateInactive", DesiredStateInactive, curRev.DesiredState == StateInactive},
			}
			for _, test := range boolTests {
				Convey(fmt.Sprintf("SuggestedRevisionBool(%s) should return false", test.name),
					func() {
						So(emptyAPIUser.SuggestedRevisionBool(test.fieldName), ShouldBeFalse)
					})
			}
			for _, test := range boolTests {
				Convey(fmt.Sprintf("SuggestedRevisionBool(%s) should be equal to value", test.name),
					func() {
						So(objs.Approved.obj.SuggestedRevisionBool(test.fieldName), ShouldEqual, test.value)
					})
			}
			Convey("default APIUser GetDisplay Name should have correct format", func() {
				So(objs.Approved.obj.GetDisplayName(), ShouldEqual, fmt.Sprintf("%d - %s",
					objs.Approved.obj.ID, curRev.Name))
			})
			Convey("Empty APIUser GetDisplay Name should have correct format", func() {
				So(emptyAPIUser.GetDisplayName(), ShouldEqual, fmt.Sprintf("0 - %s",
					UnPreparedAPIUserError))
			})
		})
	})
}

func TestAPIUserParseFromForm(t *testing.T) {
	t.Parallel()
	DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
		Convey("Given a HTTP request with no user set", t, func() {
			ctx := context.Background()
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			apiUser := APIUser{}
			parseError := apiUser.ParseFromForm(request, dbCache)

			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})

		Convey("Given a HTTP request with a valid user set", t, func() {
			timeBefore := TimeNow()
			ctx := context.Background()
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			request.Header.Add("REMOTE_USER", TestUser1Username)
			request.Form = make(url.Values)
			apiUser := APIUser{}
			err := dbCache.Save(&apiUser)
			So(err, ShouldBeNil)
			parseError := apiUser.ParseFromForm(request, dbCache)
			timeAfter := TimeNow()
			Convey("No error should be returned", func() {
				So(parseError, ShouldBeNil)
				Convey("The resulting approver should have the expected values", func() {
					So(apiUser.CreatedBy, ShouldEqual, TestUser1Username)
					So(apiUser.UpdatedBy, ShouldEqual, TestUser1Username)
					So(apiUser.CreatedAt, ShouldHappenOnOrBetween, timeBefore, timeAfter)
					So(apiUser.UpdatedAt, ShouldHappenOnOrBetween, timeBefore, timeAfter)
					So(apiUser.State, ShouldEqual, StateNew)
				})
			})
		})
	})
}

func TestAPIUserParseFromFormUpdate(t *testing.T) {
	t.Parallel()
	DoAPIUserDBTests(t, func(dbCache *DBCache, objs *APIUserBaseObjs) {
		Convey("Given a HTTP request with no user set", t, func() {
			ctx := context.Background()
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			apiUser := APIUser{}
			parseError := apiUser.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})

		Convey("Given a HTTP request with a valid user set", t, func() {
			ctx := context.Background()
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			request.Header.Add("REMOTE_USER", TestUser1Username)
			request.Form = make(url.Values)
			apiUser := APIUser{}
			err := dbCache.Save(&apiUser)
			So(err, ShouldBeNil)
			parseError := apiUser.ParseFromFormUpdate(request, dbCache, mustGetTestConf())
			Convey("An error should be returned", func() {
				So(parseError, ShouldNotBeNil)
			})
		})
	})
}

func genPEMCert(testingFile *TestFile) error {
	certificate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"Example"},
			OrganizationalUnit: []string{"test-org"},
		},
		Issuer: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"Example"},
			OrganizationalUnit: []string{"test-org"},
			Locality:           []string{"Somewhere"},
			Province:           []string{"SomeState"},
			StreetAddress:      []string{"123 Main St"},
			PostalCode:         []string{"00001"},
			SerialNumber:       "2",
			CommonName:         "registrar.example.com",
		},
		SignatureAlgorithm:    x509.SHA512WithRSA,
		PublicKeyAlgorithm:    x509.ECDSA,
		NotBefore:             TimeNow(),
		NotAfter:              TimeNow().AddDate(0, 0, 10),
		BasicConstraintsValid: true,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("error generting key: %w", err)
	}

	pub := &priv.PublicKey

	signedDER, err := x509.CreateCertificate(rand.Reader, certificate, certificate, pub, priv)
	if err != nil {
		return fmt.Errorf("error generating certificate: %w", err)
	}

	encodeErr := pem.Encode(testingFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: signedDER,
	})

	if encodeErr != nil {
		return fmt.Errorf("error encoding key: %w", encodeErr)
	}

	return nil
}

func getPEMCert(name string) (string, error) {
	return GetCahcedString(fmt.Sprintf("APIUser.%s.Cert.pem", name), genPEMCert)
}

func StartAPIUserApproval(dbCache *DBCache, conf Config, name string) (rev APIUserRevision, err error) {
	ctx := context.Background()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)

	request.Header.Add("REMOTE_USER", TestUser1Username)

	apiUser := APIUser{}

	if err = dbCache.Save(&apiUser); err != nil {
		return rev, err
	}

	rev.APIUserID = apiUser.ID
	rev.Name = name
	rev.Description = fmt.Sprintf("API User %s", name)

	if rev.Certificate, err = getPEMCert(name); err != nil {
		return rev, err
	}

	if rev.Serial, err = GetCertificateSerial([]byte(rev.Certificate)); err != nil {
		return rev, err
	}

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

func SetupAPIUserDB(dbCache *DBCache, _ Config) error {
	conf, err := getTestConf()
	if err != nil {
		return err
	}

	if err = BootstrapRegistrar(dbCache, conf); err != nil {
		return err
	}

	appRev, err := StartAPIUserApproval(dbCache, conf, "Approved")
	if err != nil {
		return fmt.Errorf("failed to start Approved user: %w", err)
	}

	if err = ApproveApproval(dbCache, TestUser1Username, appRev.CR.Approvals[0].ID, conf); err != nil {
		return fmt.Errorf("failed to approve Approved user: %w", err)
	}

	appRev, err = StartAPIUserApproval(dbCache, conf, "Declined")

	if err != nil {
		return fmt.Errorf("failed to start Declined user: %w", err)
	}

	if err = DeclineApproval(dbCache, TestUser1Username, appRev.CR.Approvals[0].ID, conf); err != nil {
		return fmt.Errorf("failed to approve Declined user: %w", err)
	}

	_, err = StartAPIUserApproval(dbCache, conf, "Pending")

	if err != nil {
		return fmt.Errorf("failed to start Pending user: %w", err)
	}

	return nil
}

func GetAPIUserDB(t *testing.T, roFunc TestFileFunc) {
	t.Helper()

	GetDBFunc(t, "APIUserDB", GetBootstrappedDB, SetupAPIUserDB, roFunc, mustGetTestConf())
}

func (obj *APIUserBaseObj) Load(t *testing.T, dbCache *DBCache, name string) (err error) {
	t.Helper()
	// db.DB = db.DB.Debug()
	if obj.cert, err = getPEMCert(name); err != nil {
		return
	}

	if err = dbCache.DB.Where("name = ?", name).First(&obj.rev).Error; err != nil {
		return
	}

	if err = obj.rev.Prepare(dbCache); err != nil {
		return
	}

	if err = dbCache.DB.First(&obj.obj, obj.rev.APIUserID).Error; err != nil {
		return
	}

	if err = obj.obj.Prepare(dbCache); err != nil {
		return
	}

	return nil
}

func DoAPIUserDBTests(t *testing.T, testBlock func(*DBCache, *APIUserBaseObjs)) {
	t.Helper()

	dbCache, err := DBFactory.GetDB(t, TestStateBootstrapAPIUser)
	if err != nil {
		panic(err)
	}

	objs := APIUserBaseObjs{}

	if err := objs.Approved.Load(t, dbCache, "Approved"); err != nil {
		t.Fatalf("fetching a API User records threw an error: %s", err.Error())
	}

	t.Logf("Approved Object: %v", objs.Approved)

	if err := objs.Declined.Load(t, dbCache, "Declined"); err != nil {
		t.Fatalf("fetching a API User records threw an error: %s", err.Error())
	}

	if err := objs.Pending.Load(t, dbCache, "Pending"); err != nil {
		t.Fatalf("fetching a API User records threw an error: %s", err.Error())
	}

	testBlock(dbCache, &objs)
}
