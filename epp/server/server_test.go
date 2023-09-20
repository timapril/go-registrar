package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/timapril/go-registrar/epp"
	"github.com/timapril/go-registrar/lib"

	logging "github.com/op/go-logging"

	. "github.com/smartystreets/goconvey/convey"
)

var log = logging.MustGetLogger("reg")

const (
	DEBUG              = true
	testingPort        = 1702
	testingHost        = "127.0.0.1"
	ClientTXID1        = "EPP-TEST-01"
	ClientTXID2        = "EPP-TEST-02"
	ClientTXID3        = "EPP-TEST-03"
	ClientTXID4        = "EPP-TEST-04"
	TestingDomainName1 = "EXAMPLE.COM"
	TestingDomainName2 = "EXAMPLE2.COM"
)

// ErrServerNotRunningWhenExpected indicates that the server is not running
// after it has been started.
var ErrServerNotRunningWhenExpected = errors.New("server not running when expected")

// TestNewEPPServer tries to generate a new epp server and then verifies
// that the values set are what would be expected.
func TestNewEPPServer(t *testing.T) {
	t.Parallel()
	Convey("Given a valid host, port and duration", t, func() {
		host := testingHost
		port := testingPort
		duration := 2 * time.Second
		Convey("Calling NewEPPServer should create a new server", func() {
			srv := NewEPPServer(host, port, duration)
			So(srv.Host, ShouldEqual, host)
			So(srv.Port, ShouldEqual, port)
			So(srv.Timeout, ShouldEqual, duration)
		})
	})
}

// TestEPPServerStartStop starts a server, starts a connection, closes
// the connection and then tries to make the server exit gracefully
// without returning an error to the watching thread.
func TestEPPServerStartStop(t *testing.T) {
	t.Parallel()
	Convey("Given a default server", t, func() {
		host := testingHost
		port := testingPort
		duration := 2 * time.Second

		connString := fmt.Sprintf("%s:%d", host, port)

		srv := NewEPPServer(host, port, duration)

		go ServerMainThreadWatcher(t, srv, "Server Start Stop", false)
		time.Sleep(time.Second * 2)
		conn, err := net.Dial("tcp", connString)
		So(err, ShouldBeNil)
		So(conn, ShouldNotBeNil)
		conn.Close()

		srv.KillChan <- true
		conn, err = net.Dial("tcp", connString)
		So(err, ShouldNotBeNil)
		So(conn, ShouldBeNil)
	})
}

// TestEPPServerStartStopBad starts a server, starts a session and then
// kills the server socket without notifying the server that it should
// expect to be closing. This should return an error the main thread
// watching script.
func TestEPPServerStartStopBad(t *testing.T) {
	t.Parallel()
	Convey("Given a default server", t, func() {
		host := testingHost
		port := testingPort
		duration := 2 * time.Second

		connString := fmt.Sprintf("%s:%d", host, port)

		srv := NewEPPServer(host, port, duration)

		go ServerMainThreadWatcher(t, srv, "Server Start Stop Bad", true)
		time.Sleep(time.Second * 2)
		conn, err := net.Dial("tcp", connString)
		So(err, ShouldBeNil)
		So(conn, ShouldNotBeNil)
		conn.Close()

		srv.KillChan <- false
		time.Sleep(10 * time.Millisecond)
		conn, err = net.Dial("tcp", connString)
		So(err, ShouldNotBeNil)
		So(conn, ShouldBeNil)
	})
}

// TestEPPServerStartDouble attepmts to start two servers expecting that
// the second server opening will fail due to the port being in use
// already.
func TestEPPServerStartDouble(t *testing.T) {
	t.Parallel()
	Convey("Given a default server", t, func() {
		host := testingHost
		port := testingPort
		duration := 2 * time.Second

		srv := NewEPPServer(host, port, duration)
		srv2 := NewEPPServer(host, port, duration)

		go ServerMainThreadWatcher(t, srv, "Server Start Double 1", false)
		time.Sleep(10 * time.Millisecond)
		go ServerMainThreadWatcher(t, srv2, "Server Start Double 2", true)

		time.Sleep(10 * time.Millisecond)

		srv.KillChan <- true
		srv2.KillChan <- true
	})
}

// TestGreeting starts the server, creates a client connection and
// expects a greeting message to be sent by the server to the client
// before the connectino times out.
func TestGreeting(t *testing.T) {
	t.Parallel()

	Convey("Given a default server and no packets sent a greeting should be sent after connection", t, func() {
		srv := GetDefaultServer()
		go ServerMainThreadWatcher(t, srv, "Test Greeting", true)
		run := <-srv.Running
		So(run, ShouldBeTrue)
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		timeout := GetTimeout(10 * time.Second)
		select {
		case msg := <-simpleClient.RecvChannel:
			So(msg.GreetingObject, ShouldNotBeNil)
		case timeout := <-timeout:
			So(timeout, ShouldBeFalse)
		}
		simpleClient.Close()
		srv.KillChan <- true
	})
}

// TestWaitingForClientTimeout starts a server, creates a client and
// waits until a timeout should occur and checks to make sure that the
// server sent the expected greeting message.
func TestWaitingForClientTimeout(t *testing.T) {
	t.Parallel()

	srv := GetDefaultServer()
	go ServerMainThreadWatcher(t, srv, "Client timeout", false)
	run := <-srv.Running

	if run != true {
		t.Error(ErrServerNotRunningWhenExpected)
	}

	simpleClient := GetDefaultSimpleClient()

	err := simpleClient.Start()
	So(err, ShouldBeNil)

	time.Sleep(srv.Timeout)

	msg := <-simpleClient.RecvChannel

	if msg.MessageType() != epp.GreetingType {
		t.Errorf("Expected a EPP.Greeting message but got %s", msg.MessageType())
	}

	t.Log("Timeout should have passed")
	srv.KillChan <- true
}

// TestNonCommand stars a server, creates a client and sends a byte
// string that is not a valid server message. The server should do
// nothing and the server should be able to exit gracefully.
func TestNonCommand(t *testing.T) {
	t.Parallel()

	srv := GetDefaultServer()
	go ServerMainThreadWatcher(t, srv, "Non Command", false)
	run := <-srv.Running

	if run != true {
		t.Error(ErrServerNotRunningWhenExpected)
	}

	simpleClient := GetDefaultSimpleClient()

	err := simpleClient.Start()
	So(err, ShouldBeNil)

	simpleClient.RawSendChannel <- "aaaa"

	time.Sleep(time.Second * 1)

	timeout := GetTimeout(time.Millisecond * 100)

	select {
	case msg := <-simpleClient.RecvChannel:
		if msg.MessageType() == epp.GreetingType {
			t.Log("Found greeting message as expected")
		} else {
			t.Errorf("Unexpected message type %s, expected greeting type", msg.MessageType())
		}
	case <-timeout:
		t.Error("Expected to get a greeting message but nothing found")
	}

	timeout = GetTimeout(time.Millisecond * 100)

	select {
	case msg := <-simpleClient.RecvChannel:
		t.Errorf("Got %s message when none was expected", msg.MessageType())
	case <-timeout:
		t.Log("No message found, as expected")
	}

	srv.KillChan <- true
}

