package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPHello(t *testing.T) {
	t.Parallel()
	Convey("Creating a hello object with the same parameters as the default hello in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHello()
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHello)
			So(eppErr, ShouldBeNil)
		})
	})
}

// Note: the ending hello tag was added rather than closing the original
// tag with a /.
// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHello = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <hello xmlns="urn:ietf:params:xml:ns:epp-1.0"></hello>
</epp>`
