package objects

import (
	"errors"

	"github.com/timapril/go-registrar/lib"
)

// Host holds the data required to generate a WHOIS response.
type Host struct {
	ID int64

	HostName      string
	HostROID      string
	HostAddresses []string
}

// WHOISHostFromExport takes a lib.HostExport object and extracts
// the current values that are required to display the valid WHOIS
// information.
func WHOISHostFromExport(libhost *lib.HostExport) (Host, error) {
	h := Host{}
	h.ID = libhost.ID

	currentRevision := libhost.CurrentRevision
	if currentRevision.ID == 0 {
		return h, errors.New("Unable to find current revision")
	}

	// Hostname
	h.HostName = libhost.HostName

	// Host Addresses
	for _, host := range currentRevision.HostAddresses {
		h.HostAddresses = append(h.HostAddresses, host.IPAddress)
	}

	return h, nil
}
