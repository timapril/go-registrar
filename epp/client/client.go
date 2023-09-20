package client

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/timapril/go-registrar/epp"

	logging "github.com/op/go-logging"
)

// EPPConnTimeout is the length of time before we assume that the
// connection has timed out or is about to.
const EPPConnTimeout = 90 * time.Second

// ChanBufferSize limits the size of EPP message buffers.
const ChanBufferSize = 25

// channelSize represents the size of the channel used to manage the FSM.
const channelSize = 1000

// ErrTimeout is the error that is returned when a timeout occurs.
var ErrTimeout = errors.New("a timeout occurred while waiting for the server to respond")

// ErrNotPreparedClient indicates that the client was not prepared
// when attempting to use the client.
var ErrNotPreparedClient = errors.New("cannot start a client that is not prepared")

// ErrUnknownState indicates that the FSM was reporting an unknown state.
var ErrUnknownState = errors.New("unknown finite state machine state")

// ErrUnexpectedMessageType indicates that an unexpected EPP message was
// received.
var ErrUnexpectedMessageType = errors.New("unexpected EPP message")

// EPPClient is used to communicate with the EPP Server.
type EPPClient struct {
	EPPConn net.Conn

	Writer *bufio.Writer

	ClientConfig Config

	RecvChannel chan epp.Epp
	SendChannel chan epp.Epp

	NewWork      chan epp.Epp
	WorkResponse chan epp.Epp

	SendErrorChannel chan error

	StopChannel   chan bool
	LogoutChannel chan bool
	StatusChannel chan StatusMessage

	RecentGreeting   epp.Greeting
	LastGreetingTime time.Time

	Log *logging.Logger

	loginCallback  func(epp.Epp) error
	logoutCallback func(epp.Epp) error

	currentState EPPClientState

	prepared bool

	senderStop  chan bool
	shouldlogin bool
}

// StatusMessage is used to report the status of the client to consumers.
type StatusMessage struct {
	Status EPPClientState
}

// Prepare will setup the.
func (c *EPPClient) Prepare(conf Config, log *logging.Logger, loginCallback func(epp.Epp) error, logoutCallback func(epp.Epp) error) {
	c.Log = log

	c.ClientConfig = conf
	c.StatusChannel = make(chan StatusMessage, channelSize)
	c.LogoutChannel = make(chan bool, 1)

	c.loginCallback = loginCallback
	c.logoutCallback = logoutCallback

	c.prepared = true
	c.shouldlogin = true
}

// Start initializes the connection as well as the RX and TX go routines
// for the EPP Client.
func (c *EPPClient) Start() error {
	if c.prepared {
		c.Log.Debug("Client is prepared, starting the FSM")

		go func() {
			err := c.StartFSM()
			if err != nil {
				c.Log.Error(err)
			}
		}()

		return nil
	}

	return ErrNotPreparedClient
}

// Stop will trigger the EPP client to disconnect from the server.
func (c *EPPClient) Stop() {
	c.StopChannel <- true
}

// EPPClientState is a type that has a string defining the states of the
// EPP Client.
type EPPClientState string

const (
	// ClosedConnection represents the "connection closed" state in the
	// client FSM.
	ClosedConnection EPPClientState = "closedConnection"

	// NewConnection represents the "new connection" state in the client
	// FSM.
	NewConnection EPPClientState = "newConnection"

	// OpenedConnection represents the "opened connection" state in the
	// client FSM.
	OpenedConnection EPPClientState = "openedConnection"

	// PreparePreLoginHello represents the "prepare hello" prior to login
	// state in the client FSM.
	PreparePreLoginHello EPPClientState = "preparePreLoginHello"

	// PrepareLogin represents the "prepare connection" state in the
	// client FSM.
	PrepareLogin EPPClientState = "prepareLogin"

	// WaitLogin represents the "wait for login response" state in the
	// client FSM.
	WaitLogin EPPClientState = "waitForLoginResponse"

	// WaitForWork represents the "wait for work" state in the
	// client FSM.
	WaitForWork EPPClientState = "waitingForWork"

	// WaitForWorkHello represents the "prepare hello" of the "wait
	// for work" state in the client FSM.
	WaitForWorkHello EPPClientState = "waitingForWorkHello"

	// WaitForWorkHelloResp represents the "process greeting" state in
	// the client FSM.
	WaitForWorkHelloResp EPPClientState = "waitingForWorkHelloResp"

	// WaitForCommandResponse represents the "wait for response" state
	// in the client FSM.
	WaitForCommandResponse EPPClientState = "waitForCommandResponse"

	// PrepareLogout represents the "Prepare <logout>" state in the
	// client FSM.
	PrepareLogout EPPClientState = "preparelogout"

	// WaitLogout represents the state where the system is waiting for
	// the server to return a response after attempting to logout.
	WaitLogout EPPClientState = "waitLogout"

	// ShutdownRequested is a state that is used to indicate that a
	// shutdown has been requested outside the normal FSM process.
	ShutdownRequested EPPClientState = "shutdownRequested"
)

