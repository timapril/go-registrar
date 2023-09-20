package epp

import (
	"encoding/xml"
	"errors"
)

var (
	// ErrNotDomainResponse indicates that the object treated as a domain
	// response was some other object.
	ErrNotDomainResponse = errors.New("not domain response")

	// ErrNotHostResponse indicates that the object treated as a host
	// response was some other object.
	ErrNotHostResponse = errors.New("not host response")

	// ErrNotContactResponse indicates that the object treated as a contact
	// response was some other object.
	ErrNotContactResponse = errors.New("not contact response")
)

// GenericInfDataResp is used to receive a InfoResponse from a server.
type GenericInfDataResp struct {
	XMLNSDomain         string              `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSHost           string              `xml:"host,attr" json:"xmlns.host"`
	XMLNSContact        string              `xml:"contact,attr" json:"xmlns.contact"`
	XMLNsSchemaLocation string              `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string              `xml:"name" json:"name"`
	ID                  string              `xml:"id" json:"id"`
	ROID                string              `xml:"roid" json:"roid"`
	Status              []GenericStatus     `xml:"status" json:"status"`
	RegistrantID        string              `xml:"registrant" json:"registrant"`
	Contacts            []GenericContact    `xml:"contact" json:"contact"`
	NSHosts             []string            `xml:"ns>hostObj" json:"hostObj"`
	HostAddrs           []GenericHostAddr   `xml:"addr" json:"addr"`
	Hosts               []string            `xml:"host" json:"host"`
	ClientID            string              `xml:"clID" json:"clID"`
	CreateID            string              `xml:"crID" json:"crID"`
	CreateDate          string              `xml:"crDate" json:"crDate"`
	ExpireDate          string              `xml:"exDate" json:"exDate"`
	UpdateID            string              `xml:"upID" json:"upID"`
	UpdateDate          string              `xml:"upDate" json:"upDate"`
	TransferDate        string              `xml:"trDate" json:"trDate"`
	Voice               GenericPhone        `xml:"voice" json:"voice"`
	Fax                 GenericPhone        `xml:"fax" json:"fax"`
	AuthPW              string              `xml:"authInfo>pw" json:"authInfoPW"`
	Email               string              `xml:"email" json:"email"`
	Disclose            *GenericDisclose    `xml:"disclose" json:"disclosure"`
	PostalInfos         []GenericPostalInfo `xml:"postalInfo" json:"postalInfo"`
}

// GenericPhone is used receive a Phone number without the namespacing
// being paid attention to.
type GenericPhone struct {
	Extension string `xml:"x,attr" json:"x"`
	Number    string `xml:",chardata" json:"Number"`
}

// ToDomainInfDataResp attempts to copy the required values from
// the called oject into a DomainInfDataResp object.
func (g GenericInfDataResp) ToDomainInfDataResp() (d DomainInfDataResp, err error) {
	if g.XMLNSDomain != "" {
		d.XMLNSDomain = g.XMLNSDomain
		d.XMLNsSchemaLocation = g.XMLNsSchemaLocation
		d.Name = g.Name
		d.ROID = g.ROID
		d.RegistrantID = g.RegistrantID
		d.ClientID = g.ClientID
		d.CreateID = g.CreateID
		d.CreateDate = g.CreateDate
		d.UpdateID = g.UpdateID
		d.UpdateDate = g.UpdateDate
		d.ExpireDate = g.ExpireDate
		d.TransferDate = g.TransferDate

		if g.AuthPW != "" {
			d.AuthPW = &DomainAuth{}
			d.AuthPW.Password = g.AuthPW
		}

		d.Hosts = append(d.Hosts, g.Hosts...)

		for _, h := range g.NSHosts {
			d.NSHosts.Hosts = append(d.NSHosts.Hosts, DomainHost{Value: h})
		}

		for _, s := range g.Status {
			d.Status = append(d.Status, DomainStatus{StatusFlag: s.Value})
		}

		for _, c := range g.Contacts {
			var ct ContactType
			ct, err = ContactTypeFromString(c.Type)

			if err == nil {
				d.Contacts = append(d.Contacts, DomainContact{Type: ct, Value: c.Value})
			}
		}

		return d, nil
	}

	return d, ErrNotDomainResponse
}

// ToHostInfDataResp attempts to copy the required values from
// the called oject into a HostInfDataResp object.
func (g GenericInfDataResp) ToHostInfDataResp() (h HostInfDataResp, err error) {
	if g.XMLNSHost != "" {
		h.XMLNSHost = g.XMLNSHost
		h.XMLNsSchemaLocation = g.XMLNsSchemaLocation
		h.Name = g.Name
		h.ROID = g.ROID
		h.ClientID = g.ClientID
		h.CreateID = g.CreateID
		h.CreateDate = g.CreateDate
		h.UpdateID = g.UpdateID
		h.UpdateDate = g.UpdateDate
		h.TransferDate = g.TransferDate

		for _, s := range g.Status {
			h.Status = append(h.Status, HostStatus{StatusFlag: s.Value})
		}

		for _, a := range g.HostAddrs {
			h.Addresses = append(h.Addresses, HostAddress{IPVersion: a.IPVersion, Address: a.Address})
		}

		return h, nil
	}

	return h, ErrNotHostResponse
}

// ToContactInfDataResp attempts to copy the required values from
// the called oject into a ContactInfDataResp object.
func (g GenericInfDataResp) ToContactInfDataResp() (c ContactInfDataResp, err error) {
	if g.XMLNSContact != "" {
		c.XMLNSContact = g.XMLNSContact
		c.XMLNsSchemaLocation = g.XMLNsSchemaLocation
		c.ID = g.ID
		c.ROID = g.ROID
		c.Voice = PhoneNumber{Number: g.Voice.Number, Extension: g.Voice.Extension}
		c.Fax = PhoneNumber{Number: g.Fax.Number, Extension: g.Fax.Extension}
		c.Email = g.Email
		c.ClientID = g.ClientID
		c.CreateID = g.CreateID
		c.CreateDate = g.CreateDate
		c.UpdateID = g.UpdateID
		c.UpdateDate = g.UpdateDate
		c.TransferDate = g.TransferDate

		if g.AuthPW != "" {
			c.AuthPW = &ContactAuth{}
			c.AuthPW.Password = g.AuthPW
		}

		if g.Disclose != nil {
			dis := g.Disclose.ToContactDisclose()
			c.ContactDisclose = &dis
		}

		for _, s := range g.Status {
			c.Status = append(c.Status, ContactStatus{StatusFlag: s.Value})
		}

		for _, postalInfoObj := range g.PostalInfos {
			postalInfo := PostalInfo{}
			postalInfo.PostalInfoType = postalInfoObj.Type
			postalInfo.Name = postalInfoObj.Name
			postalInfo.Org = postalInfoObj.Org

			postalInfo.Address.Street = postalInfoObj.Addr.Streets
			postalInfo.Address.City = postalInfoObj.Addr.City
			postalInfo.Address.Sp = postalInfoObj.Addr.StateProv
			postalInfo.Address.Pc = postalInfoObj.Addr.PostalCode
			postalInfo.Address.Cc = postalInfoObj.Addr.Country
			c.PostalInfos = append(c.PostalInfos, postalInfo)
		}

		return c, nil
	}

	return c, ErrNotContactResponse
}

// GenericDisclose is used to receive a Disclose object from the server.
type GenericDisclose struct {
	Flag      int                 `xml:"flag,attr" json:"flag"`
	Names     []ContactTypedField `xml:"name" json:"name"`
	Orgs      []ContactTypedField `xml:"org" json:"org"`
	Addresses []ContactTypedField `xml:"addr" json:"addr"`
	Voice     string              `xml:"voice" json:"voice"`
	Fax       string              `xml:"fax" json:"fax"`
	Email     string              `xml:"email" json:"email"`
}

