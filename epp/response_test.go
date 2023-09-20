package epp

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetObjectType(t *testing.T) {
	t.Parallel()
	Convey("A Info response object that has a domain namespace", t, func() {
		ir := GenericInfDataResp{}
		ir.XMLNSDomain = DomainXMLNS
		Convey("Should return domain", func() {
			So(ir.GetObjectType(), ShouldEqual, "domain")
		})
	})

	Convey("A Info response object that has a host namespace", t, func() {
		ir := GenericInfDataResp{}
		ir.XMLNSHost = HostXMLNS
		Convey("Should return host", func() {
			So(ir.GetObjectType(), ShouldEqual, "host")
		})
	})

	Convey("A Info response object that has a contact namespace", t, func() {
		ir := GenericInfDataResp{}
		ir.XMLNSContact = ContactXMLNS
		Convey("Should return contact", func() {
			So(ir.GetObjectType(), ShouldEqual, "contact")
		})
	})

	Convey("A Info response object that does not have a host, domain or contact namespace", t, func() {
		ir := GenericInfDataResp{}
		Convey("Should return an empty string for the object type", func() {
			So(ir.GetObjectType(), ShouldEqual, "")
		})
	})
}

const CommandFailedMessage = "Command failed; server closing connection"

func TestIsError(t *testing.T) {
	t.Parallel()
	Convey("Given a response with an error set (code 2500)", t, func() {
		resp := Response{}
		resp.Result = &Result{}
		resp.Result.Code = 2500
		resp.Result.Msg = CommandFailedMessage
		Convey("IsError should return true", func() {
			So(resp.IsError(), ShouldBeTrue)
		})
	})

	Convey("Given a response without an error set (code 1000)", t, func() {
		resp := Response{}
		resp.Result = &Result{}
		resp.Result.Code = 1000
		resp.Result.Msg = ""
		Convey("IsError should return false", func() {
			So(resp.IsError(), ShouldBeFalse)
		})
	})
}

func TestIsFatalError(t *testing.T) {
	t.Parallel()
	Convey("Given a response with an error set (code 2500)", t, func() {
		resp := Response{}
		resp.Result = &Result{}
		resp.Result.Code = 2500
		resp.Result.Msg = CommandFailedMessage
		Convey("IsFatalError should return true", func() {
			So(resp.IsFatalError(), ShouldBeTrue)
		})
	})

	Convey("Given a response with an error set (code 2500)", t, func() {
		resp := Response{}
		resp.Result = &Result{}
		resp.Result.Code = 2400
		resp.Result.Msg = "Command failed"
		Convey("IsFatalError should return false", func() {
			So(resp.IsFatalError(), ShouldBeFalse)
		})
	})

	Convey("Given a response without an error set (code 1000)", t, func() {
		resp := Response{}
		resp.Result = &Result{}
		resp.Result.Code = 1000
		resp.Result.Msg = ""
		Convey("IsFatalError should return false", func() {
			So(resp.IsFatalError(), ShouldBeFalse)
		})
	})
}

func TestGetError(t *testing.T) {
	t.Parallel()
	Convey("Given a response with an error set (code 2500)", t, func() {
		errorMessage := CommandFailedMessage
		resp := Response{}
		resp.Result = &Result{}
		resp.Result.Code = 2500
		resp.Result.Msg = errorMessage
		Convey("GetError should return an error with the provided error message", func() {
			So(resp.GetError(), ShouldNotBeNil)
			So(resp.GetError().Error(), ShouldEqual, errorMessage)
		})
	})

	Convey("Given a response without an error set (code 1000)", t, func() {
		resp := Response{}
		resp.Result = &Result{}
		resp.Result.Code = 1000
		resp.Result.Msg = ""
		Convey("GetError should return nil", func() {
			So(resp.GetError(), ShouldBeNil)
		})
	})
}

