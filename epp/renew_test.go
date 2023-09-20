package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainRenew(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain renew object with the same parameters as the default domain renew in the Verisign testing tool", t, func() {
		renewLen := GetEPPDomainPeriod(DomainPeriodYear, 1)
		verisignEPP := GetEPPDomainRenew("example.com", "2008-12-25", renewLen, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainRenew)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain renew object with the same parameters (switching to .net) as the default domain renew in the Verisign testing tool", t, func() {
		renewLen := GetEPPDomainPeriod(DomainPeriodYear, 1)
		verisignEPP := GetEPPDomainRenew("example.net", "2008-12-25", renewLen, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainRenewDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestRenewTypedMessageDomain(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "renew", "domain", verisignDomainRenew, CommandRenewDomainType)
}

func TestRenewMessageType(t *testing.T) {
	t.Parallel()
	Convey("Given an empty renew object", t, func() {
		emptyRenew := GetEPPRenew("123")
		So(emptyRenew.MessageType(), ShouldEqual, CommandRenewType)
	})
	Convey("Given a domain renew object", t, func() {
		domainRenew := GetEPPDomainRenew("example.com", "2008-12-25", GetEPPDomainPeriod(DomainPeriodYear, 1), "123")
		So(domainRenew.MessageType(), ShouldEqual, CommandRenewDomainType)
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainRenew = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <renew>
      <domain:renew xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:curExpDate>2008-12-25</domain:curExpDate>
        <domain:period unit="y">1</domain:period>
      </domain:renew>
    </renew>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainRenewDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <renew>
      <domain:renew xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
        <domain:curExpDate>2008-12-25</domain:curExpDate>
        <domain:period unit="y">1</domain:period>
      </domain:renew>
    </renew>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