// GenericPostalInfo is used to receive a PostalInfo object from the
// server.
type GenericPostalInfo struct {
	Type string      `xml:"type,attr" json:"type"`
	Name string      `xml:"name" json:"name"`
	Org  string      `xml:"org,omitempty" json:"org"`
	Addr GenericAddr `xml:"addr" json:"addr"`
}

// GenericAddr is used to receive a PostalInfo>Address object from the
// server.
type GenericAddr struct {
	Streets    []string `xml:"street" json:"street"`
	City       string   `xml:"city" json:"city"`
	StateProv  string   `xml:"sp,omitempty" json:"sp"`
	PostalCode string   `xml:"pc,omitempty" json:"pc"`
	Country    string   `xml:"cc" json:"cc"`
}

// ToContactDisclose attempts to copy the required values from
// the called oject into a ContactDisclose object.
func (g GenericDisclose) ToContactDisclose() (c ContactDisclose) {
	c.Flag = g.Flag
	c.Voice = g.Voice
	c.Fax = g.Fax
	c.Email = g.Email
	c.Names = g.Names
	c.Orgs = g.Orgs
	c.Addresses = g.Addresses

	return
}

// ContactDisclose is used to represent a ContactDisclose section of a
// response with proper namespacing.
type ContactDisclose struct {
	Flag      int                 `xml:"flag,attr" json:"flag"`
	Names     []ContactTypedField `xml:"contact:name" json:"names"`
	Orgs      []ContactTypedField `xml:"contact:org" json:"orgs"`
	Addresses []ContactTypedField `xml:"contact:addr" json:"addresses"`
	Voice     string              `xml:"contact:voice" json:"voice"`
	Fax       string              `xml:"contact:fax" json:"fax"`
	Email     string              `xml:"contact:email" json:"email"`
}

// ContactTypedField is used to receive a Contact identifier with a type
// (usually "loc" or "int") from the server.
type ContactTypedField struct {
	Type string `xml:"type,attr" json:"type"`
}

// DomainInfDataResp is used to represent a domain:infData response
// object from the server.
type DomainInfDataResp struct {
	XMLNSDomain         string          `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNsSchemaLocation string          `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string          `xml:"domain:name" json:"domain.name"`
	ROID                string          `xml:"domain:roid" json:"domain:roid"`
	Status              []DomainStatus  `xml:"domain:status" json:"domain.status"`
	RegistrantID        string          `xml:"domain:registrant" json:"domain.registrant"`
	Contacts            []DomainContact `xml:"domain:contact" json:"domain.contact"`
	NSHosts             DomainHostList  `xml:"domain:ns" json:"domain.ns"`
	Hosts               []string        `xml:"domain:host" json:"domain.host"`
	ClientID            string          `xml:"domain:clID" json:"domain.clID"`
	CreateID            string          `xml:"domain:crID" json:"domain.crID"`
	CreateDate          string          `xml:"domain:crDate" json:"domain.crDate"`
	UpdateID            string          `xml:"domain:upID,omitempty" json:"domain.upID"`
	UpdateDate          string          `xml:"domain:upDate,omitempty" json:"domain.upDate"`
	ExpireDate          string          `xml:"domain:exDate" json:"domain.exDate"`
	TransferDate        string          `xml:"domain:trDate,omitempty" json:"domain.trDate"`
	AuthPW              *DomainAuth     `xml:"domain:authInfo" json:"domain.authInfo"`
}

