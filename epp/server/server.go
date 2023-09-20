package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/timapril/go-registrar/epp"
	"github.com/timapril/go-registrar/lib"

	logging "github.com/op/go-logging"
)

// TODO: Add logging

const (
	// eppServerRecvBufferSize sets the buffer size for receive buffers.
	eppServerRecvBufferSize = 100
)

// EPPServer contains the infomration required to run a testing EPP
// server.
type EPPServer struct {
	Host    string
	Port    int
	Timeout time.Duration

	KillChan   chan bool
	KilledChan chan bool

	socket net.Listener

	Running chan bool

	Logins                 map[string]LoginObject
	MaxFailedLoginAttempts int

	NextConnectionID int

	Contacts map[string]lib.Contact
	Hosts    map[string]lib.Host
	Domains  map[string]lib.Domain

	Log *logging.Logger
}

var (
	ErrNoContactFound    = errors.New("no contact found")
	ErrNoHostFound       = errors.New("no host found")
	ErrNoDomainFound     = errors.New("no domain found")
	ErrClientTimeout     = errors.New("client timeout")
	ErrNotImplemented    = errors.New("not implemented")
	ErrUnhandledTransfer = errors.New("unhandled transfer")
)

// LoginObject is used to hold the information about a specific Login
// in to the epp server.
type LoginObject struct {
	LoginID     string
	Password    string
	RegistrarID string
}

// ContactByID searches for contacts by the RegistryID.
func (srv EPPServer) ContactByID(id string) (*lib.Contact, error) {
	for _, cont := range srv.Contacts {
		if cont.ContactRegistryID == id {
			return &cont, nil
		}
	}

	return nil, ErrNoContactFound
}

// HostByName searches for hosts by the HostName.
func (srv EPPServer) HostByName(name string) (*lib.Host, error) {
	for _, hos := range srv.Hosts {
		if hos.HostName == name {
			return &hos, nil
		}
	}

	return nil, ErrNoHostFound
}

// DomainByName searches for domain by the DomainName.
func (srv EPPServer) DomainByName(name string) (*lib.Domain, error) {
	for _, dom := range srv.Domains {
		if dom.DomainName == name {
			return &dom, nil
		}
	}

	return nil, ErrNoDomainFound
}

// EPPServerConnection has the required information to handle
// communication with an epp client.
type EPPServerConnection struct {
	Conn        net.Conn
	Conf        *EPPServer
	RecvChannel chan epp.Epp

	FailedLoginAttempts int

	TXID         int
	ConnectionID int

	LoggedIn *LoginObject

	Log *logging.Logger
}

// maximumFailedLoginAttempts is the number of failed logins before the server
// will close a connectoin.
const maximumFailedLoginAttempts = 2

// firstConnectionID is the default connection ID for new servers for their
// first connection attempt.
const firstConnectionID = 1

// NewEPPServer will take the vital information related to the server
// like the host address, port and connection timeout duration that it
// should use for communicating with clients.
func NewEPPServer(host string, port int, timeout time.Duration) EPPServer {
	srv := EPPServer{
		Host:       host,
		Port:       port,
		Timeout:    timeout,
		KillChan:   make(chan bool, 1),
		KilledChan: make(chan bool, 1),
		Running:    make(chan bool, 1),

		Logins:                 make(map[string]LoginObject),
		MaxFailedLoginAttempts: maximumFailedLoginAttempts,
		NextConnectionID:       firstConnectionID,

		Contacts: make(map[string]lib.Contact),
		Hosts:    make(map[string]lib.Host),
		Domains:  make(map[string]lib.Domain),

		Log: logging.MustGetLogger("eppserver"),
	}

	return srv
}

// watchForKill listens on the chan used to kill the server and will
// close the socket associated with the server resulting in the server
// terminating.
func (srv *EPPServer) watchForKill() {
	good := <-srv.KillChan

	if good {
		srv.KilledChan <- true
	}

	srv.socket.Close()
}

// Start will initiate the server and block until the server is killed
// or an error occurs while stariting up or running.
func (srv *EPPServer) Start() error {
	var err error

	srv.socket, err = net.Listen("tcp", fmt.Sprintf("%s:%d", srv.Host, srv.Port))
	if err != nil {
		srv.Log.Error(err)

		return fmt.Errorf("error starting server: %w", err)
	}

	srv.Running <- true

	go srv.watchForKill()

	for {
		conn, err := srv.socket.Accept()
		if err != nil {
			select {
			case <-srv.KilledChan:
				return nil
			default:
				return fmt.Errorf("accept error: %w", err)
			}
		}

		eppConn := EPPServerConnection{
			Conn:                conn,
			Conf:                srv,
			RecvChannel:         make(chan epp.Epp, eppServerRecvBufferSize),
			TXID:                1,
			ConnectionID:        srv.NextConnectionID,
			FailedLoginAttempts: 0,
			Log:                 srv.Log,
		}
		srv.NextConnectionID = srv.NextConnectionID + 1

		go func() {
			err := eppConn.Listen()
			if err != nil {
				srv.Log.Error(err)
			}
		}()

		go func() {
			err := eppConn.Handle()
			if err != nil {
				srv.Log.Error(err)
			}
		}()
	}
}