// TestLogin will start a server and a client and send a login message.
// It is expected that the login will work and the server will respond
// with a 1000 result code and no error.
func TestLogin(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()

		go ServerMainThreadWatcher(t, srv, "Login Test", false)

		run := <-srv.Running

		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}

		simpleClient := GetDefaultSimpleClient()

		err := simpleClient.Start()
		So(err, ShouldBeNil)

		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)

		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())

		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		srv.KillChan <- true
	})
}

// TestLoginInvalidCommand will start a server and client and then send
// a request for a contact object before logging in. The server should
// return an error indicating that the command was not allowed without
// logging in.
func TestLoginInvalidCommand(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a invalid command", t, func() {
		srv := GetDefaultServer()

		go ServerMainThreadWatcher(t, srv, "Login invald command", false)

		run := <-srv.Running

		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}

		simpleClient := GetDefaultSimpleClient()

		err := simpleClient.Start()
		So(err, ShouldBeNil)

		msg := <-simpleClient.RecvChannel

		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)

		cltxid := ClientTXID1
		login := epp.GetEPPContactInfo("123", cltxid, "")

		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2002)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2002)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		srv.KillChan <- true
	})
}

// TestLoginInvalidLogout will start a server and a client and then send
// a logout message before sending a login message. The server should
// respond with an error.
func TestLoginInvalidLogout(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a invalid command", t, func() {
		srv := GetDefaultServer()

		go ServerMainThreadWatcher(t, srv, "Login invalid logout", false)

		run := <-srv.Running

		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}

		simpleClient := GetDefaultSimpleClient()

		err := simpleClient.Start()
		So(err, ShouldBeNil)

		msg := <-simpleClient.RecvChannel

		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)

		cltxid := ClientTXID1
		login := epp.GetEPPLogout(cltxid)

		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2002)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2002)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		srv.KillChan <- true
	})
}

// TestLoginNonCommand will start a server and a client and then try to
// send an empty epp message before logging in. An error is expected to
// be returned.
func TestLoginNonCommand(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a invalid command", t, func() {
		srv := GetDefaultServer()
		go ServerMainThreadWatcher(t, srv, "Login non command", false)
		run := <-srv.Running

		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}

		simpleClient := GetDefaultSimpleClient()

		err := simpleClient.Start()
		So(err, ShouldBeNil)

		msg := <-simpleClient.RecvChannel

		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)

		login := epp.GetEPP()

		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2002)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2002)
		srv.KillChan <- true
	})
}

// TestLoginFail will start a server and a client and then try and login
// three times with a bad password. The first two times a 2200 result
// code should be returned and for the third response a 2501 result code
// should be retrned.
func TestLoginFail(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()
		go ServerMainThreadWatcher(t, srv, "Login Fail", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "bad-password", cltxid, epp.GetDefaultServiceMenu())
		// First Login attempt - should result in 2200
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2200)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2200)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		// Second Login attempt - should result in 2200
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2200)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2200)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		// Third Login attempt - should result in 2501
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2501)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2501)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		srv.KillChan <- true
	})
}

// TestWaitingForCommandTimeout will start a server and a client and
// login. Once the session is open the client will wait the standard
// timeout length and then check to ensure that the server does not
// error after waiting the timeout period.
func TestWaitingForCommandTimeout(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()
		go ServerMainThreadWatcher(t, srv, "Waiting for command timeout", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		time.Sleep(srv.Timeout + 1*time.Second)
		t.Log("The server should not error when waiting past the timeout")
		srv.KillChan <- true
	})
}

// TestWaitingForCommandLogout will start a server and a client and
// login and then send a logout request. The server should repond with a
// result code of 1500 and no error.
func TestWaitingForCommandLogout(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()
		go ServerMainThreadWatcher(t, srv, "Waiting for command logout", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		cltxid = ClientTXID2
		logout := epp.GetEPPLogout(cltxid)
		simpleClient.SendChannel <- logout

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1500)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1500)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		t.Log("Timeout should have passed")
		srv.KillChan <- true
	})
}

// TestWaitingForCommandCheckContact will start a server and a client
// and login and then send requests to check two contacts, one that is
// in the database and one that is not. Result codes of 1000 for each of
// the check, with the existing contact showing as not available and
// the non-existent one showing as available, are expected with no
// errors.
func TestWaitingForCommandCheckContact(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()

		cont := GetTestingContactOwned()

		srv.Contacts[cont.ContactRegistryID] = cont

		go ServerMainThreadWatcher(t, srv, "Waiting for command check contact", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		cltxid = ClientTXID2
		contactCheck := epp.GetEPPContactCheck("123", cltxid)
		simpleClient.SendChannel <- contactCheck

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseContactCheckType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.ContactChkDataResp, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.ContactChkDataResp.ContactIDs, ShouldNotBeEmpty)
		So(len(msg.ResponseObject.ResultData.ContactChkDataResp.ContactIDs), ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.ContactChkDataResp.ContactIDs[0].ID.Available, ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.ContactChkDataResp.ContactIDs[0].ID.Value, ShouldEqual, "123")

		cltxid = ClientTXID3
		contactCheck2 := epp.GetEPPContactCheck("1234", cltxid)
		simpleClient.SendChannel <- contactCheck2

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseContactCheckType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.ContactChkDataResp, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.ContactChkDataResp.ContactIDs, ShouldNotBeEmpty)
		So(len(msg.ResponseObject.ResultData.ContactChkDataResp.ContactIDs), ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.ContactChkDataResp.ContactIDs[0].ID.Available, ShouldEqual, 0)
		So(msg.ResponseObject.ResultData.ContactChkDataResp.ContactIDs[0].ID.Value, ShouldEqual, "1234")

		srv.KillChan <- true
	})
}

// TestWaitingForCommandCheckDomain will start a server and a client
// and login and then send requests to check two domains, one that is in
// the database and one that is not. Result codes of 1000 for each of
// the check, with the existing domain showing as not available and the
// non-existent one showing as available, are expected with no errors.
func TestWaitingForCommandCheckDomain(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()

		dom := lib.Domain{}
		dom.DomainName = "BAR.COM"

		srv.Domains[dom.DomainName] = dom

		go ServerMainThreadWatcher(t, srv, "Waiting for command check domain", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		cltxid = ClientTXID2
		domainCheck := epp.GetEPPDomainCheck("NONEXIST.COM", cltxid)
		simpleClient.SendChannel <- domainCheck

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseDomainCheckType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.DomainChkDataResp, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.DomainChkDataResp.Domains, ShouldNotBeEmpty)
		So(len(msg.ResponseObject.ResultData.DomainChkDataResp.Domains), ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.DomainChkDataResp.Domains[0].Name.Available, ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.DomainChkDataResp.Domains[0].Name.Value, ShouldEqual, "NONEXIST.COM")

		cltxid = ClientTXID3
		domainCheck2 := epp.GetEPPDomainCheck("BAR.COM", cltxid)
		simpleClient.SendChannel <- domainCheck2

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseDomainCheckType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.DomainChkDataResp, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.DomainChkDataResp.Domains, ShouldNotBeEmpty)
		So(len(msg.ResponseObject.ResultData.DomainChkDataResp.Domains), ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.DomainChkDataResp.Domains[0].Name.Available, ShouldEqual, 0)
		So(msg.ResponseObject.ResultData.DomainChkDataResp.Domains[0].Name.Value, ShouldEqual, "BAR.COM")

		srv.KillChan <- true
	})
}