// HostInfDataResp is used to represent a host:infData response
// object from the server.
type HostInfDataResp struct {
	XMLNSHost           string        `xml:"xmlns:host,attr" json:"xmlns.host"`
	XMLNsSchemaLocation string        `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string        `xml:"host:name" json:"host.name"`
	ROID                string        `xml:"host:roid" json:"host.roid"`
	Status              []HostStatus  `xml:"host:status" json:"host.status"`
	Addresses           []HostAddress `xml:"host:addr" json:"host.addr"`
	ClientID            string        `xml:"host:clID" json:"host.clID"`
	CreateID            string        `xml:"host:crID" json:"host.crID"`
	CreateDate          string        `xml:"host:crDate" json:"host.crDate"`
	UpdateID            string        `xml:"host:upID,omitempty" json:"host.upID"`
	UpdateDate          string        `xml:"host:upDate,omitempty" json:"upDate"`
	TransferDate        string        `xml:"host:trDate,omitempty" json:"trDate"`
}

// ContactInfDataResp is used to represent a contact:infData response
// object from the server.
type ContactInfDataResp struct {
	XMLNSContact        string           `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNsSchemaLocation string           `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	ID                  string           `xml:"contact:id" json:"contact.id"`
	ROID                string           `xml:"contact:roid" json:"contact.roid"`
	Status              []ContactStatus  `xml:"contact:status" json:"contact.status"`
	PostalInfos         []PostalInfo     `xml:"contact:postalInfo" json:"contact.postalInfo"`
	Voice               PhoneNumber      `xml:"contact:voice,omitempty" json:"contact.voice"`
	Fax                 PhoneNumber      `xml:"contact:fax,omitempty" json:"contact.fax"`
	Email               string           `xml:"contact:email" json:"contact.email"`
	ClientID            string           `xml:"contact:clID" json:"contact.clID"`
	CreateID            string           `xml:"contact:crID" json:"contact.crID"`
	CreateDate          string           `xml:"contact:crDate" json:"contact.crDate"`
	UpdateID            string           `xml:"contact:upID,omitempty" json:"contact.upID"`
	UpdateDate          string           `xml:"contact:upDate,omitempty" json:"contact.upDate"`
	TransferDate        string           `xml:"contact:trDate,omitempty" json:"contact.trDate"`
	AuthPW              *ContactAuth     `xml:"contact:authInfo,omitempty" json:"contact.authInfo"`
	ContactDisclose     *ContactDisclose `xml:"contact:disclose,omitempty" json:"contact.disclose"`
}

// GenericHostAddr is used to receive a Host Address object from the
// server.
type GenericHostAddr struct {
	IPVersion IPVersionName `xml:"ip,attr" json:"ip"`
	Address   string        `xml:",chardata" json:"address"`
}

// GenericContact is used to receive information related to a contact.
type GenericContact struct {
	Type  string `xml:"type,attr" json:"type"`
	Value string `xml:",chardata" json:"value"`
}

// GenericStatus is used to receive object status indicators from the
// server.
type GenericStatus struct {
	Value string `xml:"s,attr" json:"value"`
}

// GetObjectType is used to determine which type of info response object
// was created.
func (g GenericInfDataResp) GetObjectType() string {
	if g.XMLNSDomain != "" {
		return "domain"
	}

	if g.XMLNSHost != "" {
		return "host"
	}

	if g.XMLNSContact != "" {
		return "contact"
	}

	return ""
}

// MessageQueue is used to construct and receive <msgQ> messages as part
// of a respons message.
type MessageQueue struct {
	XMLName xml.Name `xml:"msgQ" json:"-"`
	Count   int64    `xml:"count,attr" json:"count"`
	ID      string   `xml:"id,attr" json:"id"`
	QDate   string   `xml:"qDate,omitempty" json:"qDate,omitempty"`
	Message string   `xml:"msg,omitempty" json:"msg,omitempty"`
}

// Response is used to construct and receive <response> messages.
type Response struct {
	XMLName       xml.Name           `xml:"response" json:"-"`
	Result        *Result            `xml:"result" json:"result"`
	MessageQueue  *MessageQueue      `xml:"msgQ" json:"msgQ"`
	ResultData    *ResultData        `xml:"resData,omitempty" json:"resData,omitempty"`
	Extension     *ResponseExtension `xml:"extension,omitempty" json:"extension,omitempty"`
	TransactionID struct {
		XMLName             xml.Name `xml:"trID" json:"-"`
		ClientTransactionID string   `xml:"clTRID" json:"clTRID"`
		ServerTransactionID string   `xml:"svTRID" json:"svTRID"`
	} `xml:"trID" json:"trID"`
}

const (
	// ResponseType represents a <response> message.
	ResponseType string = "epp.response"
)

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (r Response) TypedMessage() Response {
	out := Response{}

	out.XMLName = r.XMLName

	if r.Result != nil {
		result := r.Result.TypedMessage()
		out.Result = &result
	}

	if r.ResultData != nil {
		resultData := r.ResultData.TypedMessage()
		out.ResultData = &resultData
	}

	if r.Extension != nil {
		extension := r.Extension.TypedMessage()
		out.Extension = &extension
	}

	if r.MessageQueue != nil {
		out.MessageQueue = r.MessageQueue
	}

	out.TransactionID = r.TransactionID

	return out
}

// HasMessageQueue will returned true if the message has a msgQ
// component to it.
func (r Response) HasMessageQueue() bool {
	return r.MessageQueue != nil
}

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (r Response) MessageType() string {
	if r.ResultData != nil {
		return r.ResultData.MessageType()
	}

	return ResponseType
}

// GetEPPResponse Returns an uninitialized EPP Response object.
func GetEPPResponse(ClientTXID string, ServerTXID string) Epp {
	epp := GetEPP()
	epp.ResponseObject = new(Response)
	epp.ResponseObject.TransactionID.ClientTransactionID = ClientTXID
	epp.ResponseObject.TransactionID.ServerTransactionID = ServerTXID

	return epp
}

// Result is used to receive a result object fromt he server containing
// a message code and a message if applicable.
type Result struct {
	Code int    `xml:"code,attr" json:"code"`
	Msg  string `xml:"msg" json:"msg"`
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (r Result) TypedMessage() Result {
	return r
}

// GetEPPResponseResult Returns an populated EPP Response Result object.
func GetEPPResponseResult(ClientTXID string, ServerTXID string, Code int, Message string) Epp {
	epp := GetEPPResponse(ClientTXID, ServerTXID)
	epp.ResponseObject.Result = new(Result)
	epp.ResponseObject.Result.Code = Code
	epp.ResponseObject.Result.Msg = Message

	return epp
}

// ResultData is used to receive a resultData object from the server
// or to serialize an object once the values have been converted to the
// non generic version of each object.
type ResultData struct {
	GenericChkDataResp *GenericChkDataResp `xml:"chkData,omitempty" json:"chkData,omitempty"`
	GenericCreDataResp *GenericCreDataResp `xml:"creData,omitempty" json:"creData,omitempty"`
	GenericInfDataResp *GenericInfDataResp `xml:"infData,omitempty" json:"infData,omitempty"`
	GenericRenDataResp *GenericRenDataResp `xml:"renData,omitempty" json:"renData,omitempty"`
	GenericTrnDataResp *GenericTrnDataResp `xml:"trnData,omitempty" json:"trnData,omitempty"`

	DomainChkDataResp *DomainChkDataResp `xml:"domain:chkData,omitempty" json:"domain:chkData,omitempty"`
	DomainCreDataResp *DomainCreDataResp `xml:"domain:creData,omitempty" json:"domain:creData,omitempty"`
	DomainInfDataResp *DomainInfDataResp `xml:"domain:infData,omitempty" json:"domain:infData,omitempty"`
	DomainRenDataResp *DomainRenDataResp `xml:"domain:renData,omitempty" json:"domain:renData,omitempty"`
	DomainTrnDataResp *DomainTrnDataResp `xml:"domain:trnData,omitempty" json:"domain:trnData,omitempty"`

	HostChkDataResp *HostChkDataResp `xml:"host:chkData,omitempty" json:"host:chkData,omitempty"`
	HostCreDataResp *HostCreDataResp `xml:"host:creData,omitempty" json:"host:creData,omitempty"`
	HostInfDataResp *HostInfDataResp `xml:"host:infData,omitempty" json:"host:infData,omitempty"`

	ContactChkDataResp *ContactChkDataResp `xml:"contact:chkData,omitempty" json:"contact:chkData,omitempty"`
	ContactCreDataResp *ContactCreDataResp `xml:"contact:creData,omitempty" json:"contact:creData,omitempty"`
	ContactInfDataResp *ContactInfDataResp `xml:"contact:infData,omitempty" json:"contact:infData,omitempty"`
	ContactTrnDataResp *ContactTrnDataResp `xml:"contact:trnData,omitempty" json:"contact:trnData,omitempty"`
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (r ResultData) TypedMessage() ResultData {
	out := ResultData{}

	if r.GenericInfDataResp != nil {
		if r.GenericInfDataResp.XMLNSDomain != "" {
			typed, err := r.GenericInfDataResp.ToDomainInfDataResp()

			if err == nil {
				out.DomainInfDataResp = &typed
			}
		}

		if r.GenericInfDataResp.XMLNSHost != "" {
			typed, err := r.GenericInfDataResp.ToHostInfDataResp()

			if err == nil {
				out.HostInfDataResp = &typed
			}
		}

		if r.GenericInfDataResp.XMLNSContact != "" {
			typed, err := r.GenericInfDataResp.ToContactInfDataResp()

			if err == nil {
				out.ContactInfDataResp = &typed
			}
		}
	}

	if r.GenericChkDataResp != nil {
		if r.GenericChkDataResp.XMLNSDomain != "" {
			domainCheckResp := &DomainChkDataResp{}
			domainCheckResp.XMLNS = DomainXMLNS
			domainCheckResp.XMLNsSchemaLocation = DomainSchema

			domainCheckResp.Domains = make([]CheckDomain, 0)

			if r.GenericChkDataResp.Items != nil {
				for _, dom := range r.GenericChkDataResp.Items {
					domainCheckResp.Domains = append(domainCheckResp.Domains, dom.ToCheckDomain())
				}
			}

			out.DomainChkDataResp = domainCheckResp
		}

		if r.GenericChkDataResp.XMLNSHost != "" {
			hostCheckResp := &HostChkDataResp{}
			hostCheckResp.XMLNS = HostXMLNS
			hostCheckResp.XMLNsSchemaLocation = HostSchema

			hostCheckResp.Hosts = make([]CheckHost, 0)

			if r.GenericChkDataResp.Items != nil {
				for _, hos := range r.GenericChkDataResp.Items {
					hostCheckResp.Hosts = append(hostCheckResp.Hosts, hos.ToCheckHost())
				}
			}

			out.HostChkDataResp = hostCheckResp
		}

		if r.GenericChkDataResp.XMLNSContact != "" {
			contactCheckResp := &ContactChkDataResp{}
			contactCheckResp.XMLNS = ContactXMLNS
			contactCheckResp.XMLNsSchemaLocation = ContactSchema

			contactCheckResp.ContactIDs = make([]CheckContact, 0)

			if r.GenericChkDataResp.Items != nil {
				for _, id := range r.GenericChkDataResp.Items {
					contactCheckResp.ContactIDs = append(contactCheckResp.ContactIDs, id.ToCheckContact())
				}
			}

			out.ContactChkDataResp = contactCheckResp
		}
	}

	if r.GenericRenDataResp != nil {
		if r.GenericRenDataResp.XMLNSDomain != "" {
			grd := r.GenericRenDataResp

			drd := &DomainRenDataResp{}
			drd.XMLNSDomain = grd.XMLNSDomain
			drd.XMLNsSchemaLocation = grd.XMLNsSchemaLocation
			drd.Name = grd.Name
			drd.ExpireDate = grd.ExpireDate
			out.DomainRenDataResp = drd
		}
	}

	if r.GenericTrnDataResp != nil {
		if r.GenericTrnDataResp.XMLNSDomain != "" {
			gtd := r.GenericTrnDataResp

			dtd := &DomainTrnDataResp{}
			dtd.XMLNSDomain = gtd.XMLNSDomain
			dtd.XMLNsSchemaLocation = gtd.XMLNsSchemaLocation
			dtd.Name = gtd.Name
			dtd.TrStatus = gtd.TrStatus
			dtd.ReID = gtd.ReID
			dtd.ReDate = gtd.ReDate
			dtd.AcID = gtd.AcID
			dtd.AcDate = gtd.AcDate
			dtd.ExpireDate = gtd.ExpireDate
			out.DomainTrnDataResp = dtd
		}

		if r.GenericTrnDataResp.XMLNSContact != "" {
			gtd := r.GenericTrnDataResp

			ctd := &ContactTrnDataResp{}
			ctd.XMLNSContact = gtd.XMLNSContact
			ctd.XMLNsSchemaLocation = gtd.XMLNsSchemaLocation
			ctd.ID = gtd.ID
			ctd.TrStatus = gtd.TrStatus
			ctd.ReID = gtd.ReID
			ctd.ReDate = gtd.ReDate
			ctd.AcID = gtd.AcID
			ctd.AcDate = gtd.AcDate
			out.ContactTrnDataResp = ctd
		}
	}

	if r.GenericCreDataResp != nil {
		if r.GenericCreDataResp.XMLNSDomain != "" {
			gcd := r.GenericCreDataResp

			dcd := &DomainCreDataResp{}
			dcd.XMLNSDomain = gcd.XMLNSDomain
			dcd.XMLNsSchemaLocation = gcd.XMLNsSchemaLocation
			dcd.Name = gcd.Name
			dcd.CreateDate = gcd.CreateDate
			dcd.ExpireDate = gcd.ExpireDate
			out.DomainCreDataResp = dcd
		}

		if r.GenericCreDataResp.XMLNSHost != "" {
			gcd := r.GenericCreDataResp

			hcd := &HostCreDataResp{}
			hcd.XMLNSHost = gcd.XMLNSHost
			hcd.XMLNsSchemaLocation = gcd.XMLNsSchemaLocation
			hcd.Name = gcd.Name
			hcd.CreateDate = gcd.CreateDate
			out.HostCreDataResp = hcd
		}

		if r.GenericCreDataResp.XMLNSContact != "" {
			gcd := r.GenericCreDataResp

			ccd := &ContactCreDataResp{}
			ccd.XMLNSContact = gcd.XMLNSContact
			ccd.XMLNsSchemaLocation = gcd.XMLNsSchemaLocation
			ccd.ID = gcd.ID
			ccd.CreateDate = gcd.CreateDate
			out.ContactCreDataResp = ccd
		}
	}

	return out
}

const (
	// ResponseDomainType represents a <response> domain message.
	ResponseDomainType string = "epp.response.domain"

	// ResponseDomainCheckType represents a <response> domain check message.
	ResponseDomainCheckType string = "epp.response.domain.check"

	// ResponseDomainCreateType represents a <response> domain create message.
	ResponseDomainCreateType string = "epp.response.domain.create"

	// ResponseDomainInfoType represents a <response> domain info message.
	ResponseDomainInfoType string = "epp.response.domain.info"

	// ResponseDomainRenewType represents a <response> domain renew message.
	ResponseDomainRenewType string = "epp.response.domain.renew"

	// ResponseDomainTransferType represents a <response> domain transfer message.
	ResponseDomainTransferType string = "epp.response.domain.transfer"

	// ResponseHostType represents a <response> host message.
	ResponseHostType string = "epp.response.host"

	// ResponseHostCheckType represents a <response> host check message.
	ResponseHostCheckType string = "epp.response.host.check"

	// ResponseHostCreateType represents a <response> host create message.
	ResponseHostCreateType string = "epp.response.host.create"

	// ResponseHostInfoType represents a <response> host info message.
	ResponseHostInfoType string = "epp.response.host.info"

	// ResponseContactType represents a <response> contact message.
	ResponseContactType string = "epp.response.contact"

	// ResponseContactCheckType represents a <response> contact check message.
	ResponseContactCheckType string = "epp.response.contact.check"

	// ResponseContactCreateType represents a <response> contact create message.
	ResponseContactCreateType string = "epp.response.contact.create"

	// ResponseContactInfoType represents a <response> contact info message.
	ResponseContactInfoType string = "epp.response.contact.info"

	// ResponseContactTransferType represents a <response> contact transfer message.
	ResponseContactTransferType string = "epp.response.contact.transfer"
)

// MessageType is used to determine what the contained message type is
// based on the objects that are present in the structure.
func (r ResultData) MessageType() string {
	isDomain := r.DomainChkDataResp != nil || r.DomainCreDataResp != nil || r.DomainInfDataResp != nil || r.DomainRenDataResp != nil || r.DomainTrnDataResp != nil
	isHost := r.HostChkDataResp != nil || r.HostCreDataResp != nil || r.HostInfDataResp != nil
	isContact := r.ContactChkDataResp != nil || r.ContactCreDataResp != nil || r.ContactInfDataResp != nil || r.ContactTrnDataResp != nil

	if isDomain && !isHost && !isContact {
		if r.DomainChkDataResp != nil && r.DomainCreDataResp == nil &&
			r.DomainInfDataResp == nil && r.DomainRenDataResp == nil &&
			r.DomainTrnDataResp == nil {
			return ResponseDomainCheckType
		}

		if r.DomainChkDataResp == nil && r.DomainCreDataResp != nil &&
			r.DomainInfDataResp == nil && r.DomainRenDataResp == nil &&
			r.DomainTrnDataResp == nil {
			return ResponseDomainCreateType
		}

		if r.DomainChkDataResp == nil && r.DomainCreDataResp == nil &&
			r.DomainInfDataResp != nil && r.DomainRenDataResp == nil &&
			r.DomainTrnDataResp == nil {
			return ResponseDomainInfoType
		}

		if r.DomainChkDataResp == nil && r.DomainCreDataResp == nil &&
			r.DomainInfDataResp == nil && r.DomainRenDataResp != nil &&
			r.DomainTrnDataResp == nil {
			return ResponseDomainRenewType
		}

		if r.DomainChkDataResp == nil && r.DomainCreDataResp == nil &&
			r.DomainInfDataResp == nil && r.DomainRenDataResp == nil &&
			r.DomainTrnDataResp != nil {
			return ResponseDomainTransferType
		}
	}

	if !isDomain && isHost && !isContact {
		if r.HostChkDataResp != nil && r.HostCreDataResp == nil &&
			r.HostInfDataResp == nil {
			return ResponseHostCheckType
		}

		if r.HostChkDataResp == nil && r.HostCreDataResp != nil &&
			r.HostInfDataResp == nil {
			return ResponseHostCreateType
		}

		if r.HostChkDataResp == nil && r.HostCreDataResp == nil &&
			r.HostInfDataResp != nil {
			return ResponseHostInfoType
		}
	}

	if !isDomain && !isHost && isContact {
		if r.ContactChkDataResp != nil && r.ContactCreDataResp == nil &&
			r.ContactInfDataResp == nil && r.ContactTrnDataResp == nil {
			return ResponseContactCheckType
		}

		if r.ContactChkDataResp == nil && r.ContactCreDataResp != nil &&
			r.ContactInfDataResp == nil && r.ContactTrnDataResp == nil {
			return ResponseContactCreateType
		}

		if r.ContactChkDataResp == nil && r.ContactCreDataResp == nil &&
			r.ContactInfDataResp != nil && r.ContactTrnDataResp == nil {
			return ResponseContactInfoType
		}

		if r.ContactChkDataResp == nil && r.ContactCreDataResp == nil &&
			r.ContactInfDataResp == nil && r.ContactTrnDataResp != nil {
			return ResponseContactTransferType
		}
	}

	return ResponseType
}

// GenericTrnDataResp is used to receive a generic version of a trnData
// object from the server.
type GenericTrnDataResp struct {
	XMLNSDomain         string `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSContact        string `xml:"contact,attr" json:"xml.contact"`
	XMLNsSchemaLocation string `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string `xml:"name" json:"name"`
	ID                  string `xml:"id" json:"id"`
	TrStatus            string `xml:"trStatus" json:"trStatus"`
	ReID                string `xml:"reID" json:"reID"`
	ReDate              string `xml:"reDate" json:"reDate"`
	AcID                string `xml:"acID" json:"acID"`
	AcDate              string `xml:"acDate" json:"acDate"`
	ExpireDate          string `xml:"exDate" json:"exDate"`
}

// DomainTrnDataResp is the non generic form of GenericTrnDataResp
// containing the server responses from the transfer request for the
// domain.
type DomainTrnDataResp struct {
	XMLNSDomain         string `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNsSchemaLocation string `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string `xml:"domain:name" json:"domain.name"`
	TrStatus            string `xml:"domain:trStatus" json:"domain.trStatus"`
	ReID                string `xml:"domain:reID" json:"domain.reID"`
	ReDate              string `xml:"domain:reDate" json:"domain.reDate"`
	AcID                string `xml:"domain:acID" json:"domain.acID"`
	AcDate              string `xml:"domain:acDate" json:"domain.acDate"`
	ExpireDate          string `xml:"domain:exDate" json:"domain.exDate"`
}

// ContactTrnDataResp is the non generic form of GenericTrnDataResp
// containing the server responses from the transfer request for the
// contact.
type ContactTrnDataResp struct {
	XMLNSContact        string `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNsSchemaLocation string `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	ID                  string `xml:"contact:id" json:"contact.id"`
	TrStatus            string `xml:"contact:trStatus" json:"contact.trStatus"`
	ReID                string `xml:"contact:reID" json:"contact.reID"`
	ReDate              string `xml:"contact:reDate" json:"contact.reDate"`
	AcID                string `xml:"contact:acID" json:"contact.acID"`
	AcDate              string `xml:"contact:acDate" json:"contact.acDate"`
}

// GenericRenDataResp is used to receive a generic version of a renData
// object from the server.
type GenericRenDataResp struct {
	XMLNSDomain         string `xml:"domain,attr" json:"xmlns.domain"`
	XMLNsSchemaLocation string `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string `xml:"name" json:"name"`
	ExpireDate          string `xml:"exDate" json:"exDate"`
}

