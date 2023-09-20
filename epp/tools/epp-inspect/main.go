package main

import (
	"flag"
	"log"
	"os"

	"github.com/timapril/go-registrar/epp"
)

var eppfile = flag.String("msg", "", "the message to process")

func main() {
	flag.Parse()

	if len(*eppfile) == 0 {
		log.Fatal("No file name given")

		return
	}

	data, err := os.ReadFile(*eppfile)
	if err != nil {
		log.Fatalf("An error occurred trying to open the provided file: %s", err)

		return
	}

	msg, eppUnmarshallErr := epp.UnmarshalMessage(data)
	if eppUnmarshallErr != nil {
		log.Fatal(eppUnmarshallErr)

		return
	}

	tmsg := msg.TypedMessage()

	message, ErrToStr := tmsg.ToString()
	if ErrToStr != nil {
		log.Fatal(ErrToStr)

		return
	}

	log.Default().Printf("Message Type: %s", tmsg.MessageType())

	log.Default().Printf("Message Body:\n%s", message)
}
