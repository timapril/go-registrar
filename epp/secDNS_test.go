package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainDSUpdate(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain ds update request message with the same parameters as the default in Verisign testing tool", t, func() {
		addDS := DSData{Alg: 5, DigestType: 1, KeyTag: 1655, Digest: "1971674BFF957211D129B0DFE9410AF753559D4B"}
		remDS := DSData{Alg: 5, DigestType: 1, KeyTag: 965, Digest: "3801674BFF957211D129B0DFE9410AF753559D4B"}
		verisignEPP := GetEPPDomainSecDNSUpdate("EXAMPLE.COM", []DSData{addDS}, []DSData{remDS}, "ABC-12345-XYZ")

		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainDSUpdate)
			So(eppErr, ShouldBeNil)
		})
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainDSUpdate = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
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
      <secDNS:update xmlns:secDNS="urn:ietf:params:xml:ns:secDNS-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:secDNS-1.1 secDNS-1.1.xsd">
        <secDNS:rem>
          <secDNS:dsData>
            <secDNS:keyTag>965</secDNS:keyTag>
            <secDNS:alg>5</secDNS:alg>
            <secDNS:digestType>1</secDNS:digestType>
            <secDNS:digest>3801674BFF957211D129B0DFE9410AF753559D4B</secDNS:digest>
          </secDNS:dsData>
        </secDNS:rem>
        <secDNS:add>
          <secDNS:dsData>
            <secDNS:keyTag>1655</secDNS:keyTag>
            <secDNS:alg>5</secDNS:alg>
            <secDNS:digestType>1</secDNS:digestType>
            <secDNS:digest>1971674BFF957211D129B0DFE9410AF753559D4B</secDNS:digest>
          </secDNS:dsData>
        </secDNS:add>
      </secDNS:update>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
