package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetEPPDomainTransferRequest(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain transfer request message with the same parameters as the default in the Verisign testing tool", t, func() {
		period := GetEPPDomainPeriod(DomainPeriodYear, 1)
		verisignEPP := GetEPPDomainTransferRequest("example.com", period, "sampleAuthInfo-1", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferRequest)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain transfer request message with the same parameters (switchted to dot net) as the default in the Verisign testing tool", t, func() {
		period := GetEPPDomainPeriod(DomainPeriodYear, 1)
		verisignEPP := GetEPPDomainTransferRequest("example.net", period, "sampleAuthInfo-1", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferRequestDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPDomainTransferQuery(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain transfer query message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainTransferQuery("example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferQuery)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain transfer query message with the same parameters (switchted to dot net) as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainTransferQuery("example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferQueryDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPDomainTransferApprove(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain transfer approve message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainTransferApprove("example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferApprove)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain transfer approve message with the same parameters (switchted to dot net) as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainTransferApprove("example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferApproveDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPDomainTransferReject(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain transfer reject message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainTransferReject("example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferReject)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain transfer reject message with the same parameters (switchted to dot net) as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainTransferReject("example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferRejectDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPDomainTransferCancel(t *testing.T) {
	t.Parallel()
	Convey("Creating a domain transfer cancel message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainTransferCancel("example.com", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferCancel)
			So(eppErr, ShouldBeNil)
		})
	})

	Convey("Creating a domain transfer cancel message with the same parameters (switchted to dot net) as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPDomainTransferCancel("example.net", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignDomainTransferCancelDotNet)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactTransferRequest(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact transfer request message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPContactTransferRequest("8013", "sampleAuthInfo-1", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactTransferRequest)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactTransferQuery(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact transfer query message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPContactTransferQuery("8013", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactTransferQuery)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactTransferApprove(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact transfer approve message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPContactTransferApprove("8013", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactTransferApprove)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactTransferReject(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact transfer reject message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPContactTransferReject("8013", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactTransferReject)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetEPPContactTransferCancel(t *testing.T) {
	t.Parallel()
	Convey("Creating a contact transfer cancel message with the same parameters as the default in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPContactTransferCancel("8013", "ABC-12345-XYZ")
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignContactTransferCancel)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestTransferTypedMessageDomainRequest(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "domain.request", verisignDomainTransferRequest, CommandTransferDomainRequestType)
}

func TestTransferTypedMessageDomainQuery(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "domain.query", verisignDomainTransferQuery, CommandTransferDomainQueryType)
}

func TestTransferTypedMessageDomainApprove(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "domain.approve", verisignDomainTransferApprove, CommandTransferDomainApproveType)
}

func TestTransferTypedMessageDomainReject(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "domain.reject", verisignDomainTransferReject, CommandTransferDomainRejectType)
}

func TestTransferTypedMessageDomainCancel(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "domain.cancel", verisignDomainTransferCancel, CommandTransferDomainCancelType)
}

func TestTransferTypedMessageContactRequest(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "contact.request", verisignContactTransferRequest, CommandTransferContactRequestType)
}

func TestTransferTypedMessageContactQuery(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "contact.query", verisignContactTransferQuery, CommandTransferContactQueryType)
}

func TestTransferTypedMessageContactApprove(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "contact.approve", verisignContactTransferApprove, CommandTransferContactApproveType)
}

func TestTransferTypedMessageContactReject(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "contact.reject", verisignContactTransferReject, CommandTransferContactRejectType)
}

func TestTransferTypedMessageContactCancel(t *testing.T) {
	t.Parallel()
	UnMashalMarshalTest(t, "transfer", "contact.cancel", verisignContactTransferCancel, CommandTransferContactCancelType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferRequest = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="request">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:period unit="y">1</domain:period>
        <domain:authInfo>
          <domain:pw>sampleAuthInfo-1</domain:pw>
        </domain:authInfo>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferRequestDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="request">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
        <domain:period unit="y">1</domain:period>
        <domain:authInfo>
          <domain:pw>sampleAuthInfo-1</domain:pw>
        </domain:authInfo>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferQuery = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="query">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferQueryDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="query">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferApprove = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="approve">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferApproveDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="approve">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferReject = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="reject">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferRejectDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="reject">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferCancel = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="cancel">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignDomainTransferCancelDotNet = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="cancel">
      <domain:transfer xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.NET</domain:name>
      </domain:transfer>
    </transfer>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Note: For all contact transfer operations the extension object has
// been removed. An example can be found below
//
// <extension>
//
//	<namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
//	  <namestoreExt:subProduct>EXAMPLE</namestoreExt:subProduct>
//	</namestoreExt:namestoreExt>
//
// </extension>

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactTransferRequest = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="request">
      <contact:transfer xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
        <contact:authInfo>
          <contact:pw>sampleAuthInfo-1</contact:pw>
        </contact:authInfo>
      </contact:transfer>
    </transfer>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactTransferQuery = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="query">
      <contact:transfer xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
      </contact:transfer>
    </transfer>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactTransferApprove = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="approve">
      <contact:transfer xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
      </contact:transfer>
    </transfer>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactTransferReject = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="reject">
      <contact:transfer xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
      </contact:transfer>
    </transfer>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignContactTransferCancel = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <command xmlns="urn:ietf:params:xml:ns:epp-1.0">
    <transfer op="cancel">
      <contact:transfer xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
      </contact:transfer>
    </transfer>
    <clTRID>ABC-12345-XYZ</clTRID>
  </command>
</epp>`