// TestWaitingForCommandCheckHost will start a server and a client
// and login and then send requests to check two hosts, one that is in
// the database and one that is not. Result codes of 1000 for each of
// the check, with the existing host showing as not available and the
// non-existent one showing as available, are expected with no errors.
func TestWaitingForCommandCheckHost(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()

		hos := lib.Host{}
		hos.HostName = "NS2.FOO.COM"

		srv.Hosts[hos.HostName] = hos

		go ServerMainThreadWatcher(t, srv, "Waiting for command check host", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		cltxid = ClientTXID2
		hostCheck := epp.GetEPPHostCheck("NS1.FOO.COM", cltxid)
		simpleClient.SendChannel <- hostCheck

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseHostCheckType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.HostChkDataResp, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.HostChkDataResp.Hosts, ShouldNotBeEmpty)
		So(len(msg.ResponseObject.ResultData.HostChkDataResp.Hosts), ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.HostChkDataResp.Hosts[0].Name.Available, ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.HostChkDataResp.Hosts[0].Name.Value, ShouldEqual, "NS1.FOO.COM")

		cltxid = ClientTXID3
		hostCheck2 := epp.GetEPPHostCheck("NS2.FOO.COM", cltxid)
		simpleClient.SendChannel <- hostCheck2

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseHostCheckType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.HostChkDataResp, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.HostChkDataResp.Hosts, ShouldNotBeEmpty)
		So(len(msg.ResponseObject.ResultData.HostChkDataResp.Hosts), ShouldEqual, 1)
		So(msg.ResponseObject.ResultData.HostChkDataResp.Hosts[0].Name.Available, ShouldEqual, 0)
		So(msg.ResponseObject.ResultData.HostChkDataResp.Hosts[0].Name.Value, ShouldEqual, "NS2.FOO.COM")

		srv.KillChan <- true
	})
}

// TestWaitingForCommandInfoContact will start a server and a client
// and login and then send requests for info about two contacts that
// exist in the database (one owned by the user and one that is not)
// and 1 that does not exist. The first two should return the expected
// data and the third should return an error.
func TestWaitingForCommandInfoContact(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()

		go ServerMainThreadWatcher(t, srv, "Waiting for command info contact", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		cltxid = ClientTXID2
		contactInfo := epp.GetEPPContactInfo("1234", cltxid, "foo")
		simpleClient.SendChannel <- contactInfo

		cont, getErr := srv.ContactByID("1234")
		So(getErr, ShouldBeNil)

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseContactInfoType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.ContactInfDataResp, ShouldNotBeNil)

		contactInfoResp := msg.ResponseObject.ResultData.ContactInfDataResp
		So(contactInfoResp.ID, ShouldEqual, cont.ContactRegistryID)
		So(contactInfoResp.ROID, ShouldEqual, cont.ContactROID)
		So(contactInfoResp.Email, ShouldEqual, cont.CurrentRevision.EmailAddress)
		So(contactInfoResp.Fax.Extension, ShouldEqual, cont.CurrentRevision.FaxPhoneExtension)
		So(contactInfoResp.Fax.Number, ShouldEqual, cont.CurrentRevision.FaxPhoneNumber)
		So(contactInfoResp.Voice.Extension, ShouldEqual, cont.CurrentRevision.VoicePhoneExtension)
		So(contactInfoResp.Voice.Number, ShouldEqual, cont.CurrentRevision.VoicePhoneNumber)
		So(len(contactInfoResp.PostalInfos), ShouldEqual, 1)
		postalInfo := contactInfoResp.PostalInfos[0]
		streetAddrLen := 0
		if cont.CurrentRevision.AddressStreet1 != "" {
			streetAddrLen = streetAddrLen + 1
		}
		if cont.CurrentRevision.AddressStreet2 != "" {
			streetAddrLen = streetAddrLen + 1
		}
		if cont.CurrentRevision.AddressStreet3 != "" {
			streetAddrLen = streetAddrLen + 1
		}
		So(len(postalInfo.Address.Street), ShouldEqual, streetAddrLen)

		curStreetID := 0
		if cont.CurrentRevision.AddressStreet1 != "" {
			So(postalInfo.Address.Street[curStreetID], ShouldEqual, cont.CurrentRevision.AddressStreet1)
			curStreetID = curStreetID + 1
		}
		if cont.CurrentRevision.AddressStreet2 != "" {
			So(postalInfo.Address.Street[curStreetID], ShouldEqual, cont.CurrentRevision.AddressStreet2)
			curStreetID = curStreetID + 1
		}
		if cont.CurrentRevision.AddressStreet3 != "" {
			So(postalInfo.Address.Street[curStreetID], ShouldEqual, cont.CurrentRevision.AddressStreet3)
		}
		So(postalInfo.Name, ShouldEqual, cont.CurrentRevision.Name)
		So(postalInfo.Org, ShouldEqual, cont.CurrentRevision.Org)
		So(postalInfo.Address.City, ShouldEqual, cont.CurrentRevision.AddressCity)
		So(postalInfo.Address.Sp, ShouldEqual, cont.CurrentRevision.AddressState)
		So(postalInfo.Address.Cc, ShouldEqual, cont.CurrentRevision.AddressCountry)
		So(postalInfo.Address.Pc, ShouldEqual, cont.CurrentRevision.AddressPostalCode)
		So(postalInfo.PostalInfoType, ShouldEqual, "int")
		So(contactInfoResp.Email, ShouldEqual, cont.CurrentRevision.EmailAddress)
		So(len(contactInfoResp.Status), ShouldEqual, 8)
		So(contactInfoResp.AuthPW, ShouldNotBeNil)
		So(contactInfoResp.AuthPW.Password, ShouldEqual, "testpassword")
		So(contactInfoResp.CreateDate, ShouldEqual, cont.CreateDate.Format(epp.EPPTimeFormat))
		So(contactInfoResp.CreateID, ShouldEqual, cont.CreateClientID)
		if cont.UpdateDate.Unix() > 0 {
			So(contactInfoResp.UpdateDate, ShouldEqual, cont.UpdateDate.Format(epp.EPPTimeFormat))
			So(contactInfoResp.UpdateID, ShouldEqual, cont.UpdateClientID)
		} else {
			So(contactInfoResp.UpdateDate, ShouldBeEmpty)
			So(contactInfoResp.UpdateID, ShouldBeEmpty)
		}
		if cont.TransferDate.Unix() > 0 {
			So(contactInfoResp.TransferDate, ShouldEqual, cont.TransferDate.Format(epp.EPPTimeFormat))
		} else {
			So(contactInfoResp.TransferDate, ShouldBeEmpty)
		}
		So(contactInfoResp.AuthPW, ShouldNotBeNil)

		cltxid = ClientTXID3
		contactInfo2 := epp.GetEPPContactInfo("8013", cltxid, "foo")
		simpleClient.SendChannel <- contactInfo2
		contE, getErrE := srv.ContactByID("8013")
		So(getErrE, ShouldBeNil)

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseContactInfoType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.ContactInfDataResp, ShouldNotBeNil)
		contactInfoResp = msg.ResponseObject.ResultData.ContactInfDataResp
		So(contactInfoResp.ID, ShouldEqual, contE.ContactRegistryID)
		So(contactInfoResp.ROID, ShouldEqual, contE.ContactROID)
		So(contactInfoResp.Email, ShouldEqual, contE.CurrentRevision.EmailAddress)
		So(contactInfoResp.Fax.Extension, ShouldEqual, contE.CurrentRevision.FaxPhoneExtension)
		So(contactInfoResp.Fax.Number, ShouldEqual, contE.CurrentRevision.FaxPhoneNumber)
		So(contactInfoResp.Voice.Extension, ShouldEqual, contE.CurrentRevision.VoicePhoneExtension)
		So(contactInfoResp.Voice.Number, ShouldEqual, contE.CurrentRevision.VoicePhoneNumber)
		So(len(contactInfoResp.PostalInfos), ShouldEqual, 1)
		postalInfo = contactInfoResp.PostalInfos[0]
		streetAddrLen = 0
		if contE.CurrentRevision.AddressStreet1 != "" {
			streetAddrLen = streetAddrLen + 1
		}
		if contE.CurrentRevision.AddressStreet2 != "" {
			streetAddrLen = streetAddrLen + 1
		}
		if contE.CurrentRevision.AddressStreet3 != "" {
			streetAddrLen = streetAddrLen + 1
		}
		So(len(postalInfo.Address.Street), ShouldEqual, streetAddrLen)

		curStreetID = 0
		if contE.CurrentRevision.AddressStreet1 != "" {
			So(postalInfo.Address.Street[curStreetID], ShouldEqual, contE.CurrentRevision.AddressStreet1)
			curStreetID = curStreetID + 1
		}
		if contE.CurrentRevision.AddressStreet2 != "" {
			So(postalInfo.Address.Street[curStreetID], ShouldEqual, contE.CurrentRevision.AddressStreet2)
			curStreetID = curStreetID + 1
		}
		if contE.CurrentRevision.AddressStreet3 != "" {
			So(postalInfo.Address.Street[curStreetID], ShouldEqual, contE.CurrentRevision.AddressStreet3)
			curStreetID = curStreetID + 1
		}
		So(postalInfo.Name, ShouldEqual, contE.CurrentRevision.Name)
		So(postalInfo.Org, ShouldEqual, contE.CurrentRevision.Org)
		So(curStreetID, ShouldEqual, 2)
		So(postalInfo.Address.City, ShouldEqual, contE.CurrentRevision.AddressCity)
		So(postalInfo.Address.Sp, ShouldEqual, contE.CurrentRevision.AddressState)
		So(postalInfo.Address.Cc, ShouldEqual, contE.CurrentRevision.AddressCountry)
		So(postalInfo.Address.Pc, ShouldEqual, contE.CurrentRevision.AddressPostalCode)
		So(postalInfo.PostalInfoType, ShouldEqual, "int")
		So(contactInfoResp.Email, ShouldEqual, contE.CurrentRevision.EmailAddress)
		So(len(contactInfoResp.Status), ShouldEqual, 2)
		So(contactInfoResp.AuthPW, ShouldBeNil)
		So(contactInfoResp.CreateDate, ShouldEqual, contE.CreateDate.Format(epp.EPPTimeFormat))
		So(contactInfoResp.CreateID, ShouldEqual, contE.CreateClientID)
		if contE.UpdateDate.Unix() > 0 {
			So(contactInfoResp.UpdateDate, ShouldEqual, contE.UpdateDate.Format(epp.EPPTimeFormat))
			So(contactInfoResp.UpdateID, ShouldEqual, contE.UpdateClientID)
		} else {
			So(contactInfoResp.UpdateDate, ShouldBeEmpty)
			So(contactInfoResp.UpdateID, ShouldBeEmpty)
		}
		if contE.TransferDate.Unix() > 0 {
			So(contactInfoResp.TransferDate, ShouldEqual, contE.TransferDate.Format(epp.EPPTimeFormat))
		} else {
			So(contactInfoResp.TransferDate, ShouldBeEmpty)
		}
		So(contactInfoResp.AuthPW, ShouldBeNil)

		cltxid = ClientTXID4
		contactInfo3 := epp.GetEPPContactInfo("5678", cltxid, "")
		simpleClient.SendChannel <- contactInfo3

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2303)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2303)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldBeNil)

		srv.KillChan <- true
	})
}