// EPPServerState is a type that is used to represent the current state
// of the EPP server according to the http://tools.ietf.org/html/rfc5730.
type EPPServerState string

var (
	// EPPServerPrepareGreeting represents the "Prepare Greeting" state of
	// RFC5730.
	EPPServerPrepareGreeting EPPServerState = "PrepareGreeting"

	// EPPServerWaitingForClient represents the "Waiting For Client
	// Authentication" state of RFC5730.
	EPPServerWaitingForClient EPPServerState = "WaitingForClient"

	// EPPServerWaitingForCommand represents the "Waiting For Command
	// or hello" state of RFC5730.
	EPPServerWaitingForCommand EPPServerState = "WaitingForCommand"

	// EPPServerCloseConnection represents the "End Session" state of
	// RFC5730.
	EPPServerCloseConnection EPPServerState = "CloseConnection"
)

// Listen will try to receive messages from the socket associated with
// a connection.
func (conn *EPPServerConnection) Listen() error {
	scanner := bufio.NewScanner(conn.Conn)
	scanner.Split(epp.WireSplit)

	for scanner.Scan() {
		text := scanner.Text()
		outObj, unmarshallErr := epp.UnmarshalMessage([]byte(text))

		if unmarshallErr != nil {
			conn.Log.Error(unmarshallErr)

			return fmt.Errorf("error unmarshalling request: %w", unmarshallErr)
		}

		conn.RecvChannel <- outObj
	}

	err := scanner.Err()
	if err != nil {
		conn.Log.Error(err)
	}

	return nil
}

// GetNextTransactionID generates a new Transaction ID for the server's
// commuication.
func (conn *EPPServerConnection) GetNextTransactionID() string {
	txid := fmt.Sprintf("SRV-%d-%d", conn.ConnectionID, conn.TXID)
	conn.TXID = conn.TXID + 1

	return txid
}

// HandleWaitingForClient handles the Waiting for client state of the
// EPP Server Connection.
func (conn *EPPServerConnection) HandleWaitingForClient(timeout chan bool) (state EPPServerState, err error) {
	state = EPPServerWaitingForClient
	select {
	case msg := <-conn.RecvChannel:
		msgType := msg.MessageType()
		switch msgType {
		case epp.CommandLoginType:
			if conn.Conf.Logins[msg.CommandObject.LoginObject.ClientID].Password == msg.CommandObject.LoginObject.Password {
				client := conn.Conf.Logins[msg.CommandObject.LoginObject.ClientID]
				cltxid, _ := msg.GetTransactionID()
				srvtxid := conn.GetNextTransactionID()
				res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeCommandSuccessful, epp.ResponseCode1000)
				err = conn.WriteEPP(res)

				if err != nil {
					return state, err
				}

				state = EPPServerWaitingForCommand
				conn.LoggedIn = &client
			} else {
				if conn.FailedLoginAttempts >= conn.Conf.MaxFailedLoginAttempts {
					cltxid, _ := msg.GetTransactionID()
					srvtxid := conn.GetNextTransactionID()
					res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeAuthorizationErrorClosing, epp.ResponseCode2501)
					err = conn.WriteEPP(res)

					if err != nil {
						return state, err
					}

					state = EPPServerCloseConnection
				} else {
					cltxid, _ := msg.GetTransactionID()
					srvtxid := conn.GetNextTransactionID()
					res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeAuthenticationError, epp.ResponseCode2200)
					err = conn.WriteEPP(res)

					if err != nil {
						return state, err
					}

					conn.FailedLoginAttempts = conn.FailedLoginAttempts + 1
				}
			}
		default:
			cltxid, _ := msg.GetTransactionID()
			srvtxid := conn.GetNextTransactionID()
			res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeCommandUseError, epp.ResponseCode2002)
			err = conn.WriteEPP(res)

			if err != nil {
				return state, err
			}

			state = EPPServerCloseConnection
		}
	case <-timeout:
		err := conn.CloseConnection(nil)
		if err != nil {
			return state, err
		}

		return state, ErrClientTimeout
	}

	return state, err
}

