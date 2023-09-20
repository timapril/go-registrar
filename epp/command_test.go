package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPCommand(t *testing.T) {
	t.Parallel()
	Convey("Creating a command object with a transaction id set", t, func() {
		verisignEPP := GetEPPCommand("ABC-12345-XYZ")
		Convey("The output should match the default template for a "+
			"command", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, eppCommandTemplate)
			So(eppErr, ShouldBeNil)
		})
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var eppCommandTemplate = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

func TestCommandMessageType(t *testing.T) {
	t.Parallel()
	Convey("Given a login command object", t, func() {
		obj := GetEPPLogin("username", "password", "txid", GetDefaultServiceMenu())
		Convey("The MessageType function should return a login message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandLoginType)
		})
	})
	Convey("Given a logout command object", t, func() {
		obj := GetEPPLogout("txid")
		Convey("The MessageType function should return a logout message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandLogoutType)
		})
	})
	Convey("Given a check command object", t, func() {
		obj := GetEPPCheck("txid")
		Convey("The MessageType function should return a check message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandCheckType)
		})
	})
	Convey("Given a create command object", t, func() {
		obj := GetEPPCreate("txid")
		Convey("The MessageType function should return a create message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandCreateType)
		})
	})
	Convey("Given a info command object", t, func() {
		obj := GetEPPInfo("txid")
		Convey("The MessageType function should return a info message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandInfoType)
		})
	})
	Convey("Given a delete command object", t, func() {
		obj := GetEPPDelete("txid")
		Convey("The MessageType function should return a delete message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandDeleteType)
		})
	})
	Convey("Given a update command object", t, func() {
		obj := GetEPPUpdate("txid")
		Convey("The MessageType function should return a update message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandUpdateType)
		})
	})
	Convey("Given a transfer command object", t, func() {
		obj := GetEPPTransfer(TransferRequest, "txid")
		Convey("The MessageType function should return a transfer message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandTransferType)
		})
	})
	Convey("Given a renew command object", t, func() {
		obj := GetEPPRenew("txid")
		Convey("The MessageType function should return a renew message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandRenewType)
		})
	})
	Convey("Given a generic command object", t, func() {
		obj := GetEPPCommand("txid")
		Convey("The MessageType function should return a generic message", func() {
			So(obj.CommandObject.MessageType(), ShouldEqual, CommandType)
		})
	})
}