// IsRunningEPPState returns true if the state passed corresponds to
// a running epp client (authenticated and active).
func IsRunningEPPState(state EPPClientState) bool {
	return state == WaitForWork || state == WaitForWorkHello || state == WaitForWorkHelloResp
}

type eppClientFSMStateHandler func(*EPPClient) (EPPClientState, error)

var fsmStateMap = map[EPPClientState]eppClientFSMStateHandler{
	ClosedConnection:       (*EPPClient).closedConnectionState,
	NewConnection:          (*EPPClient).newConnectionState,
	OpenedConnection:       (*EPPClient).openedConnectionState,
	PrepareLogin:           (*EPPClient).prepareLoginState,
	WaitLogin:              (*EPPClient).waitLoginState,
	WaitForWork:            (*EPPClient).waitForWorkState,
	WaitForWorkHello:       (*EPPClient).waitForWorkHelloState,
	WaitForWorkHelloResp:   (*EPPClient).waitForWorkHelloRespState,
	WaitForCommandResponse: (*EPPClient).waitForCommandResponseState,
	PrepareLogout:          (*EPPClient).prepareLogoutState,
	WaitLogout:             (*EPPClient).waitLogoutState,
	ShutdownRequested:      (*EPPClient).shutdownState,
}

// StartFSM will starts and runs the fininte state machine associated
// with the EPP client. If this function returns nil if the FMS exits
// gracefully. If an error occurs, an error will be returned.
func (c *EPPClient) StartFSM() error {
	c.Log.Notice("Starting Registrar Client")
	c.currentState = ClosedConnection

	for {
		var err error

		var nextState EPPClientState

		c.Log.Notice(fmt.Sprintf("Current State %s", c.currentState))
		select {
		case msg := <-c.SendErrorChannel:
			c.Log.Errorf("%s", msg.Error())
		case <-c.StopChannel:
			c.currentState = ShutdownRequested
		default:
		}

		nextState, err = fsmStateMap[c.currentState](c)

		if nextState == ClosedConnection {
			return nil
		}

		if nextState != c.currentState {
			c.Log.Notice(fmt.Sprintf("State Transition to State %s -> %s", c.currentState, nextState))
		}

		c.currentState = nextState
		c.StatusChannel <- StatusMessage{Status: c.currentState}

		if err != nil {
			c.Log.Error(err.Error())

			return err
		}
	}
}

func (c *EPPClient) waitForCommandResponseState() (nextState EPPClientState, err error) {
	timeout := makeTimeout(EPPConnTimeout)

	c.Log.Debug("waitForCommandResponse: Start")

	select {
	case <-c.StopChannel:
		c.Log.Debug("waitForCommandResponse: shutdown request")

		return ShutdownRequested, nil
	case msg := <-c.RecvChannel:
		c.Log.Debug("waitForCommandResponse: got message")
		c.WorkResponse <- msg

		return WaitForWork, nil
	case <-timeout:
		c.Log.Debug("waitForCommandResponse: timeout")

		return PrepareLogout, ErrTimeout
	}
}

func (c *EPPClient) waitForWorkHelloRespState() (nextState EPPClientState, err error) {
	timeout := makeTimeout(EPPConnTimeout)

	select {
	case <-c.StopChannel:
		return ShutdownRequested, nil
	case msg := <-c.RecvChannel:
		if msg.MessageType() == epp.GreetingType {
			c.RecentGreeting = *msg.GreetingObject
			c.LastGreetingTime = time.Now()

			return WaitForWork, nil
		}

	case <-timeout:
		return ClosedConnection, ErrTimeout
	}

	return WaitForWorkHelloResp, nil
}

func (c *EPPClient) waitForWorkHelloState() (nextState EPPClientState, err error) {
	HelloObj := epp.GetEPPHello()

	c.SendChannel <- HelloObj

	return WaitForWorkHelloResp, nil
}

const (
	eppKeepalivePeriodNomenator   = 80
	eppKeepalivePeriodDenomenator = 100
)

func (c *EPPClient) waitForWorkState() (nextState EPPClientState, err error) {
	timeoutTime := (EPPConnTimeout * eppKeepalivePeriodNomenator / eppKeepalivePeriodDenomenator) - time.Since(c.LastGreetingTime)

	if timeoutTime.Seconds() < 0 {
		return WaitForWorkHello, nil
	}

	c.Log.Debugf("WaitForWorkState: Timeout Remaining before next hello %d s", timeoutTime.Seconds())
	timeout := makeTimeout(timeoutTime)

	select {
	case <-c.StopChannel:
		return ShutdownRequested, nil
	case msg := <-c.RecvChannel:
		c.WorkResponse <- msg
	case <-timeout:
		return WaitForWorkHello, ErrTimeout
	case msg := <-c.NewWork:
		c.SendChannel <- msg
		c.Log.Debug("WaitForWorkState: Command message sent")

		return WaitForCommandResponse, nil
	case <-c.LogoutChannel:
		c.shouldlogin = false

		return PrepareLogout, nil
	}

	return WaitForWork, nil
}

