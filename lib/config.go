// Package lib provides the objects required to operate registrar
package lib

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"

	"gopkg.in/gcfg.v1"
	// "github.com/msbranco/goconfig".
	"github.com/op/go-logging"
)

// DBTypeMySQL is a constant to describe a MySQL database type.
const DBTypeMySQL string = "mysql"

// DBTypeSqlite is a constant to describe a sqlite database type.
const DBTypeSqlite string = "sqlite"

// DBTypePostgres is a constant to describe a postgres database type.
const DBTypePostgres string = "pq"

// A Config object holds the runtime configuration parameters set in the
// configuration file which is read at startup. Common configuration
// settings like database connection information is required and
// bootstraping settings may also be included.
type Config struct {
	Server struct {
		Port              int64
		TemplatePath      string
		BasePath          string
		AppURL            string
		CertHeader        string
		DefaultUserDomain string
	}

	Database struct {
		Type      string
		Host      string
		Port      string
		User      string
		Password  string
		Database  string
		Path      string
		CertPath  string `gcfg:"certpath"`
		KeyPath   string `gcfg:"keypath"`
		ChainPath string `gcfg:"chainpath"`
		CertAuth  bool
		MaxRTT    float64
	}

	Logging struct {
		File              string `gcfg:"logFile"`
		DatabaseDebugging bool
		LogLevelRaw       string        `gcfg:"logLevel"`
		LogLevel          logging.Level `gcfg:"logLevelParsed"`
	}

	Email struct {
		Server    string
		FromEmail string
		FromName  string
		Announce  string
		CC        string
		Enabled   bool
	}

	Bootstrap struct {
		Name                  string
		Username              string
		EmployeeID            int64
		EmailAddress          string
		Role                  string
		Department            string
		DefaultSetTitle       string
		DefaultSetDescription string
		Fingerprint           string
		Pubkeyfile            string
		PubkeyContents        []byte `gcfg:""`
	}

	CSRF struct {
		ValidityTime     int64
		ValidityDuration time.Duration `gcfg:""`
		MACKey           string
	}

	Registrar struct {
		ID string
	}
}

// LoadConfig will attempt to load the configuration at the path
// provided and will return the parsed config or an error.
func LoadConfig(path string) (Config, error) {
	conf := &Config{}
	err := conf.LoadConfig(path)

	return *conf, err
}

// LoadConfig will open the file (arg1) parse the required fields before
// starting the application.
func (con *Config) LoadConfig(path string) error {
	err := gcfg.ReadFileInto(con, path)
	if err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}

	con.CSRF.ValidityDuration = time.Duration(con.CSRF.ValidityTime) * time.Second

	con.Logging.LogLevel, err = logging.LogLevel(con.Logging.LogLevelRaw)
	if err != nil {
		return fmt.Errorf("error configuring logging: %w", err)
	}

	con.Bootstrap.PubkeyContents, err = os.ReadFile(con.Bootstrap.Pubkeyfile)

	if err != nil {
		return fmt.Errorf("error prasing bootstrap key: %w", err)
	}

	return nil
}

// GetValidityPeriod returns the period for which CSRF tokens will be
// valid for.
func (con Config) GetValidityPeriod() time.Duration {
	return con.CSRF.ValidityDuration
}

// GetHMACKey returns the key that is used for.
func (con Config) GetHMACKey() []byte {
	return []byte(con.CSRF.MACKey)
}

// GetMailHosts will resolve the MX records for a domain name and return the
// list of hosts sorted by preference if MX records are sent, otherwise the
// domain name is returned as a fallback A or AAAA record.
func GetMailHosts(domain string) (hosts []string) {
	mxs, err := net.LookupMX(domain)
	if err == nil {
		for _, mx := range mxs {
			hosts = append(hosts, mx.Host)
		}

		return
	}

	return []string{domain}
}

