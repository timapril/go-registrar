package csrf

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

type TestingConf struct {
	ErrType string
}

const (
	// ErrTypeTimeError indicates that the testing configuration object
	// should return a bad timestamp validity.
	ErrTypeTimeError = "TimeError"

	// ErrTypeShort indicates that the testing configuration object
	// should return a short duration validity.
	ErrTypeShort = "ShortTime"

	// ErrTypeShort indicates that the testing configuration object
	// should return a normal duration validity.
	ErrTypeNone = "None"
)

func (t TestingConf) GetValidityPeriod() time.Duration {
	if t.ErrType == ErrTypeTimeError {
		return time.Minute * -30
	} else if t.ErrType == ErrTypeShort {
		return time.Millisecond * 20
	}

	return time.Minute * 30
}

func (t TestingConf) GetHMACKey() []byte {
	return []byte("testkey")
}

func TestGenerateCSRF(t *testing.T) {
	t.Parallel()
	Convey("Given a valid username and a positive time duration", t, func() {
		token, err := GenerateCSRF("testuser", TestingConf{ErrType: ErrTypeNone})
		So(err, ShouldBeNil)
		So(token, ShouldNotBeEmpty)

		validCsrf, checkerr := CheckCSRF("testuser", TestingConf{ErrType: ErrTypeNone}, token)
		So(checkerr, ShouldBeNil)
		So(validCsrf, ShouldBeTrue)
	})
}

func TestCheckCSRF(t *testing.T) {
	t.Parallel()
	Convey("Given a short lived token waiting until after the token expires should return an error", t, func() {
		conf := TestingConf{ErrType: ErrTypeShort}
		token, err := GenerateCSRF("testuser", conf)
		So(err, ShouldBeNil)
		So(token, ShouldNotBeEmpty)

		time.Sleep(conf.GetValidityPeriod() * 2)

		validCsrf, checkerr := CheckCSRF("testuser", TestingConf{ErrType: ErrTypeNone}, token)
		So(checkerr, ShouldNotBeNil)
		So(validCsrf, ShouldBeFalse)
	})

	Convey("Given a poorly formatted token, CheckCSRF should return an error", t, func() {
		validCsrf, checkerr := CheckCSRF("testuser", TestingConf{ErrType: ErrTypeNone}, "bad token:")
		So(checkerr, ShouldNotBeNil)
		So(validCsrf, ShouldBeFalse)
	})

	Convey("Given a correct token with the wrong username, CheckCSRF should return an error", t, func() {
		token, err := GenerateCSRF("testuser", TestingConf{ErrType: ErrTypeNone})
		So(err, ShouldBeNil)
		So(token, ShouldNotBeEmpty)

		validCsrf, checkerr := CheckCSRF("testuser2", TestingConf{ErrType: ErrTypeNone}, token)
		So(checkerr, ShouldNotBeNil)
		So(validCsrf, ShouldBeFalse)
	})

	Convey("Given a correct token with the wrong checksum, CheckCSRF should return an error", t, func() {
		token, err := GenerateCSRF("testuser", TestingConf{ErrType: ErrTypeNone})
		So(err, ShouldBeNil)
		So(token, ShouldNotBeEmpty)

		validCsrf, checkerr := CheckCSRF("testuser", TestingConf{ErrType: ErrTypeNone}, token+"1234")
		So(checkerr, ShouldNotBeNil)
		So(validCsrf, ShouldBeFalse)
	})
}
