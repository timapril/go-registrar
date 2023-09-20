// Package lib provides the objects required to operate registrar
package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_LoadConfig_BogusFile(t *testing.T) {
	t.Parallel()

	conf := Config{}

	err := conf.LoadConfig("bogusfile")
	if err == nil {
		t.Error("Expected error when trying to open a bogus file")
	}

	t.Log(err)
}

func genTestingGPGKey(path string, keyFilename string, entityName string, entityComent string, _ string) (string, error) {
	privPath := filepath.Join(path, fmt.Sprintf("%s.key", keyFilename))
	pubPath := filepath.Join(path, fmt.Sprintf("%s.pub", keyFilename))

	var entity *openpgp.Entity

	entity, err := openpgp.NewEntity(entityName, entityComent, entityComent, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create gpg entity: %w", err)
	}

	// Add more identities here if you wish

	// Sign all the identities
	for _, id := range entity.Identities {
		err := id.SelfSignature.SignUserId(id.UserId.Id, entity.PrimaryKey, entity.PrivateKey, nil)
		if err != nil {
			return "", fmt.Errorf("unable to selfsign gpg identity: %w", err)
		}
	}

	pubKeyBuf := new(bytes.Buffer)
	privKeyBuf := new(bytes.Buffer)

	pubKeyWriter, err := armor.Encode(pubKeyBuf, openpgp.PublicKeyType, nil)
	if err != nil {
		return "", fmt.Errorf("unable to armor encode the entity's public key: %w", err)
	}

	err = entity.Serialize(pubKeyWriter)
	if err != nil {
		return "", fmt.Errorf("unable to serialize the entity's public key: %w", err)
	}

	err = pubKeyWriter.Close()
	if err != nil {
		return "", fmt.Errorf("unable to close the entity's public key writer: %w", err)
	}

	err = os.WriteFile(pubPath, pubKeyBuf.Bytes(), 0o600)
	if err != nil {
		return "", fmt.Errorf("unable to write the entity's public key: %w", err)
	}

	privKeyWriter, err2 := armor.Encode(privKeyBuf, openpgp.PrivateKeyType, nil)
	if err2 != nil {
		return "", fmt.Errorf("unable to armor the entity's private key: %w", err)
	}

	err = entity.SerializePrivate(privKeyWriter, nil)
	if err != nil {
		return "", fmt.Errorf("unable to serialize the entity's private key: %w", err)
	}

	err = privKeyWriter.Close()
	if err != nil {
		return "", fmt.Errorf("unable to close the entity's private key writer: %w", err)
	}

	err = os.WriteFile(privPath, privKeyBuf.Bytes(), 0o600)
	if err != nil {
		return "", fmt.Errorf("unable to write the entity's private key: %w", err)
	}

	fp := entity.PrimaryKey.Fingerprint

	PGPFingerprint := fmt.Sprintf("%0X %0X %0X %0X %0X  %0X %0X %0X %0X %0X", fp[0:2], fp[2:4], fp[4:6], fp[6:8], fp[8:10], fp[10:12], fp[12:14], fp[14:16], fp[16:18], fp[18:20])

	return PGPFingerprint, nil
}

func CreateDummyConfigString(pubkeyFilename, _, dbtype, logLevel, testingDBPath string) (dummyConfig string) {
	dummyConfig = `[server]
port=6000
templatePath=./templates
basePath=./
appURL=http://appurl/

`
	dummyMysqlConfig := `[database]
type=mysql
host=127.0.0.1
port=3306
user=root
password=toor
database=registrar

`

	dummySQLiteConfigFormat := `[database]
type=sqlite
path=%s

`
	dummySQLiteConfig := fmt.Sprintf(dummySQLiteConfigFormat, testingDBPath)

	if dbtype == "mysql" {
		dummyConfig += dummyMysqlConfig
	} else {
		dummyConfig += dummySQLiteConfig
	}

	dummyLoggingConfigFormat := `[logging]
logFile=./logfile
databaseDebugging=false
logLevel=%s

`

	dummyLoggingConfigBogus := fmt.Sprintf(dummyLoggingConfigFormat, "BOGUS")
	dummyLoggingConfigError := fmt.Sprintf(dummyLoggingConfigFormat, "ERROR")

	if logLevel == bogusState {
		dummyConfig += dummyLoggingConfigBogus
	} else {
		dummyConfig += dummyLoggingConfigError
	}

	dummyBootstrapConfigFormat := `[bootstrap]
name=Test User
username=test
employeeID=1
emailaddress=test@example.com
role=test
department=testing
defaultsettitle=Testing Approver Set
defaultsetdescription=default aprovers
fingerprint=%s
pubkeyfile=%s

`
	entity, errKey := getKey(BaseTestingKey)

	var dummyBootstrapConfig string

	if errKey == nil {
		fp := entity.PrimaryKey.Fingerprint
		PGPFingerprint := fmt.Sprintf("%0X %0X %0X %0X %0X  %0X %0X %0X %0X %0X", fp[0:2], fp[2:4], fp[4:6], fp[6:8], fp[8:10], fp[10:12], fp[12:14], fp[14:16], fp[16:18], fp[18:20])
		dummyBootstrapConfig = fmt.Sprintf(dummyBootstrapConfigFormat, PGPFingerprint, pubkeyFilename)
	} else {
		dummyBootstrapConfig = fmt.Sprintf(dummyBootstrapConfigFormat, "1234 1234 1234 1234 1234  1234 1234 1234 1234 1234", pubkeyFilename)
	}

	dummyConfig += dummyBootstrapConfig

	dummyCSRFConfig := `[csrf]
validityTime=1800
MACKey=testingmackey

`
	dummyConfig += dummyCSRFConfig

	dummyEmailConfig := `[email]
server=smtp-server
fromEmail=noreply@example.com
fromName=Registrar Testing
announce=foo@example.com
cc=bar@example.com
enabled=false

`

	dummyConfig += dummyEmailConfig

	return dummyConfig
}

