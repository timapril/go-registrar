package epp

import (
	"encoding/xml"
)

// ServiceMenu allows the EPP Service menu object to be desreialized
// This object usually exists in the greeting message and values from
// it are used in resuests such as the login message.
type ServiceMenu struct {
	XMLName               xml.Name `xml:"svcMenu" json:"-"`
	Version               string   `xml:"version" json:"version"`
	Language              string   `xml:"lang" json:"lang"`
	URIs                  []string `xml:"objURI" json:"objURI"`
	ServiceExtensionsURIs []string `xml:"svcExtension>extURI" json:"svcExtension.extURI"`
}

// GetDefaultServiceMenu generates a default service menu as described
// by the verisign EPP testing tool.
func GetDefaultServiceMenu() ServiceMenu {
	svc := ServiceMenu{}
	svc.Language = "en"
	svc.Version = "1.0"
	svc.URIs = []string{
		"http://www.verisign.com/epp/lowbalance-poll-1.0",
		"urn:ietf:params:xml:ns:contact-1.0",
		"http://www.verisign.com/epp/balance-1.0",
		"urn:ietf:params:xml:ns:domain-1.0",
		"http://www.verisign.com/epp/registry-1.0",
		"http://www.nic.name/epp/nameWatch-1.0",
		"http://www.verisign-grs.com/epp/suggestion-1.1",
		"http://www.verisign.com/epp/rgp-poll-1.0",
		"http://www.nic.name/epp/defReg-1.0",
		"urn:ietf:params:xml:ns:host-1.0",
		"http://www.nic.name/epp/emailFwd-1.0",
		"http://www.verisign.com/epp/whowas-1.0",
	}

	svc.ServiceExtensionsURIs = []string{
		"urn:ietf:params:xml:ns:secDNS-1.1",
		"http://www.verisign.com/epp/whoisInf-1.0",
		"urn:ietf:params:xml:ns:secDNS-1.0",
		"http://www.verisign.com/epp/idnLang-1.0",
		"http://www.nic.name/epp/persReg-1.0",
		"http://www.verisign.com/epp/jobsContact-1.0",
		"urn:ietf:params:xml:ns:coa-1.0",
		"http://www.verisign-grs.com/epp/namestoreExt-1.1",
		"http://www.verisign.com/epp/sync-1.0",
		"http://www.verisign.com/epp/premiumdomain-1.0",
		"http://www.verisign.com/epp/relatedDomain-1.0",
		"urn:ietf:params:xml:ns:launch-1.0",
		"urn:ietf:params:xml:ns:rgp-1.0",
	}

	return svc
}
