package epp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetCOMNamestoreExtension(t *testing.T) {
	t.Parallel()
	Convey("Creating a .COM Namestore Extension", t, func() {
		ext := GetCOMNamestoreExtension()
		Convey("The output should match the Namestore Extension format for .COM", func() {
			str, err := interfaceToXMLString(ext)
			So(str, ShouldEqual, namestoreCOMExtension)
			So(err, ShouldBeNil)
		})
	})
}

func TestGetNETNamestoreExtension(t *testing.T) {
	t.Parallel()
	Convey("Creating a .NET Namestore Extension", t, func() {
		ext := GetNETNamestoreExtension()
		Convey("The output should match the Namestore Extension format for .NET", func() {
			str, err := interfaceToXMLString(ext)
			So(str, ShouldEqual, namestoreNETExtension)
			So(err, ShouldBeNil)
		})
	})
}

var namestoreCOMExtension = `<extension>
  <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
    <namestoreExt:subProduct>dotCOM</namestoreExt:subProduct>
  </namestoreExt:namestoreExt>
</extension>`

var namestoreNETExtension = `<extension>
  <namestoreExt:namestoreExt xmlns:namestoreExt="http://www.verisign-grs.com/epp/namestoreExt-1.1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.verisign-grs.com/epp/namestoreExt-1.1 namestoreExt-1.1.xsd">
    <namestoreExt:subProduct>dotNET</namestoreExt:subProduct>
  </namestoreExt:namestoreExt>
</extension>`
