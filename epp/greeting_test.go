package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGreetingTypedMessage(t *testing.T) {
	t.Parallel()
	Convey("Given a greeting epp message, unmarshalling it and getting the typed version should return a greeting message", t, func() {
		obj, err := UnmarshalMessage([]byte(verisignGreeting))
		So(err, ShouldBeNil)
		msg := obj.TypedMessage()
		So(msg.MessageType(), ShouldEqual, GreetingType)

		textVersion, err := msg.ToString()
		So(err, ShouldBeNil)
		So(textVersion, ShouldEqual, verisignGreeting)
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignGreeting = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <greeting>
    <svID>EPP Server Stub</svID>
    <svDate>2015-09-22T23:53:39.854Z</svDate>
    <svcMenu>
      <version>1.0</version>
      <lang>en</lang>
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
    </svcMenu>
    <dcp>
      <access>
        <all></all>
      </access>
      <statement>
        <purpose>
          <admin></admin>
          <prov></prov>
        </purpose>
        <recipient>
          <ours></ours>
          <public></public>
        </recipient>
        <retention>
          <stated></stated>
        </retention>
      </statement>
    </dcp>
  </greeting>
</epp>`