// TestWaitingForCommandInfoHost will start a server and a client
// and login and then send requests for info about two hosts that
// exist in the database (one owned by the user and one that is not)
// and 1 that does not exist. The first two should return the expected
// data and the third should return an error.
func TestWaitingForCommandInfoHost(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()

		go ServerMainThreadWatcher(t, srv, "waiting for command info host", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		cltxid = ClientTXID2
		hostInfo := epp.GetEPPHostInfo("NS2.EXAMPLE.COM", cltxid)
		simpleClient.SendChannel <- hostInfo

		host, getErr := srv.HostByName("NS2.EXAMPLE.COM")
		So(getErr, ShouldBeNil)

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseHostInfoType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.HostInfDataResp, ShouldNotBeNil)
		hostInfoResp := msg.ResponseObject.ResultData.HostInfDataResp
		So(hostInfoResp.ROID, ShouldEqual, host.HostROID)
		So(hostInfoResp.Name, ShouldEqual, host.HostName)
		So(len(hostInfoResp.Status), ShouldEqual, 8)
		So(len(hostInfoResp.Addresses), ShouldEqual, len(host.CurrentRevision.HostAddresses))
		So(hostInfoResp.CreateDate, ShouldEqual, host.CreateDate.Format(epp.EPPTimeFormat))
		So(hostInfoResp.CreateID, ShouldEqual, host.CreateClientID)
		if host.UpdateDate.Unix() > 0 {
			So(hostInfoResp.UpdateDate, ShouldEqual, host.UpdateDate.Format(epp.EPPTimeFormat))
			So(hostInfoResp.UpdateID, ShouldEqual, host.UpdateClientID)
		} else {
			So(hostInfoResp.UpdateDate, ShouldBeEmpty)
			So(hostInfoResp.UpdateID, ShouldBeEmpty)
		}
		if host.TransferDate.Unix() > 0 {
			So(hostInfoResp.TransferDate, ShouldEqual, host.TransferDate.Format(epp.EPPTimeFormat))
		} else {
			So(hostInfoResp.TransferDate, ShouldBeEmpty)
		}

		cltxid = ClientTXID3
		hostInfo2 := epp.GetEPPHostInfo("NS1.EXAMPLE.COM", cltxid)
		simpleClient.SendChannel <- hostInfo2
		hostE, getErrE := srv.HostByName("NS1.EXAMPLE.COM")
		So(getErrE, ShouldBeNil)

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseHostInfoType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.HostInfDataResp, ShouldNotBeNil)
		hostInfoResp = msg.ResponseObject.ResultData.HostInfDataResp
		So(hostInfoResp.ROID, ShouldEqual, hostE.HostROID)
		So(hostInfoResp.Name, ShouldEqual, hostE.HostName)
		So(len(hostInfoResp.Status), ShouldEqual, 3)
		So(len(hostInfoResp.Addresses), ShouldEqual, len(hostE.CurrentRevision.HostAddresses))
		So(hostInfoResp.CreateDate, ShouldEqual, hostE.CreateDate.Format(epp.EPPTimeFormat))
		So(hostInfoResp.CreateID, ShouldEqual, hostE.CreateClientID)
		if hostE.UpdateDate.Unix() > 0 {
			So(hostInfoResp.UpdateDate, ShouldEqual, hostE.UpdateDate.Format(epp.EPPTimeFormat))
			So(hostInfoResp.UpdateID, ShouldEqual, hostE.UpdateClientID)
		} else {
			So(hostInfoResp.UpdateDate, ShouldBeEmpty)
			So(hostInfoResp.UpdateID, ShouldBeEmpty)
		}
		if hostE.TransferDate.Unix() > 0 {
			So(hostInfoResp.TransferDate, ShouldEqual, hostE.TransferDate.Format(epp.EPPTimeFormat))
		} else {
			So(hostInfoResp.TransferDate, ShouldBeEmpty)
		}

		cltxid = ClientTXID4
		contactInfo3 := epp.GetEPPHostInfo("NS1.FOO.COM", cltxid)
		simpleClient.SendChannel <- contactInfo3

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2303)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2303)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldBeNil)

		srv.KillChan <- true
	})
}