func CreateDummyConfig(confFilename, pubkeyFilename, ignore, dbtype, logLevel string) {
	dummyConfig := CreateDummyConfigString(pubkeyFilename, ignore, dbtype, logLevel, GetTestingDBPath())

	err := os.WriteFile(confFilename, []byte(dummyConfig), 0o600)
	if err != nil {
		logger.Errorf("error creating dummy config file: %s", err.Error())
	}
}

func Test_LoadConfig_ForErrors(t *testing.T) {
	t.Parallel()
	// Create the files
	tfp := GetTestFilenamer()

	confFilename, err := tfp.TempFilename("conf")
	if err != nil {
		t.Fatalf("Unable to allocate temp config file: %s", err.Error())
	}

	pubkeyFilename, err := tfp.TempFilenameWith("pubkey", "test")
	if err != nil {
		t.Fatalf("Unable to allocate temp pubkey file: %s", err.Error())
	}
	//FIXME: this doesn't appear to be used...
	//emptyPubkeyFilename, err := tfp.TempFilename("empty_pk")
	//if err != nil {
	//	t.Fatalf("Unable to allocate temp empty pubkey file: %s", err.Error())
	//}

	// Testing to make sure that an unknown log level returns an error
	CreateDummyConfig(confFilename, pubkeyFilename, "logWrongLevel", DBTypeMySQL, bogusState)

	conf := Config{}

	err = conf.LoadConfig(confFilename)
	if err == nil {
		if conf.Logging.LogLevel != logging.WARNING {
			t.Error("Expected logLevel to default to Warning when set incorrectly")
		}
	}

	tfp.Cleanup()
}

func TestGetValidityPeriod(t *testing.T) {
	t.Parallel()
	Convey("Given a testing config the validity time should be greater than 0 seconds", t, func() { // Create the files
		tfp := GetTestFilenamer()

		confFilename, err := tfp.TempFilename("conf")
		if err != nil {
			t.Fatalf("Unable to allocate temp config file: %s", err.Error())
		}

		pubkeyFilename, err := tfp.TempFilenameWith("pubkey", "test")
		if err != nil {
			t.Fatalf("Unable to allocate temp pubkey file: %s", err.Error())
		}

		CreateDummyConfig(confFilename, pubkeyFilename, "", DBTypeMySQL, "")
		conf := Config{}

		err = conf.LoadConfig(confFilename)
		if err != nil {
			t.Error(err)
		}

		validTime := conf.GetValidityPeriod()
		So(validTime.Seconds(), ShouldBeGreaterThan, 0)
		tfp.Cleanup()
	})
}

func TestGetHMACKey(t *testing.T) {
	t.Parallel()
	Convey("Given a testing config the HMAC key should be non zero in length", t, func() { // Create the files
		tfp := GetTestFilenamer()

		confFilename, err := tfp.TempFilename("conf")
		if err != nil {
			t.Fatalf("Unable to allocate temp config file: %s", err.Error())
		}

		pubkeyFilename, err := tfp.TempFilenameWith("pubkey", "test")
		if err != nil {
			t.Fatalf("Unable to allocate temp pubkey file: %s", err.Error())
		}

		CreateDummyConfig(confFilename, pubkeyFilename, "", DBTypeMySQL, "")
		conf := Config{}
		err = conf.LoadConfig(confFilename)
		if err != nil {
			t.Error(err)
		}

		key := conf.GetHMACKey()
		So(len(key), ShouldBeGreaterThan, 0)
		defer os.Remove(pubkeyFilename)
		tfp.Cleanup()
	})
}
