package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPContactUpdate(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact update object with the same values as the testing tool from verisign", t, func() {
		add := GetEPPContactUpdateAddRemove([]string{"clientDeleteProhibited"})
		rem := GetEPPContactUpdateAddRemove([]string{"clientUpdateProhibited"})
		pi := GetEPPPostalInfo("int", "John Doe", "Example Inc.", "123 Example Dr.", "Suite 100", "Wing C", "Dulles", "VA", "20166-6503", "US")
		voice := GetEPPPhoneNumber("+1.7035555555", "1234")
		fax := GetEPPPhoneNumber("+1.7035555556", "5678")
		chg := GetEPPContactUpdateChange(&pi, &voice, &fax, "jdoe@example.com", "sampleAuthInfo-1")
		verisignEPP := GetEPPContactUpdate("8013", add, rem, chg, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactUpdate)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a contact update object with the no values as the testing tool from verisign", t, func() {
		add := GetEPPContactUpdateAddRemove([]string{})
		rem := GetEPPContactUpdateAddRemove([]string{})
		chg := GetEPPContactUpdateChange(nil, nil, nil, "", "")
		verisignEPP := GetEPPContactUpdate("8013", add, rem, chg, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactUpdateEmpty)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPHostUpdate(t *testing.T) {
	t.Parallel()
	Convey("Creating a host update object with some values in each of the add, remove and change options", t, func() {
		addAddresses := []HostAddress{}
		addAddresses = append(addAddresses, HostAddress{IPVersion: IPv4, Address: "192.0.2.223"})
		addAddresses = append(addAddresses, HostAddress{IPVersion: IPv6, Address: "2001:db8::ff24:2c74:42c5:5dfc"})
		remAddresses := []HostAddress{}
		remAddresses = append(remAddresses, HostAddress{IPVersion: IPv6, Address: "2001:db8::1eab:b52d:27e4:5da6"})
		remAddresses = append(remAddresses, HostAddress{IPVersion: IPv4, Address: "203.0.113.56"})
		add := GetEPPHostUpdateAddRemove(addAddresses, []string{"clientUpdateProhibited"})
		rem := GetEPPHostUpdateAddRemove(remAddresses, []string{"clientDeleteProhibited"})
		chg := GetEPPHostUpdateChange("ns2.example.com")
		verisignEPP := GetEPPHostUpdate("ns1.example.com", &add, &rem, &chg, "ABC-12345-XYZ")

		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostUpdate)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a host update object with some values in each of the add, remove and change options (dot net version)", t, func() {
		addAddresses := []HostAddress{}
		addAddresses = append(addAddresses, HostAddress{IPVersion: IPv4, Address: "192.0.2.223"})
		addAddresses = append(addAddresses, HostAddress{IPVersion: IPv6, Address: "2001:db8::ff24:2c74:42c5:5dfc"})
		remAddresses := []HostAddress{}
		remAddresses = append(remAddresses, HostAddress{IPVersion: IPv6, Address: "2001:db8::1eab:b52d:27e4:5da6"})
		remAddresses = append(remAddresses, HostAddress{IPVersion: IPv4, Address: "203.0.113.56"})
		add := GetEPPHostUpdateAddRemove(addAddresses, []string{"clientUpdateProhibited"})
		rem := GetEPPHostUpdateAddRemove(remAddresses, []string{"clientDeleteProhibited"})
		chg := GetEPPHostUpdateChange("ns2.example.net")
		verisignEPP := GetEPPHostUpdate("ns1.example.net", &add, &rem, &chg, "ABC-12345-XYZ")

		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostUpdateDotNet)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a host update object with no values in each of the add, remove and change options", t, func() {
		addAddresses := []HostAddress{}
		remAddresses := []HostAddress{}
		add := GetEPPHostUpdateAddRemove(addAddresses, []string{})
		rem := GetEPPHostUpdateAddRemove(remAddresses, []string{})
		chg := GetEPPHostUpdateChange("")
		verisignEPP := GetEPPHostUpdate("ns1.example.com", &add, &rem, &chg, "ABC-12345-XYZ")

		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignHostUpdateEmpty)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPDomainUpdate(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain update object with some values in each of the add, remove and change options", t, func() {
		regID := "8013"
		passwd := "samplePassword-1"
		duc := GetEPPDomainUpdateChange(&regID, &passwd)
		addContacts := []DomainContact{}
		addContacts = append(addContacts, DomainContact{Type: Admin, Value: "T09571012"})
		addContacts = append(addContacts, DomainContact{Type: Tech, Value: "T09571012"})
		addContacts = append(addContacts, DomainContact{Type: Billing, Value: "T09571012"})
		add := GetEPPDomainUpdateAddRemove([]string{"ns1.example.com"}, addContacts, []string{"clientHold", "clientUpdateProhibited"})
		remContacts := []DomainContact{}
		remContacts = append(remContacts, DomainContact{Type: Admin, Value: "T09571011"})
		remContacts = append(remContacts, DomainContact{Type: Tech, Value: "T09571011"})
		remContacts = append(remContacts, DomainContact{Type: Billing, Value: "T09571011"})
		rem := GetEPPDomainUpdateAddRemove([]string{"ns2.example.com"}, remContacts, []string{"clientDeleteProhibited", "clientTransferProhibited"})
		verisignEPP := GetEPPDomainUpdate("example.com", &add, &rem, &duc, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainUpdate)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain update object with some values in each of the add, remove and change options (dot net version)", t, func() {
		regID := "8013"
		passwd := "samplePassword-1"
		duc := GetEPPDomainUpdateChange(&regID, &passwd)
		addContacts := []DomainContact{}
		addContacts = append(addContacts, DomainContact{Type: Admin, Value: "T09571012"})
		addContacts = append(addContacts, DomainContact{Type: Tech, Value: "T09571012"})
		addContacts = append(addContacts, DomainContact{Type: Billing, Value: "T09571012"})
		add := GetEPPDomainUpdateAddRemove([]string{"ns1.example.net"}, addContacts, []string{"clientHold", "clientUpdateProhibited"})
		remContacts := []DomainContact{}
		remContacts = append(remContacts, DomainContact{Type: Admin, Value: "T09571011"})
		remContacts = append(remContacts, DomainContact{Type: Tech, Value: "T09571011"})
		remContacts = append(remContacts, DomainContact{Type: Billing, Value: "T09571011"})
		rem := GetEPPDomainUpdateAddRemove([]string{"ns2.example.net"}, remContacts, []string{"clientDeleteProhibited", "clientTransferProhibited"})
		verisignEPP := GetEPPDomainUpdate("example.net", &add, &rem, &duc, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainUpdateDotNet)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain update object with no add, rem or change options", t, func() {
		duc := GetEPPDomainUpdateChange(nil, nil)
		addContacts := []DomainContact{}
		add := GetEPPDomainUpdateAddRemove([]string{}, addContacts, []string{})
		remContacts := []DomainContact{}
		rem := GetEPPDomainUpdateAddRemove([]string{}, remContacts, []string{})
		verisignEPP := GetEPPDomainUpdate("example.com", &add, &rem, &duc, "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainUpdateEmpty)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestUpdateTypedMessageDomain(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "update", "domain", verisignDomainUpdate, CommandUpdateDomainType)
}

func TestUpdateTypedMessageHost(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "update", "host", verisignHostUpdate, CommandUpdateHostType)
}

func TestUpdateTypedMessageContact(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "update", "contact", verisignContactUpdate, CommandUpdateContactType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainUpdate = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <domain:update xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:add>
          <domain:ns>
            <domain:hostObj>ns1.example.com</domain:hostObj>
          </domain:ns>
          <domain:contact type="admin">T09571012</domain:contact>
          <domain:contact type="tech">T09571012</domain:contact>
          <domain:contact type="billing">T09571012</domain:contact>
          <domain:status s="clientHold"></domain:status>
          <domain:status s="clientUpdateProhibited"></domain:status>
        </domain:add>
        <domain:rem>
          <domain:ns>
            <domain:hostObj>ns2.example.com</domain:hostObj>
          </domain:ns>
          <domain:contact type="admin">T09571011</domain:contact>
          <domain:contact type="tech">T09571011</domain:contact>
          <domain:contact type="billing">T09571011</domain:contact>
          <domain:status s="clientDeleteProhibited"></domain:status>
          <domain:status s="clientTransferProhibited"></domain:status>
        </domain:rem>
        <domain:chg>
          <domain:registrant>8013</domain:registrant>
          <domain:authInfo>
            <domain:pw>samplePassword-1</domain:pw>
          </domain:authInfo>
        </domain:chg>
      </domain:update>
    </update>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainUpdateDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <domain:update xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
        <domain:add>
          <domain:ns>
            <domain:hostObj>ns1.example.net</domain:hostObj>
          </domain:ns>
          <domain:contact type="admin">T09571012</domain:contact>
          <domain:contact type="tech">T09571012</domain:contact>
          <domain:contact type="billing">T09571012</domain:contact>
          <domain:status s="clientHold"></domain:status>
          <domain:status s="clientUpdateProhibited"></domain:status>
        </domain:add>
        <domain:rem>
          <domain:ns>
            <domain:hostObj>ns2.example.net</domain:hostObj>
          </domain:ns>
          <domain:contact type="admin">T09571011</domain:contact>
          <domain:contact type="tech">T09571011</domain:contact>
          <domain:contact type="billing">T09571011</domain:contact>
          <domain:status s="clientDeleteProhibited"></domain:status>
          <domain:status s="clientTransferProhibited"></domain:status>
        </domain:rem>
        <domain:chg>
          <domain:registrant>8013</domain:registrant>
          <domain:authInfo>
            <domain:pw>samplePassword-1</domain:pw>
          </domain:authInfo>
        </domain:chg>
      </domain:update>
    </update>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainUpdateEmpty = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
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
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostUpdate = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <host:update xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
        <host:add>
          <host:addr ip="v4">192.0.2.223</host:addr>
          <host:addr ip="v6">2001:db8::ff24:2c74:42c5:5dfc</host:addr>
          <host:status s="clientUpdateProhibited"></host:status>
        </host:add>
        <host:rem>
          <host:addr ip="v6">2001:db8::1eab:b52d:27e4:5da6</host:addr>
          <host:addr ip="v4">203.0.113.56</host:addr>
          <host:status s="clientDeleteProhibited"></host:status>
        </host:rem>
        <host:chg>
          <host:name>NS2.EXAMPLE.COM</host:name>
        </host:chg>
      </host:update>
    </update>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostUpdateDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <host:update xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.NET</host:name>
        <host:add>
          <host:addr ip="v4">192.0.2.223</host:addr>
          <host:addr ip="v6">2001:db8::ff24:2c74:42c5:5dfc</host:addr>
          <host:status s="clientUpdateProhibited"></host:status>
        </host:add>
        <host:rem>
          <host:addr ip="v6">2001:db8::1eab:b52d:27e4:5da6</host:addr>
          <host:addr ip="v4">203.0.113.56</host:addr>
          <host:status s="clientDeleteProhibited"></host:status>
        </host:rem>
        <host:chg>
          <host:name>NS2.EXAMPLE.NET</host:name>
        </host:chg>
      </host:update>
    </update>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHostUpdateEmpty = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <host:update xmlns:host="urn:ietf:params:xml:ns:host-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
        <host:add></host:add>
        <host:rem></host:rem>
        <host:chg></host:chg>
      </host:update>
    </update>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Note: removed extension from testing and adjusted to add
// contact:status close tags
//
// <extension>
//
//	<namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
//	  <namestoreExt:subProduct>EXAMPLE</namestoreExt:subProduct>
//	</namestoreExt:namestoreExt>
//
// </extension>

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactUpdate = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <contact:update xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
        <contact:add>
          <contact:status s="clientDeleteProhibited"></contact:status>
        </contact:add>
        <contact:rem>
          <contact:status s="clientUpdateProhibited"></contact:status>
        </contact:rem>
        <contact:chg>
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
        </contact:chg>
      </contact:update>
    </update>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Note: removed extension from testing
//
// <extension>
//
//	<namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
//	  <namestoreExt:subProduct>EXAMPLE</namestoreExt:subProduct>
//	</namestoreExt:namestoreExt>
//
// </extension>

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactUpdateEmpty = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <update>
      <contact:update xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
        <contact:add></contact:add>
        <contact:rem></contact:rem>
        <contact:chg></contact:chg>
      </contact:update>
    </update>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