// DomainRenDataResp is the non generic form of GenericRenDataResp
// containing the server responses from the renew request for the
// domain.
type DomainRenDataResp struct {
	XMLNSDomain         string `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNsSchemaLocation string `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string `xml:"domain:name" json:"name"`
	ExpireDate          string `xml:"domain:exDate" json:"expireDate"`
}

// GenericCreDataResp is used to receive a generic version of a creData
// object from the server.
type GenericCreDataResp struct {
	XMLNSDomain         string `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSHost           string `xml:"host,attr" json:"xmlns.host"`
	XMLNSContact        string `xml:"contact,attr" json:"xmlns.contact"`
	XMLNsSchemaLocation string `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string `xml:"name" json:"name"`
	ID                  string `xml:"id" json:"id"`
	CreateDate          string `xml:"crDate" json:"crDate"`
	ExpireDate          string `xml:"exDate" json:"exDate"`
}

// DomainCreDataResp is the non generic form of GenericCreDataResp
// containing the server responses from the create request for the
// domain.
type DomainCreDataResp struct {
	XMLNSDomain         string `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNsSchemaLocation string `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string `xml:"domain:name" json:"domain.name"`
	CreateDate          string `xml:"domain:crDate" json:"domain.crDate"`
	ExpireDate          string `xml:"domain:exDate" json:"domain.exDate"`
}

