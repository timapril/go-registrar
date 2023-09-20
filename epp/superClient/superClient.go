package superclient

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/epp"
	"github.com/timapril/go-registrar/epp/client"
	"github.com/timapril/go-registrar/lib"
)

var (
	// ErrResponseTimeout represnts a tiemout waiting for a server to
	// respond to a request.
	ErrResponseTimeout = errors.New("timeout occurred waiting for server response")

	// ErrUnknownCheckAvailableReturn is returned when a check request
	// returns something other than 0 or 1 for its availability.
	ErrUnknownCheckAvailableReturn = errors.New("check returned a non 0 or 1 value for availability")

	// ErrInvalidDomainReturned is returned when the domain that was
	// requested is not the domain that is returned.
	ErrInvalidDomainReturned = errors.New("domain name in resopnse was not the domain name requested")

	// ErrInvalidContactReturned is returned when the contact that was
	// requested is not the contact that is returned.
	ErrInvalidContactReturned = errors.New("contact name in resopnse was not the contact name requested")

	// ErrInvalidHostReturned is returned when the host that was requested
	// is not the host that is returned.
	ErrInvalidHostReturned = errors.New("host name in resopnse was not the host name requested")

	// ErrUnhandledResponse is returned when a response that is unhandled
	// by the application is found.
	ErrUnhandledResponse = errors.New("unhandled response")

	// ErrUnhandledPollMessage is returend when a poll message is not
	// handled properly.
	ErrUnhandledPollMessage = errors.New("unhandled poll messages")

	// ErrUnknownPollResponseCode indicates that the EPP response to a
	// poll message returned an unexpected respose code.
	ErrUnknownPollResponseCode = errors.New("unknown epp poll response code")

	// ErrUnexpectedReponse indicates that the EPP response received was
	// unexpected.
	ErrUnexpectedReponse = errors.New("unexpected epp response")
)

const (
	// superclientTimeout is the timeot that the client will wait before
	// a timeout occurs.
	superclientTimeout = time.Second * 90
)

// SuperClient is used to handle a easier interface to the EPP server.
type SuperClient struct {
	client client.EPPClient

	Timeout time.Duration
}

// NewSuperClient takes the configuration information and a logger
// interface and will start a new EPP client and wait for the client
// to be ready for use before returning. If an error occurs while
// setting up the client, it will be returned.
func NewSuperClient(config client.Config, log *logging.Logger, loginCallback func(lib.EPPAction), logoutCallback func(lib.EPPAction)) (sc *SuperClient, err error) {
	sc = &SuperClient{}
	sc.client = client.NewEPPClient()
	sc.client.Prepare(config, log, func(eppMsg epp.Epp) error {
		txid := ""

		if eppMsg.ResponseObject != nil {
			txid = eppMsg.ResponseObject.TransactionID.ClientTransactionID
		}

		action := lib.NewEPPAction(txid)
		action.SetAction(lib.EPPLogActionLogin, "")
		err := action.HandleResponse(eppMsg)
		if err != nil {
			return fmt.Errorf("error connecting to server: %w", err)
		}

		loginCallback(action)

		return nil
	}, func(eppMsg epp.Epp) error {
		txid := ""

		if eppMsg.ResponseObject != nil {
			txid = eppMsg.ResponseObject.TransactionID.ClientTransactionID
		}

		action := lib.NewEPPAction(txid)
		action.SetAction(lib.EPPLogActionLogout, "")
		err := action.HandleResponse(eppMsg)
		if err != nil {
			return fmt.Errorf("error disconnecting from the epp server: %w", err)
		}

		logoutCallback(action)

		return nil
	})

	err = sc.client.Start()

	if err != nil {
		return sc, fmt.Errorf("error starting client: %w", err)
	}

	sc.Timeout = superclientTimeout

	blockUntilWaitForWork(sc.client)

	go sc.StatusWatcher()

	return sc, nil
}

// blockUntilWaitForWork will block from doing anything until the client
// is in the state where it is ready to use.
func blockUntilWaitForWork(cli client.EPPClient) {
	for status := range cli.StatusChannel {
		if status.Status == client.WaitForWork {
			return
		}
	}
}