func (c *EPPClient) waitLoginState() (nextState EPPClientState, err error) {
	timeout := makeTimeout(EPPConnTimeout)
	select {
	case <-c.StopChannel:
		return ShutdownRequested, nil
	case msg := <-c.RecvChannel:
		if c.loginCallback != nil {
			err := c.loginCallback(msg)
			if err != nil {
				return nextState, err
			}
		}

		if msg.MessageType() == epp.ResponseType {
			if msg.ResponseObject.IsError() {
				return PrepareLogin, fmt.Errorf("epp response error: %w", msg.ResponseObject.GetError())
			}

			return WaitForWork, nil
		}

	case <-timeout:
		return ClosedConnection, ErrTimeout
	}

	return WaitLogin, nil
}

func (c *EPPClient) waitLogoutState() (nextState EPPClientState, err error) {
	timeout := makeTimeout(EPPConnTimeout)
	select {
	case <-c.StopChannel:
		return ShutdownRequested, nil
	case msg := <-c.RecvChannel:
		if c.logoutCallback != nil {
			err := c.logoutCallback(msg)
			if err != nil {
				return nextState, err
			}
		}

		if msg.MessageType() == epp.ResponseType {
			if msg.ResponseObject.IsError() {
				return ClosedConnection, fmt.Errorf("epp response error: %w", msg.ResponseObject.GetError())
			}

			return ClosedConnection, nil
		}
	case <-timeout:
		return ClosedConnection, ErrTimeout
	}

	return ClosedConnection, nil
}

func (c *EPPClient) shutdownState() (nextState EPPClientState, err error) {
	c.senderStop <- true

	return ClosedConnection, nil
}

func (c *EPPClient) prepareLoginState() (nextState EPPClientState, err error) {
	LoginObj := epp.GetEPPLogin(c.ClientConfig.Username,
		c.ClientConfig.Password,
		c.ClientConfig.GetNewTransactionID(),
		c.RecentGreeting.SvcMenu)

	c.SendChannel <- LoginObj

	return WaitLogin, nil
}

func (c *EPPClient) prepareLogoutState() (nextState EPPClientState, err error) {
	LogoutObj := epp.GetEPPLogout(c.ClientConfig.GetNewTransactionID())
	c.SendChannel <- LogoutObj

	return WaitLogout, nil
}

func (c *EPPClient) openedConnectionState() (nextState EPPClientState, err error) {
	timeout := makeTimeout(EPPConnTimeout)
	select {
	case <-c.StopChannel:
		return ShutdownRequested, nil
	case msg := <-c.RecvChannel:
		if msg.MessageType() == epp.GreetingType {
			c.RecentGreeting = *msg.GreetingObject
			c.LastGreetingTime = time.Now()

			return PrepareLogin, nil
		}

		c.Log.Errorf("Unexpected message - type: %s", msg.MessageType())

		messageString, _ := msg.ToStringCS("S:")

		c.Log.Infof("Received Message: %s", messageString)

		return "", ErrUnexpectedMessageType

	case <-timeout:
		c.Log.Info("A timeout has occurred when waiting for a greeting")

		return PreparePreLoginHello, ErrTimeout
	}
}

func (c *EPPClient) newConnectionState() (nextState EPPClientState, err error) {
	c.Log.Debug(fmt.Sprintf("Opening connection to %s", c.ClientConfig.GetConnectionString()))

	conn, connectionErr := net.Dial("tcp", c.ClientConfig.GetConnectionString())
	if connectionErr != nil {
		c.Log.Critical(fmt.Sprintf("An error occurred opening the connection: %s", connectionErr.Error()))

		return ClosedConnection, fmt.Errorf("error opening epp connection: %w", connectionErr)
	}

	c.EPPConn = conn
	c.Writer = bufio.NewWriter(conn)

	c.senderStop = make(chan bool)

	go c.EPPListener()
	go c.EPPSender(c.senderStop)

	return OpenedConnection, nil
}

func (c *EPPClient) closedConnectionState() (EPPClientState, error) {
	if c.shouldlogin {
		return NewConnection, nil
	}

	return ShutdownRequested, nil
}

