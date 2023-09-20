package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEPPGetPollRequest(t *testing.T) {
	t.Parallel()
	Convey("Creating a poll request with the same parameters as the default poll in the versign testing tool", t, func() {
		verisignEPP := GetEPPPollRequest("ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignPollRequest)
			So(eppErr, ShouldBeNil)
		})

		Convey("The epp message should have the type of an epp poll", func() {
			So(verisignEPP.MessageType(), ShouldEqual, PollType)
		})
	})
}

func TestEPPGetPollAck(t *testing.T) {
	t.Parallel()
	Convey("Creating a poll request with the same parameters as the default poll in the versign testing tool", t, func() {
		verisignEPP := GetEPPPollAcknowledge("06BB9D358D8A4088E043AC19A0464088", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignPollAck)
			So(eppErr, ShouldBeNil)
		})

		Convey("The epp message should have the type of an epp poll", func() {
			So(verisignEPP.MessageType(), ShouldEqual, PollType)
		})
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignPollRequest = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <poll op="req"></poll>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignPollAck = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <poll msgID="06BB9D358D8A4088E043AC19A0464088" op="ack"></poll>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
