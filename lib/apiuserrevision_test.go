package lib

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAPIUserRevisionExportToJSON(t *testing.T) {
	t.Parallel()
	Convey("Given an APIUser Revision Export object with an ID of 0", t, func() {
		a := APIUserRevisionExport{ID: 0}
		_, err := a.ToJSON()
		Convey("Calling ToJSON should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
	Convey("Given an APIUser Revision Export object with an ID of -1", t, func() {
		a := APIUserRevisionExport{ID: -1}
		_, err := a.ToJSON()
		Convey("Calling ToJSON should return an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
}