// HostCreDataResp is the non generic form of GenericCreDataResp
// containing the server responses from the create request for the
// host.
type HostCreDataResp struct {
	XMLNSHost           string `xml:"xmlns:host,attr" json:"xmlns.host"`
	XMLNsSchemaLocation string `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Name                string `xml:"host:name" json:"host.name"`
	CreateDate          string `xml:"host:crDate" json:"host.crDate"`
}

// ContactCreDataResp is the non generic form of GenericCreDataResp
// containing the server responses from the create request for the
// contact.
type ContactCreDataResp struct {
	XMLNSContact        string `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNsSchemaLocation string `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	ID                  string `xml:"contact:id" json:"contact.id"`
	CreateDate          string `xml:"contact:crDate" json:"crDate"`
}

// ResponseExtension is used to receive a extension object from the
// server as part of the response or to serialize an object once the
// values have been converted to the non generic version of each object.
type ResponseExtension struct {
	GenericChkDataRespExt *GenericChkDataRespExt `xml:"chkData" json:"chkData"`
	GenericNamestore      *GenericNamestore      `xml:"namestoreExt" json:"namespaceExt"`
	GenericInfDataResp    *GenericInfDataRespExt `xml:"infData" json:"infData"`
	GenericUpData         *GenericUpData         `xml:"upData" json:"upData"`

	LaunchChkData *LaunchChkData      `xml:"launch:chkData" json:"launch.chkData"`
	NameStore     *NameStoreExtension `xml:"namestoreExt:namestoreExt" json:"namestoreExt.namestoreExt"`
	SecDNSInfData *SecDNSInfData      `xml:"secDNS:infData" json:"secDNS.infData"`
	RgpUpData     *RgpUpData          `xml:"rgp:upData" json:"rgp.upData"`
	JobsContact   *JobsContact        `xml:"jobsContact:infData" json:"jobsContact.infData"`
}

// TypedMessage is used to convert generic versions of the object into
// a typed version after it is parsed.
func (r ResponseExtension) TypedMessage() ResponseExtension {
	out := ResponseExtension{}

	if r.GenericChkDataRespExt != nil {
		if r.GenericChkDataRespExt.XMLNSLaunch != "" {
			gcd := r.GenericChkDataRespExt

			lcd := &LaunchChkData{}
			lcd.LaunchPhase = gcd.LaunchPhase
			lcd.XMLNSLaunch = "urn:ietf:params:xml:ns:launch-1.0"

			for _, gcdcd := range gcd.GenericCDs {
				cd := LaunchCD{}
				cd.ClaimKey = gcdcd.ClaimKey
				cd.Name = gcdcd.Name
				lcd.LaunchCDs = append(lcd.LaunchCDs, cd)
			}

			out.LaunchChkData = lcd
		}
	}

	if r.GenericInfDataResp != nil {
		if r.GenericInfDataResp.XMLNSsecDNS != "" {
			gid := r.GenericInfDataResp
			sdid := &SecDNSInfData{}
			sdid.XMLNSsecDNS = gid.XMLNSsecDNS
			sdid.XMLNsSchemaLocation = gid.XMLNsSchemaLocation

			for _, gds := range gid.DSData {
				sdid.DSData = append(sdid.DSData, gds.ToDSData())
			}

			out.SecDNSInfData = sdid
		}

		if r.GenericInfDataResp.XMLNSJobsContact != "" {
			out.JobsContact = r.GenericInfDataResp.ToJobsContact()
		}
	}

	if r.GenericNamestore != nil {
		ns := GetDefaultNameStoreExtension()
		ns.SubProducts = append(ns.SubProducts, r.GenericNamestore.Subproducts...)
		out.NameStore = ns
	}

	if r.GenericUpData != nil {
		gud := r.GenericUpData
		if gud.XMLNSRgp != "" {
			rud := &RgpUpData{}

			rud.XMLNSRgp = gud.XMLNSRgp
			rud.XMLNsSchemaLocation = gud.XMLNsSchemaLocation

			for _, grs := range gud.GenericRgpStatuses {
				rs := RgpStatus{}
				rs.Status = grs.Status
				rud.RgpStatuses = append(rud.RgpStatuses, rs)
			}

			out.RgpUpData = rud
		}
	}

	return out
}

// GenericUpData is used to receive a generic version of a upData
// extension object from the server.
type GenericUpData struct {
	XMLNSRgp            string             `xml:"rgp,attr" json:"xmlns.rgp"`
	XMLNsSchemaLocation string             `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	GenericRgpStatuses  []GenericRgpStatus `xml:"rgpStatus" json:"rgpStatus"`
}