// StatusWatcher will watch for status channel updates and log them to the debug
// stream of the log function.
func (sc *SuperClient) StatusWatcher() {
	for msg := range sc.client.StatusChannel {
		sc.client.Log.Debug(msg.Status)

		if msg.Status == client.ShutdownRequested {
			sc.client.StatusChannel <- msg

			return
		}
	}
}

// getTransactionID is a helper funcation that will use the parent
// client to generate a new transaction id for a request.
func (sc *SuperClient) getTransactionID() string {
	return sc.client.ClientConfig.GetNewTransactionID()
}

// getTimeout is a helper function to generate a go routine that will
// send a message to a channel after a set period of time.
func (sc SuperClient) getTimeout() chan bool {
	return makeTimeout(sc.Timeout)
}

// makeTimeout will return a chan bool that will have a true written to
// it after the timeout duration has passed.
func makeTimeout(length time.Duration) chan bool {
	timeout := make(chan bool, 1)

	go func() {
		time.Sleep(length)
		timeout <- true
	}()

	return timeout
}

// PollResponse represents a poll response object and exposes fields
// that indicate what type of message was received.
type PollResponse struct {
	ID      string
	Message string

	DomainTransfer            *epp.DomainTrnDataResp
	IsDomainTransferApproved  bool
	IsDomainTransferRequested bool
	IsDomainTransferRejected  bool
	IsDomainTransferCancelled bool

	HostInf               *epp.HostInfDataResp
	IsUnusedObjectsPolicy bool
}

// AckPoll is used to acknowledge a message queue item to remove it from
// the queue.
func (sc *SuperClient) AckPoll(messageID string) (action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionPollAck, messageID)

	msg := epp.GetEPPPollAcknowledge(messageID, action.ClientTransactionID)
	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		action.SetError(err)
	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return action, ErrResponseTimeout
	}

	return
}

// PollRequest is used to send a poll request to the EPP server to get
// the first object, if any exit.
func (sc *SuperClient) PollRequest() (hasMessages bool, pr *PollResponse, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionPoll, "")

	msg := epp.GetEPPPollRequest(action.ClientTransactionID)
	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err = action.HandleResponse(respMsg)

		if err != nil {
			return hasMessages, pr, action, fmt.Errorf("error issung poll request: %w", err)
		}

		if respMsg.ResponseObject.Result.Code == epp.ResponseCodeCompletedNoMessages {
			// No messages in poll response
			return false, nil, action, nil
		}

		if respMsg.ResponseObject != nil &&
			respMsg.ResponseObject.Result != nil &&
			respMsg.ResponseObject.HasMessageQueue() {
			pr = &PollResponse{}
			pr.ID = respMsg.ResponseObject.MessageQueue.ID
			pr.Message = respMsg.ResponseObject.MessageQueue.Message

			if respMsg.ResponseObject.Result.Code == epp.ResponseCodeCompletedAckToDequeue {
				// There are messages to respond to
				switch respMsg.MessageType() {
				case epp.ResponseDomainTransferType:
					if respMsg.ResponseObject != nil && respMsg.ResponseObject.ResultData != nil && respMsg.ResponseObject.ResultData.DomainTrnDataResp != nil {
						pr.DomainTransfer = respMsg.ResponseObject.ResultData.DomainTrnDataResp

						switch respMsg.ResponseObject.MessageQueue.Message {
						case "Transfer Approved.":
							pr.IsDomainTransferApproved = true
						case "Transfer Requested.":
							pr.IsDomainTransferRequested = true
						case "Transfer Rejected.":
							pr.IsDomainTransferRejected = true
						case "Transfer Cancelled.":
							pr.IsDomainTransferCancelled = true
						case "Transfer Auto Approved.":
							pr.IsDomainTransferApproved = true
						default:
							sc.client.Log.Infof("Unhandled poll message for %s", respMsg.ResponseObject.MessageQueue.Message)

							return hasMessages, pr, action, ErrUnhandledPollMessage
						}
					}
				case epp.ResponseHostInfoType:
					if respMsg.ResponseObject != nil && respMsg.ResponseObject.ResultData != nil && respMsg.ResponseObject.ResultData.HostInfDataResp != nil {
						pr.HostInf = respMsg.ResponseObject.ResultData.HostInfDataResp

						switch respMsg.ResponseObject.MessageQueue.Message {
						case "Unused objects policy":
							pr.IsUnusedObjectsPolicy = true
						}
					}
				}

				if respMsg.MessageType() == epp.PollType {
					msg, err := respMsg.ToString()
					if err != nil {
						return hasMessages, pr, action, fmt.Errorf("error parsing epp message: %w", err)
					}

					sc.client.Log.Infof("Poll Type %s", msg)
				}

				return true, pr, action, nil
			}

			err = ErrUnknownPollResponseCode

			sc.client.Log.Errorf("Unexpected poll response code: %s", respMsg.ResponseObject.Result.Code)
			action.SetError(err)

			return false, nil, action, err
		}
	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return false, nil, action, ErrResponseTimeout
	}
	action.SetError(ErrUnhandledResponse)

	return false, nil, action, ErrUnhandledResponse
}

