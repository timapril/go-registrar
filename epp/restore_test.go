package epp

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainRestoreRequest(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain sync request message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainRestoreRequest("Example.Com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainRestoreRequest)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPDomainRestoreReport(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain sync request message with the same parameters as the default in the Verisign testing tool", t, func() {
		report := &RestoreReport{}
		report.PreWHOISData = "Pre-WhoIs Data..."
		report.PostWHOISData = "Post-WhoIs Data..."
		report.SetDeleteTime(time.Date(2008, time.January, 10, 22, 0, 0, 0, time.UTC))
		report.SetRestoreTime(time.Date(2008, time.January, 20, 22, 0, 0, 0, time.UTC))
		report.Reason = "Customer forgot to renew."
		report.Statements = append(report.Statements, "I agree that the Domain Name has not been restored in order to assume the rights to use or sell the name to myself or for any third party.")
		report.Statements = append(report.Statements, "I agree that the information provided in this Restore Report is true to the best of my knowledge, and acknowledge that intentionally supplying false information in the Restore Report shall constitute an incurable material breach of the Registry-Registrar Agreement.")
		verisignEPP := GetEPPDomainRestoreReport("Example.Com", report, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainRestoreReport)
			So(eppErr, ShouldBeNil)
		})
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainRestoreRequest = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <domain:update xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:add></domain:add>
        <domain:rem></domain:rem>
        <domain:chg></domain:chg>
      </domain:update>
    </update>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
      <rgp:update xmlns:rgp="urn:ietf:params:xml:ns:rgp-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:rgp-1.0 rgp-1.0.xsd">
        <rgp:restore op="request"></rgp:restore>
      </rgp:update>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainRestoreReport = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <domain:update xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:add></domain:add>
        <domain:rem></domain:rem>
        <domain:chg></domain:chg>
      </domain:update>
    </update>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
      <rgp:update xmlns:rgp="urn:ietf:params:xml:ns:rgp-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:rgp-1.0 rgp-1.0.xsd">
        <rgp:restore op="report">
          <rgp:report>
            <rgp:preData>Pre-WhoIs Data...</rgp:preData>
            <rgp:postData>Post-WhoIs Data...</rgp:postData>
            <rgp:delTime>2008-01-10T22:00:00.0000Z</rgp:delTime>
            <rgp:resTime>2008-01-20T22:00:00.0000Z</rgp:resTime>
            <rgp:resReason>Customer forgot to renew.</rgp:resReason>
            <rgp:statement>I agree that the Domain Name has not been restored in order to assume the rights to use or sell the name to myself or for any third party.</rgp:statement>
            <rgp:statement>I agree that the information provided in this Restore Report is true to the best of my knowledge, and acknowledge that intentionally supplying false information in the Restore Report shall constitute an incurable material breach of the Registry-Registrar Agreement.</rgp:statement>
            <rgp:other></rgp:other>
          </rgp:report>
        </rgp:restore>
      </rgp:update>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