// GenericRgpStatus is used to receive a generic version of a RgpStatus
// object from the server.
type GenericRgpStatus struct {
	Status string `xml:"s,attr" json:"status"`
}

// RgpUpData is the non generic form of GenericUpData containing the
// server responses from the create request for the Rgp object.
type RgpUpData struct {
	XMLNSRgp            string      `xml:"xmlns:rgp,attr" json:"xmlns.rgp"`
	XMLNsSchemaLocation string      `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	RgpStatuses         []RgpStatus `xml:"rgp:rgpStatus" json:"rgpStatus"`
}

// RgpStatus contains a status value associated with an rgp object.
type RgpStatus struct {
	Status string `xml:"s,attr" json:"status"`
}

// GenericInfDataRespExt is used to receive a generic version of a
// infData extension object from the server.
type GenericInfDataRespExt struct {
	XMLNSsecDNS         string          `xml:"secDNS,attr" json:"xmlns.secdns"`
	XMLNSJobsContact    string          `xml:"jobsContact,attr" json:"xmlns.jobsContact"`
	XMLNsSchemaLocation string          `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	DSData              []GenericDSData `xml:"dsData" json:"dsData"`
	Title               string          `xml:"title" json:"title"`
	Website             string          `xml:"website" json:"website"`
	IndustryType        string          `xml:"industryType" json:"industryType"`
	IsAdminContact      string          `xml:"isAdminContact" json:"isAdminContacT"`
	IsAssociationMember string          `xml:"isAssociationMember" json:"isAssociationMember"`
}

// ToJobsContact attempts to convert a GenericInfDataRespExt object into
// a JobsContact object.
func (g GenericInfDataRespExt) ToJobsContact() (j *JobsContact) {
	j = &JobsContact{}
	j.XMLNSJobsContact = g.XMLNSJobsContact
	j.XMLNsSchemaLocation = g.XMLNsSchemaLocation
	j.Title = g.Title
	j.Website = g.Website
	j.IndustryType = g.IndustryType
	j.IsAdminContact = g.IsAdminContact
	j.IsAssociationMember = g.IsAssociationMember

	return
}

// JobsContact is the non generic form of GenericInfDataRespExt
// containing the server responses from the job contact extenDomainHostListsion.
type JobsContact struct {
	XMLNSJobsContact    string `xml:"xmlns:jobsContact,attr" json:"xmlns.jobscontact"`
	XMLNsSchemaLocation string `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Title               string `xml:"jobsContact:title" json:"title"`
	Website             string `xml:"jobsContact:website" json:"website"`
	IndustryType        string `xml:"jobsContact:industryType" json:"industryType"`
	IsAdminContact      string `xml:"jobsContact:isAdminContact" json:"isAdminContact"`
	IsAssociationMember string `xml:"jobsContact:isAssociationMember" json:"isAssociationMember"`
}

// SecDNSInfData is the non generic form of the GenericInfDataRespExt
// object containing a SecDNS:infData object.
type SecDNSInfData struct {
	XMLNSsecDNS         string   `xml:"xmlns:secDNS,attr" json:"xmlns.secdns"`
	XMLNsSchemaLocation string   `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	DSData              []DSData `xml:"secDNS:dsData" json:"dsdata"`
}

// GenericDSData is used to receive a DSData object from the
// server.
type GenericDSData struct {
	KeyTag     int    `xml:"keyTag" json:"keyTag"`
	Alg        int    `xml:"alg" json:"alg"`
	DigestType int    `xml:"digestType" json:"digestType"`
	Digest     string `xml:"digest" json:"digest"`
}

// ToDSData converts from the generic version of GenericDSData to the
// specific version in the form of DSData.
func (g GenericDSData) ToDSData() DSData {
	dsRecord := DSData{}
	dsRecord.Alg = g.Alg
	dsRecord.Digest = g.Digest
	dsRecord.DigestType = g.DigestType
	dsRecord.KeyTag = g.KeyTag

	return dsRecord
}

// DSData contains DS data infomration about a domain stored in an
// SecDNSInfData extension.
type DSData struct {
	KeyTag     int    `xml:"secDNS:keyTag" json:"secDNS.keyTag"`
	Alg        int    `xml:"secDNS:alg" json:"secDNS.alg"`
	DigestType int    `xml:"secDNS:digestType" json:"secDNS.digestType"`
	Digest     string `xml:"secDNS:digest" json:"secDNS.digest"`
}

// GenericNamestore is used to receive a NameStore object from the
// server.
type GenericNamestore struct {
	XMLNSNamestoreExt   string   `xml:"namestoreExt,attr" json:"namestoreExt"`
	XMLNsSchemaLocation string   `xml:"schemaLocation,attr" json:"xmlns.schemaLocation"`
	Subproducts         []string `xml:"subProduct" json:"subProduct"`
}

// GenericChkDataRespExt is used to receive a generic version of a
// chkData object from the server.
type GenericChkDataRespExt struct {
	XMLName     xml.Name    `xml:"chkData" json:"chkData"`
	XMLNSLaunch string      `xml:"launch,attr" json:"launch"`
	LaunchPhase string      `xml:"phase" json:"launchPhase"`
	GenericCDs  []GenericCD `xml:"cd" json:"genericCDs"`
}

// LaunchChkData is used to represent a launch:chkData object from the
// server.
type LaunchChkData struct {
	XMLName     xml.Name   `xml:"launch:chkData" json:"launch.chkData"`
	XMLNSLaunch string     `xml:"xmlns:launch,attr" json:"xmlns.launch"`
	LaunchPhase string     `xml:"launch:phase" json:"launch.phase"`
	LaunchCDs   []LaunchCD `xml:"launch:cd" json:"launch.cd"`
}

// LaunchCD is used to represent a launch:cd object from the server.
type LaunchCD struct {
	Name     LaunchCDName     `xml:"launch:name" json:"launch.name"`
	ClaimKey LaunchCDClaimKey `xml:"launch:claimKey" json:"launch.claimKey"`
}

// GenericCD is used to receive a generic version of a cd object from
// the server.
type GenericCD struct {
	Name     LaunchCDName     `xml:"name" json:"name"`
	ClaimKey LaunchCDClaimKey `xml:"claimKey" json:"claimKey"`
}

// LaunchCDName is used to represent a launch:cd>name object allowing
// the exists attribute to be set.
type LaunchCDName struct {
	Exists int    `xml:"exists,attr" json:"exists"`
	Name   string `xml:",chardata" json:"chardata"`
}