// DomainAvailable takes a domain name and will run a check command with
// the epp server and return true iff the domain is available and a
// reason for its availability if there is one. If an error occures
// during the process, an error is returned and the values for
// availability and reason are false and empty respectively. This
// function times out after the timeout defined by the server without a
// response.
func (sc *SuperClient) DomainAvailable(domainName string) (bool, string, lib.EPPAction, error) {
	action := lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainAvailable, domainName)

	msg := epp.GetEPPDomainCheck(domainName, action.ClientTransactionID)
	sc.client.NewWork <- msg
	timeout := sc.getTimeout()
	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		if err != nil {
			return false, "", action, fmt.Errorf("unexpected epp error: %w", err)
		}

		if respMsg.MessageType() == epp.ResponseDomainCheckType {
			domains := respMsg.ResponseObject.ResultData.DomainChkDataResp.Domains
			for _, domain := range domains {
				if domain.Name.Value == strings.ToUpper(domainName) {
					if domain.Name.Available == 1 {
						action.AddNote("Available")

						return true, domain.Reason, action, nil
					} else if domain.Name.Available == 0 {
						action.AddNote("Not Available")

						return false, domain.Reason, action, nil
					}

					action.SetError(ErrUnknownCheckAvailableReturn)

					return false, "", action, ErrUnknownCheckAvailableReturn
				}
			}
		} else {
			if respMsg.ResponseObject != nil {
				if respMsg.ResponseObject.IsError() {
					return false, "", action, fmt.Errorf("EPP error: %w", respMsg.ResponseObject.GetError())
				}
			}
		}

	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return false, "", action, ErrResponseTimeout
	}
	action.SetError(ErrInvalidDomainReturned)

	return false, "", action, ErrInvalidDomainReturned
}

// HostAvailable takes a hostname name and will run a check command with
// the epp server and return true iff the host is available. If an error
// occures during the process, an error is returned and the
// availablility will be false. This function times out after the
// timeout defined by the server without a response.
func (sc *SuperClient) HostAvailable(hostName string) (bool, lib.EPPAction, error) {
	action := lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionHostAvailable, hostName)

	msg := epp.GetEPPHostCheck(hostName, action.ClientTransactionID)

	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		if err != nil {
			return false, action, fmt.Errorf("unexpected epp error: %w", err)
		}

		if respMsg.MessageType() == epp.ResponseHostCheckType {
			hosts := respMsg.ResponseObject.ResultData.HostChkDataResp.Hosts
			for _, host := range hosts {
				if host.Name.Value == strings.ToUpper(hostName) {
					if host.Name.Available == 1 {
						action.AddNote("Available")

						return true, action, nil
					} else if host.Name.Available == 0 {
						action.AddNote("Not Available")

						return false, action, nil
					}

					action.SetError(ErrUnknownCheckAvailableReturn)

					return false, action, ErrUnknownCheckAvailableReturn
				}
			}
		} else {
			if respMsg.ResponseObject != nil {
				if respMsg.ResponseObject.IsError() {
					return false, action, fmt.Errorf("EPP error: %w", respMsg.ResponseObject.GetError())
				}
			}
		}

	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return false, action, ErrResponseTimeout
	}

	action.SetError(ErrInvalidHostReturned)

	return false, action, ErrInvalidHostReturned
}