func TestEPPToTypedResponseNonResponse(t *testing.T) {
	t.Parallel()
	Convey("Given a Non-Response object", t, func() {
		epp := GetEPPHello()
		tMsg := epp.TypedMessage()
		Convey("ToTypedResponse should return the same object", func() {
			eppStr, eppErr := epp.ToString()
			msgStr, msgErr := tMsg.ToString()
			So(eppErr, ShouldBeNil)
			So(msgErr, ShouldBeNil)
			So(eppStr, ShouldEqual, msgStr)
		})
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainAvailabilityCheckResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:chkData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:cd>
          <domain:name avail="1">EXAMPLE.COM</domain:name>
        </domain:cd>
      </domain:chkData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>12345-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainAvailabilityCheckResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Availability Check", verisignEPPDomainAvailabilityCheckResponse)
	UnMashalMarshalTest(t, "response", "domain availability check", verisignEPPDomainAvailabilityCheckResponse, ResponseDomainCheckType)
}

// Note: Added xmlns:xsi to extension>namestoreExt.
// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainClaimsCheckResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <extension>
      <launch:chkData xmlns:launch="urn:ietf:params:xml:ns:launch-1.0">
        <launch:phase>claims</launch:phase>
        <launch:cd>
          <launch:name exists="1">EXAMPLE.COM</launch:name>
          <launch:claimKey validatorID="tmch">RVhBTVBMRS5DT00=</launch:claimKey>
        </launch:cd>
      </launch:chkData>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-91367-XYZ</clTRID>
      <svTRID>54321-XYZ</svTRID>
    </trID>
  </response>
</epp>`

// TODO: Implement claims checking for gtld launch.
func TestEPPDomainClaimsCheckResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Claims Check", verisignEPPDomainClaimsCheckResponse)
	UnMashalMarshalTest(t, "response", "domain claims check", verisignEPPDomainClaimsCheckResponse, ResponseType)
}

// Note: Added xmlns:xsi to extension>namestoreExt.
// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainCreateResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:creData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:crDate>2015-06-29T18:35:59.380Z</domain:crDate>
        <domain:exDate>2015-06-29T18:35:59.380Z</domain:exDate>
      </domain:creData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainCreateResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Create", verisignEPPDomainCreateResponse)
	UnMashalMarshalTest(t, "response", "domain create", verisignEPPDomainCreateResponse, ResponseDomainCreateType)
}

// Note: Added xmlns:xsi to extension>namestoreExt.
// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainInfoResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:infData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>example.com</domain:name>
        <domain:roid>NS1EXAMPLE1-VRSN</domain:roid>
        <domain:status s="ok"></domain:status>
        <domain:status s="clientHold"></domain:status>
        <domain:registrant>jd1234</domain:registrant>
        <domain:contact type="admin">sh8013</domain:contact>
        <domain:contact type="billing">sh8013</domain:contact>
        <domain:contact type="tech">sh8013</domain:contact>
        <domain:ns>
          <domain:hostObj>ns1.example.com</domain:hostObj>
          <domain:hostObj>ns2.example.com</domain:hostObj>
        </domain:ns>
        <domain:host>ns1.example.com</domain:host>
        <domain:host>ns2.example.com</domain:host>
        <domain:clID>ClientX</domain:clID>
        <domain:crID>ClientY</domain:crID>
        <domain:crDate>2015-06-29T20:58:22.670Z</domain:crDate>
        <domain:upID>ClientZ</domain:upID>
        <domain:upDate>2015-07-29T20:58:22.670Z</domain:upDate>
        <domain:exDate>2016-06-29T20:58:22.670Z</domain:exDate>
        <domain:authInfo>
          <domain:pw>2fooBAR</domain:pw>
        </domain:authInfo>
      </domain:infData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
      <secDNS:infData xmlns:secDNS="urn:ietf:params:xml:ns:secDNS-1.1" xsi:schemaLocation="urn:ietf:params:xml:ns:secDNS-1.1 secDNS-1.1.xsd">
        <secDNS:dsData>
          <secDNS:keyTag>34095</secDNS:keyTag>
          <secDNS:alg>5</secDNS:alg>
          <secDNS:digestType>1</secDNS:digestType>
          <secDNS:digest>6BD4FFFF11566D6E6A5BA44ED0018797564AA289</secDNS:digest>
        </secDNS:dsData>
        <secDNS:dsData>
          <secDNS:keyTag>10563</secDNS:keyTag>
          <secDNS:alg>5</secDNS:alg>
          <secDNS:digestType>1</secDNS:digestType>
          <secDNS:digest>9C20674BFF957211D129B0DFE9410AF753559D4B</secDNS:digest>
        </secDNS:dsData>
      </secDNS:infData>
    </extension>
    <trID>
      <clTRID>ABC-26599-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainInfoResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Info", verisignEPPDomainInfoResponse)
	UnMashalMarshalTest(t, "response", "domain info", verisignEPPDomainInfoResponse, ResponseDomainInfoType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainUpdateResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainUpdateResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Update", verisignEPPDomainUpdateResponse)
	UnMashalMarshalTest(t, "response", "domain update", verisignEPPDomainUpdateResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDoaminDeleteResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainDeleteResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Delete", verisignEPPDoaminDeleteResponse)
	UnMashalMarshalTest(t, "response", "domain delete", verisignEPPDoaminDeleteResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainRenewResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:renData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:exDate>2015-06-30T15:31:32.389Z</domain:exDate>
      </domain:renData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainRenewResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Renew", verisignEPPDomainRenewResponse)
	UnMashalMarshalTest(t, "response", "domain renew", verisignEPPDomainRenewResponse, ResponseDomainRenewType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainTransferRequestResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:trnData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:trStatus>pending</domain:trStatus>
        <domain:reID>ClientX</domain:reID>
        <domain:reDate>2015-06-30T15:42:25.461Z</domain:reDate>
        <domain:acID>ClientY</domain:acID>
        <domain:acDate>2015-06-30T15:42:25.461Z</domain:acDate>
        <domain:exDate>2015-06-30T15:42:25.461Z</domain:exDate>
      </domain:trnData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainTransferRequestResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Transfer Request", verisignEPPDomainTransferRequestResponse)
	UnMashalMarshalTest(t, "response", "domain transfer request", verisignEPPDomainTransferRequestResponse, ResponseDomainTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainTransferQueryResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:trnData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:trStatus>pending</domain:trStatus>
        <domain:reID>ClientX</domain:reID>
        <domain:reDate>2015-06-30T15:44:02.007Z</domain:reDate>
        <domain:acID>ClientY</domain:acID>
        <domain:acDate>2015-06-30T15:44:02.007Z</domain:acDate>
        <domain:exDate>2015-06-30T15:44:02.007Z</domain:exDate>
      </domain:trnData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainTransferQueryResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Transfer Query", verisignEPPDomainTransferQueryResponse)
	UnMashalMarshalTest(t, "response", "domain transfer query", verisignEPPDomainTransferQueryResponse, ResponseDomainTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainTransferApproveResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:trnData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:trStatus>pending</domain:trStatus>
        <domain:reID>ClientX</domain:reID>
        <domain:reDate>2015-06-30T15:45:16.408Z</domain:reDate>
        <domain:acID>ClientY</domain:acID>
        <domain:acDate>2015-06-30T15:45:16.408Z</domain:acDate>
        <domain:exDate>2015-06-30T15:45:16.408Z</domain:exDate>
      </domain:trnData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainTransferApproveResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Transfer Approve", verisignEPPDomainTransferApproveResponse)
	UnMashalMarshalTest(t, "response", "domain transfer approve", verisignEPPDomainTransferApproveResponse, ResponseDomainTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainTransferRejectResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:trnData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:trStatus>pending</domain:trStatus>
        <domain:reID>ClientX</domain:reID>
        <domain:reDate>2015-06-30T15:48:34.277Z</domain:reDate>
        <domain:acID>ClientY</domain:acID>
        <domain:acDate>2015-06-30T15:48:34.277Z</domain:acDate>
        <domain:exDate>2015-06-30T15:48:34.277Z</domain:exDate>
      </domain:trnData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainTransferRejectResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Transfer Reject", verisignEPPDomainTransferRejectResponse)
	UnMashalMarshalTest(t, "response", "domain transfer reject", verisignEPPDomainTransferRejectResponse, ResponseDomainTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainTransferCancelResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:trnData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:trStatus>pending</domain:trStatus>
        <domain:reID>ClientX</domain:reID>
        <domain:reDate>2015-06-30T15:46:35.299Z</domain:reDate>
        <domain:acID>ClientY</domain:acID>
        <domain:acDate>2015-06-30T15:46:35.299Z</domain:acDate>
        <domain:exDate>2015-06-30T15:46:35.299Z</domain:exDate>
      </domain:trnData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainTransferCancelResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Transfer Cancel", verisignEPPDomainTransferCancelResponse)
	UnMashalMarshalTest(t, "response", "domain transfer cancel", verisignEPPDomainTransferCancelResponse, ResponseDomainTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainSyncResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainSyncResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Sync", verisignEPPDomainSyncResponse)
	UnMashalMarshalTest(t, "response", "domain sync", verisignEPPDomainSyncResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainRestoreRequestResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
      <rgp:upData xmlns:rgp="urn:ietf:params:xml:ns:rgp-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:rgp-1.0 rgp-1.0.xsd">
        <rgp:rgpStatus s="pendingRestore"></rgp:rgpStatus>
      </rgp:upData>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainRestoreRequestResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Restore Request", verisignEPPDomainRestoreRequestResponse)
	UnMashalMarshalTest(t, "response", "domain restore request", verisignEPPDomainRestoreRequestResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainRestoreReportResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainRestoreReportResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Restore Report", verisignEPPDomainRestoreReportResponse)
	UnMashalMarshalTest(t, "response", "domain restore report", verisignEPPDomainRestoreReportResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPHostCheckResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <host:chkData xmlns:host="urn:ietf:params:xml:ns:host-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:cd>
          <host:name avail="0">NS1.EXAMPLE.COM</host:name>
        </host:cd>
      </host:chkData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPHostCheckResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Host Check", verisignEPPHostCheckResponse)
	UnMashalMarshalTest(t, "response", "host check", verisignEPPHostCheckResponse, ResponseHostCheckType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPHostCreateResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <host:creData xmlns:host="urn:ietf:params:xml:ns:host-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
        <host:crDate>2015-06-30T17:25:49.450Z</host:crDate>
      </host:creData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPHostCreateResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Host Create", verisignEPPHostCreateResponse)
	UnMashalMarshalTest(t, "response", "host create", verisignEPPHostCreateResponse, ResponseHostCreateType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPHostInfoResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <host:infData xmlns:host="urn:ietf:params:xml:ns:host-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd">
        <host:name>NS1.EXAMPLE.COM</host:name>
        <host:roid>NS1EXAMPLE1-VRSN</host:roid>
        <host:status s="linked"></host:status>
        <host:status s="clientUpdateProhibited"></host:status>
        <host:addr ip="v4">192.1.2.3</host:addr>
        <host:clID>clientidX</host:clID>
        <host:crID>createdByLloyd</host:crID>
        <host:crDate>2015-06-30T17:27:08.958Z</host:crDate>
        <host:upID>updatedX</host:upID>
        <host:upDate>2015-06-30T17:27:08.959Z</host:upDate>
      </host:infData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPHostInfoResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Host Info", verisignEPPHostInfoResponse)
	UnMashalMarshalTest(t, "response", "host info", verisignEPPHostInfoResponse, ResponseHostInfoType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPHostUpdateResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPHostUpdateResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Host Update", verisignEPPHostUpdateResponse)
	UnMashalMarshalTest(t, "response", "host update", verisignEPPHostUpdateResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPHostDeleteResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPHostDeleteResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Host Delete", verisignEPPHostDeleteResponse)
	UnMashalMarshalTest(t, "response", "host delete", verisignEPPHostDeleteResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactCheckResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:chkData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:cd>
          <contact:id avail="0">8013</contact:id>
        </contact:cd>
      </contact:chkData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactCheckResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Check", verisignEPPContactCheckResponse)
	UnMashalMarshalTest(t, "response", "contact check", verisignEPPContactCheckResponse, ResponseContactCheckType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactCreateResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:creData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>8013</contact:id>
        <contact:crDate>2015-06-30T18:47:57.968Z</contact:crDate>
      </contact:creData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactCreateResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Create", verisignEPPContactCreateResponse)
	UnMashalMarshalTest(t, "response", "contact create", verisignEPPContactCreateResponse, ResponseContactCreateType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactInfoResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:infData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>sh8013</contact:id>
        <contact:roid>SH8013-VRSN</contact:roid>
        <contact:status s="linked"></contact:status>
        <contact:status s="clientUpdateProhibited"></contact:status>
        <contact:postalInfo type="loc">
          <contact:name>John Doe</contact:name>
          <contact:org>Example Inc.</contact:org>
          <contact:addr>
            <contact:street>123 Example Dr.</contact:street>
            <contact:street>Suite 100</contact:street>
            <contact:city>Dulles</contact:city>
            <contact:sp>VA</contact:sp>
            <contact:pc>20166-6503</contact:pc>
            <contact:cc>US</contact:cc>
          </contact:addr>
        </contact:postalInfo>
        <contact:postalInfo type="int">
          <contact:name>i15d John Doe</contact:name>
          <contact:org>i15d Example Inc.</contact:org>
          <contact:addr>
            <contact:street>i15d 123 Example Dr.</contact:street>
            <contact:street>i15d Suite 100</contact:street>
            <contact:city>Dulles</contact:city>
            <contact:sp>VA</contact:sp>
            <contact:pc>20166-6503</contact:pc>
            <contact:cc>US</contact:cc>
          </contact:addr>
        </contact:postalInfo>
        <contact:voice x="123">+1.7035555555</contact:voice>
        <contact:fax x="456">+1.7035555556</contact:fax>
        <contact:email>jdoe@example.com</contact:email>
        <contact:clID>ClientY</contact:clID>
        <contact:crID>ClientX</contact:crID>
        <contact:crDate>2015-06-30T18:48:34.837Z</contact:crDate>
        <contact:authInfo>
          <contact:pw>2fooBAR</contact:pw>
        </contact:authInfo>
        <contact:disclose flag="1">
          <contact:name type="int"></contact:name>
          <contact:org type="loc"></contact:org>
          <contact:org type="int"></contact:org>
          <contact:addr type="loc"></contact:addr>
          <contact:addr type="int"></contact:addr>
          <contact:voice></contact:voice>
          <contact:fax></contact:fax>
          <contact:email></contact:email>
        </contact:disclose>
      </contact:infData>
    </resData>
    <extension>
      <jobsContact:infData xmlns:jobsContact="http://www.verisign.com/epp/jobsContact-1.0" xsi:schemaLocation="http://www.verisign.com/epp/jobsContact-1.0 jobsContact-1.0.xsd">
        <jobsContact:title>Title</jobsContact:title>
        <jobsContact:website>http://www.verisign.jobs</jobsContact:website>
        <jobsContact:industryType>IT</jobsContact:industryType>
        <jobsContact:isAdminContact>Yes</jobsContact:isAdminContact>
        <jobsContact:isAssociationMember>Yes</jobsContact:isAssociationMember>
      </jobsContact:infData>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactInfoResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Info", verisignEPPContactInfoResponse)
	UnMashalMarshalTest(t, "response", "contact into", verisignEPPContactInfoResponse, ResponseContactInfoType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactUpdateResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactUpdateResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Update", verisignEPPContactUpdateResponse)
	UnMashalMarshalTest(t, "response", "contact update", verisignEPPContactUpdateResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactDeleteResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactDeleteResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Delete", verisignEPPContactDeleteResponse)
	UnMashalMarshalTest(t, "response", "contact delete", verisignEPPContactDeleteResponse, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactTransferRequestResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:trnData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>SH0000</contact:id>
        <contact:trStatus>pending</contact:trStatus>
        <contact:reID>ClientX</contact:reID>
        <contact:reDate>2015-06-30T18:51:08.497Z</contact:reDate>
        <contact:acID>ClientY</contact:acID>
        <contact:acDate>2015-06-30T18:51:08.497Z</contact:acDate>
      </contact:trnData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactTransferRequestResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Transfer Request", verisignEPPContactTransferRequestResponse)
	UnMashalMarshalTest(t, "response", "contact transfer request", verisignEPPContactTransferRequestResponse, ResponseContactTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactTransferQueryResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:trnData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>SH0000</contact:id>
        <contact:trStatus>pending</contact:trStatus>
        <contact:reID>ClientX</contact:reID>
        <contact:reDate>2015-06-30T18:51:36.238Z</contact:reDate>
        <contact:acID>ClientY</contact:acID>
        <contact:acDate>2015-06-30T18:51:36.238Z</contact:acDate>
      </contact:trnData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactTransferQueryResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Transfer Query", verisignEPPContactTransferQueryResponse)
	UnMashalMarshalTest(t, "response", "contact transfer query", verisignEPPContactTransferQueryResponse, ResponseContactTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactTransferApproveResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:trnData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>SH0000</contact:id>
        <contact:trStatus>pending</contact:trStatus>
        <contact:reID>ClientX</contact:reID>
        <contact:reDate>2015-06-30T18:52:07.496Z</contact:reDate>
        <contact:acID>ClientY</contact:acID>
        <contact:acDate>2015-06-30T18:52:07.496Z</contact:acDate>
      </contact:trnData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactTransferApproveResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Transfer Approve", verisignEPPContactTransferApproveResponse)
	UnMashalMarshalTest(t, "response", "contact transfer approve", verisignEPPContactTransferApproveResponse, ResponseContactTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactTransferRejectResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:trnData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>SH0000</contact:id>
        <contact:trStatus>pending</contact:trStatus>
        <contact:reID>ClientX</contact:reID>
        <contact:reDate>2015-06-30T18:53:04.506Z</contact:reDate>
        <contact:acID>ClientY</contact:acID>
        <contact:acDate>2015-06-30T18:53:04.506Z</contact:acDate>
      </contact:trnData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactTransferRejectResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Transfer Reject", verisignEPPContactTransferRejectResponse)
	UnMashalMarshalTest(t, "response", "contact transfer reject", verisignEPPContactTransferRejectResponse, ResponseContactTransferType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPContactTransferCancelResponse = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <contact:trnData xmlns:contact="urn:ietf:params:xml:ns:contact-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd">
        <contact:id>SH0000</contact:id>
        <contact:trStatus>pending</contact:trStatus>
        <contact:reID>ClientX</contact:reID>
        <contact:reDate>2015-06-30T18:53:31.016Z</contact:reDate>
        <contact:acID>ClientY</contact:acID>
        <contact:acDate>2015-06-30T18:53:31.016Z</contact:acDate>
      </contact:trnData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPContactTransferCancelResponse(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Contact Transfer Cancel", verisignEPPContactTransferCancelResponse)
	UnMashalMarshalTest(t, "response", "contact transfer cancel", verisignEPPContactTransferCancelResponse, ResponseContactTransferType)
}

func ResponseTransformHelper(t *testing.T, msgType string, originalMessage string) {
	t.Helper()
	Convey(fmt.Sprintf("Given a response from an %s", msgType), t, func() {
		msg, err := UnmarshalMessage([]byte(originalMessage))
		So(err, ShouldBeNil)
		tMsg := msg.TypedMessage()
		Convey("The typed response should match the response", func() {
			str, err := tMsg.ToString()
			So(err, ShouldBeNil)
			So(str, ShouldEqual, originalMessage)
		})
	})
}

func TestToTypeInfDataResp(t *testing.T) {
	t.Parallel()
	Convey("Each of the To<type>InfDataResp functions can return an error, test to make sure they do", t, func() {
		genericInfo := GenericInfDataResp{}
		Convey("ToDomainInfDataResp should return an error", func() {
			_, err := genericInfo.ToDomainInfDataResp()
			So(err, ShouldNotBeNil)
		})

		Convey("ToHostInfDataResp should return an error", func() {
			_, err := genericInfo.ToHostInfDataResp()
			So(err, ShouldNotBeNil)
		})

		Convey("ToContactInfDataResp should return an error", func() {
			_, err := genericInfo.ToContactInfDataResp()
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGetEPPResponseResult(t *testing.T) {
	t.Parallel()
	Convey("Creating a response result object with the same parameters as the default 1000 response result in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPResponseResult("ABC-12345-XYZ", "SRV-12345-XYZ", 1000, ResponseCode1000)
		Convey("The output should match the output from the testing tool", func() {
			eppStr, eppErr := verisignEPP.ToString()
			So(eppStr, ShouldEqual, verisignEPPResponse1000)
			So(eppErr, ShouldBeNil)
		})
	})
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPResponse1000 = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>SRV-12345-XYZ</svTRID>
    </trID>
  </response>
</epp>`

// verisignEPPDomainOverloaded is used to make sure that responses that
// multiple result data objects will return the generic ResponseType.
// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPDomainOverloaded = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1000">
      <msg>Command completed successfully</msg>
    </result>
    <resData>
      <domain:creData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:crDate>2015-06-29T18:35:59.380Z</domain:crDate>
        <domain:exDate>2015-06-29T18:35:59.380Z</domain:exDate>
      </domain:creData>
      <domain:renData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:exDate>2015-06-30T15:31:32.389Z</domain:exDate>
      </domain:renData>
    </resData>
    <extension>
      <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
        <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
      </namestoreExt:namestoreExt>
    </extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPDomainOverloaded(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Domain Overloaded", verisignEPPDomainOverloaded)
	UnMashalMarshalTest(t, "response", "domain overloaded", verisignEPPDomainOverloaded, ResponseType)
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPMessageQTransferApproved = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1301">
      <msg>Command completed successfully; ack to dequeue</msg>
    </result>
    <msgQ count="2" id="182575">
      <qDate>2016-04-05T20:53:25Z</qDate>
      <msg>Transfer Approved.</msg>
    </msgQ>
    <resData>
      <domain:trnData xmlns:domain="urn:ietf:params:xml:ns:domain-1.0" xsi:schemaLocation="urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd">
        <domain:name>EXAMPLE.COM</domain:name>
        <domain:trStatus>clientApproved</domain:trStatus>
        <domain:reID>204218</domain:reID>
        <domain:reDate>2016-04-05T20:45:13Z</domain:reDate>
        <domain:acID>goreg1-admin</domain:acID>
        <domain:acDate>2016-04-05T20:53:25Z</domain:acDate>
        <domain:exDate>2018-04-05T19:53:34Z</domain:exDate>
      </domain:trnData>
    </resData>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignEPPMessageQUnusedObjectsPolicy = `<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
  <response>
    <result code="1301">
      <msg>Command completed successfully; ack to dequeue</msg>
    </result>
    <msgQ count="33" id="79900023">
      <qDate>2019-12-20T16:12:48Z</qDate>
      <msg>Unused objects policy</msg>
    </msgQ>
    <resData>
      <host:infData xmlns:host="urn:ietf:params:xml:ns:host-1.0" xsi:schemaLocation="">
        <host:name>NS1.EXAMPLE.COM</host:name>
        <host:roid>NS1EXAMPLE1-VRSN</host:roid>
        <host:status s="clientDeleteProhibited"></host:status>
        <host:status s="clientUpdateProhibited"></host:status>
        <host:addr ip="v4">192.1.2.3</host:addr>
        <host:clID>clientidX</host:clID>
        <host:crID>createdByLloyd</host:crID>
        <host:crDate>2015-06-30T17:27:08.958Z</host:crDate>
        <host:upID>updatedX</host:upID>
        <host:upDate>2015-06-30T17:27:08.959Z</host:upDate>
      </host:infData>
    </resData>
    <extension></extension>
    <trID>
      <clTRID>ABC-12345-XYZ</clTRID>
      <svTRID>54322-XYZ</svTRID>
    </trID>
  </response>
</epp>`

func TestEPPPoll(t *testing.T) {
	t.Parallel()
	ResponseTransformHelper(t, "EPP Poll Transfer Approved", verisignEPPMessageQTransferApproved)
	UnMashalMarshalTest(t, "response", "epp poll", verisignEPPMessageQTransferApproved, ResponseDomainTransferType)

	Convey("Given a transfer approved poll response message", t, func() {
		raw, err := UnmarshalMessage([]byte(verisignEPPMessageQTransferApproved))
		So(err, ShouldBeNil)
		typed := raw.TypedMessage()
		So(typed.ResponseObject.HasMessageQueue(), ShouldBeTrue)
	})

	ResponseTransformHelper(t, "EPP Poll Unused Objects Policy", verisignEPPMessageQUnusedObjectsPolicy)
	UnMashalMarshalTest(t, "response", "epp poll", verisignEPPMessageQUnusedObjectsPolicy, ResponseHostInfoType)

	Convey("Given an unused objects policy poll response message", t, func() {
		raw, err := UnmarshalMessage([]byte(verisignEPPMessageQUnusedObjectsPolicy))
		So(err, ShouldBeNil)
		typed := raw.TypedMessage()
		So(typed.ResponseObject.HasMessageQueue(), ShouldBeTrue)
		So(typed.ResponseObject.MessageQueue.Message, ShouldEqual, "Unused objects policy")
		So(typed.ResponseObject.ResultData.HostInfDataResp.Name, ShouldEqual, "NS1.EXAMPLE.COM")
	})

	Convey("Given a non poll response message", t, func() {
		raw, err := UnmarshalMessage([]byte(verisignEPPResponse1000))
		So(err, ShouldBeNil)
		typed := raw.TypedMessage()
		So(typed.ResponseObject.HasMessageQueue(), ShouldBeFalse)
	})
}