// TestWaitingForCommandDomainInfo will start a server and a client
// and login and then send requests for info about two domains that
// exist in the database (one owned by the user and one that is not)
// and 1 that does not exist. The first two should return the expected
// data and the third should return an error.
func TestWaitingForCommandDomainInfo(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()

		go ServerMainThreadWatcher(t, srv, "waiting for command domain info", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		cltxid = ClientTXID2
		domainInfo1 := epp.GetEPPDomainInfo("NONEXIST.COM", cltxid, "", epp.DomainInfoHostsAll)
		simpleClient.SendChannel <- domainInfo1

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 2303)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode2303)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldBeNil)

		cltxid = ClientTXID3
		domainInfo2 := epp.GetEPPDomainInfo("FOO.COM", cltxid, "", epp.DomainInfoHostsAll)
		simpleClient.SendChannel <- domainInfo2

		dom, getErr := srv.DomainByName("FOO.COM")
		So(getErr, ShouldBeNil)

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseDomainInfoType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.DomainInfDataResp, ShouldNotBeNil)
		domainInfoResp := msg.ResponseObject.ResultData.DomainInfDataResp
		So(domainInfoResp.ROID, ShouldEqual, dom.DomainROID)
		So(len(domainInfoResp.Status), ShouldEqual, 4)
		So(domainInfoResp.AuthPW, ShouldNotBeNil)
		So(domainInfoResp.RegistrantID, ShouldEqual, "1234")
		So(len(domainInfoResp.Contacts), ShouldEqual, 3)
		So(domainInfoResp.Name, ShouldEqual, dom.DomainName)
		So(domainInfoResp.CreateID, ShouldEqual, dom.CreateClientID)
		So(domainInfoResp.CreateDate, ShouldEqual, dom.CreateDate.Format(epp.EPPTimeFormat))
		So(domainInfoResp.UpdateID, ShouldEqual, dom.UpdateClientID)
		So(domainInfoResp.UpdateDate, ShouldEqual, dom.UpdateDate.Format(epp.EPPTimeFormat))
		So(domainInfoResp.TransferDate, ShouldEqual, dom.TransferDate.Format(epp.EPPTimeFormat))
		So(domainInfoResp.ExpireDate, ShouldEqual, dom.ExpireDate.Format(epp.EPPTimeFormat))
		So(len(domainInfoResp.Hosts), ShouldEqual, len(dom.CurrentRevision.Hostnames))

		cltxid = ClientTXID4
		domainInfo3 := epp.GetEPPDomainInfo("BAR.COM", cltxid, "", epp.DomainInfoHostsAll)
		simpleClient.SendChannel <- domainInfo3

		dom2, getErr := srv.DomainByName("BAR.COM")
		So(getErr, ShouldBeNil)

		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseDomainInfoType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)
		So(msg.ResponseObject.ResultData, ShouldNotBeNil)
		So(msg.ResponseObject.ResultData.DomainInfDataResp, ShouldNotBeNil)
		di2 := msg.ResponseObject.ResultData.DomainInfDataResp
		So(di2.ROID, ShouldEqual, dom2.DomainROID)
		So(len(di2.Status), ShouldEqual, 8)
		So(di2.AuthPW, ShouldBeNil)
		So(di2.RegistrantID, ShouldEqual, "8013")
		So(len(di2.Contacts), ShouldEqual, 3)
		So(di2.Name, ShouldEqual, dom2.DomainName)
		So(di2.CreateID, ShouldEqual, dom2.CreateClientID)
		So(di2.CreateDate, ShouldEqual, dom2.CreateDate.Format(epp.EPPTimeFormat))
		So(di2.UpdateID, ShouldEqual, dom2.UpdateClientID)
		So(di2.UpdateDate, ShouldEqual, dom2.UpdateDate.Format(epp.EPPTimeFormat))
		So(di2.TransferDate, ShouldEqual, dom2.TransferDate.Format(epp.EPPTimeFormat))
		So(di2.ExpireDate, ShouldEqual, dom2.ExpireDate.Format(epp.EPPTimeFormat))
		So(len(di2.Hosts), ShouldEqual, len(dom2.CurrentRevision.Hostnames))

		srv.KillChan <- true
	})
}

