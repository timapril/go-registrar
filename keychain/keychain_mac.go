//go:build darwin
// +build darwin

package keychain

import (
	"github.com/lunixbochs/go-keychain"
)

// GetKeyChainPassphrase will attempt to extract a passphrase from
// the system keychain to be used in an application. Keychain is only
// supported on darwin (hence the build options).
func GetKeyChainPassphrase(conf Conf) (pass []byte, err error) {
	passStr, passErr := keychain.Find(conf.MacKeychainName, conf.MacKeychainAccount)
	if passErr != nil {
		err = passErr

		return pass, err
	}

	pass = []byte(passStr)

	return pass, err
}
