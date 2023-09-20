package lib

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql" // MySQL drivers.
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"           // Postgres drivers.
	_ "github.com/mattn/go-sqlite3" // SQLite3 drivers.
	"github.com/op/go-logging"
)

// LoadDB is used to start the connection to the database and set up
// logging as defined in the configuration file. If an error occures
// when getting the database connection, an error is returned.
func LoadDB(conf Config, logger *logging.Logger) (db *gorm.DB, err error) {
	db = &gorm.DB{}

	switch conf.Database.Type {
	case DBTypeMySQL:
		*db, err = gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True", conf.Database.User, conf.Database.Password, conf.Database.Host, conf.Database.Port, conf.Database.Database))
	case DBTypeSqlite:
		*db, err = gorm.Open("sqlite3", conf.Database.Path)
	case DBTypePostgres:
		connectionString := fmt.Sprintf("user=%s host=%s dbname=%s port=%s", conf.Database.User, conf.Database.Host, conf.Database.Database, conf.Database.Port)
		if conf.Database.CertAuth {
			connectionString = fmt.Sprintf("user=%s host=%s dbname=%s port=%s sslcert=%s sslkey=%s sslrootcert=%s sslmode=verify-full", conf.Database.User, conf.Database.Host, conf.Database.Database, conf.Database.Port, conf.Database.CertPath, conf.Database.KeyPath, conf.Database.ChainPath)
		}

		*db, err = gorm.Open("postgres", connectionString)

	default:
		err = fmt.Errorf("unknown DB type %s", conf.Database.Type)
	}

	if err != nil {
		return db, err
	}

	wrap := LogWrapper{Logger: logger}

	db.SetLogger(wrap)

	if conf.Logging.DatabaseDebugging {
		logger.Debug("Database logging has been enabled")
		db.LogMode(true)
	}

	return db, err
}