func TestWaitingForCommandDomainUpdate(t *testing.T) {
	t.Parallel()
	Convey("Given a default client and server and a valid login", t, func() {
		srv := GetDefaultServer()

		go ServerMainThreadWatcher(t, srv, "Waiting for command domain update", false)
		run := <-srv.Running
		if run != true {
			t.Error(ErrServerNotRunningWhenExpected)
		}
		simpleClient := GetDefaultSimpleClient()
		err := simpleClient.Start()
		So(err, ShouldBeNil)
		msg := <-simpleClient.RecvChannel
		stxid, err := msg.GetServerTransactionID()
		So(err, ShouldBeNil)
		So(stxid, ShouldNotBeEmpty)
		cltxid := ClientTXID1
		login := epp.GetEPPLogin("username", "password", cltxid, epp.GetDefaultServiceMenu())
		simpleClient.SendChannel <- login
		msg = <-simpleClient.RecvChannel
		So(msg.MessageType(), ShouldEqual, epp.ResponseType)
		So(msg.ResponseObject.Result, ShouldNotBeNil)
		So(msg.ResponseObject.Result.Code, ShouldEqual, 1000)
		So(msg.ResponseObject.Result.Msg, ShouldEqual, epp.ResponseCode1000)
		So(msg.ResponseObject.TransactionID.ClientTransactionID, ShouldEqual, cltxid)

		cltxid = ClientTXID1
		domainAdd := epp.DomainUpdateAddRemove{}
		addCont := epp.DomainContact{}
		addCont.Type = "admin"
		addCont.Value = "4321"
		domainAdd.Contacts = append(domainAdd.Contacts, addCont)
		addStatus := epp.DomainStatus{}
		addStatus.StatusFlag = "clientHold"
		domainAdd.Statuses = append(domainAdd.Statuses, addStatus)
		domainRem := epp.DomainUpdateAddRemove{}
		delCont := epp.DomainContact{}
		delCont.Type = "admin"
		delCont.Value = "1234"
		domainRem.Contacts = append(domainRem.Contacts, delCont)
		delStatus := epp.DomainStatus{}
		delStatus.StatusFlag = "clientDeleteProhibited"
		domainRem.Statuses = append(domainRem.Statuses, delStatus)

		domainChg := epp.DomainUpdateChange{}

		domainUpdate1 := epp.GetEPPDomainUpdate("FOO.COM", &domainAdd, &domainRem, &domainChg, cltxid)
		simpleClient.SendChannel <- domainUpdate1

		time.Sleep(time.Second * 2)
		srv.KillChan <- true
	})
}

// GetTestingDomainOwned creates a testing domain that is owned by the
// testing client. The Domain will have a current revision and be valid
// for the next 5 years.
func GetTestingDomainOwned() lib.Domain {
	dom := lib.Domain{}
	dom.ID = 1
	dom.RegistryDomainID = TestingDomainName1
	dom.DomainROID = TestingDomainName1
	dom.DomainName = TestingDomainName1

	dom.CurrentRevision = GetTestingDomainRevisionOwned()

	dom.SponsoringClientID = "1"

	dom.CreateClientID = "5"
	dom.CreateDate = time.Now().Add(-1 * 24 * 365 * time.Hour)
	dom.UpdateClientID = "6"
	dom.UpdateDate = time.Now().Add(-1 * 24 * 180 * time.Hour)
	dom.TransferDate = time.Now().Add(-1 * 24 * 90 * time.Hour)
	dom.ExpireDate = time.Now().Add(5 * 24 * 365 * time.Hour)

	return dom
}

// GetTestingDomainRevisionOwned generates a domain revision for a
// domain that is owned by the testing client with the contacts and
// hosts associated also being owned by the testing client.
func GetTestingDomainRevisionOwned() lib.DomainRevision {
	domainRevision := lib.DomainRevision{}
	domainRevision.ID = 1
	domainRevision.DomainID = 1

	domainRevision.ClientDeleteProhibitedStatus = true
	domainRevision.ServerDeleteProhibitedStatus = false
	domainRevision.ClientHoldStatus = false
	domainRevision.ServerHoldStatus = false
	domainRevision.ClientRenewProhibitedStatus = false
	domainRevision.ServerRenewProhibitedStatus = false
	domainRevision.ClientTransferProhibitedStatus = true
	domainRevision.ServerTransferProhibitedStatus = false
	domainRevision.ClientUpdateProhibitedStatus = true
	domainRevision.ServerUpdateProhibitedStatus = false

	domainRevision.DomainRegistrant = GetTestingContactOwned()
	domainRevision.DomainAdminContact = GetTestingContactOwned()
	domainRevision.DomainTechContact = GetTestingContactOwned()
	domainRevision.DomainBillingContact = GetTestingContactOwned()

	dsData := lib.DSDataEntry{}
	dsData.DomainRevisionID = 1
	dsData.KeyTag = 1234
	dsData.Algorithm = 1
	dsData.DigestType = 1
	dsData.Digest = "0123456789ABCDEF"

	domainRevision.DSDataEntries = append(domainRevision.DSDataEntries, dsData)

	domainRevision.Hostnames = append(domainRevision.Hostnames, GetTestingHostOwned())

	return domainRevision
}

// GetTestingDomainExternal creates a testing domain that is not owned
// by the testing client. The Domain will have a current revision and be
// valid for the next 5 years.
func GetTestingDomainExternal() lib.Domain {
	dom := lib.Domain{}
	dom.ID = 2
	dom.RegistryDomainID = TestingDomainName2
	dom.DomainROID = TestingDomainName2
	dom.DomainName = TestingDomainName2

	dom.CurrentRevision = GetTestingDomainRevisionExternal()

	dom.SponsoringClientID = "2"

	dom.CreateClientID = "7"
	dom.CreateDate = time.Now().Add(-1 * 24 * 365 * time.Hour)
	dom.UpdateClientID = "8"
	dom.UpdateDate = time.Now().Add(-1 * 24 * 180 * time.Hour)
	dom.TransferDate = time.Now().Add(-1 * 24 * 90 * time.Hour)
	dom.ExpireDate = time.Now().Add(5 * 24 * 365 * time.Hour)

	return dom
}

// GetTestingDomainRevisionExternal generates a domain revision for a
// domain that is not owned by the testing client with the contacts and
// hosts associated also not being owned by the testing client.
func GetTestingDomainRevisionExternal() lib.DomainRevision {
	domainRevision := lib.DomainRevision{}
	domainRevision.ID = 2
	domainRevision.DomainID = 2

	domainRevision.ClientDeleteProhibitedStatus = false
	domainRevision.ServerDeleteProhibitedStatus = true
	domainRevision.ClientHoldStatus = true
	domainRevision.ServerHoldStatus = true
	domainRevision.ClientRenewProhibitedStatus = true
	domainRevision.ServerRenewProhibitedStatus = true
	domainRevision.ClientTransferProhibitedStatus = false
	domainRevision.ServerTransferProhibitedStatus = true
	domainRevision.ClientUpdateProhibitedStatus = false
	domainRevision.ServerUpdateProhibitedStatus = true

	domainRevision.DomainRegistrant = GetTestingContactExternal()
	domainRevision.DomainAdminContact = GetTestingContactExternal()
	domainRevision.DomainTechContact = GetTestingContactExternal()
	domainRevision.DomainBillingContact = GetTestingContactExternal()

	dsData := lib.DSDataEntry{}
	dsData.DomainRevisionID = 2
	dsData.KeyTag = 1234
	dsData.Algorithm = 1
	dsData.DigestType = 1
	dsData.Digest = "0123456789ABCDEF"

	domainRevision.DSDataEntries = append(domainRevision.DSDataEntries, dsData)

	domainRevision.Hostnames = append(domainRevision.Hostnames, GetTestingHostExternal())

	return domainRevision
}