// DomainInfo takes a domain name and an indicator of which type of
// host information is desired and will attempt to send a domain info
// request to the epp server. If no error occurs, the domain info
// response object will be returned, otherwise an error will be returned.
func (sc *SuperClient) DomainInfo(domainName string, hosts epp.DomainInfoHosts, authInfo *string) (*epp.DomainInfDataResp, *epp.Response, lib.EPPAction, error) {
	action := lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainInfo, domainName)

	var msg epp.Epp

	if authInfo != nil {
		action.AddNote("AuthInfo value is set")
		msg = epp.GetEPPDomainInfo(domainName, action.ClientTransactionID, *authInfo, hosts)
	} else {
		action.AddNote("AuthInfo value not is set")
		msg = epp.GetEPPDomainInfo(domainName, action.ClientTransactionID, "", hosts)
	}

	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		if err != nil {
			return nil, nil, action, fmt.Errorf("unexpected epp error: %w", err)
		}

		if respMsg.MessageType() == epp.ResponseDomainInfoType {
			domainInfo := respMsg.ResponseObject.ResultData.DomainInfDataResp
			if domainInfo.Name == strings.ToUpper(domainName) {
				return domainInfo, respMsg.ResponseObject, action, nil
			}
		} else {
			if respMsg.ResponseObject != nil {
				if respMsg.ResponseObject.IsError() {
					return nil, nil, action, fmt.Errorf("EPP error: %w", respMsg.ResponseObject.GetError())
				}
			}
		}

	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return nil, nil, action, ErrResponseTimeout
	}

	action.SetError(ErrInvalidDomainReturned)

	return nil, nil, action, ErrInvalidDomainReturned
}

// HostInfo takes a hostname and will attempt to send a host info
// request to the epp server. If no error occurs, the host info response
// object will be returned, otherwise an error will be returned.
func (sc *SuperClient) HostInfo(hostName string) (*epp.HostInfDataResp, *epp.Response, lib.EPPAction, error) {
	action := lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionHostInfo, hostName)

	msg := epp.GetEPPHostInfo(hostName, action.ClientTransactionID)
	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		if err != nil {
			return nil, nil, action, fmt.Errorf("unexpected epp error: %w", err)
		}

		if respMsg.MessageType() == epp.ResponseHostInfoType {
			hostInfo := respMsg.ResponseObject.ResultData.HostInfDataResp
			if hostInfo.Name == strings.ToUpper(hostName) {
				return hostInfo, respMsg.ResponseObject, action, nil
			}
		} else {
			if respMsg.ResponseObject != nil {
				if respMsg.ResponseObject.IsError() {
					return nil, nil, action, fmt.Errorf("EPP error: %w", respMsg.ResponseObject.GetError())
				}
			}
		}

	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return nil, nil, action, ErrResponseTimeout
	}

	action.SetError(ErrInvalidHostReturned)

	return nil, nil, action, ErrInvalidHostReturned
}

// Logout will send a logout request to the underlying client in order
// to shut the client down and disconnect from the server.
func (sc *SuperClient) Logout() (lib.EPPAction, error) {
	action := lib.NewEPPAction("")
	action.SetAction(lib.EPPLogActionLogout, "")

	sc.client.LogoutChannel <- true

	return action, sc.waitForShutdown()
}

const (
	// shutdownTimeout is the constant amount of time that the super
	// client should wait before potentially shutting down.
	shutdownTimeout = 2
)

// waitForShutdown will block until the client has either ended or the
// system timeout has elapsed.
func (sc SuperClient) waitForShutdown() error {
	timeout := makeTimeout(sc.Timeout * shutdownTimeout)

	for {
		select {
		case status := <-sc.client.StatusChannel:
			if status.Status == client.ShutdownRequested {
				return nil
			}
		case <-timeout:
			return ErrResponseTimeout
		}
	}
}

