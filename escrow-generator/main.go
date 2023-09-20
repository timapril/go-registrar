package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/csv"
	"flag"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/op/go-logging"
	"github.com/timapril/go-registrar/client"
	"github.com/timapril/go-registrar/epp"
	"github.com/timapril/go-registrar/keychain"
	"github.com/timapril/go-registrar/lib"
	"gopkg.in/gcfg.v1"
)

var (
	// certFile   = flag.String("cert", "", "A PEM eoncoded certificate file.")
	// keyFile    = flag.String("key", "", "A PEM encoded private key file.")
	// caFile     = flag.String("CA", "", "A PEM eoncoded CA's certificate file.")
	configPath = flag.String("conf", "~/.registrar", "A configuration file to provide default values for the Registrar client application")

	// keychainEnabled = flag.Bool("keychain.enabled", false, "If keychain should be used for auth or not")
	// keychainName    = flag.String("keychain.name", "", "The name of the keychain entry holding the key passphrase")
	// keychainAccount = flag.String("keychain.account", "", "The account name for the keychain entry holding the key passphrase")

	// appServer = flag.String("server", "", "The application server to connect to")

	// getWhois       = flag.Bool("whois", false, "Should the WHOIS block be output")
	// defaultContact = flag.Int64("default_contact", 1, "The ID of the default contact that should be set")

	verbose = flag.Bool("v", false, "Verbose logging")
)

// Config is an object that holds the configuration for the client
// application. gcfg parses the configuarion file into this format of
// object
type Config struct {
	Registrar struct {
		Server         string
		Port           int64
		UseHTTPS       bool
		TrustAnchor    []string
		RegistrarID    int64
		RDEFingerprint string
		RDEKeyID       string
	}

	Certs struct {
		CACertPath string
		CertPath   string
		KeyPath    string
	}
	Defaults struct {
		ApproverID int64
		AppServer  string
	}

	Mac keychain.Conf

	Testing struct {
		SpoofCert  string
		CertHeader string
	}
	Output struct {
		Path string
	}

	CacheConfig client.DiskCacheConfig
}

var conf Config

// GetConnectionURL will return the URL that can be used to connect to the
// Registrar server as defined by the parameters in the configuartion
func (c Config) GetConnectionURL() string {
	if c.Registrar.UseHTTPS {
		return fmt.Sprintf("https://%s:%d", c.Registrar.Server, c.Registrar.Port)
	}
	return fmt.Sprintf("http://%s:%d", c.Registrar.Server, c.Registrar.Port)
}

// GetTrustAnchor will use the information in the config object to create the
// trust anchor set for the application and return the trustanchors or an error
// if an error occurs.
func (c Config) GetTrustAnchor() (client.TrustAnchors, error) {
	ta := client.TrustAnchors{}
	for _, anchor := range c.Registrar.TrustAnchor {
		pubkey, readErr := os.ReadFile(anchor)
		if readErr != nil {
			return ta, readErr
		}
		err := ta.AddKey(string(pubkey))
		if err != nil {
			return ta, err
		}
	}
	return ta, nil
}

// GetRegistrarClient will use the configuration object and generate and
// return an Registrar client object. If an error occurs when generating
// the client, the error is returned
func (c Config) GetRegistrarClient() (cli client.Client, err error) {
	cli.TrustAnchor, err = c.GetTrustAnchor()
	if err != nil {
		return
	}

	cli.Prepare(c.GetConnectionURL(), log, c.CacheConfig)
	if c.Testing.SpoofCert != "" {
		spoofCert, readErr := os.ReadFile(c.Testing.SpoofCert)
		if readErr != nil {
			err = readErr
			return
		}
		cli.SpoofCertificateForTesting(string(spoofCert), conf.Testing.CertHeader)
	} else {
		cli.PrepareSSL(c.GetConnectionURL(), c.Certs.CertPath, c.Certs.KeyPath, c.Certs.CACertPath, c.Mac, log, c.CacheConfig)
	}

	return cli, nil
}