// SendAllEmail will iterate through all of the users in the to field and will
// separate the users into lists of users for each domain and then send one
// email for each set of domain users using the domain lookup for the MX of
// the domains.
func (con Config) SendAllEmail(subject string, message string, recipients []string) (err error) {
	if !con.Email.Enabled {
		return nil
	}

	toDomains := make(map[string][]string)

	for _, toaddr := range recipients {
		emailAddr, err := mail.ParseAddress(toaddr)
		if err != nil {
			return fmt.Errorf("error parsing email address: %w", err)
		}

		if strings.Count(emailAddr.Address, "@") == 1 {
			tokens := strings.Split(emailAddr.Address, "@")
			toDomains[tokens[1]] = append(toDomains[tokens[1]], emailAddr.Address)
		}
	}

	for domain, users := range toDomains {
		mxs := GetMailHosts(domain)
		sent := false

		for _, mx := range mxs {
			sendErr := SendEmail(mx, con.Email.FromEmail, con.Email.FromName, subject, message, users)
			if sendErr == nil {
				sent = true

				break
			}
		}

		if !sent {
			logger.Errorf("unable to send mail to %s", domain)
		}
	}

	return nil
}

// SendEmail takes a subject, message and a list of recipients and will
// attempt to send an email. If the email fails to send, an error is
// returned, otherwise nil is returned.
func SendEmail(server string, fromEmail string, fromName string, subject string, message string, sendTo []string) (err error) {
	conn, smptErr := smtp.Dial(fmt.Sprintf("%s:25", server))
	if smptErr != nil {
		return fmt.Errorf("unable to dial SMTP connection: %w", smptErr)
	}

	defer func() {
		if err != nil {
			err = conn.Close()
		}
	}()

	buf := bytes.NewBufferString(fmt.Sprintf("From: %s <%s>\n", fromName, fromEmail))

	if err = conn.Mail(fromEmail); err != nil {
		return fmt.Errorf("unable to set sender: %w", err)
	}

	for _, rcpt := range sendTo {
		if rcptErr := conn.Rcpt(rcpt); rcptErr != nil {
			return fmt.Errorf("unable to set recipient: %w", rcptErr)
		}
	}

	if _, writerErr := buf.WriteString(fmt.Sprintf("To: %s\n", strings.Join(sendTo, ","))); writerErr != nil {
		return fmt.Errorf("error writing email message: %w", writerErr)
	}

	dataWriter, err := conn.Data()
	if err != nil {
		return fmt.Errorf("unable to open data connection: %w", err)
	}

	defer func() {
		if err != nil {
			err = dataWriter.Close()
		}
	}()

	if _, writerErr3 := buf.WriteString(fmt.Sprintf("Subject: %s\n\n", subject)); writerErr3 != nil {
		return fmt.Errorf("unable to write subject: %w", writerErr3)
	}

	if _, writerErr4 := buf.WriteString(message); writerErr4 != nil {
		return fmt.Errorf("unable to write message: %w", writerErr4)
	}

	if _, writerErr5 := buf.WriteTo(dataWriter); writerErr5 != nil {
		return fmt.Errorf("unable to write to field: %w", writerErr5)
	}

	err = conn.Quit()
	if err != nil {
		return fmt.Errorf("error closing email connection: %w", err)
	}

	return nil
}

var (
	// ErrDatabaseCertPathNotSet is used to indicate that the cert path
	// for the application to communicate with the database is missing.
	ErrDatabaseCertPathNotSet = errors.New("database cert path was not set in the configuration")

	// ErrUnableToSelectMaxIndex is used to indicate that there was an
	// error finding the maximum secret index for the certificates.
	ErrUnableToSelectMaxIndex = errors.New("unable to select maximum secret index")
)

var instance Config

// GetConfig returns the instance of Config thats has been created if
// an instance has been created. If Config.IsLoaded is not set it was
// not read to be used.
// TODO: try and see if there is a way to use this pattern
// http://marcio.io/2015/07/singleton-pattern-in-go/
func GetConfig() Config {
	return instance
}
