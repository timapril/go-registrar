package whois

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

// Response is a data structre that contains the information that has
// been parsed from the WHOIS response.
type Response struct {
	ObjectType                string
	DomainName                string
	Registrar                 string
	SponsoringRegistrarIANAID string
	WhoisServer               string
	ReferralURL               string
	NameServers               []string
	Statuses                  []string
	UpdatedDate               time.Time
	CreationDate              time.Time
	ExpirationDate            time.Time
	RegistrarURL              string
	TechContactName           string
	BillingContactName        string
	AdminContactName          string
	RegistrantContactName     string
	FullLookup                string
	DNSSECSigned              bool
}

// ParseFromWhois tries to extract WHOIS infomration from well formed
// WHOIS infomration into the data structure that is passed in.
// TODO: Split up function.
// TODO: switch the ifelse tree to a switch statment.
func (r *Response) ParseFromWhois(data string, debug bool) (string, []error) {
	var nextServer string

	var errs []error

	var tmpErr error

	r.ObjectType = "domain"
	r.DNSSECSigned = false

	r.FullLookup = fmt.Sprintf("%s\n%s", r.FullLookup, data)

	lines := strings.Split(data, "\n")
	for idx, line := range lines {
		trimmedLine := strings.Trim(line, " \t")
		if strings.HasPrefix(trimmedLine, "Whois Server: ") {
			r.WhoisServer = trimmedLine[len("Whois Server: "):]
			nextServer = r.WhoisServer
		} else if strings.HasPrefix(trimmedLine, "Domain Name: ") {
			r.DomainName = trimmedLine[len("Domain Name: "):]
		} else if strings.HasPrefix(trimmedLine, "Registrar: ") {
			r.Registrar = trimmedLine[len("Registrar: "):]
		} else if strings.HasPrefix(trimmedLine, "Sponsoring Registrar IANA ID: ") {
			r.SponsoringRegistrarIANAID = trimmedLine[len("Sponsoring Registrar IANA ID: "):]
		} else if strings.HasPrefix(trimmedLine, "Referral URL: ") {
			r.ReferralURL = trimmedLine[len("Referral URL: "):]
		} else if strings.HasPrefix(trimmedLine, "Registrar URL: ") {
			r.RegistrarURL = trimmedLine[len("Registrar URL: "):]
		} else if strings.HasPrefix(trimmedLine, "Registrant Name: ") {
			r.TechContactName = trimmedLine[len("Registrant Name: "):]
		} else if strings.HasPrefix(trimmedLine, "Admin Name: ") {
			r.AdminContactName = trimmedLine[len("Admin Name: "):]
		} else if strings.HasPrefix(trimmedLine, "Tech Name: ") {
			r.RegistrantContactName = trimmedLine[len("Tech Name: "):]
		} else if strings.HasPrefix(trimmedLine, "Billing Name: ") {
			r.BillingContactName = trimmedLine[len("Billing Name: "):]
		} else if strings.HasPrefix(trimmedLine, "Name Server: ") {
			r.AddNameserver(trimmedLine[len("Name Server: "):])
		} else if strings.HasPrefix(trimmedLine, "Status: ") {
			values := strings.Split(trimmedLine[len("Status: "):], " ")
			if len(values) >= 1 {
				r.Statuses = append(r.Statuses, values[0])
			}
		} else if strings.HasPrefix(trimmedLine, "Updated Date: ") {
			dateStr := trimmedLine[len("Updated Date: "):]
			r.UpdatedDate, tmpErr = ParseWhoisDate(dateStr)
			if tmpErr != nil {
				errs = append(errs, tmpErr)
			}
		} else if strings.HasPrefix(trimmedLine, "Creation Date: ") {
			dateStr := trimmedLine[len("Creation Date: "):]
			r.CreationDate, tmpErr = ParseWhoisDate(dateStr)
			if tmpErr != nil {
				errs = append(errs, tmpErr)
			}
		} else if strings.HasPrefix(trimmedLine, "Expiration Date: ") {
			dateStr := trimmedLine[len("Expiration Date: "):]
			r.ExpirationDate, tmpErr = ParseWhoisDate(dateStr)
			if tmpErr != nil {
				errs = append(errs, tmpErr)
			}
		} else if strings.HasPrefix(trimmedLine, "DNSSEC: ") {
			signedString := strings.ToLower(trimmedLine[len("DNSSEC: "):])
			if signedString == "signed" || signedString == "yes" {
				r.DNSSECSigned = true
			} else {
				r.DNSSECSigned = false
			}
		} else if debug {
			log.Printf("% 3d: %s\n", idx, trimmedLine)
		}
	}

	return nextServer, errs
}

