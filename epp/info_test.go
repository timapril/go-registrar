package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainInfo(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain info object with the same parameters as the default domain info in the "+
		"Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainInfo("example.com", "ABC-12345-XYZ", "sampleAuthInfo-1", DomainInfoHostsAll)
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainInfo)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain info object with the same parameters as the default domain (switching to .NET) "+
		"info in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainInfo("example.net", "ABC-12345-XYZ", "sampleAuthInfo-1", DomainInfoHostsAll)
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainInfoDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPHostInfo(t *testing.T) {
	t.Parallel()
	Convey("Creating a host info object with the same parameters as the default host info in the Verisign "+
		"testing tool", t, func() {
		verisignEPP := GetEPPHostInfo("ns1.example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostInfo)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a host info object with the same parameters as the default host (switching to .NET) info "+
		"in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHostInfo("ns1.example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostInfoDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactInfo(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact info object with the same parameters as the default contact info in the "+
		"Verisign testing tool", t, func() {
		verisignEPP := GetEPPContactInfo("8013", "ABC-12345-XYZ", "sampleAuthInfo-1")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactInfo)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestInfoTypedMessageContact(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "info", "contact", verisignContactInfo, CommandInfoContactType)
}

func TestInfoTypedMessageDomain(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "info", "domain", verisignDomainInfo, CommandInfoDomainType)
}

func TestInfoTypedMessageHost(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "info", "host", verisignHostInfo, CommandInfoHostType)
}

func TestInfoTypedMessageHostAuth(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "info", "host", verisignHostInfoAuth, CommandInfoHostType)
}

// xmlns omitted due to issue with unmarshaling epp. Not relevant for the processing
// in this library. The EPP envelope would have would contain
// `xmlns="urn:ietf:params:xml:ns:epp-1.0"`
var verisignDomainInfo = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <info>
      <domain:info xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name hosts="all">EXAMPLE.COM</domain:name>
        <domain:authInfo>
          <domain:pw>sampleAuthInfo-1</domain:pw>
        </domain:authInfo>
      </domain:info>
    </info>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// xmlns omitted due to issue with unmarshaling epp. Not relevant for the processing
// in this library. The EPP envelope would have would contain
// `xmlns="urn:ietf:params:xml:ns:epp-1.0"`
var verisignDomainInfoDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <info>
      <domain:info xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name hosts="all">EXAMPLE.NET</domain:name>
        <domain:authInfo>
          <domain:pw>sampleAuthInfo-1</domain:pw>
        </domain:authInfo>
      </domain:info>
    </info>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// xmlns omitted due to issue with unmarshaling epp. Not relevant for the processing
// in this library. The EPP envelope would have would contain
// `xmlns="urn:ietf:params:xml:ns:epp-1.0"`
var verisignHostInfo = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <info>
      <host:info xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
      </host:info>
    </info>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// xmlns omitted due to issue with unmarshaling epp. Not relevant for the processing
// in this library. The EPP envelope would have would contain
// `xmlns="urn:ietf:params:xml:ns:epp-1.0"`
var verisignHostInfoAuth = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <info>
      <host:info xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
        <host:authInfo>
          <host:pw>sampleAuthInfo-1</host:pw>
        </host:authInfo>
      </host:info>
    </info>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// xmlns omitted due to issue with unmarshaling epp. Not relevant for the processing
// in this library. The EPP envelope would have would contain
// `xmlns="urn:ietf:params:xml:ns:epp-1.0"`
var verisignHostInfoDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <info>
      <host:info xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.NET</host:name>
      </host:info>
    </info>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// xmlns omitted due to issue with unmarshaling epp. Not relevant for the processing
// in this library. The EPP envelope would have would contain
// `xmlns="urn:ietf:params:xml:ns:epp-1.0"`
var verisignContactInfo = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <info>
      <contact:info xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
        <contact:authInfo>
          <contact:pw>sampleAuthInfo-1</contact:pw>
        </contact:authInfo>
      </contact:info>
    </info>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
