package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainCheck(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain check object with the same parameters as the "+
		"default domain check in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainCheck("example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainCheck)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain check object with the same parameters as the "+
		"default domain check (switching to .NET) in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainCheck("example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainCheckDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPHostCheck(t *testing.T) {
	t.Parallel()
	Convey("Creating a host check object with the same parameters as the "+
		"default host check in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHostCheck("ns1.example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostCheck)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a host check object with the same parameters as the "+
		"default host check (switching to .NET) in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHostCheck("ns1.example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostCheckDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactCheck(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact check object with the same parameters as the "+
		"default contact check in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPContactCheck("8013", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactCheck)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestCheckMessageType(t *testing.T) {
	t.Parallel()
	Convey("Given a contact check message", t, func() {
		contCheck := GetEPPContactCheck("123", "123")
		So(contCheck.MessageType(), ShouldEqual, CommandCheckContactType)
	})

	Convey("Given a host check message", t, func() {
		contCheck := GetEPPHostCheck("123", "123")
		So(contCheck.MessageType(), ShouldEqual, CommandCheckHostType)
	})

	Convey("Given a domain check message", t, func() {
		contCheck := GetEPPDomainCheck("123", "123")
		So(contCheck.MessageType(), ShouldEqual, CommandCheckDomainType)
	})
}

func TestCheckTypedMessageContact(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "check", "contact", verisignContactCheck, CommandCheckContactType)
}

func TestCheckTypedMessageDomain(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "check", "domain", verisignDomainCheck, CommandCheckDomainType)
}

func TestCheckTypedMessageHost(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "check", "host", verisignHostCheck, CommandCheckHostType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainCheck = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <check>
      <domain:check xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
      </domain:check>
    </check>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainCheckDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <check>
      <domain:check xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
      </domain:check>
    </check>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostCheck = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <check>
      <host:check xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
      </host:check>
    </check>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostCheckDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <check>
      <host:check xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.NET</host:name>
      </host:check>
    </check>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Note: Removed the following extension
// <extension>
//
//	<namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
//		<namestoreExt:subProduct>EXAMPLE</namestoreExt:subProduct>
//	</namestoreExt:namestoreExt>
//
// </extension>

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactCheck = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <check>
      <contact:check xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
      </contact:check>
    </check>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