var log = logging.MustGetLogger("registrar")
var format = logging.MustStringFormatter(
	//"%{color}%{level:.4s}  ▶ %{time:15:04:05.000000} %{shortfile} § %{longfunc} %{id:03x}%{color:reset} %{message}",
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

	confErr := gcfg.ReadFileInto(&conf, *configPath)
	if confErr != nil {
		log.Fatal(confErr.Error())
	}

	log.Error(conf.Output.Path)
	createErr := os.Mkdir(conf.Output.Path, 0700)
	if createErr != nil {
		fmt.Println(createErr)
		return
	}

	cli, err := conf.GetRegistrarClient()
	if err != nil {
		log.Error(err)
		return
	}
	errs := cli.PrepareObjectDirectory()
	if len(errs) != 0 {
		for _, err := range errs {
			log.Error(err)
		}
		return
	}

	domains := make(map[int64]*lib.DomainExport)
	contacts := make(map[int64]*lib.ContactExport)

	domGetErr := GetDomainInfo(&cli, &domains, &contacts)
	if domGetErr != nil {
		log.Error(domGetErr)
		return
	}

	conGetErr := GetContacts(&cli, &contacts)
	if conGetErr != nil {
		log.Error(conGetErr)
		return
	}

	handleFile, handleErr := CreateHandlesFile(contacts)
	if handleErr != nil {
		log.Error(handleErr)
		return
	}

	domainFile, domainErr := CreateDomainFile(domains)
	if domainErr != nil {
		log.Error(domainErr)
		return
	}

	var hashes []string

	for filename, hash := range domainFile.Sums {
		_, fn := filepath.Split(filename)
		line := fmt.Sprintf("%s %s", hash, fn)
		hashes = append(hashes, line)
	}
	for filename, hash := range handleFile.Sums {
		_, fn := filepath.Split(filename)
		line := fmt.Sprintf("%s %s", hash, fn)
		hashes = append(hashes, line)
	}

	fileName, _ := FileNames(conf.Output.Path, conf.Registrar.RegistrarID, FileTypeHash, 0, "txt")
	err = os.WriteFile(fileName, []byte(strings.Join(hashes, "\n")+"\n"), 0644)
	if err != nil {
		log.Error(err)
		return
	}

}

// GetContacts will get all current valid contacts and save the contacts into
// the map that is passed indexed by id
func GetContacts(cli *client.Client, cons *map[int64]*lib.ContactExport) error {
	for id := range *cons {
		verified, conErrs, con := cli.GetVerifiedContact(id, time.Now().Unix())
		if !verified {
			log.Errorf("Unable to verifiy contact %d", id)
		}
		if len(conErrs) != 0 {
			for _, err := range conErrs {
				log.Error(err.Error())
			}
			return fmt.Errorf("error getting contact %d", id)
		}

		(*cons)[id] = con
	}

	return nil
}

// GetDomainInfo will get all current valid domains and save the domains into
// the map that is passed indexed by id
func GetDomainInfo(cli *client.Client, doms *map[int64]*lib.DomainExport, cons *map[int64]*lib.ContactExport) error {
	// domIds, _, domGetAllErrs := cli.GetAll(lib.DomainType)
	// if len(domGetAllErrs) != 0 {
	// 	for _, err := range domGetAllErrs {
	// 		log.Error(err.Error())
	// 	}
	// 	return errors.New("error getting domain list")
	// }

	for _, domain := range cli.ObjectDir.DomainIDs {
		verified, domErrs, dom := cli.GetVerifiedDomain(domain, time.Now().Unix())
		if !verified {
			log.Errorf("Unable to verifiy domain %d", domain)
		}
		if len(domErrs) != 0 {
			for _, err := range domErrs {
				log.Error(err.Error())
			}
			return fmt.Errorf("error getting domain %d", domain)
		}
		(*doms)[domain] = dom
		CheckOrAdd(dom.CurrentRevision.DomainAdminContact.ID, cons)
		CheckOrAdd(dom.CurrentRevision.DomainBillingContact.ID, cons)
		CheckOrAdd(dom.CurrentRevision.DomainRegistrant.ID, cons)
		CheckOrAdd(dom.CurrentRevision.DomainTechContact.ID, cons)

	}

	return nil
}

// CheckOrAdd will check to see if the contact ID is alread in the list of
// conacts that need to be processed and add it if it is not present
func CheckOrAdd(contactID int64, cons *map[int64]*lib.ContactExport) {
	if _, ok := (*cons)[contactID]; !ok {
		(*cons)[contactID] = nil
	}
}

// CreateHandlesFile will generate the escrow handle file mapping contacts to
// handle IDs that are used in the Domain File. The RDE file is returned
// following the processing along with any erros that may occur during the
// process.
// NOTE: This does not handle the chunking required for Domain Files over 1GB
// as required by the RDE standard for large escrows
func CreateHandlesFile(cons map[int64]*lib.ContactExport) (file *RDEFile, err error) {
	f, rdefileErr := NewRDEFile(conf.Output.Path, conf.Registrar.RegistrarID, FileTypeHandle, "csv")
	if rdefileErr != nil {
		log.Error(rdefileErr)
		return f, rdefileErr
	}
	defer f.Close()
	writer := csv.NewWriter(f)
	err = writer.Write([]string{"handleid", "name", "postalAddress", "email", "phone", "fax"})
	if err != nil {
		return file, err
	}
	for _, con := range cons {
		row := []string{
			fmt.Sprintf("REG-%d", con.ID),
			con.CurrentRevision.Name,
			con.CurrentRevision.EscrowAddress(),
			con.CurrentRevision.EmailAddress,
			con.CurrentRevision.VoiceNumber(),
			con.CurrentRevision.FaxNumber(),
		}
		err = writer.Write(row)
		if err != nil {
			return file, err
		}
	}
	writer.Flush()
	return f, nil
}

