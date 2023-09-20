package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPLogout(t *testing.T) {
	t.Parallel()
	Convey("Creating a logout object with the same parameters as the default logout in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPLogout("ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignLogout)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestLogoutTypedMessage(t *testing.T) {
	t.Parallel()
	Convey("Given a logout epp message, unmarshalling it and getting the typed version should return a logout message", t, func() {
		obj, err := UnmarshalMessage([]byte(verisignLogout))
		So(err, ShouldBeNil)
		msg := obj.TypedMessage()
		So(msg.MessageType(), ShouldEqual, CommandLogoutType)

		textVersion, err := msg.ToString()
		So(err, ShouldBeNil)
		So(textVersion, ShouldEqual, verisignLogout)
	})
}

func TestGetEPPLogin(t *testing.T) {
	t.Parallel()
	Convey("Creating a login object with the same parameters as the default login in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPLogin("username", "password", "ABC-12345-XYZ", GetDefaultServiceMenu())
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignLogin)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestLoginTypedMessage(t *testing.T) {
	t.Parallel()
	Convey("Given a login epp message, unmarshalling it and getting the typed version should return a login message", t, func() {
		obj, err := UnmarshalMessage([]byte(verisignLogin))
		So(err, ShouldBeNil)
		msg := obj.TypedMessage()
		So(msg.MessageType(), ShouldEqual, CommandLoginType)

		textVersion, err := msg.ToString()
		So(err, ShouldBeNil)
		So(textVersion, ShouldEqual, verisignLogin)
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignLogin = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <login>
      <clID>username</clID>
      <pw>password</pw>
      <options>
        <version>1.0</version>
        <lang>en</lang>
      </options>
      <svcs>
        <objURI>http://www.verisign.com/epp/lowbalance-poll-1.0</objURI>
        <objURI>urn:ietf:params:xml:ns:contact-1.0</objURI>
        <objURI>http://www.verisign.com/epp/balance-1.0</objURI>
        <objURI>urn:ietf:params:xml:ns:domain-1.0</objURI>
        <objURI>http://www.verisign.com/epp/registry-1.0</objURI>
        <objURI>http://www.nic.name/epp/nameWatch-1.0</objURI>
        <objURI>http://www.verisign-grs.com/epp/suggestion-1.1</objURI>
        <objURI>http://www.verisign.com/epp/rgp-poll-1.0</objURI>
        <objURI>http://www.nic.name/epp/defReg-1.0</objURI>
        <objURI>urn:ietf:params:xml:ns:host-1.0</objURI>
        <objURI>http://www.nic.name/epp/emailFwd-1.0</objURI>
        <objURI>http://www.verisign.com/epp/whowas-1.0</objURI>
        <svcExtension>
          <extURI>urn:ietf:params:xml:ns:secDNS-1.1</extURI>
          <extURI>http://www.verisign.com/epp/whoisInf-1.0</extURI>
          <extURI>urn:ietf:params:xml:ns:secDNS-1.0</extURI>
          <extURI>http://www.verisign.com/epp/idnLang-1.0</extURI>
          <extURI>http://www.nic.name/epp/persReg-1.0</extURI>
          <extURI>http://www.verisign.com/epp/jobsContact-1.0</extURI>
          <extURI>urn:ietf:params:xml:ns:coa-1.0</extURI>
          <extURI>http://www.verisign-grs.com/epp/namestoreExt-1.1</extURI>
          <extURI>http://www.verisign.com/epp/sync-1.0</extURI>
          <extURI>http://www.verisign.com/epp/premiumdomain-1.0</extURI>
          <extURI>http://www.verisign.com/epp/relatedDomain-1.0</extURI>
          <extURI>urn:ietf:params:xml:ns:launch-1.0</extURI>
          <extURI>urn:ietf:params:xml:ns:rgp-1.0</extURI>
        </svcExtension>
      </svcs>
    </login>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignLogout = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <logout></logout>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