func (conn *EPPServerConnection) generateCheckResponseObject(msg epp.Epp) error {
	cltxid, err := msg.GetTransactionID()
	if err != nil {
		return fmt.Errorf("transaction id not available: %w", err)
	}

	srvtxid := conn.GetNextTransactionID()
	res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeCommandSuccessful, epp.ResponseCode1000)

	resData := &(epp.ResultData{})

	switch msg.MessageType() {
	case epp.CommandCheckContactType:
		resData.ContactChkDataResp = &(epp.ContactChkDataResp{})

		resData.ContactChkDataResp.XMLNS = "urn:ietf:params:xml:ns:contact-1.0"
		resData.ContactChkDataResp.XMLNsSchemaLocation = "urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd"

		for _, cont := range msg.CommandObject.CheckObject.ContactChecks {
			for _, contactID := range cont.ContactIDs {
				_, err := conn.Conf.ContactByID(contactID)
				cChk := epp.CheckContact{}

				if err != nil {
					cChk.ID.Available = 1
					cChk.ID.Value = contactID
				} else {
					cChk.ID.Available = 0
					cChk.ID.Value = contactID
				}

				resData.ContactChkDataResp.ContactIDs = append(resData.ContactChkDataResp.ContactIDs, cChk)
			}
		}

	case epp.CommandCheckHostType:
		resData.HostChkDataResp = &(epp.HostChkDataResp{})

		resData.HostChkDataResp.XMLNS = "urn:ietf:params:xml:ns:host-1.0"
		resData.HostChkDataResp.XMLNsSchemaLocation = "urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd"

		for _, host := range msg.CommandObject.CheckObject.HostChecks {
			for _, name := range host.HostNames {
				_, err := conn.Conf.HostByName(name)
				hChk := epp.CheckHost{}

				if err != nil {
					hChk.Name.Available = 1
					hChk.Name.Value = name
				} else {
					hChk.Name.Available = 0
					hChk.Name.Value = name
				}

				resData.HostChkDataResp.Hosts = append(resData.HostChkDataResp.Hosts, hChk)
			}
		}
	case epp.CommandCheckDomainType:
		resData.DomainChkDataResp = &(epp.DomainChkDataResp{})

		resData.DomainChkDataResp.XMLNS = "urn:ietf:params:xml:ns:domain-1.0"
		resData.DomainChkDataResp.XMLNsSchemaLocation = "urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd"

		for _, dom := range msg.CommandObject.CheckObject.DomainChecks {
			for _, name := range dom.DomainNames {
				_, err := conn.Conf.DomainByName(name)
				dChk := epp.CheckDomain{}

				if err != nil {
					dChk.Name.Available = 1
					dChk.Name.Value = name
				} else {
					dChk.Name.Available = 0
					dChk.Name.Value = name
				}

				resData.DomainChkDataResp.Domains = append(resData.DomainChkDataResp.Domains, dChk)
			}
		}
	}

	res.ResponseObject.ResultData = resData

	return conn.WriteEPP(res)
}

// func (conn *EPPServerConnection) handleDomainCheck(msg epp.Epp) error {
// 	cltxid, err := msg.GetTransactionID()
// 	if err != nil {
// 		return fmt.Errorf("transaction id not available: %w", err)
// 	}
// 	srvtxid := conn.GetNextTransactionID()
// 	res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeCommandSuccessful, epp.ResponseCode1000)
// 	resData := &(epp.ResultData{})
// 	resData.DomainChkDataResp = &(epp.DomainChkDataResp{})

// 	resData.DomainChkDataResp.XMLNS = "urn:ietf:params:xml:ns:domain-1.0"
// 	resData.DomainChkDataResp.XMLNsSchemaLocation = "urn:ietf:params:xml:ns:domain-1.0 domain-1.0.xsd"

// 	for _, dom := range msg.CommandObject.CheckObject.DomainChecks {
// 		for _, name := range dom.DomainNames {
// 			_, err := conn.Conf.DomainByName(name)
// 			dChk := epp.CheckDomain{}

// 			if err != nil {
// 				dChk.Name.Available = 1
// 				dChk.Name.Value = name
// 			} else {
// 				dChk.Name.Available = 0
// 				dChk.Name.Value = name
// 			}

// 			resData.DomainChkDataResp.Domains = append(resData.DomainChkDataResp.Domains, dChk)
// 		}
// 	}

// 	res.ResponseObject.ResultData = resData

// 	return conn.WriteEPP(res)
// }

// func (conn *EPPServerConnection) handleHostCheck(msg epp.Epp) error {
// 	cltxid, err := msg.GetTransactionID()
// 	if err != nil {
// 		return fmt.Errorf("transaction id not available: %w", err)
// 	}
// 	srvtxid := conn.GetNextTransactionID()
// 	res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeCommandSuccessful, epp.ResponseCode1000)
// 	resData := &(epp.ResultData{})
// 	resData.HostChkDataResp = &(epp.HostChkDataResp{})

// 	resData.HostChkDataResp.XMLNS = "urn:ietf:params:xml:ns:host-1.0"
// 	resData.HostChkDataResp.XMLNsSchemaLocation = "urn:ietf:params:xml:ns:host-1.0 host-1.0.xsd"

// 	for _, host := range msg.CommandObject.CheckObject.HostChecks {
// 		for _, name := range host.HostNames {
// 			_, err := conn.Conf.HostByName(name)
// 			hChk := epp.CheckHost{}