// LaunchCDClaimKey is used to represent a launch:cd>claimKey object
// allowing the validatorID attribute to be set.
type LaunchCDClaimKey struct {
	ValidatorID string `xml:"validatorID,attr" json:"validatorID"`
	Value       string `xml:",chardata" json:"value"`
}

// GenericChkDataResp is used to receive check responses where it is difficult
// to determine the type of the object due to a lacking feature in Go.
type GenericChkDataResp struct {
	XMLNamespace string              `xml:"xmlns:domain,attr" json:"xmlns,omitempty"`
	XMLNSDomain  string              `xml:"domain,attr" json:"xmlns.domain"`
	XMLNSHost    string              `xml:"host,attr" json:"host"`
	XMLNSContact string              `xml:"contact,attr" json:"contact"`
	Items        []CheckValueUntyped `xml:"cd" json:"cd"`
}

// DomainChkDataResp is used to receive a check domain response.
type DomainChkDataResp struct {
	XMLNS               string        `xml:"xmlns:domain,attr" json:"xmlns.domain"`
	XMLNsSchemaLocation string        `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Domains             []CheckDomain `xml:"domain:cd" json:"domain.cd"`
}

// HostChkDataResp is used to receive a check host response.
type HostChkDataResp struct {
	XMLNS               string      `xml:"xmlns:host,attr" json:"xmlns.host"`
	XMLNsSchemaLocation string      `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	Hosts               []CheckHost `xml:"host:cd" json:"host.cd"`
}

// ContactChkDataResp is used to receive a check contact response.
type ContactChkDataResp struct {
	XMLNS               string         `xml:"xmlns:contact,attr" json:"xmlns.contact"`
	XMLNsSchemaLocation string         `xml:"xsi:schemaLocation,attr" json:"xmlns.schemaLocation"`
	ContactIDs          []CheckContact `xml:"host:cd" json:"host.cd"`
}

// CheckValueUntyped is used to receive an individual domain check
// response.
type CheckValueUntyped struct {
	XMLName xml.Name `xml:"cd" json:"-"`
	Name    struct {
		Available int    `xml:"avail,attr" json:"avail"`
		Value     string `xml:",chardata" json:"value"`
	} `xml:"name" json:"name"`
	ID struct {
		Available int    `xml:"avail,attr" json:"avail"`
		Value     string `xml:",chardata" json:"value"`
	} `xml:"id" json:"id"`
	Reason string `xml:"reason,omitempty" json:"reason"`
}

// ToCheckDomain takes a CheckValueUntyped object and turns it into a
// CheckDomain object with a type.
func (c CheckValueUntyped) ToCheckDomain() CheckDomain {
	cd := CheckDomain{}
	cd.Name = c.Name
	cd.Reason = c.Reason

	return cd
}

// ToCheckHost takes a CheckValueUntyped object and turns it into a
// CheckHost object with a type.
func (c CheckValueUntyped) ToCheckHost() CheckHost {
	cd := CheckHost{}
	cd.Name = c.Name

	return cd
}

// ToCheckContact takes a CheckValueUntyped object and turns it into a
// CheckContact object with a type.
func (c CheckValueUntyped) ToCheckContact() CheckContact {
	cd := CheckContact{}
	cd.ID = c.ID

	return cd
}

// CheckContact is used to receive an individual contact check response.
type CheckContact struct {
	XMLName xml.Name `xml:"contact:cd" json:"contact.cd"`
	ID      struct {
		Available int    `xml:"avail,attr" json:"avail"`
		Value     string `xml:",chardata" json:"value"`
	} `xml:"contact:id" json:"contact.id"`
}

// CheckHost is used to receive an individual host check response.
type CheckHost struct {
	XMLName xml.Name `xml:"host:cd" json:"host.cd"`
	Name    struct {
		Available int    `xml:"avail,attr" json:"avail"`
		Value     string `xml:",chardata" json:"value"`
	} `xml:"host:name" json:"host.name"`
}

// CheckDomain is used to receive an individual domain check response.
type CheckDomain struct {
	XMLName xml.Name `xml:"domain:cd" json:"domain.cd"`
	Name    struct {
		Available int    `xml:"avail,attr" json:"avail"`
		Value     string `xml:",chardata" json:"value"`
	} `xml:"domain:name" json:"domain.name"`
	Reason string `xml:"urn:ietf:params:xml:ns:domain-1.0 reason,omitempty" json:"reason"`
}

const (
	// startFatalCodeRange is the start of result codes for 25xx which indicates
	// a fatal condition.
	startFatalCodeRange = 2500

	// endFatalCodeRange is the end of result codes for 25xx which indicates
	// a fatal condition.
	endFatalCodeRange = 2599

	// startErrorCodeRange is the start of the result code range for 2xxx which
	// indicates and error.
	startErrorCodeRange = 2000

	// endErrorCodeRange is the end of the result code range for 2xxx which
	// indicates and error.
	endErrorCodeRange = 2999
)

// IsFatalError will return true if the response code indicates an error
// that results in a closed connection.
func (r Response) IsFatalError() bool {
	return r.Result.Code >= startFatalCodeRange && r.Result.Code <= endFatalCodeRange
}

// IsError will return true if the response code indicates an error.
func (r Response) IsError() bool {
	return r.Result.Code >= startErrorCodeRange && r.Result.Code <= endErrorCodeRange
}

// GetError will create an error object with the message contained in
// the response body.
func (r Response) GetError() error {
	if r.IsError() {
		return errors.New(r.Result.Msg)
	}

	return nil
}

