package client

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetConnectionString(t *testing.T) {
	t.Parallel()
	Convey("Given no config, GetConnectionString should return 127.0.0.1:1700", t, func() {
		conf := Config{}
		So(conf.GetConnectionString(), ShouldEqual, "127.0.0.1:1700")
	})

	Convey("Given a config with only the host set, GetConnectionString should return host:1700", t, func() {
		conf := Config{Host: "example.com"}
		So(conf.GetConnectionString(), ShouldEqual, "example.com:1700")
	})

	Convey("Given a config with only the port set, GetConnectionString should return 127.0.0.1:port", t, func() {
		conf := Config{Port: 1800}
		So(conf.GetConnectionString(), ShouldEqual, "127.0.0.1:1800")
	})

	Convey("Given a config with both host and port set, GetConnectionString should return host:port", t, func() {
		conf := Config{Host: "example.com", Port: 1800}
		So(conf.GetConnectionString(), ShouldEqual, "example.com:1800")
	})
}