// 			if err != nil {
// 				hChk.Name.Available = 1
// 				hChk.Name.Value = name
// 			} else {
// 				hChk.Name.Available = 0
// 				hChk.Name.Value = name
// 			}

// 			resData.HostChkDataResp.Hosts = append(resData.HostChkDataResp.Hosts, hChk)
// 		}
// 	}

// 	res.ResponseObject.ResultData = resData

// 	return conn.WriteEPP(res)
// }

// func (conn *EPPServerConnection) handleContactCheck(msg epp.Epp) error {
// 	cltxid, _ := msg.GetTransactionID()
// 	srvtxid := conn.GetNextTransactionID()
// 	res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeCommandSuccessful, epp.ResponseCode1000)
// 	resData := &(epp.ResultData{})
// 	resData.ContactChkDataResp = &(epp.ContactChkDataResp{})

// 	resData.ContactChkDataResp.XMLNS = "urn:ietf:params:xml:ns:contact-1.0"
// 	resData.ContactChkDataResp.XMLNsSchemaLocation = "urn:ietf:params:xml:ns:contact-1.0 contact-1.0.xsd"

// 	for _, cont := range msg.CommandObject.CheckObject.ContactChecks {
// 		for _, contactID := range cont.ContactIDs {
// 			_, err := conn.Conf.ContactByID(contactID)
// 			cChk := epp.CheckContact{}

// 			if err != nil {
// 				cChk.ID.Available = 1
// 				cChk.ID.Value = contactID
// 			} else {
// 				cChk.ID.Available = 0
// 				cChk.ID.Value = contactID
// 			}

// 			resData.ContactChkDataResp.ContactIDs = append(resData.ContactChkDataResp.ContactIDs, cChk)
// 		}
// 	}

// 	res.ResponseObject.ResultData = resData

// 	return conn.WriteEPP(res)
// }

func (conn *EPPServerConnection) handleContactInfo(msg epp.Epp) error {
	cltxid, _ := msg.GetTransactionID()
	srvtxid := conn.GetNextTransactionID()

	responseCode := epp.ResponseCodeCommandSuccessful
	responseError := epp.ResponseCode1000

	resData := &(epp.ResultData{})
	resData.ContactInfDataResp = &epp.ContactInfDataResp{}

	resData.ContactInfDataResp.XMLNSContact = epp.ContactXMLNS
	resData.ContactInfDataResp.XMLNsSchemaLocation = epp.ContactSchema

	if msg.CommandObject.InfoObject.ContactInfoObj != nil {
		obj := msg.CommandObject.InfoObject.ContactInfoObj
		cont, err := conn.Conf.ContactByID(obj.ID)

		if err == nil {
			resData.ContactInfDataResp.ID = cont.ContactRegistryID
			resData.ContactInfDataResp.ROID = cont.ContactROID
			resData.ContactInfDataResp.Email = cont.CurrentRevision.EmailAddress
			resData.ContactInfDataResp.Voice.Extension = cont.CurrentRevision.VoicePhoneExtension
			resData.ContactInfDataResp.Voice.Number = cont.CurrentRevision.VoicePhoneNumber
			resData.ContactInfDataResp.Fax.Extension = cont.CurrentRevision.FaxPhoneExtension
			resData.ContactInfDataResp.Fax.Number = cont.CurrentRevision.FaxPhoneNumber
			postalInfo := epp.PostalInfo{}
			postalInfo.PostalInfoType = "int"

			if cont.CurrentRevision.AddressStreet1 != "" {
				postalInfo.Address.Street = append(postalInfo.Address.Street, cont.CurrentRevision.AddressStreet1)
			}

			if cont.CurrentRevision.AddressStreet2 != "" {
				postalInfo.Address.Street = append(postalInfo.Address.Street, cont.CurrentRevision.AddressStreet2)
			}

			if cont.CurrentRevision.AddressStreet3 != "" {
				postalInfo.Address.Street = append(postalInfo.Address.Street, cont.CurrentRevision.AddressStreet3)
			}

			postalInfo.Address.Sp = cont.CurrentRevision.AddressState
			postalInfo.Address.Cc = cont.CurrentRevision.AddressCountry
			postalInfo.Address.City = cont.CurrentRevision.AddressCity
			postalInfo.Address.Pc = cont.CurrentRevision.AddressPostalCode
			postalInfo.Org = cont.CurrentRevision.Org
			postalInfo.Name = cont.CurrentRevision.Name

			if cont.LinkedStatus {
				resData.ContactInfDataResp.Status = append(resData.ContactInfDataResp.Status, epp.ContactStatus{StatusFlag: "linked"})
			}

			if cont.CurrentRevision.ClientDeleteProhibitedStatus {
				resData.ContactInfDataResp.Status = append(resData.ContactInfDataResp.Status, epp.ContactStatus{StatusFlag: "clientDeleteProhibited"})
			}

			if cont.CurrentRevision.ClientTransferProhibitedStatus {
				resData.ContactInfDataResp.Status = append(resData.ContactInfDataResp.Status, epp.ContactStatus{StatusFlag: "clientTransferProhibited"})
			}

			if cont.CurrentRevision.ClientUpdateProhibitedStatus {
				resData.ContactInfDataResp.Status = append(resData.ContactInfDataResp.Status, epp.ContactStatus{StatusFlag: "clientUpdateProhibited"})
			}

			if cont.CurrentRevision.ServerDeleteProhibitedStatus {
				resData.ContactInfDataResp.Status = append(resData.ContactInfDataResp.Status, epp.ContactStatus{StatusFlag: "serverDeleteProhibited"})
			}

			if cont.CurrentRevision.ServerTransferProhibitedStatus {
				resData.ContactInfDataResp.Status = append(resData.ContactInfDataResp.Status, epp.ContactStatus{StatusFlag: "serverTransferProhibited"})
			}

			if cont.CurrentRevision.ServerUpdateProhibitedStatus {
				resData.ContactInfDataResp.Status = append(resData.ContactInfDataResp.Status, epp.ContactStatus{StatusFlag: "serverUpdateProhibited"})
			}

			resData.ContactInfDataResp.Status = append(resData.ContactInfDataResp.Status, epp.ContactStatus{StatusFlag: "OK"})

			resData.ContactInfDataResp.PostalInfos = append(resData.ContactInfDataResp.PostalInfos, postalInfo)

			if cont.SponsoringClientID == conn.LoggedIn.RegistrarID {
				resData.ContactInfDataResp.AuthPW = &epp.ContactAuth{}
				resData.ContactInfDataResp.AuthPW.Password = "testpassword"
			}

			resData.ContactInfDataResp.CreateID = cont.CreateClientID
			resData.ContactInfDataResp.CreateDate = cont.CreateDate.Format(epp.EPPTimeFormat)

			if cont.UpdateClientID != "" {
				resData.ContactInfDataResp.UpdateID = cont.UpdateClientID
				resData.ContactInfDataResp.UpdateDate = cont.UpdateDate.Format(epp.EPPTimeFormat)
			}

			if cont.TransferDate.Unix() > 0 {
				resData.ContactInfDataResp.TransferDate = cont.TransferDate.Format(epp.EPPTimeFormat)
			}
		} else {
			responseCode = epp.ResponseCodeObjectDoesNotExist
			responseError = epp.ResponseCode2303
		}
	}

	res := epp.GetEPPResponseResult(cltxid, srvtxid, responseCode, responseError)

	if responseCode == epp.ResponseCodeCommandSuccessful {
		res.ResponseObject.ResultData = resData
	}

	return conn.WriteEPP(res)
}

