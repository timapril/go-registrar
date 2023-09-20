package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"

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
	if app.ObjectType == lib.DomainType {
		dom := lib.DomainExport{}
		err = json.Unmarshal(app.ExportRev, &dom)
		if err != nil {
			log.Error(err)
			return
		}

		rawNSes, err := net.LookupNS(dom.DomainName)
		if err != nil {
			log.Error(err)
			return
		}
		nses := []string{}
		for _, ns := range rawNSes {
			nses = append(nses, ns.Host)
		}

		recordNames := []string{}
		for _, hos := range dom.PendingRevision.Hostnames {
			recordNames = append(recordNames, hos.HostName)
		}

		pendingRev, err := json.MarshalIndent(dom.PendingRevision, "", "  ")
		if err != nil {
			fmt.Println("Error marshaling the pending revision")
		} else {
			fmt.Println(string(pendingRev))
		}
		fmt.Println("")

		sort.Strings(recordNames)
		sort.Strings(nses)

		fmt.Printf("Domain Name: %s\n\n", dom.DomainName)

		fmt.Println("Existing NSes:")
		for _, name := range nses {
			fmt.Printf("\t%s\n", name)
		}
		fmt.Println("")
		fmt.Println("Proposed NSes:")
		for _, name := range recordNames {
			fmt.Printf("\t%s\n", name)
		}
		fmt.Println("")

		cleanExit := true

		clientFlagsOK := false
		serverFlagsOK := false

		flagsOK := true
		if dom.PendingRevision.ClientRenewProhibitedStatus {
			fmt.Println("\tWARNING: Unexpected ClientRenewProhibited set")
			flagsOK = false
		}
		if dom.PendingRevision.ClientHoldStatus {
			fmt.Println("\tWARNING: Unexpected ClientHold set")
			flagsOK = false
		}
		if !dom.PendingRevision.ClientDeleteProhibitedStatus {
			fmt.Println("\tWARNING: Unexpected ClientDeleteProhibited not set")
			flagsOK = false
		}
		if !dom.PendingRevision.ClientTransferProhibitedStatus {
			fmt.Println("\tWARNING: Unexpected ClientTransferProhibited not set")
			flagsOK = false
		}
		if !dom.PendingRevision.ClientUpdateProhibitedStatus {
			fmt.Println("\tWARNING: Unexpected ClientUpdateProhibited not set")
			flagsOK = false
		}

		if flagsOK {
			fmt.Println("Client Status Flags: OK")
			clientFlagsOK = true
		} else {
			fmt.Println("Client Status Flags: Warning")
			cleanExit = false
		}

		flagsOK = true
		serverFlagsSet := false
		if dom.CurrentRevision.ServerDeleteProhibitedStatus || dom.CurrentRevision.ServerTransferProhibitedStatus || dom.CurrentRevision.ServerUpdateProhibitedStatus ||
			dom.PendingRevision.ServerDeleteProhibitedStatus || dom.PendingRevision.ServerTransferProhibitedStatus || dom.PendingRevision.ServerUpdateProhibitedStatus || dom.PendingRevision.Class == "deployed-critical" ||
			dom.PendingRevision.Class == "corp-critical" {
			serverFlagsSet = true
		}
		if dom.PendingRevision.ServerRenewProhibitedStatus {
			fmt.Println("\tWARNING: Unexpected ServerRenewProhibited set")
			flagsOK = false
		}
		if dom.PendingRevision.ServerHoldStatus {
			fmt.Println("\tWARNING: Unexpected ServerHold set")
			flagsOK = false
		}
		if serverFlagsSet && !dom.PendingRevision.ServerDeleteProhibitedStatus {
			fmt.Println("\tWARNING: Unexpected ServerDeleteProhibited not set")
			flagsOK = false
		}
		if serverFlagsSet && !dom.PendingRevision.ServerTransferProhibitedStatus {
			fmt.Println("\tWARNING: Unexpected ServerTransferProhibited not set")
			flagsOK = false
		}
		if serverFlagsSet && !dom.PendingRevision.ServerUpdateProhibitedStatus {
			fmt.Println("\tWARNING: Unexpected ServerUpdateProhibited not set")
			flagsOK = false
		}

		if serverFlagsSet {
			fmt.Println("Server Flags: Enabled")
		} else {
			fmt.Println("Server Flags: Disabled")
		}
		if flagsOK {
			fmt.Println("Server Status Flags: OK")
			serverFlagsOK = true
		} else {
			fmt.Println("Server Status Flags: Warning")
			cleanExit = false
		}

		if len(strings.TrimSpace(dom.PendingRevision.Owners)) == 0 {
			fmt.Println("Warning: Owners is empty")
			cleanExit = false
		}

		fmt.Println("")
		overlap, match := listMatchDualString(recordNames, nses)
		if match {
			fmt.Printf("Records Match\n\n\n")
		} else {
			fmt.Printf("Record Mismatch\n\n\n")
			if overlap == 0 && len(nses) > 0 {
				fmt.Println("WARNING: All host records are changing, this is not advised and could break some resolvers")
			}
			cleanExit = false
		}

		if match && clientFlagsOK && serverFlagsOK {
			fmt.Println("\n\nOK: Domain Checks out")
		} else {
			fmt.Println("\n\nError: Domain appears to have an error")
			cleanExit = false
		}

		if !cleanExit {
			os.Exit(1)
		}
	}
}

func listsMatchString(list1, list2 []string) (int64, bool) {
	var overlap int64
	allFoundSofar := true
	for _, i1 := range list1 {
		found := false
		for _, i2 := range list2 {
			if strings.TrimRight(strings.ToUpper(i1), ".") == strings.TrimRight(strings.ToUpper(i2), ".") {
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

func listMatchDualString(list1, list2 []string) (int64, bool) {
	o1, m1 := listsMatchString(list1, list2)
	if !m1 {
		return o1, false
	}
	return listsMatchString(list2, list1)
}
