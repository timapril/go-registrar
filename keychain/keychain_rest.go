//go:build !darwin
// +build !darwin

package keychain

// GetKeyChainPassphrase can only be used on darwin. Please see
// keychain_mac.go for full details and working code.
func GetKeyChainPassphrase(conf KeyChainConf) (pass []byte, err error) {
	return
}