const (
	ipv4EPPProtocolIdentifier = 4
	ipv6EPPProtocolIdentifier = 6
)

func (conn *EPPServerConnection) handleHostInfo(msg epp.Epp) error {
	cltxid, _ := msg.GetTransactionID()
	srvtxid := conn.GetNextTransactionID()

	responseCode := epp.ResponseCodeCommandSuccessful
	responseError := epp.ResponseCode1000

	resData := &(epp.ResultData{})
	resData.HostInfDataResp = &epp.HostInfDataResp{}

	resData.HostInfDataResp.XMLNSHost = epp.HostXMLNS
	resData.HostInfDataResp.XMLNsSchemaLocation = epp.HostSchema

	if msg.CommandObject.InfoObject.HostInfoObj != nil {
		obj := msg.CommandObject.InfoObject.HostInfoObj
		host, err := conn.Conf.HostByName(obj.Name)

		if err == nil {
			resData.HostInfDataResp.ROID = host.HostROID

			resData.HostInfDataResp.Name = host.HostName

			for _, hostAddress := range host.CurrentRevision.HostAddresses {
				addr := epp.HostAddress{}
				addr.Address = hostAddress.IPAddress

				if hostAddress.Protocol == ipv4EPPProtocolIdentifier {
					addr.IPVersion = "v4"
				}

				if hostAddress.Protocol == ipv6EPPProtocolIdentifier {
					addr.IPVersion = "v6"
				}

				resData.HostInfDataResp.Addresses = append(resData.HostInfDataResp.Addresses, addr)
			}

			if host.LinkedStatus {
				resData.HostInfDataResp.Status = append(resData.HostInfDataResp.Status, epp.HostStatus{StatusFlag: "linked"})
			}

			if host.CurrentRevision.ClientDeleteProhibitedStatus {
				resData.HostInfDataResp.Status = append(resData.HostInfDataResp.Status, epp.HostStatus{StatusFlag: "clientDeleteProhibited"})
			}

			if host.CurrentRevision.ClientTransferProhibitedStatus {
				resData.HostInfDataResp.Status = append(resData.HostInfDataResp.Status, epp.HostStatus{StatusFlag: "clientTransferProhibited"})
			}

			if host.CurrentRevision.ClientUpdateProhibitedStatus {
				resData.HostInfDataResp.Status = append(resData.HostInfDataResp.Status, epp.HostStatus{StatusFlag: "clientUpdateProhibited"})
			}

			if host.CurrentRevision.ServerDeleteProhibitedStatus {
				resData.HostInfDataResp.Status = append(resData.HostInfDataResp.Status, epp.HostStatus{StatusFlag: "serverDeleteProhibited"})
			}

			if host.CurrentRevision.ServerTransferProhibitedStatus {
				resData.HostInfDataResp.Status = append(resData.HostInfDataResp.Status, epp.HostStatus{StatusFlag: "serverTransferProhibited"})
			}

			if host.CurrentRevision.ServerUpdateProhibitedStatus {
				resData.HostInfDataResp.Status = append(resData.HostInfDataResp.Status, epp.HostStatus{StatusFlag: "serverUpdateProhibited"})
			}

			resData.HostInfDataResp.Status = append(resData.HostInfDataResp.Status, epp.HostStatus{StatusFlag: "OK"})

			resData.HostInfDataResp.CreateID = host.CreateClientID
			resData.HostInfDataResp.CreateDate = host.CreateDate.Format(epp.EPPTimeFormat)

			if host.UpdateClientID != "" {
				resData.HostInfDataResp.UpdateID = host.UpdateClientID
				resData.HostInfDataResp.UpdateDate = host.UpdateDate.Format(epp.EPPTimeFormat)
			}

			if host.TransferDate.Unix() > 0 {
				resData.HostInfDataResp.TransferDate = host.TransferDate.Format(epp.EPPTimeFormat)
			}
		} else {
			responseCode = epp.ResponseCodeObjectDoesNotExist
			responseError = epp.ResponseCode2303
		}
	}

	res := epp.GetEPPResponseResult(cltxid, srvtxid, responseCode, responseError)

	if responseCode == epp.ResponseCodeCommandSuccessful {
		res.ResponseObject.ResultData = resData
	}

	return conn.WriteEPP(res)
}

