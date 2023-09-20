package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainCreate(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain create object with the same parameters as the default domain create in the Verisign testing tool", t, func() {
		period := GetEPPDomainPeriod(DomainPeriodYear, 1)
		hosts := []DomainHost{}
		hosts = append(hosts, DomainHost{Value: "ns1.example.com"})
		hosts = append(hosts, DomainHost{Value: "ns2.example.com"})
		contactID := "8013"
		verisignEPP := GetEPPDomainCreate("example.com", period, hosts, &contactID, &contactID, &contactID, &contactID, "sampleAuthInfo-1", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainCreate)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain create object with the same parameters as the default domain (switching to .net) create in the Verisign testing tool", t, func() {
		period := GetEPPDomainPeriod(DomainPeriodYear, 1)
		hosts := []DomainHost{}
		hosts = append(hosts, DomainHost{Value: "ns1.example.net"})
		hosts = append(hosts, DomainHost{Value: "ns2.example.net"})
		contactID := "8013"
		verisignEPP := GetEPPDomainCreate("example.net", period, hosts, &contactID, &contactID, &contactID, &contactID, "sampleAuthInfo-1", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainCreateDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPHostCreate(t *testing.T) {
	t.Parallel()
	Convey("Creating a host create object with the same parameters as the default host create in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHostCreate("NS1.EXAMPLE.COM", []string{"192.0.2.2"}, []string{}, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostCreate)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a host create object with the same parameters as the default host create (and an additional v6 address) in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHostCreate("NS1.EXAMPLE.COM", []string{"192.0.2.2"}, []string{"::1"}, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostCreateWithV6)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a host create object with the same parameters as the default host create (switching to .NET) in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHostCreate("NS1.EXAMPLE.NET", []string{"192.0.2.2"}, []string{}, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostCreateDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactCreate(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact create object with the same parameters as the default contact create in the Verising testing tool", t, func() {
		pi := GetEPPPostalInfo("int", "John Doe", "Example Inc.", "123 Example Dr.", "Suite 100", "Wing C", "Dulles", "VA", "20166-6503", "US")
		voice := GetEPPPhoneNumber("+1.7035555555", "1234")
		fax := GetEPPPhoneNumber("+1.7035555556", "5678")
		verisignEPP := GetEPPContactCreate("8013", pi, "jdoe@example.com", voice, fax, "sampleAuthInfo-1", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactCreate)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPPostalInfo(t *testing.T) {
	t.Parallel()
	Convey("Creating a postal info object with the same parameters as the default in the Verisgn testing tool", t, func() {
		pi := GetEPPPostalInfo("int", "John Doe", "Example Inc.", "123 Example Dr.", "Suite 100", "Wing C", "Dulles", "VA", "20166-6503", "US")
		Convey("The output should match the output from the testing tool", func() {
			xml, err := interfaceToXMLString(pi)
			So(err, ShouldBeNil)
			So(xml, ShouldEqual, verisignPostalInfo)
		})
	})
}

func TestContactTypeFromString(t *testing.T) {
	t.Parallel()
	Convey("Requesting each of the contact types (and one that is not valid) should return the expected results", t, func() {
		Convey("Test Tech Contact Type", func() {
			c, err := ContactTypeFromString("Tech")
			So(err, ShouldBeNil)
			So(c, ShouldEqual, Tech)
		})

		Convey("Test Billing Contact Type", func() {
			c, err := ContactTypeFromString("Billing")
			So(err, ShouldBeNil)
			So(c, ShouldEqual, Billing)
		})

		Convey("Test Admin Contact Type", func() {
			c, err := ContactTypeFromString("Admin")
			So(err, ShouldBeNil)
			So(c, ShouldEqual, Admin)
		})

		Convey("Test Bogus Contact Type", func() {
			_, err := ContactTypeFromString("Bogus")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestCreateTypedMessageDomain(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "create", "domain", verisignDomainCreate, CommandCreateDomainType)
}

func TestCreateTypedMessageContact(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "create", "contact", verisignContactCreate, CommandCreateContactType)
}

func TestCreateTypedMessageHost(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "create", "host", verisignHostCreate, CommandCreateHostType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainCreate = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <create>
      <domain:create xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:period unit="y">1</domain:period>
        <domain:ns>
          <domain:hostObj>ns1.example.com</domain:hostObj>
          <domain:hostObj>ns2.example.com</domain:hostObj>
        </domain:ns>
        <domain:registrant>8013</domain:registrant>
        <domain:contact type="admin">8013</domain:contact>
        <domain:contact type="tech">8013</domain:contact>
        <domain:contact type="billing">8013</domain:contact>
        <domain:authInfo>
          <domain:pw>sampleAuthInfo-1</domain:pw>
        </domain:authInfo>
      </domain:create>
    </create>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainCreateDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <create>
      <domain:create xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
        <domain:period unit="y">1</domain:period>
        <domain:ns>
          <domain:hostObj>ns1.example.net</domain:hostObj>
          <domain:hostObj>ns2.example.net</domain:hostObj>
        </domain:ns>
        <domain:registrant>8013</domain:registrant>
        <domain:contact type="admin">8013</domain:contact>
        <domain:contact type="tech">8013</domain:contact>
        <domain:contact type="billing">8013</domain:contact>
        <domain:authInfo>
          <domain:pw>sampleAuthInfo-1</domain:pw>
        </domain:authInfo>
      </domain:create>
    </create>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostCreate = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <create>
      <host:create xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
        <host:addr ip="v4">192.0.2.2</host:addr>
      </host:create>
    </create>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostCreateDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <create>
      <host:create xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.NET</host:name>
        <host:addr ip="v4">192.0.2.2</host:addr>
      </host:create>
    </create>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostCreateWithV6 = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <create>
      <host:create xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
        <host:addr ip="v4">192.0.2.2</host:addr>
        <host:addr ip="v6">::1</host:addr>
      </host:create>
    </create>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Note: the extension object was removed for testing
// <extension>
//
//	<namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
//		<namestoreExt:subProduct>EXAMPLE</namestoreExt:subProduct>
//	</namestoreExt:namestoreExt>
//
// </extension>

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactCreate = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <create>
      <contact:create xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
        <contact:postalInfo type="int">
          <contact:name>John Doe</contact:name>
          <contact:org>Example Inc.</contact:org>
          <contact:addr>
            <contact:street>123 Example Dr.</contact:street>
            <contact:street>Suite 100</contact:street>
            <contact:street>Wing C</contact:street>
            <contact:city>Dulles</contact:city>
            <contact:sp>VA</contact:sp>
            <contact:pc>20166-6503</contact:pc>
            <contact:cc>US</contact:cc>
          </contact:addr>
        </contact:postalInfo>
        <contact:voice x="1234">+1.7035555555</contact:voice>
        <contact:fax x="5678">+1.7035555556</contact:fax>
        <contact:email>jdoe@example.com</contact:email>
        <contact:authInfo>
          <contact:pw>sampleAuthInfo-1</contact:pw>
        </contact:authInfo>
      </contact:create>
    </create>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

var verisignPostalInfo = `<contact:postalInfo type="int">
  <contact:name>John Doe</contact:name>
  <contact:org>Example Inc.</contact:org>
  <contact:addr>
    <contact:street>123 Example Dr.</contact:street>
    <contact:street>Suite 100</contact:street>
    <contact:street>Wing C</contact:street>
    <contact:city>Dulles</contact:city>
    <contact:sp>VA</contact:sp>
    <contact:pc>20166-6503</contact:pc>
    <contact:cc>US</contact:cc>
  </contact:addr>
</contact:postalInfo>`