const (

	// ResponseCode1000 represents the message associated with the 1000
	// response code.
	ResponseCode1000 string = "Command completed successfully"

	// ResponseCodeCommandSuccessful represents the response code for
	// Command completed successfully from the epp server.
	ResponseCodeCommandSuccessful int = 1000

	// ResponseCode1001 represents the message associated with the 1001
	// response code.
	ResponseCode1001 string = "Command completed successfully; action pending"

	// ResponseCodeCommandSuccessfulPending represents the response code
	// for Command completed successfully with action pending from the
	// epp server.
	ResponseCodeCommandSuccessfulPending int = 1001

	// ResponseCode1300 represents the message associated with the 1300
	// response code.
	ResponseCode1300 string = "Command completed successfully; no messages"

	// ResponseCodeCompletedNoMessages represents the response code
	// for Command completed successfully with no messages from the
	// epp server.
	ResponseCodeCompletedNoMessages int = 1300

	// ResponseCode1301 represents the message associated with the 1301
	// response code.
	ResponseCode1301 string = "Command completed successfully; ack to dequeue"

	// ResponseCodeCompletedAckToDequeue represents the response code
	// for Command completed successfully where a the message
	// received can be dequeued by an ack message.
	ResponseCodeCompletedAckToDequeue int = 1301

	// ResponseCode1500 represents the message associated with the 1500
	// response code.
	ResponseCode1500 string = "Command completed successfully; ending session"

	// ResponseCodeCompletedEndingSession represents the response code
	// for Command completed successfully, ending the session.
	ResponseCodeCompletedEndingSession int = 1500

	// ResponseCode2000 represents the message associated with the 2000
	// response code.
	ResponseCode2000 string = "Unknown command"

	// ResponseCodeUnknownCommand represents the response code for an
	// unknown command message.
	ResponseCodeUnknownCommand int = 2000

	// ResponseCode2001 represents the message associated with the 2001
	// response code.
	ResponseCode2001 string = "Command syntax error"

	// ResponseCodeCommandSyntaxError represents the response code for a
	// command syntax error message.
	ResponseCodeCommandSyntaxError int = 2001

	// ResponseCode2002 represents the message associated with the 2002
	// response code.
	ResponseCode2002 string = "Command use error"

	// ResponseCodeCommandUseError represnets the response code for a
	// command use error message.
	ResponseCodeCommandUseError int = 2002

	// ResponseCode2003 represents the message associated with the 2003
	// response code.
	ResponseCode2003 string = "Required parameter missing"

	// ResponseCodeRequireParameterMissing represents the response code
	// for a required parametere missing error message.
	ResponseCodeRequireParameterMissing int = 2003

	// ResponseCode2004 represents the message associated with the 2004
	// response code.
	ResponseCode2004 string = "Parameter value range error"

	// ResponseCodeParameterValueRangeError represnets the response code
	// for a parametere value range error message.
	ResponseCodeParameterValueRangeError int = 2004

	// ResponseCode2005 represents the message associated with the 2005
	// response code.
	ResponseCode2005 string = "Parameter value syntax error"

	// ResponseCodeParameterValueSyntaxError represnets the response code
	// for a parametere value syntax error message.
	ResponseCodeParameterValueSyntaxError int = 2005

	// ResponseCode2100 represents the message associated with the 1200
	// response code.
	ResponseCode2100 string = "Unimplemented protocol version"

	// ResponseCodeUnimplementedProtocolVersion represents the response
	// code for an unimplemented protocol version error message.
	ResponseCodeUnimplementedProtocolVersion int = 2100

	// ResponseCode2101 represents the message associated with the 2101
	// response code.
	ResponseCode2101 string = "Unimplemented command"

	// ResponseCodeUnimplementedCommand represents the response code for
	// an unimplemented command message.
	ResponseCodeUnimplementedCommand int = 2101

	// ResponseCode2102 represents the message associated with the 2102
	// response code.
	ResponseCode2102 string = "Unimplemented option"

	// ResponseCodeUnimplementedOption represents the response code for
	// an unimplemented option message.
	ResponseCodeUnimplementedOption int = 2102

	// ResponseCode2103 represents the message associated with the 2103
	// response code.
	ResponseCode2103 string = "Unimplemented extension"

	// ResponseCodeUnimplementedExtension represents the response code
	// for an unimlpemented extension message.
	ResponseCodeUnimplementedExtension int = 2103

	// ResponseCode2104 represents the message associated with the 2104
	// response code.
	ResponseCode2104 string = "Billing failure"

	// ResponseCodeBillingFailure represents the response code for a
	// billing failure message.
	ResponseCodeBillingFailure int = 2104

	// ResponseCode2105 represents the message associated with the 2105
	// response code.
	ResponseCode2105 string = "Object is not eligible for renewal"

	// ResponseCodeOjectNotEligibleForRenewal represents the response
	// code for a not eligible for renewal message.
	ResponseCodeOjectNotEligibleForRenewal int = 2105

	// ResponseCode2106 represents the message associated with the 2106
	// response code.
	ResponseCode2106 string = "Object is not eligible for transfer"

	// ResponseCodeObjectNotEligibleForTransfer represents the response
	// code for a not eligible for transfer message.
	ResponseCodeObjectNotEligibleForTransfer int = 2106

	// ResponseCode2200 represents the message associated with the 2200
	// response code.
	ResponseCode2200 string = "Authentication error"

	// ResponseCodeAuthenticationError represents the response code for
	// an authentication error message.
	ResponseCodeAuthenticationError int = 2200

	// ResponseCode2201 represents the message associated with the 2201
	// response code.
	ResponseCode2201 string = "Authorization error"

	// ResponseCodeAuthorizationError represents the response code for
	// an authorization error.
	ResponseCodeAuthorizationError int = 2201

	// ResponseCode2202 represents the message associated with the 2202
	// response code.
	ResponseCode2202 string = "Invalid authorization information"

	// ResponseCodeInvalidAuthorizationInformation represents the response
	// code for invalid authorization information messages.
	ResponseCodeInvalidAuthorizationInformation int = 2202

	// ResponseCode2300 represents the message associated with the 2300
	// response code.
	ResponseCode2300 string = "Object pending transfer"

	// ResponseCodeObjectPendingTransfer represnets the response code for
	// a object pending transfer message.
	ResponseCodeObjectPendingTransfer int = 2300

	// ResponseCode2301 represents the message associated with the 2301
	// response code.
	ResponseCode2301 string = "Object not pending transfer"

	// ResponseCodeObjectNotPendingTransfer represents the response code
	// for a object not pending transfer message.
	ResponseCodeObjectNotPendingTransfer int = 2301

	// ResponseCode2302 represents the message associated with the 2302
	// response code.
	ResponseCode2302 string = "Object exists"

	// ResponseCodeObjectExists represents the response code for a
	// object exists message.
	ResponseCodeObjectExists int = 2302

	// ResponseCode2303 represents the message associated with the 2303
	// response code.
	ResponseCode2303 string = "Object does not exist"

	// ResponseCodeObjectDoesNotExist represents a response code for a
	// object does not exist message.
	ResponseCodeObjectDoesNotExist int = 2303

	// ResponseCode2304 represents the message associated with the 2304
	// response code.
	ResponseCode2304 string = "Object status prohibits operation"

	// ResponseCodeObjectStatusProhibited represents the response code
	// for a object status prohibited operation message.
	ResponseCodeObjectStatusProhibited int = 2304

	// ResponseCode2305 represents the message associated with the 2305
	// response code.
	ResponseCode2305 string = "Object association prohibits operation"

	// ResponseCodeObjectAssociationProhibitsOperation represnets the
	// response code for a object association prohibits operation
	// message.
	ResponseCodeObjectAssociationProhibitsOperation int = 2305

	// ResponseCode2306 represents the message associated with the 2306
	// response code.
	ResponseCode2306 string = "Parameter value policy error"

	// ResponseCodeParameterValuePolicyError represents the response code
	// for a parameter value policy error message.
	ResponseCodeParameterValuePolicyError int = 2306

	// ResponseCode2307 represents the message associated with the 2307
	// response code.
	ResponseCode2307 string = "Unimplemented object service"

	// ResponseCodeUnimplementedObjectService represents the response code
	// for an unimplemented object service  message.
	ResponseCodeUnimplementedObjectService int = 2307

	// ResponseCode2308 represents the message associated with the 2308
	// response code.
	ResponseCode2308 string = "Data management policy violation"

	// ResponseCodeDataManagemnetPolicyViolation represents the response code
	// for a data management policy violation message.
	ResponseCodeDataManagemnetPolicyViolation int = 2308

	// ResponseCode2400 represents the message associated with the 2400
	// response code.
	ResponseCode2400 string = "Command failed"

	// ResponseCodeCommandFailed represents the response code for a
	// command failure message.
	ResponseCodeCommandFailed int = 2400

	// ResponseCode2500 represents the message associated with the 2500
	// response code.
	ResponseCode2500 string = "Command failed; server closing connection"

	// ResponseCodeCommandFailedClosing represents the response code for
	// a command failed, closing message.
	ResponseCodeCommandFailedClosing int = 2500

	// ResponseCode2501 represents the message associated with the 2501
	// response code.
	ResponseCode2501 string = "Authentication error; server closing connection"

	// ResponseCodeAuthorizationErrorClosing represnets the response code
	// for a Authentication error message that closes the connection.
	ResponseCodeAuthorizationErrorClosing int = 2501

	// ResponseCode2502 represents the message associated with the 2502
	// response code.
	ResponseCode2502 string = "Session limit exceeded; server closing connection"

	// ResponseCodeSessionLimitExceededClosing represents the response code
	// for session limit exceeded message that closes the connection.
	ResponseCodeSessionLimitExceededClosing int = 2502
)
