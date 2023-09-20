package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainDelete(t *testing.T) {
	t.Parallel()
	Convey("Creatign a domain delete object with the same parameters as the default domain delete in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainDelete("example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainDelete)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creatign a domain delete object with the same parameters as the default domain delete (switching to .NET) in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainDelete("example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainDeleteDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPHostDelete(t *testing.T) {
	t.Parallel()
	Convey("Creatign a host delete object with the same parameters as the default host delete in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHostDelete("ns1.example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostDelete)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creatign a host delete object with the same parameters as the default host delete (switching to .NET) in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHostDelete("ns1.example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostDeleteDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactDelete(t *testing.T) {
	t.Parallel()
	Convey("Creatign a contact delete object with the same parameters as the default contact delete in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPContactDelete("8013", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactDelete)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestDeleteTypedMessageDomain(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "delete", "domain", verisignDomainDelete, CommandDeleteDomainType)
}

func TestDeleteTypedMessageHost(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "delete", "host", verisignHostDelete, CommandDeleteHostType)
}

func TestDeleteTypedMessageContact(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "delete", "contact", verisignContactDelete, CommandDeleteContactType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainDelete = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <delete>
      <domain:delete xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
      </domain:delete>
    </delete>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainDeleteDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <delete>
      <domain:delete xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
      </domain:delete>
    </delete>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostDelete = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <delete>
      <host:delete xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
      </host:delete>
    </delete>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostDeleteDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <delete>
      <host:delete xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.NET</host:name>
      </host:delete>
    </delete>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Note, an extension has been removed:
// <extension>
//
//	<namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
//	  <namestoreExt:subProduct>EXAMPLE</namestoreExt:subProduct>
//	</namestoreExt:namestoreExt>
//
// </extension>

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactDelete = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <delete>
      <contact:delete xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
      </contact:delete>
    </delete>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
