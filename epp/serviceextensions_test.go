package epp

import (
	"encoding/xml"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func serviceMenuToString(svc ServiceMenu) (string, error) {
	message, err := xml.MarshalIndent(svc, "", "  ")

	return string(message), err
}

func TestGetDefaultServiceMenu(t *testing.T) {
	t.Parallel()
	Convey("Creating a Default Service Menu", t, func() {
		menu := GetDefaultServiceMenu()
		Convey("When Serialized to XML should match the service menu from a testing login message", func() {
			eppStr, eppErr := serviceMenuToString(menu)
			So(eppStr, ShouldEqual, verisignServiceMenu)
			So(eppErr, ShouldBeNil)
		})
	})
}

var verisignServiceMenu = `<svcMenu>
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
</svcMenu>`