// RequestDomainTransfer will take a domain name and the authorization
// info password and attempt to request a domain transfer for the
// given domain. If the transfer request does not succeed, a response
// code and an error will be returned otherwise the domain transfer
// object will returned.
func (sc *SuperClient) RequestDomainTransfer(domainName string, authInfo string) (trResp *epp.DomainTrnDataResp, responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainTransferRequest, domainName)

	per := epp.DomainPeriod{}
	per.Unit = epp.DomainPeriodYear
	per.Value = 1

	msg := epp.GetEPPDomainTransferRequest(domainName, per, authInfo, action.ClientTransactionID)
	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		if err != nil {
			return nil, 0, action, fmt.Errorf("unexpected epp error: %w", err)
		}

		switch respMsg.MessageType() {
		case epp.ResponseDomainTransferType:
			return respMsg.ResponseObject.ResultData.DomainTrnDataResp, respMsg.ResponseObject.Result.Code, action, nil
		case epp.ResponseType:
			return nil, respMsg.ResponseObject.Result.Code, action, fmt.Errorf("EPP error: %w", respMsg.ResponseObject.GetError())
		}

		err = ErrUnexpectedReponse

		action.SetError(err)

		return nil, 0, action, err
	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return nil, 0, action, ErrResponseTimeout
	}
}

// RejectDomainTransfer takes a dmoain name and will attempt to reject
// the domain transfer request. If the transfer reject does not
// succeed an error and response code will be returned.
func (sc *SuperClient) RejectDomainTransfer(domainName string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainTransferReject, domainName)

	msg := epp.GetEPPDomainTransferReject(domainName, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// ApproveDomainTransfer takes a domain name and will attempt to approve
// the domain transfer request for that domain. If the transfer approval
// does not succeed, an error and response code will be returned.
func (sc *SuperClient) ApproveDomainTransfer(domainName string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainTransferApprove, domainName)

	msg := epp.GetEPPDomainTransferApprove(domainName, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// QueryDomainTransfer takes a domain name and will query its transfer
// status and return the doamin transfer object if it exists, otherwise
// an error is returned.
func (sc *SuperClient) QueryDomainTransfer(domainName string) (trResp *epp.DomainTrnDataResp, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainTransferQuery, domainName)

	msg := epp.GetEPPDomainTransferQuery(domainName, action.ClientTransactionID)
	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		if err != nil {
			return nil, action, fmt.Errorf("unexpected epp error: %w", err)
		}

		if respMsg.MessageType() == epp.ResponseDomainTransferType {
			return respMsg.ResponseObject.ResultData.DomainTrnDataResp, action, nil
		}

		err = ErrUnexpectedReponse

		action.SetError(err)

		return nil, action, err
	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return nil, action, ErrResponseTimeout
	}
}

