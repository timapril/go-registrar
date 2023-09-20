package lib

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/op/go-logging"
)

// Format log.
var sqlRegexp = regexp.MustCompile(`(\$\d+)|\?`)

var loggerFormat = logging.MustStringFormatter(
	"%{level:.4s} - %{time:15:04:05.000000} %{shortfile} - %{longfunc} %{id:04x} %{message}",
)

type debugLogger interface {
	Debug(...interface{})
}

type LogWrapper struct {
	Logger debugLogger
}

// MustGetLogger wrapps the logging must get logging method that will
// get and return a logger object.
func MustGetLogger(name string) *logging.Logger {
	return logging.MustGetLogger(name)
}

// defaultLogFileMode is the default file mode for the log files.
const defaultLogFileMode = 0o666

// ConfigureLogging will prepare the logging infrastructure.
func ConfigureLogging(conf Config) error {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, loggerFormat)
	backendLevel := logging.AddModuleLevel(backendFormatter)
	backendLevel.SetLevel(conf.Logging.LogLevel, "")

	if len(conf.Logging.File) > 0 {
		if conf.Logging.File == "" {
			return errors.New("config missing value for LogFile")
		}

		outfile, outFileErr := os.OpenFile(conf.Logging.File, os.O_RDWR|os.O_CREATE, defaultLogFileMode)
		if outFileErr != nil {
			return fmt.Errorf("error creating log file: %w", outFileErr)
		}

		fileBackend := logging.NewLogBackend(outfile, "", 0)
		backendFormatter2 := logging.NewBackendFormatter(fileBackend, loggerFormat)
		backendLevel2 := logging.AddModuleLevel(backendFormatter2)
		backendLevel2.SetLevel(conf.Logging.LogLevel, "")
		logging.SetBackend(backendLevel, backendLevel2)
	} else {
		logging.SetBackend(backendLevel)
	}

	return nil
}

// Print will take a list of values passed in the form of the
// Gorm.Logger output format and display them as a signle line in a
// way that can be pushed to a output logger (like go-logging).
func (l LogWrapper) Print(values ...interface{}) {
	if len(values) > 1 {
		level := values[0]
		msg := fmt.Sprintf("%s - %v - ", level, values[1])

		if level == "sql" {
			// duration
			// fmt.Sprintf("%s - %.2fms", msg, float64(values[2].(time.Duration).Nanoseconds()/1e4)/100.0)
			// sql
			var formatedValues []interface{}

			for _, value := range values[4].([]interface{}) {
				indirectValue := reflect.Indirect(reflect.ValueOf(value))
				if indirectValue.IsValid() {
					value = indirectValue.Interface()
					if t, ok := value.(time.Time); ok {
						formatedValues = append(formatedValues, fmt.Sprintf("'%v'", t.Format(time.RFC3339)))
					} else if b, ok := value.([]byte); ok {
						formatedValues = append(formatedValues, fmt.Sprintf("'%v'", string(b)))
					} else if r, ok := value.(driver.Valuer); ok {
						if value, err := r.Value(); err == nil && value != nil {
							formatedValues = append(formatedValues, fmt.Sprintf("'%v'", value))
						} else {
							formatedValues = append(formatedValues, "NULL")
						}
					} else {
						formatedValues = append(formatedValues, fmt.Sprintf("'%v'", value))
					}
				} else {
					formatedValues = append(formatedValues, fmt.Sprintf("'%v'", value))
				}
			}

			formatedSQL := fmt.Sprintf(sqlRegexp.ReplaceAllString(values[3].(string), "%v"), formatedValues...)
			msg = fmt.Sprintf("%s %s", msg, formatedSQL)
		} else {
			for _, value := range values[2:] {
				msg = fmt.Sprintf("%s %q", msg, value)
			}
		}

		l.Logger.Debug(msg)
	}
}
