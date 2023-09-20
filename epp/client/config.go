package client

import (
	"fmt"

	logging "github.com/op/go-logging"
)

// Config stores the information required to connect to the EPP
// server or the auth proxy.
type Config struct {
	Host string
	Port int64

	LogLevel logging.Level

	Username string
	Password string

	TransactionPrefix  string
	TransactionStartID int64

	RegistrarID string

	currentTransactionID int64
}

const (
	defaultEPPPort = 1700
	defaultEPPIP   = "127.0.0.1"
)

// GetConnectionString returns the connection string for the EPP server
// or auth proxy. If the host or port are not set correctly the
// respecitive default values are "127.0.0.1" and 1700.
func (c Config) GetConnectionString() string {
	host := c.Host
	port := c.Port

	if len(host) == 0 {
		host = defaultEPPIP
	}

	if port <= 0 || port >= 65535 {
		port = defaultEPPPort
	}

	return fmt.Sprintf("%s:%d", host, port)
}

// GetNewTransactionID is used to generate a new transaction ID for the
// session (unique within session, may not be unique between session).
func (c *Config) GetNewTransactionID() string {
	if c.currentTransactionID == 0 {
		c.currentTransactionID = c.TransactionStartID
	}

	txid := fmt.Sprintf("%s%d", c.TransactionPrefix, c.currentTransactionID)

	c.currentTransactionID = c.currentTransactionID + 1

	return txid
}