// AddStatus will check if a status exists in the list of statuses
// already and if not it will add it.
func (r *Response) AddStatus(status string) {
	found := false

	for _, testStatus := range r.Statuses {
		if status == testStatus {
			found = true

			break
		}
	}

	if !found {
		r.Statuses = append(r.Statuses, status)
	}
}

// AddNameserver will check if a name server exists in the list of name
// servers already and if not it will add it.
func (r *Response) AddNameserver(name string) {
	found := false

	for _, testName := range r.NameServers {
		if name == testName {
			found = true

			break
		}
	}

	if !found {
		r.NameServers = append(r.NameServers, name)
	}
}

// numTokensInDate represents the number of valid tokens in a valid
// ISO 8601 date.
const numTokensInDate = 3

// ParseWhoisDate will take a date from a WHOIS response and turn it
// into a time object. If the conversion fails, an error will be
// returned.
func ParseWhoisDate(date string) (rettime time.Time, err error) {
	dateTokens := strings.Split(date, "-")
	if len(dateTokens) == numTokensInDate {
		if date[len(date)-1:] == "Z" {
			rettime, err = time.Parse("2006-01-02T15:04:05Z", date)
		} else {
			month := fmt.Sprintf("%s%s", strings.ToUpper(dateTokens[1][0:1]), dateTokens[1][1:])
			dateTokens[1] = month
			date = strings.Join(dateTokens, "-")
			rettime, err = time.Parse("02-Jan-2006", date)
		}
	}

	return
}

// whoisTimeout represents how long a whois query should take a most to
// return.
const whoisTimeout = 30 * time.Second

func sendQuery(domain, host string) ([]byte, []error) {
	var data []byte

	var errs []error

	conn, connErr := net.DialTimeout("tcp", host+":43", whoisTimeout)
	if connErr != nil {
		errs = append(errs, connErr)

		return data, errs
	}

	defer conn.Close()

	var query []byte

	if strings.Contains(host, "verisign") {
		query = []byte("=" + domain + "\r\n")
	} else {
		query = []byte(domain + "\r\n")
	}

	_, writeErr := conn.Write(query)

	if writeErr != nil {
		errs = append(errs, writeErr)

		return data, errs
	}

	data, readErr := io.ReadAll(conn)
	if readErr != nil {
		errs = append(errs, readErr)

		return data, errs
	}

	return data, errs
}

// Query takes a domain to lookup and then returns a response and a list
// of errors if any.
//
// TODO: Switch the lookup to a map lookup rather than having to keep
// checking.
func Query(domain string) (resp Response, errs []error) {
	whoisServer := "whois.iana.org"
	ldomain := strings.ToLower(domain)

	// Find more with `whois -h whois.iana.org <domain>`
	if strings.HasSuffix(ldomain, ".net") || strings.HasSuffix(ldomain, ".com") {
		whoisServer = "whois.verisign-grs.com"
	}

	var parseErrs []error

	for {
		data, derrs := sendQuery(ldomain, whoisServer)
		errs = append(errs, derrs...)

		whoisServer, parseErrs = resp.ParseFromWhois(string(data), false)
		errs = append(errs, parseErrs...)

		if whoisServer == "" {
			break
		}
	}

	return
}