func (conn *EPPServerConnection) handleDomainInfo(msg epp.Epp) error {
	cltxid, _ := msg.GetTransactionID()
	srvtxid := conn.GetNextTransactionID()

	responseCode := epp.ResponseCodeCommandSuccessful
	responseError := epp.ResponseCode1000

	resData := &(epp.ResultData{})
	resData.DomainInfDataResp = &epp.DomainInfDataResp{}

	resData.DomainInfDataResp.XMLNSDomain = epp.DomainXMLNS
	resData.DomainInfDataResp.XMLNsSchemaLocation = epp.DomainSchema

	if msg.CommandObject.InfoObject.DomainInfoObj != nil {
		obj := msg.CommandObject.InfoObject.DomainInfoObj
		dom, err := conn.Conf.DomainByName(obj.Domain.Name)

		if err == nil {
			resData.DomainInfDataResp.Name = dom.DomainName
			resData.DomainInfDataResp.ROID = dom.DomainROID

			if dom.CurrentRevision.DomainAdminContact.ID != 0 {
				cont := epp.DomainContact{}
				cont.Type = epp.Admin
				cont.Value = dom.CurrentRevision.DomainAdminContact.ContactRegistryID
				resData.DomainInfDataResp.Contacts = append(resData.DomainInfDataResp.Contacts, cont)
			}

			if dom.CurrentRevision.DomainBillingContact.ID != 0 {
				cont := epp.DomainContact{}
				cont.Type = epp.Billing
				cont.Value = dom.CurrentRevision.DomainBillingContact.ContactRegistryID
				resData.DomainInfDataResp.Contacts = append(resData.DomainInfDataResp.Contacts, cont)
			}

			if dom.CurrentRevision.DomainTechContact.ID != 0 {
				cont := epp.DomainContact{}
				cont.Type = epp.Tech
				cont.Value = dom.CurrentRevision.DomainTechContact.ContactRegistryID
				resData.DomainInfDataResp.Contacts = append(resData.DomainInfDataResp.Contacts, cont)
			}

			if dom.CurrentRevision.DomainRegistrant.ID != 0 {
				resData.DomainInfDataResp.RegistrantID = dom.CurrentRevision.DomainRegistrant.ContactRegistryID
			}

			// :ns></domain:ns>
			for _, hos := range dom.CurrentRevision.Hostnames {
				resData.DomainInfDataResp.Hosts = append(resData.DomainInfDataResp.Hosts, hos.HostName)
				resData.DomainInfDataResp.NSHosts.Hosts = append(resData.DomainInfDataResp.NSHosts.Hosts, epp.DomainHost{Value: hos.HostName})
			}

			resData.DomainInfDataResp.CreateID = dom.CreateClientID
			resData.DomainInfDataResp.CreateDate = dom.CreateDate.Format(epp.EPPTimeFormat)

			if dom.UpdateClientID != "" {
				resData.DomainInfDataResp.UpdateID = dom.UpdateClientID
				resData.DomainInfDataResp.UpdateDate = dom.UpdateDate.Format(epp.EPPTimeFormat)
			}

			if dom.TransferDate.Unix() > 0 {
				resData.DomainInfDataResp.TransferDate = dom.TransferDate.Format(epp.EPPTimeFormat)
			}

			resData.DomainInfDataResp.ExpireDate = dom.ExpireDate.Format(epp.EPPTimeFormat)

			if dom.CurrentRevision.ClientDeleteProhibitedStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "clientDeleteProhibited"})
			}

			if dom.CurrentRevision.ClientHoldStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "clientHold"})
			}

			if dom.CurrentRevision.ClientRenewProhibitedStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "clientRenewProhibited"})
			}

			if dom.CurrentRevision.ClientTransferProhibitedStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "clientTransferProhibited"})
			}

			if dom.CurrentRevision.ClientUpdateProhibitedStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "clientUpdateProhibited"})
			}

			if dom.CurrentRevision.ServerDeleteProhibitedStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "serverDeleteProhibited"})
			}

			if dom.CurrentRevision.ServerHoldStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "serverHold"})
			}

			if dom.CurrentRevision.ServerRenewProhibitedStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "serverRenewProhibited"})
			}

			if dom.CurrentRevision.ServerTransferProhibitedStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "serverTransferProhibited"})
			}

			if dom.CurrentRevision.ServerUpdateProhibitedStatus {
				resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "serverUpdateProhibited"})
			}

			resData.DomainInfDataResp.Status = append(resData.DomainInfDataResp.Status, epp.DomainStatus{StatusFlag: "OK"})

			if dom.SponsoringClientID == conn.LoggedIn.RegistrarID {
				resData.DomainInfDataResp.AuthPW = &epp.DomainAuth{}
				resData.DomainInfDataResp.AuthPW.Password = "testpassword"
			}
		} else {
			responseCode = epp.ResponseCodeObjectDoesNotExist
			responseError = epp.ResponseCode2303
		}
	}

	res := epp.GetEPPResponseResult(cltxid, srvtxid, responseCode, responseError)

	if responseCode == epp.ResponseCodeCommandSuccessful {
		res.ResponseObject.ResultData = resData
	}

	return conn.WriteEPP(res)
}

