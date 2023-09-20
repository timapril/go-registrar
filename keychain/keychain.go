package keychain

// KeyChainConf holds the values required to search for a keychain entry
// in the mac keychain.
type Conf struct {
	MacKeychainEnabled bool
	MacKeychainName    string
	MacKeychainAccount string
}
