package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"

	"github.com/op/go-logging"
)

var verbose = flag.Bool("v", false, "Verbose logging")

var log = logging.MustGetLogger("registrar")
var format = logging.MustStringFormatter(
	"%{color}%{level:.4s} %{time:15:04:05.000000} %{shortfile} %{callpath} %{longfunc} %{id:03x}%{color:reset} %{message}",
)

// prepareLogging sets up logging so the application can have
// configurable logging
func prepareLogging(level logging.Level) {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLevel := logging.AddModuleLevel(backendFormatter)
	backendLevel.SetLevel(level, "")
	logging.SetBackend(backendLevel)
}

func main() {
	flag.Parse()

	ll := logging.ERROR
	if *verbose {
		ll = logging.DEBUG
	}

	prepareLogging(ll)

	data, err := os.ReadFile("domains.txt")
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, row := range strings.Split(string(data), "\n") {
		if len(row) == 0 {
			continue
		}
		rawNSes, err := net.LookupNS(row)
		if err != nil {
			log.Error(err)
			return
		}
		nses := []string{}
		for _, ns := range rawNSes {
			nses = append(nses, strings.ToUpper(ns.Host))
		}

		sort.Strings(nses)

		fmt.Printf("%s;%s\n", strings.ToUpper(row), strings.Join(nses, ","))

	}
}