// GetTestingHostOwned creates a testing host that is owned by the
// testing client.
func GetTestingHostOwned() lib.Host {
	host := lib.Host{}
	host.ID = 1
	host.HostName = "NS1.EXAMPLE.NET"
	host.HostROID = "1234-H-REGISTRY"
	host.LinkedStatus = true
	host.SponsoringClientID = "1"
	host.CreateDate = time.Now().Add(-1 * 24 * 365 * time.Hour)
	host.UpdateClientID = "10"
	host.UpdateDate = time.Now().Add(-1 * 24 * 180 * time.Hour)
	host.TransferDate = time.Now().Add(-1 * 24 * 90 * time.Hour)
	host.CurrentRevision = GetTestingHostRevisionOwned()

	return host
}

// GetTestingHostRevisionOwned generates a host revision for a
// host that is owned by the testing client.
func GetTestingHostRevisionOwned() lib.HostRevision {
	hostRevision := lib.HostRevision{}
	hostRevision.ID = 1
	hostRevision.HostID = 1
	hostRevision.RevisionState = lib.StateActive
	hostRevision.DesiredState = lib.StateActive
	hostRevision.HostStatus = "OK"
	hostRevision.ClientDeleteProhibitedStatus = true
	hostRevision.ServerDeleteProhibitedStatus = true
	hostRevision.ClientTransferProhibitedStatus = true
	hostRevision.ServerTransferProhibitedStatus = true
	hostRevision.ClientUpdateProhibitedStatus = true
	hostRevision.ServerUpdateProhibitedStatus = true

	ha4 := lib.HostAddress{}
	ha4.Protocol = 4
	ha4.IPAddress = testingHost

	ha6 := lib.HostAddress{}
	ha6.Protocol = 6
	ha6.IPAddress = "::1"

	hostRevision.HostAddresses = append(hostRevision.HostAddresses, ha4)
	hostRevision.HostAddresses = append(hostRevision.HostAddresses, ha6)

	return hostRevision
}

// GetTestingHostExternal creates a testing host that is not owned by
// the testing client.
func GetTestingHostExternal() lib.Host {
	host := lib.Host{}
	host.ID = 2
	host.HostName = "NS1.EXAMPLE1.COM"
	host.HostROID = "NS1EXAMPLE1-REGISTRY"
	host.LinkedStatus = true
	host.SponsoringClientID = "clientidX"
	host.CreateClientID = "createdByLloyd"
	loc, _ := time.LoadLocation("UTC")
	host.CreateDate = time.Date(2015, 6, 30, 17, 27, 8, 958, loc)
	host.UpdateClientID = "updatedX"
	host.UpdateDate = time.Date(2015, 6, 30, 17, 27, 8, 958, loc)
	host.CurrentRevision = GetTestingHostRevisionExternal()

	return host
}

// GetTestingHostRevisionExternal generates a host revision for a
// host that is not owned by the testing client.
func GetTestingHostRevisionExternal() lib.HostRevision {
	hostRevision := lib.HostRevision{}
	hostRevision.ID = 2
	hostRevision.HostID = 2
	hostRevision.RevisionState = lib.StateActive
	hostRevision.DesiredState = lib.StateActive
	hostRevision.HostStatus = "OK"
	hostRevision.ClientDeleteProhibitedStatus = false
	hostRevision.ServerDeleteProhibitedStatus = false
	hostRevision.ClientTransferProhibitedStatus = false
	hostRevision.ServerTransferProhibitedStatus = false
	hostRevision.ClientUpdateProhibitedStatus = true
	hostRevision.ServerUpdateProhibitedStatus = false

	ha4 := lib.HostAddress{}
	ha4.Protocol = 4
	ha4.IPAddress = "192.1.2.3"

	ha6 := lib.HostAddress{}
	ha6.Protocol = 6
	ha6.IPAddress = "::2"

	hostRevision.HostAddresses = append(hostRevision.HostAddresses, ha4)
	hostRevision.HostAddresses = append(hostRevision.HostAddresses, ha6)

	return hostRevision
}

// GetTestingContactOwned creates a testing contact that is owned by the
// testing client.
func GetTestingContactOwned() lib.Contact {
	cont := lib.Contact{}
	cont.ID = 1
	cont.ContactRegistryID = "1234"
	cont.ContactROID = "1234-VRSN"
	cont.LinkedStatus = true
	cont.SponsoringClientID = "1"
	cont.CreateClientID = "10"
	cont.CreateDate = time.Now().Add(-1 * 24 * 365 * time.Hour)
	cont.UpdateClientID = "10"
	cont.UpdateDate = time.Now().Add(-1 * 24 * 180 * time.Hour)
	cont.TransferDate = time.Now().Add(-1 * 24 * 90 * time.Hour)
	cont.CurrentRevision = GetTestingContactRevisionOwned()

	return cont
}

// GetTestingContactRevisionOwned generates a contact revision for a
// contact that is owned by the testing client.
func GetTestingContactRevisionOwned() lib.ContactRevision {
	contactRevision := lib.ContactRevision{}
	contactRevision.ID = 1
	contactRevision.ContactID = 1
	contactRevision.RevisionState = lib.StateActive
	contactRevision.DesiredState = lib.StateActive
	contactRevision.ContactStatus = "OK"
	contactRevision.ClientDeleteProhibitedStatus = true
	contactRevision.ServerDeleteProhibitedStatus = true
	contactRevision.ClientTransferProhibitedStatus = true
	contactRevision.ServerTransferProhibitedStatus = true
	contactRevision.ClientUpdateProhibitedStatus = true
	contactRevision.ServerUpdateProhibitedStatus = true
	contactRevision.Name = "Domain Admin"
	contactRevision.Org = "Example"
	contactRevision.AddressStreet1 = "123 Main St"
	contactRevision.AddressStreet2 = "line2"
	contactRevision.AddressStreet3 = "line3"
	contactRevision.AddressCity = "Somewhere"
	contactRevision.AddressState = "SomeState"
	contactRevision.AddressPostalCode = "00001"
	contactRevision.AddressCountry = "US"

	contactRevision.VoicePhoneNumber = "+1.1235551212"
	contactRevision.VoicePhoneExtension = ""
	contactRevision.FaxPhoneNumber = "+1.1235551213"
	contactRevision.FaxPhoneExtension = ""

	contactRevision.EmailAddress = "domain-admin@example.com"

	return contactRevision
}

// GetTestingContactExternal creates a testing contact that is not owned
// by the testing client.
func GetTestingContactExternal() lib.Contact {
	cont := lib.Contact{}
	cont.ID = 2
	cont.ContactRegistryID = "8013"
	cont.ContactROID = "SH8013-VRSN"
	cont.LinkedStatus = false
	cont.SponsoringClientID = "ClientY"
	cont.CreateClientID = "ClientX"
	loc, _ := time.LoadLocation("UTC")
	cont.CreateDate = time.Date(2015, 6, 30, 18, 48, 34, 837, loc)
	cont.CurrentRevision = GetTestingContactRevisionExternal()

	return cont
}

