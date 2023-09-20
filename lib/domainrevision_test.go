package lib

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDomainRevisionExportToJSON(t *testing.T) {
	t.Parallel()
	Convey("Given an Domain Revision Export object with an ID of 0", t, func() {
		a := DomainRevisionExport{ID: 0}
		_, err := a.ToJSON()
		Convey("Calling ToJSON should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
	Convey("Given an Domain Revision Export object with an ID of -1", t, func() {
		a := DomainRevisionExport{ID: -1}
		_, err := a.ToJSON()
		Convey("Calling ToJSON should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
}