func (conn *EPPServerConnection) handleContactUpdate(msg epp.Epp) error {
	conn.Log.Info("Need to handle Contact Update")

	updateObj := msg.CommandObject.UpdateObject.ContactUpdateObj

	conn.Log.Debug(updateObj)

	return ErrNotImplemented
}

func (conn *EPPServerConnection) handleDomainUpdate(msg epp.Epp) error {
	conn.Log.Info("Need to handle Domain Update")

	updateObj := msg.CommandObject.UpdateObject.DomainUpdateObj

	conn.Log.Debug(updateObj)

	cltxid, _ := msg.GetTransactionID()
	srvtxid := conn.GetNextTransactionID()
	// res := epp.GetEPPResponseResult(cltxid, srvtxid, Code int, Message string)

	if msg.MessageType() == epp.CommandUpdateDomainType && msg.CommandObject.UpdateObject.DomainUpdateObj != nil {
		obj := msg.CommandObject.UpdateObject.DomainUpdateObj

		dom, err := conn.Conf.DomainByName(obj.DomainName)
		if err != nil {
			res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeObjectDoesNotExist, epp.ResponseCode2303)

			err = conn.WriteEPP(res)
			if err != nil {
				return err
			}
		}

		if dom.SponsoringClientID == conn.LoggedIn.RegistrarID {
			// Add
			for _, contact := range obj.AddObject.Contacts {
				conn.Log.Debug(contact)
			}

			for _, status := range obj.AddObject.Statuses {
				conn.Log.Debug(status)
			}

			if obj.AddObject.Hosts != nil {
				for _, host := range obj.AddObject.Hosts.Hosts {
					conn.Log.Debug(host)
				}
			}

			// Remove

			// Change

			res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeCommandSuccessful, epp.ResponseCode1000)

			return conn.WriteEPP(res)
		}
	}

	return nil
}

func (conn *EPPServerConnection) handleHostUpdate(msg epp.Epp) error {
	conn.Log.Info("Need to handle Host Update")

	updateObj := msg.CommandObject.UpdateObject.HostUpdateObj

	conn.Log.Debug(updateObj)

	return ErrNotImplemented
}