// EPPSender starts and listens to a channel for messages to send to the
// EPP server.
func (c *EPPClient) EPPSender(stopChan chan bool) {
	c.Log.Info("Starting EPP Sender Thread")

	for {
		timeout := makeTimeout(EPPConnTimeout)
		select {
		case <-stopChan:
			c.Log.Debug("Sender: got stop message")

			return
		case msg := <-c.SendChannel:
			c.Log.Debug("Sender: got new message to send")

			outputBytes, encodeErr := msg.EncodeEPP()

			if encodeErr != nil {
				c.SendErrorChannel <- encodeErr

				return
			}

			_, writeErr := c.Writer.Write(outputBytes)

			if writeErr != nil {
				c.SendErrorChannel <- writeErr

				return
			}

			flushErr := c.Writer.Flush()

			if flushErr != nil {
				c.SendErrorChannel <- flushErr

				return
			}

			c.printEPPMessage(msg, "C:")
			c.Log.Debug("Sender: message sent")
		case <-timeout:
			c.Log.Error("Sending thread reached timeout")
		}
	}
}

// EPPListener starts to listen on the connection from the server and
// processes the messages that it gets back.
func (c *EPPClient) EPPListener() {
	scanner := bufio.NewScanner(c.EPPConn)
	scanner.Split(epp.WireSplit)

	c.Log.Info("Starting to listen for incoming messages")

	for scanner.Scan() {
		text := scanner.Text()

		c.Log.Debug(text)

		outObj, unmarshallErr := epp.UnmarshalMessage([]byte(text))

		if unmarshallErr != nil {
			c.Log.Error(unmarshallErr.Error())

			return
		}

		typed := outObj.TypedMessage()

		c.printEPPMessage(typed, "S:")

		c.RecvChannel <- typed
		c.Log.Debug("Listen: Message sent to the channel")
	}
}

// GetCurrentState will get the current state that the client is in.
func (c *EPPClient) GetCurrentState() EPPClientState {
	return c.currentState
}

// NewEPPClient prepares and returns a new EPPClient.
func NewEPPClient() EPPClient {
	eppClient := EPPClient{}
	eppClient.SendChannel = make(chan epp.Epp, ChanBufferSize)
	eppClient.RecvChannel = make(chan epp.Epp, ChanBufferSize)
	eppClient.SendErrorChannel = make(chan error, ChanBufferSize)
	eppClient.StopChannel = make(chan bool, 1)
	eppClient.LogoutChannel = make(chan bool, 1)
	eppClient.NewWork = make(chan epp.Epp, ChanBufferSize)
	eppClient.WorkResponse = make(chan epp.Epp, ChanBufferSize)

	eppClient.prepared = false
	eppClient.currentState = ClosedConnection

	return eppClient
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

func (c *EPPClient) printEPPMessage(msg epp.Epp, prefix string) {
	messageString, _ := msg.ToStringCS(prefix)
	for _, line := range strings.Split(messageString, "\n") {
		c.Log.Debug(line)
	}
}

// GetTXID will return a new transaction ID for use with the client.
func (c *EPPClient) GetTXID() string {
	return c.ClientConfig.GetNewTransactionID()
}

// Server State Machine
// From http://tools.ietf.org/html/rfc5730
//
//
// .     |
// .     V
// .+-----------------+                  +-----------------+
// .|   Waiting for   |     Connected    |     Prepare     |
// .|      Client     |----------------->|     Greeting    |
// .+-----------------+    or <hello>    +-----------------+
// .   ^                                           |
// .   | Close Connection                     Send |
// .   |     or Idle                      Greeting |
// .+-----------------+                            V
// .|       End       |     Timeout      +-----------------+
// .|     Session     |<-----------------|   Waiting for   |
// .+-----------------+                  |      Client     |
// .   ^    ^    ^        Send +-------->|  Authentication |
// .   |    |    |    Response |         +-----------------+
// .   |    |    |     +--------------+            |
// .   |    |    |     | Prepare Fail |            | <login>
// .   |    |    +-----|   Response   |            | Received
// .   |    |    Send  +--------------+            V
// .   |    |    2501          ^         +-----------------+
// .   |    |   Response       |         |   Processing    |
// .   |    |                  +---------|     <login>     |
// .   |    |                  Auth Fail +-----------------+
// .   |    |       Timeout                         |
// .   |    +-------------------------------+       | Auth OK
// .   |                                    |       V
// .   |   +-----------------+  <hello>  +-----------------+
// .   |   |     Prepare     |<----------|   Waiting for   |
// .   |   |     Greeting    |---------->|   Command or    |
// .   |   +-----------------+   Send    |     <hello>     |
// .   | Send x5xx             Greeting  +-----------------+
// .   | Response  +-----------------+  Send    ^  |
// .   +-----------|     Prepare     | Response |  | Command
// .	   	          |     Response    |----------+  | Received
// .               +-----------------+             V
// .							^          +-----------------+
// .					Command |          |   Processing    |
// .				  Processed +----------|     Command     |
// .									   +-----------------+