// CreateDomainFile will generate and return an RDEfile using all of the
// domains that are passed in. If an error occurs generating the file it will
// be returned
// NOTE: This does not handle the chunking required for Domain Files over 1GB
// as required by the RDE standard for large escrows
func CreateDomainFile(doms map[int64]*lib.DomainExport) (file *RDEFile, err error) {
	f, rdefileErr := NewRDEFile(conf.Output.Path, conf.Registrar.RegistrarID, FileTypeFull, "csv")
	if rdefileErr != nil {
		log.Error(rdefileErr)
		return f, rdefileErr
	}
	defer f.Close()
	writer := csv.NewWriter(f)
	err = writer.Write([]string{"registeredName", "nameservers", "expirationDate", "registrant", "technical", "admin", "billing"})
	if err != nil {
		return file, err
	}

	for _, dom := range doms {
		hostnames := []string{}
		for _, host := range dom.CurrentRevision.Hostnames {
			hostnames = append(hostnames, host.HostName)
		}
		row := []string{
			dom.DomainName,
			strings.Join(hostnames, " "),
			dom.ExpireDate.Format(epp.EPPTimeFormat),
			fmt.Sprintf("REG-%d", dom.CurrentRevision.DomainRegistrant.ID),
			fmt.Sprintf("REG-%d", dom.CurrentRevision.DomainTechContact.ID),
			fmt.Sprintf("REG-%d", dom.CurrentRevision.DomainAdminContact.ID),
			fmt.Sprintf("REG-%d", dom.CurrentRevision.DomainBillingContact.ID),
		}
		err = writer.Write(row)
		if err != nil {
			return file, err
		}
	}

	writer.Flush()
	return f, nil
}

const (
	// FileTypeFull represents a full data dump rather than incremental
	FileTypeFull = "full"

	// FileTypeIncremental represents an incremental dump of data from the
	// previous data dump
	FileTypeIncremental = "inc"

	// FileTypeHandle represents a Contact to Handle dump
	FileTypeHandle = "handle"

	// FileTypeHash represents the has file dump
	FileTypeHash = "hash"
)

// FileName takes the required fields for the RDEFile file name formation for
// ICANN provided data escrow provider and will generate the output file name
// needed
func FileNames(outputPath string, IANAID int64, filetype string, index int64, extension string) (gzFilename string, csvFilename string) {
	indexString := ""
	suffix := ""
	if filetype != FileTypeHash {
		indexString = fmt.Sprintf("_%d", index)
		suffix = ".gz"
	}

	gzFilename = fmt.Sprintf("%s/%d_RDE_%s_%s%s.%s%s", outputPath, IANAID, time.Now().Format("2006-01-02"), filetype, indexString, extension, suffix)
	csvFilename = fmt.Sprintf("%s/%d_RDE_%s_%s%s.%s", outputPath, IANAID, time.Now().Format("2006-01-02"), filetype, indexString, extension)

	return
}

// RDEFile represents a file that is being written for the Registry Data Escorw
// procedure. This implementation focuses on the ICANN provided escrow provider
// which is Iron Mountain
type RDEFile struct {
	IANAID      int64
	FileType    string
	Extension   string
	index       int64
	FileName    string
	RawFilename string
	Sums        map[string]string
	OutputPath  string

	outputFile *os.File
	zipWriter  *gzip.Writer
	hash       hash.Hash
}

// NewRDEFile will initialize the RDE file object and prepare the file for
// writing. If an error occurs during the process, it will be returned
func NewRDEFile(OutPath string, IANAID int64, FileType string, Extension string) (*RDEFile, error) {
	file := &RDEFile{OutputPath: OutPath, IANAID: IANAID, FileType: FileType, Extension: Extension}
	err := file.Prepare()
	return file, err
}

// Prepare will open the file, prepare the checksum and prepare the zip layer.
// If any error occurs while preparing the file, the error will be returned
func (r *RDEFile) Prepare() error {
	r.Sums = make(map[string]string)
	r.FileName, r.RawFilename = FileNames(r.OutputPath, r.IANAID, r.FileType, r.index, r.Extension)

	r.hash = sha256.New()

	var createErr error
	r.outputFile, createErr = os.Create(r.FileName)
	if createErr != nil {
		log.Error(createErr)
		return createErr
	}
	r.zipWriter = gzip.NewWriter(r.outputFile)
	return nil
}

// Write will write the provided buffer to the file and return the number of
// bytes written and an error if it occured
func (r *RDEFile) Write(p []byte) (n int, err error) {
	hashn, hasherr := r.hash.Write(p)
	if hasherr != nil {
		return hashn, hasherr
	}

	return r.zipWriter.Write(p)
}

// Close will flush and close the zip layer and then close the outpuf file while
// also finalizing the checksum for the file
func (r *RDEFile) Close() {
	r.zipWriter.Flush()
	r.zipWriter.Close()
	r.outputFile.Close()

	r.Sums[r.RawFilename] = fmt.Sprintf("%x", r.hash.Sum(nil))
}