// GetTestingContactRevisionExternal generates a contact revision for a
// contact that is not owned by the testing client.
func GetTestingContactRevisionExternal() lib.ContactRevision {
	contactRevision := lib.ContactRevision{}
	contactRevision.ID = 2
	contactRevision.ContactID = 2
	contactRevision.RevisionState = lib.StateActive
	contactRevision.DesiredState = lib.StateActive
	contactRevision.ContactStatus = "OK"
	contactRevision.ClientDeleteProhibitedStatus = false
	contactRevision.ServerDeleteProhibitedStatus = false
	contactRevision.ClientTransferProhibitedStatus = false
	contactRevision.ServerTransferProhibitedStatus = false
	contactRevision.ClientUpdateProhibitedStatus = true
	contactRevision.ServerUpdateProhibitedStatus = false
	contactRevision.Name = "John Doe"
	contactRevision.Org = "Example Inc."
	contactRevision.AddressStreet1 = "123 Example Dr."
	contactRevision.AddressStreet2 = "Suite 100"
	contactRevision.AddressStreet3 = ""
	contactRevision.AddressCity = "Dulles"
	contactRevision.AddressState = "VA"
	contactRevision.AddressPostalCode = "20166-6503"
	contactRevision.AddressCountry = "US"

	contactRevision.VoicePhoneNumber = "+1.7035555555"
	contactRevision.VoicePhoneExtension = "123"
	contactRevision.FaxPhoneNumber = "+1.7035555556"
	contactRevision.FaxPhoneExtension = "456"

	contactRevision.EmailAddress = "jdoe@example.com"

	return contactRevision
}

// GetDefaultServer will generate a default testing server with basic
// data and set to listen on localhost only.
func GetDefaultServer() EPPServer {
	host := testingHost
	port := testingPort
	duration := 2 * time.Second

	srv := NewEPPServer(host, port, duration)

	li := LoginObject{}
	li.Password = "password"
	li.LoginID = "username"
	li.RegistrarID = "1"

	srv.Logins["username"] = li

	con := GetTestingContactOwned()
	srv.Contacts[con.ContactRegistryID] = con
	con = GetTestingContactExternal()
	srv.Contacts[con.ContactRegistryID] = con

	hos := GetTestingHostOwned()
	srv.Hosts[hos.HostName] = hos
	hos = GetTestingHostExternal()
	srv.Hosts[hos.HostName] = hos

	dom := GetTestingDomainOwned()
	srv.Domains[dom.DomainName] = dom
	dom = GetTestingDomainExternal()
	srv.Domains[dom.DomainName] = dom

	return srv
}

// ServerMainThreadWatcher, when run as a go routine will start the given
// server and wait for it to stop running and print an error depending
// on if the server was expected to exit with an error or not.
func ServerMainThreadWatcher(t *testing.T, srv EPPServer, testName string, errorExpected bool) {
	t.Helper()

	err := srv.Start()

	if errorExpected {
		if err == nil {
			t.Errorf("Expected to get an error from the server main loop: %s", testName)
		}
	} else {
		if err != nil {
			t.Errorf("Expected not to get an error from the server main loop: %s - %s", testName, err)
		}
	}
}

// SimpleClient is used for testing the server application and exposes
// the required channels to communicate with the server.
type SimpleClient struct {
	EPPConn *net.Conn

	Log *logging.Logger

	Writer *bufio.Writer

	StopChan chan bool

	SendErrorChannel chan error
	ErrorChannel     chan error

	RecvChannel    chan epp.Epp
	SendChannel    chan epp.Epp
	RawSendChannel chan string

	Host string
	Port int

	Timeout time.Duration
}

// GetDefaultSimpleClient will generate a simple client with default
// settings that can be used to communicate with the server that is
// generated for testing.
func GetDefaultSimpleClient() SimpleClient {
	return NewSimpleClient(testingHost, testingPort, log, 2*time.Second)
}

// NewSimpleClient will create a SimpleClient and initialize all of the
// required fields.
func NewSimpleClient(host string, port int, log *logging.Logger, timeout time.Duration) SimpleClient {
	return SimpleClient{
		Log:              log,
		Host:             host,
		Port:             port,
		RecvChannel:      make(chan epp.Epp, 100),
		SendChannel:      make(chan epp.Epp, 100),
		RawSendChannel:   make(chan string, 100),
		SendErrorChannel: make(chan error, 100),
		ErrorChannel:     make(chan error, 100),
		StopChan:         make(chan bool),
		Timeout:          timeout,
	}
}

// Stop will suspend the actions of the DummyClient which should stop
// all of its lisenting threads.
func (c *SimpleClient) Stop() error {
	c.StopChan <- true

	return nil
}

// Start will open a connection with the server as defined by the
// DummyClient and start the listening and sending threads.
func (c *SimpleClient) Start() error {
	conn, connectionErr := net.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port))

	if connectionErr != nil {
		return fmt.Errorf("error starting server: %w", connectionErr)
	}

	c.EPPConn = &conn
	c.Writer = bufio.NewWriter(conn)

	go c.Listener()
	go c.EPPSender(c.StopChan, c.Timeout)

	return nil
}

// Close will stop the DummyClient and close the connections that the
// DummyClient has to the server.
func (c *SimpleClient) Close() {
	c.StopChan <- true
	(*c.EPPConn).Close()
}

// EPPListener starts to listen on the connection from the server and
// processes the messages that it gets back.
func (c *SimpleClient) Listener() {
	scanner := bufio.NewScanner(*c.EPPConn)
	scanner.Split(epp.WireSplit)

	for scanner.Scan() {
		text := scanner.Text()

		if DEBUG {
			c.Log.Debug(text)
		}

		outObj, unmarshallErr := epp.UnmarshalMessage([]byte(text))

		if unmarshallErr != nil {
			c.Log.Error(unmarshallErr.Error())
			c.ErrorChannel <- unmarshallErr

			return
		}

		c.RecvChannel <- outObj.TypedMessage()
	}

	err := scanner.Err()
	if err != nil {
		c.ErrorChannel <- err
	}
}

// EPPSender starts and listens to a channel for messages to send to the
// EPP server.
func (c *SimpleClient) EPPSender(stopChan chan bool, to time.Duration) {
	for {
		timeout := GetTimeout(to)

		select {
		case <-stopChan:
			return
		case msg := <-c.RawSendChannel:
			var buffer bytes.Buffer

			err := binary.Write(&buffer, binary.BigEndian, int32(len(msg)))
			if err != nil {
				c.Log.Error(err)
			}

			buffer.WriteString(msg)

			_, err = c.Writer.Write(buffer.Bytes())
			So(err, ShouldBeNil)

			c.Writer.Flush()
		case msg := <-c.SendChannel:
			outputBytes, encodeErr := msg.EncodeEPP()
			if encodeErr != nil {
				c.SendErrorChannel <- encodeErr

				return
			}

			if DEBUG {
				c.Log.Debug(string(outputBytes))
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
		case <-timeout:
			c.Log.Debug("Sending thread reached timeout")
		}
	}
}

// GetTimeout will create and return a channel for a bool that will be
// written to when the timeout passes.
func GetTimeout(sleep time.Duration) chan bool {
	ret := make(chan bool, 1)
	go timeoutProc(sleep, ret)

	return ret
}

// timeoutProc waits the provided amount of time and writes true to the
// channel provided to indicate to another process that a timeout has
// occurred.
func timeoutProc(sleep time.Duration, ch chan bool) {
	time.Sleep(sleep)
	ch <- true
}
