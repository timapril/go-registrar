package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/timapril/go-registrar/whois-parse"
)

var test = flag.String("testfile", "", "when set, the file name provided will be fed to the parser")
var debug = flag.Bool("debug", false, "when set, debugging will be enabled")

func main() {

	flag.Parse()

	if len(*test) != 0 {
		fmt.Println(*test)
		data, readErr := os.ReadFile(*test)
		if readErr != nil {
			fmt.Println(readErr)
			return
		}
		fmt.Println(string(data))

		parsedResp := whois.Response{}
		_, parseErrs := parsedResp.ParseFromWhois(string(data), *debug)
		if len(parseErrs) != 0 {
			fmt.Println(parseErrs)
		}
		fmt.Println(parsedResp.AdminContactName)
		return
	}
	domains := flag.Args()
	if len(domains) != 1 {
		if len(domains) > 1 {
			fmt.Println("Only one domain may be looked up at a time")
		} else {
			fmt.Println("One domain name is required")
		}
		return
	}

	resp, errs := whois.Query(domains[0])
	if len(errs) != 0 {
		for err := range errs {
			fmt.Println(err)
			return
		}
	} else {
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(data))
	}
}
