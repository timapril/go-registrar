package main

import (
	"flag"
	"log"
	"time"

	"github.com/timapril/go-registrar/epp"
	"github.com/timapril/go-registrar/epp/server"
)

var (
	eppPort = flag.Int("port", epp.DefaultEPPPort, "The port that the server should listen on")
	eppHost = flag.String("host", "", "The host that the server should listen on")
)

const (
	// eppTimeout sets the timeout for EPP clients to 5 seconds.
	eppTimeout time.Duration = time.Second * 5
)

func main() {
	log.Default().Print("Starting EPP Testing Server")

	srv := server.NewEPPServer(*eppHost, *eppPort, eppTimeout)

	err := srv.Start()
	if err != nil {
		log.Default().Fatal(err.Error())
	}
}