// DomainCreate takes a domain name and the number of years to register
// the domain for and attempt to register the domain. If the domain
// registration is successful a response code of 1000 and no error will
// be returned, otherwise an error code and error object will be
// returned.
func (sc *SuperClient) DomainCreate(domainName string, registrationYears int) (responseCode int, action lib.EPPAction, err error) {
	per := epp.DomainPeriod{}
	per.Unit = epp.DomainPeriodYear
	per.Value = registrationYears

	authInfo, aiErr := GetAuthInfo()
	if aiErr != nil {
		return 0, action, fmt.Errorf("unexpected authinfo error: %w", aiErr)
	}

	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainCreate, domainName)
	action.AddNote(fmt.Sprintf("Registration Years: %d", registrationYears))

	msg := epp.GetEPPDomainCreate(domainName, per, []epp.DomainHost{}, nil, nil, nil, nil, authInfo, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// DomainDelete takes a domain name and will attempt to delete the
// domain object from the registry. In the event of an error processing
// the delete request, the error code and the error body will be
// returned.
func (sc *SuperClient) DomainDelete(domainName string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainDelete, domainName)

	msg := epp.GetEPPDomainDelete(domainName, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// DomainAddHosts will attempt to add the provided hosts to a domain
// object. If an error occurs, the error and the response code will
// be returned.
func (sc *SuperClient) DomainAddHosts(domainname string, hosts []string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainAddHosts, domainname)

	for _, host := range hosts {
		action.AddNote(fmt.Sprintf("Add Host: %s", host))
	}

	addRemove := epp.GetEPPDomainUpdateAddRemove(hosts, []epp.DomainContact{}, []string{})
	msg := epp.GetEPPDomainUpdate(domainname, &addRemove, nil, nil, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// DomainRemoveHosts will attempt to remove the provided hosts from a
// domain object. If an error occurs, the error and the response code
// will be returned.
func (sc *SuperClient) DomainRemoveHosts(domainname string, hosts []string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainRemoveHosts, domainname)

	for _, host := range hosts {
		action.AddNote(fmt.Sprintf("Remove Host: %s", host))
	}

	addRemove := epp.GetEPPDomainUpdateAddRemove(hosts, []epp.DomainContact{}, []string{})
	msg := epp.GetEPPDomainUpdate(domainname, nil, &addRemove, nil, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// DomainAddStatuses will attempt to add the provided statuses to a
// domain object. If an error occurs, the error and the response code
// will be returned.
func (sc *SuperClient) DomainAddStatuses(domainname string, statuses []string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())

	action.SetAction(lib.EPPLogActionDomainAddStatuses, domainname)

	for _, status := range statuses {
		action.AddNote(fmt.Sprintf("Add Status: %s", status))
	}

	addRemove := epp.GetEPPDomainUpdateAddRemove([]string{}, []epp.DomainContact{}, statuses)
	msg := epp.GetEPPDomainUpdate(domainname, &addRemove, nil, nil, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// DomainRemoveStatuses will attempt to remove the provided statuses
// from a domain object. If an error occurs, the error and the response
// code will be returned.
func (sc *SuperClient) DomainRemoveStatuses(domainname string, statuses []string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())

	action.SetAction(lib.EPPLogActionDomainRemoveStatuses, domainname)

	for _, status := range statuses {
		action.AddNote(fmt.Sprintf("Remove Status: %s", status))
	}

	addRemove := epp.GetEPPDomainUpdateAddRemove([]string{}, []epp.DomainContact{}, statuses)
	msg := epp.GetEPPDomainUpdate(domainname, nil, &addRemove, nil, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// DomainAddDSRecords will attempt to add the ds record(s) provided to the
// domain object. If an error occurs, the error and the response code will be
// returned.
func (sc *SuperClient) DomainAddDSRecords(domainname string, records []epp.DSData) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())

	action.SetAction(lib.EPPLogActionDomainAddDSRecord, domainname)

	for _, record := range records {
		action.AddNote(fmt.Sprintf("Add DS Record: %d %d %d %s", record.KeyTag, record.Alg, record.DigestType, record.Digest))
	}

	addDS := epp.GetEPPDomainSecDNSUpdate(domainname, records, []epp.DSData{}, action.ClientTransactionID)

	return sc.expect1000Response(addDS, &action)
}

// DomainRemoveDSRecords will attempt to remove the ds record(s) provided from
// domain object. If an error occurs, the error and the response code will be
// returned.
func (sc *SuperClient) DomainRemoveDSRecords(domainname string, records []epp.DSData) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())

	action.SetAction(lib.EPPLogActionDomainRemoveDSRecord, domainname)

	for _, record := range records {
		action.AddNote(fmt.Sprintf("Remove DS Record: %d %d %d %s", record.KeyTag, record.Alg, record.DigestType, record.Digest))
	}

	remDS := epp.GetEPPDomainSecDNSUpdate(domainname, []epp.DSData{}, records, action.ClientTransactionID)

	return sc.expect1000Response(remDS, &action)
}

// DomainChangeAuthInfo will attemtp to change the domain auth info
// value at the registry. If an error occurs, the error and the response
// will be returned.
func (sc *SuperClient) DomainChangeAuthInfo(domainname string, newAuthInfo string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainChangeAuthInfo, domainname)

	change := epp.GetEPPDomainUpdateChange(nil, &newAuthInfo)
	msg := epp.GetEPPDomainUpdate(domainname, nil, nil, &change, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// DomainRenew will take a domain name, the current expire date and the
// duration for the renewal and try to send the renew request to the
// epp server. In the event that the renewal fails, an error and
// response code will be returned.
func (sc *SuperClient) DomainRenew(domainname string, currentExpireDate string, duration epp.DomainPeriod) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainRenew, domainname)
	action.AddNote(fmt.Sprintf("Current Expire Date: %s", currentExpireDate))

	switch duration.Unit {
	case epp.DomainPeriodMonth:
		action.AddNote(fmt.Sprintf("Renew Duration: %d month(s)", duration.Value))
	case epp.DomainPeriodYear:
		action.AddNote(fmt.Sprintf("Renew Duration: %d year(s)", duration.Value))
	}

	msg := epp.GetEPPDomainRenew(domainname, currentExpireDate, duration, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// HostCreate takes a hostname in addition to two lists of IP address,
// one for IPv4 and IPv6, and will attempt to create the host object. If
// the host creation fails and error and response code will be returned.
func (sc *SuperClient) HostCreate(hostname string, ipv4 []string, ipv6 []string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionHostCreate, hostname)

	for _, v4addr := range ipv4 {
		action.AddNote(fmt.Sprintf("Add IPv4 Address: %s", v4addr))
	}

	for _, v6addr := range ipv6 {
		action.AddNote(fmt.Sprintf("Add IPv6 Address: %s", v6addr))
	}

	msg := epp.GetEPPHostCreate(hostname, ipv4, ipv6, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// HostDelete takes a hostname and will attempt to delete the hostname
// from the registry. If there is an error with the delete operation
// the response code and the error will be returned.
func (sc *SuperClient) HostDelete(hostname string) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionHostDelete, hostname)

	msg := epp.GetEPPHostDelete(hostname, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// HostChangeIPAddresses will take the hostname and the list of IPs to
// either add or remove (both v4 and v6) from the host object and
// attempt to send the request to the registry. If an error occurs, the
// response code and error will be returned.
func (sc *SuperClient) HostChangeIPAddresses(hostname string, ipv4Add []string, ipv4Rem []string, ipv6Add []string, ipv6Rem []string) (responseCode int, action lib.EPPAction, err error) {
	var add *epp.HostUpdateAddRemove

	var rem *epp.HostUpdateAddRemove

	action = lib.NewEPPAction(sc.getTransactionID())

	action.SetAction(lib.EPPLogActionHostUpdate, hostname)

	if len(ipv4Add) > 0 || len(ipv6Add) > 0 {
		add = &epp.HostUpdateAddRemove{}

		for _, v4addr := range ipv4Add {
			addr := epp.HostAddress{}
			addr.IPVersion = epp.IPv4
			addr.Address = v4addr
			add.Addresses = append(add.Addresses, addr)

			action.AddNote(fmt.Sprintf("Add IPv4 Address: %s", v4addr))
		}

		for _, v6addr := range ipv6Add {
			addr := epp.HostAddress{}
			addr.IPVersion = epp.IPv6
			addr.Address = v6addr
			add.Addresses = append(add.Addresses, addr)

			action.AddNote(fmt.Sprintf("Add IPv6 Address: %s", v6addr))
		}
	}

	if len(ipv4Rem) > 0 || len(ipv6Rem) > 0 {
		rem = &epp.HostUpdateAddRemove{}

		for _, v4addr := range ipv4Rem {
			addr := epp.HostAddress{}
			addr.IPVersion = epp.IPv4
			addr.Address = v4addr
			rem.Addresses = append(rem.Addresses, addr)

			action.AddNote(fmt.Sprintf("Remove IPv4 Address: %s", v4addr))
		}

		for _, v6addr := range ipv6Rem {
			addr := epp.HostAddress{}
			addr.IPVersion = epp.IPv6
			addr.Address = v6addr
			rem.Addresses = append(rem.Addresses, addr)

			action.AddNote(fmt.Sprintf("Adding IPv6 Address: %s", v6addr))
		}
	}

	msg := epp.GetEPPHostUpdate(hostname, add, rem, nil, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// HostStatusUpdate will take the hostname and the up to two boolean pointers
// and either add or remove the status flags that correspond to each of the
// options at the registry. If an error occurs, the response code and error will
// be returned.
func (sc *SuperClient) HostStatusUpdate(hostname string, clientUpdate *bool, clientDelete *bool) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionHostUpdate, hostname)

	if clientUpdate != nil || clientDelete != nil {
		add := &epp.HostUpdateAddRemove{}
		rem := &epp.HostUpdateAddRemove{}

		needAdd := false
		needRem := false

		if clientUpdate != nil {
			status := epp.HostStatus{}
			status.StatusFlag = epp.StatusClientUpdateProhibited

			if *clientUpdate {
				action.AddNote(fmt.Sprintf("Add Status: %s", status.StatusFlag))
				add.Statuses = append(add.Statuses, status)
				needAdd = true
			} else {
				action.AddNote(fmt.Sprintf("Remove Status: %s", status.StatusFlag))
				rem.Statuses = append(rem.Statuses, status)
				needRem = true
			}
		}

		if clientDelete != nil {
			status := epp.HostStatus{}
			status.StatusFlag = epp.StatusClientDeleteProhibited

			if *clientDelete {
				action.AddNote(fmt.Sprintf("Add Status: %s", status.StatusFlag))
				add.Statuses = append(add.Statuses, status)
				needAdd = true
			} else {
				action.AddNote(fmt.Sprintf("Remove Status: %s", status.StatusFlag))
				rem.Statuses = append(rem.Statuses, status)
				needRem = true
			}
		}

		if !needAdd {
			add = nil
		}

		if !needRem {
			rem = nil
		}

		msg := epp.GetEPPHostUpdate(hostname, add, rem, nil, action.ClientTransactionID)

		return sc.expect1000Response(msg, &action)
	}

	return 0, action, nil
}

// SyncDomain is used to send a sync message to the reigstry to align a
// domain's expiration date to a specific day and month.
func (sc *SuperClient) SyncDomain(domainname string, month time.Month, day int) (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction(sc.getTransactionID())
	action.SetAction(lib.EPPLogActionDomainSync, domainname)
	action.AddNote(fmt.Sprintf("Sync Domain to %s %2d", month.String(), day))

	msg := epp.GetEPPDomainSyncUpdate(domainname, month, day, action.ClientTransactionID)

	return sc.expect1000Response(msg, &action)
}

// SendHello will send a hello message to the registry and return a
// response code and an error if an error occurs.
func (sc *SuperClient) SendHello() (responseCode int, action lib.EPPAction, err error) {
	action = lib.NewEPPAction("")
	action.SetAction(lib.EPPLogActionHello, "")

	msg := epp.GetEPPHello()
	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		if err != nil {
			return 0, action, fmt.Errorf("unexpected epp error: %w", err)
		}

		if respMsg.ResponseObject != nil {
			if respMsg.ResponseObject.IsError() {
				return respMsg.ResponseObject.Result.Code, action, fmt.Errorf("EPP error: %w", respMsg.ResponseObject.GetError())
			}
		}

		return 0, action, nil
	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return 0, action, ErrResponseTimeout
	}
}

// expect1000Response is function that is used to send an epp message
// and receive a response expecting that the response message will be
// a 1000 response code with no body.
func (sc *SuperClient) expect1000Response(msg epp.Epp, action *lib.EPPAction) (responseCode int, actionOut lib.EPPAction, err error) {
	sc.client.NewWork <- msg
	timeout := sc.getTimeout()

	select {
	case respMsg := <-sc.client.WorkResponse:
		err := action.HandleResponse(respMsg)
		if err != nil {
			return 0, *action, fmt.Errorf("unexpected epp error: %w", err)
		}

		if respMsg.ResponseObject != nil {
			if respMsg.ResponseObject.IsError() {
				return respMsg.ResponseObject.Result.Code, *action, fmt.Errorf("EPP error: %w", respMsg.ResponseObject.GetError())
			}

			if respMsg.ResponseObject.Result.Code == epp.ResponseCodeCommandSuccessful {
				// successful registration
				return respMsg.ResponseObject.Result.Code, *action, nil
			}

			err = ErrUnexpectedReponse

			action.SetError(err)

			return respMsg.ResponseObject.Result.Code, *action, err
		}
	case <-timeout:
		action.SetError(ErrResponseTimeout)

		return 0, *action, ErrResponseTimeout
	}

	action.SetError(ErrUnhandledResponse)

	return 0, *action, ErrUnhandledResponse
}
