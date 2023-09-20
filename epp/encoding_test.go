package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEncodeEPP(t *testing.T) {
	t.Parallel()
	Convey("Given a EPP Hello Message", t, func() {
		epp := GetEPPHello()
		Convey("The encoded message should be the XML version of the object with a 4 byte length header", func() {
			data, err := epp.EncodeEPP()
			So(err, ShouldBeNil)
			str, strErr := epp.ToString()
			So(strErr, ShouldBeNil)
			So(len(data), ShouldEqual, len(str)+4+len(EPPHeader))
		})
	})
}

func TestWireSplit(t *testing.T) {
	t.Parallel()
	Convey("Given an encoded EPP Hello message", t, func() {
		epp := GetEPPHello()
		data, err := epp.EncodeEPP()
		So(err, ShouldBeNil)
		Convey("WireSplit should be able to split the message", func() {
			adv, token, splitErr := WireSplit(data, true)
			So(splitErr, ShouldBeNil)
			So(string(token), ShouldEqual, string(data[4:]))
			So(adv, ShouldEqual, len(data))
		})
	})

	Convey("Given an encoded EPP Hello message that has been cut short", t, func() {
		epp := GetEPPHello()
		data, err := epp.EncodeEPP()
		So(err, ShouldBeNil)
		dataCut := data[:len(data)-10]
		Convey("WireSplit should not be able to split the message", func() {
			adv, token, splitErr := WireSplit(dataCut, false)
			So(splitErr, ShouldBeNil)
			So(string(token), ShouldEqual, "")
			So(adv, ShouldEqual, 0)
		})
	})

	Convey("Given a list of 3 bytes (less than the length header)", t, func() {
		data := []byte{0, 0, 0}
		Convey("WireSplit should not be able to split the message", func() {
			adv, token, splitErr := WireSplit(data, false)
			So(splitErr, ShouldBeNil)
			So(string(token), ShouldEqual, "")
			So(adv, ShouldEqual, 0)
		})
	})
}

func TestUnmarshalMessage(t *testing.T) {
	t.Parallel()
	Convey("Given an encoded EPP Hello message", t, func() {
		epp := GetEPPHello()
		data, err := epp.EncodeEPP()
		So(err, ShouldBeNil)
		Convey("UnmarshalMessage should return an EPP Hello Object", func() {
			So(len(data), ShouldBeGreaterThan, 4)
			obj, unmarshalErr := UnmarshalMessage(data[4:])
			So(unmarshalErr, ShouldBeNil)
			So(obj.MessageType(), ShouldEqual, HelloType)
		})
	})
}
