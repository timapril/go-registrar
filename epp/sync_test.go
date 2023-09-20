package epp

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainSyncUpdate(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain sync request message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainSyncUpdate("Example.Com", time.January, 1, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainSync)
			So(eppErr, ShouldBeNil)
		})
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainSync = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
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
      <sync:update xmlns:sync="http://www.verisign.com/epp/sync-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign.com/epp/sync-1.0 sync-1.0.xsd">
        <sync:expMonthDay>--01-01</sync:expMonthDay>
      </sync:update>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
