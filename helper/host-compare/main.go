package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"

	"github.com/op/go-logging"
	"github.com/timapril/go-registrar/lib"
)

var inFile = flag.String("in", "", "the file to parse")
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

	if *inFile == "" {
		fmt.Println("missing input file")
		return
	}

	data, err := os.ReadFile(*inFile)
	if err != nil {
		log.Error(err)
		return
	}

	app := lib.ApprovalAttestationUnmarshal{}

	err = json.Unmarshal(data, &app)
	if err != nil {
		log.Error(err)
		return
	}
	if app.ObjectType == lib.HostType {
		hos := lib.HostExport{}
		err = json.Unmarshal(app.ExportRev, &hos)
		if err != nil {
			log.Error(err)
			return
		}

		names, err := net.LookupHost(hos.HostName)
		if err != nil {
			log.Error(err)
			return
		}

		recordNames := []string{}
		for _, addr := range hos.PendingRevision.HostAddresses {
			recordNames = append(recordNames, addr.IPAddress)
		}

		sort.Strings(recordNames)
		sort.Strings(names)

		cleanExit := true

		if len(recordNames) == 0 {
			fmt.Println("Waring: Host objects should have at least one IP")
			cleanExit = false
		}

		if !hos.PendingRevision.ClientDeleteProhibitedStatus {
			fmt.Println("Warning: Missing Client Delete Prohibited")
			cleanExit = false
		}
		if !hos.PendingRevision.ClientUpdateProhibitedStatus {
			fmt.Println("Warning: Missing Client Update Prohibited")
			cleanExit = false
		}

		fmt.Printf("Hostname: %s\n\n", hos.HostName)

		fmt.Println("Existing DNS Records:")
		for _, name := range names {
			fmt.Printf("\t%s\n", name)
		}
		fmt.Println("")
		fmt.Println("Proposed DNS Records:")
		for _, name := range recordNames {
			fmt.Printf("\t%s\n", name)
		}
		fmt.Println("")

		pendingRev, err := json.MarshalIndent(hos.PendingRevision, "", "  ")
		if err != nil {
			fmt.Println("Error marshaling the pending revision")
		} else {
			fmt.Println(string(pendingRev))
		}
		fmt.Println("")

		_, match := listMatchDual(recordNames, names)
		if match {
			fmt.Printf("Records Match\n\n\n")
		} else {
			fmt.Printf("Record Mismatch\n\n\n")
			cleanExit = false
		}

		if !cleanExit {
			os.Exit(1)
		}
	} else {
		fmt.Println("Not a Host Object")
		os.Exit(1)
	}

}

func listsMatch(list1, list2 []string) (int64, bool) {
	var overlap int64
	allFoundSofar := true
	for _, i1 := range list1 {
		ip1 := net.ParseIP(i1)
		found := false
		for _, i2 := range list2 {
			ip2 := net.ParseIP(i2)
			if ip1.Equal(ip2) {
				found = true
				overlap++
			}
		}

		if !found {
			allFoundSofar = false
		}
	}
	return overlap, allFoundSofar
}

func listMatchDual(list1, list2 []string) (int64, bool) {
	o1, m1 := listsMatch(list1, list2)
	if !m1 {
		return o1, false
	}
	return listsMatch(list2, list1)
}