// HandleWaitingForCommand handles the waiting for command sate of the
// EPP server connection.
func (conn *EPPServerConnection) HandleWaitingForCommand(timeout chan bool) (state EPPServerState, err error) {
	state = EPPServerWaitingForCommand

	select {
	case rmsg := <-conn.RecvChannel:
		msg := rmsg.TypedMessage()
		msgType := msg.MessageType()

		conn.Log.Debug(msgType)

		switch msgType {
		case epp.CommandLogoutType:
			cltxid, _ := msg.GetTransactionID()
			srvtxid := conn.GetNextTransactionID()
			res := epp.GetEPPResponseResult(cltxid, srvtxid, epp.ResponseCodeCompletedEndingSession, epp.ResponseCode1500)
			err = conn.WriteEPP(res)

			if err != nil {
				return state, err
			}

			state = EPPServerCloseConnection

		// Check Commands
		case epp.CommandCheckContactType:
			// err := conn.handleContactCheck(msg)
			err := conn.generateCheckResponseObject(msg)
			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand
		case epp.CommandCheckDomainType:
			// err := conn.handleDomainCheck(msg)
			err := conn.generateCheckResponseObject(msg)
			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand
		case epp.CommandCheckHostType:
			// err = conn.handleHostCheck(msg)
			err := conn.generateCheckResponseObject(msg)
			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand

		// Info Commands
		case epp.CommandInfoContactType:
			err = conn.handleContactInfo(msg)

			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand
		case epp.CommandInfoDomainType:
			err = conn.handleDomainInfo(msg)

			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand
		case epp.CommandInfoHostType:
			err = conn.handleHostInfo(msg)

			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand

			// Update Commands
		case epp.CommandUpdateContactType:
			err = conn.handleContactUpdate(msg)

			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand
		case epp.CommandUpdateDomainType:
			err = conn.handleDomainUpdate(msg)

			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand
		case epp.CommandUpdateHostType:
			err = conn.handleHostUpdate(msg)

			if err != nil {
				return state, err
			}

			state = EPPServerWaitingForCommand

		case epp.CommandTransferType:
			if msg.CommandObject.TransferObject.ContactTransferObj != nil {
				if msg.CommandObject.TransferObject.Operation == "query" {
					// TODO: ContactQuery
					return state, ErrUnhandledTransfer
				}
			}

			if msg.CommandObject.TransferObject.DomainTransferObj != nil {
				switch msg.CommandObject.TransferObject.Operation {
				case "query":
					// TODO: DomainTransfer
					return state, ErrUnhandledTransfer
				}
			}
		}
	case <-timeout:
		err := conn.CloseConnection(nil)
		if err != nil {
			return state, err
		}

		return state, ErrClientTimeout
	}

	return state, err
}

// Handle accepts a new connection and communications with the client.
func (conn *EPPServerConnection) Handle() error {
	var err error

	state := EPPServerPrepareGreeting

	for {
		timeout := conn.makeTimeoutWatch()
		conn.Log.Debug("State %s\n", state)

		switch state {
		case EPPServerWaitingForClient:
			// Waiting for Client Authentication
			state, err = conn.HandleWaitingForClient(timeout)

			if err != nil {
				conn.Log.Errorf("Error in state", err)

				return err
			}
		case EPPServerWaitingForCommand:
			conn.Log.Debug("Waiting for command")
			state, err = conn.HandleWaitingForCommand(timeout)

			if err != nil {
				conn.Log.Errorf("Error in state", err)

				return err
			}
		case EPPServerPrepareGreeting:
			err := conn.WriteEPP(epp.GetEPPGreeting(epp.GetDefaultServiceMenu()))
			if err != nil {
				return err
			}

			state = EPPServerWaitingForClient
		case EPPServerCloseConnection:
			return conn.Close()
		}
	}
}

// WriteEPP takes an epp object and serializes it onto the socket for
// the connection.
func (conn *EPPServerConnection) WriteEPP(msg epp.Epp) error {
	data, _ := msg.EncodeEPP()
	_, err := conn.Write(data)

	return err
}

// CloseConnection closes the socket associated with the connection.
func (conn *EPPServerConnection) CloseConnection(err error) error {
	conn.Close()

	return err
}

// makeTimeoutWatch creates a go routine that will send a bool to a
// channel when the timeout period passes.
func (conn *EPPServerConnection) makeTimeoutWatch() chan bool {
	retChan := make(chan bool, 1)
	go WaitDuration(retChan, conn.Conf.Timeout)

	return retChan
}

// WaitDuration sleeps for a delay time and then sends a true to the
// channel provided.
func WaitDuration(notify chan bool, delay time.Duration) {
	time.Sleep(delay)
	notify <- true
}

// Write will accept a series of bytes and return the number of bytes
// written and an error from the underlying socket.
func (conn *EPPServerConnection) Write(b []byte) (n int, err error) {
	count, err := conn.Conn.Write(b)
	if err != nil {
		return count, fmt.Errorf("error writing to epp connection: %w", err)
	}

	return count, nil
}

// Close will close the underlying connection with the client.
func (conn *EPPServerConnection) Close() error {
	err := conn.Conn.Close()
	if err != nil {
		return fmt.Errorf("error closing epp connection: %w", err)
	}

	return nil
}
