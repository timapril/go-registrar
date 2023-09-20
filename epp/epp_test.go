package epp

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestToStringCS(t *testing.T) {
	t.Parallel()
	Convey("Creating a hello object with the same parameters as the default hello in the Verisign testing tool", t, func() {
		verisignEPP := GetEPPHello()
		Convey("The output should match the output from the testing tool prefixed with a 'C:'", func() {
			eppStr, eppErr := verisignEPP.ToStringCS("C:")
			So(eppStr, ShouldEqual, verisignHelloClient)
			So(eppErr, ShouldBeNil)
		})
	})
}

func TestGetTransactionID(t *testing.T) {
	t.Parallel()
	Convey("Given a hello object", t, func() {
		verisignEPP := GetEPPHello()
		Convey("Calling GetTransactionID should return an empty string and an error", func() {
			txid, txidErr := verisignEPP.GetTransactionID()
			So(txid, ShouldBeEmpty)
			So(txidErr, ShouldNotBeNil)
		})
	})

	Convey("Given a login object", t, func() {
		verisignEPP := GetEPPLogin("username", "password", "ABC-12345-XYZ", GetDefaultServiceMenu())
		Convey("Calling GetTransactionID should return transaction if and no error", func() {
			txid, txidErr := verisignEPP.GetTransactionID()
			So(txid, ShouldEqual, "ABC-12345-XYZ")
			So(txidErr, ShouldBeNil)
		})
	})
}

func TestMessageType(t *testing.T) {
	t.Parallel()
	Convey("Given a Greeting Object", t, func() {
		verisignEPP := GetEPPGreeting(GetDefaultServiceMenu())
		Convey("Calling MessageType on the object should return GreetingType", func() {
			So(verisignEPP.MessageType(), ShouldEqual, GreetingType)
		})
	})

	Convey("Given a Command Object", t, func() {
		verisignEPP := GetEPPCommand("ABC-12345-XYZ")
		Convey("Calling MessageType on the object should return CommandType", func() {
			So(verisignEPP.MessageType(), ShouldEqual, CommandType)
		})
	})

	Convey("Given a Hello Object", t, func() {
		verisignEPP := GetEPPHello()
		Convey("Calling MessageType on the object should return HelloType", func() {
			So(verisignEPP.MessageType(), ShouldEqual, HelloType)
		})
	})

	Convey("Given a Response Object", t, func() {
		verisignEPP := GetEPP()
		verisignEPP.ResponseObject = &Response{}
		Convey("Calling MessageType on the object should return ResponseType", func() {
			So(verisignEPP.MessageType(), ShouldEqual, ResponseType)
		})
	})

	Convey("Given a EPP Object", t, func() {
		verisignEPP := GetEPP()
		Convey("Calling MessageType on the object should return GenericEPPType", func() {
			So(verisignEPP.MessageType(), ShouldEqual, GenericEPPType)
		})
	})
}

func interfaceToXMLString(obj interface{}) (string, error) {
	message, err := xml.MarshalIndent(obj, "", "  ")

	return string(message), err
}

// Removed: xmlns="urn:ietf:params:xml:ns:epp-1.0"
var verisignHelloClient = `C:<epp xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:ietf:params:xml:ns:epp-1.0 epp-1.0.xsd">
C:  <hello xmlns="urn:ietf:params:xml:ns:epp-1.0"></hello>
C:</epp>
`

func UnMashalMarshalTest(t *testing.T, command string, objectType string, inoutxml string, commandType string) {
	t.Helper()

	re := regexp.MustCompile(`\s+`)

	Convey(fmt.Sprintf("Given a Verisign %s %s object", command, objectType), t, func() {
		raw, err := UnmarshalMessage([]byte(inoutxml))
		So(err, ShouldBeNil)
		typed := raw.TypedMessage()
		So(typed.MessageType(), ShouldEqual, commandType)

		textVersion, err := typed.ToString()
		So(err, ShouldBeNil)

		strippedmarshall := re.ReplaceAllString(textVersion, " ")
		strippedIn := re.ReplaceAllString(inoutxml, " ")

		So(strippedmarshall, ShouldEqual, strippedIn)
	})
}

func TestDateTimeToDate(t *testing.T) {
	t.Parallel()
	Convey("Given a valid date time, DateTimeToDate should not return and error and should return a valid date", t, func() {
		dateTime := "2019-04-06T19:54:31Z"
		date, err := DateTimeToDate(dateTime)
		So(err, ShouldBeNil)
		So(date, ShouldEqual, "2019-04-06")
	})

	Convey("Given a poorly formated date time, DateTimeToDate should return an error", t, func() {
		dateTime := "sdf2f2wc43"
		_, err := DateTimeToDate(dateTime)
		So(err, ShouldNotBeNil)
	})
}
