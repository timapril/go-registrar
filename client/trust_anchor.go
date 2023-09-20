package client

import (
	"bytes"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/clearsign"
	"golang.org/x/crypto/openpgp/packet"
)

// TrustAnchors are used to pin GPG keys that are trusted, often the
// set of trust anchors will be the bootstrap key or possibly the
// members of the default (first) approver set. Trust anchors can be
// added to reduce the amount of work required to verify a change set.
// At least one trust anchor will be required for verification to work
// correctly.
type TrustAnchors struct {
	Keys []*openpgp.Entity
}

// AddKey is used to add a new key to a trust anchor set
func (t *TrustAnchors) AddKey(key string) error {
	decbuf := bytes.NewBuffer([]byte(key + "\n"))
	block, err1 := armor.Decode(decbuf)
	if err1 != nil {
		return err1
	}
	packetReader := packet.NewReader(block.Body)
	entity, err2 := openpgp.ReadEntity(packetReader)
	if err2 != nil {
		return err2
	}

	t.Keys = append(t.Keys, entity)

	return nil
}

// KeysById returns the set of keys that have the given key id. This
// method is part of the interface for []openpgp.Entities
func (t TrustAnchors) KeysById(id uint64) (keys []openpgp.Key) {
	for _, e := range t.Keys {
		if e.PrimaryKey.KeyId == id {
			var selfSig *packet.Signature
			for _, ident := range e.Identities {
				if selfSig == nil {
					selfSig = ident.SelfSignature
				} else if ident.SelfSignature.IsPrimaryId != nil && *ident.SelfSignature.IsPrimaryId {
					selfSig = ident.SelfSignature
					break
				}
			}
			keys = append(keys, openpgp.Key{Entity: e, PublicKey: e.PrimaryKey, PrivateKey: e.PrivateKey, SelfSignature: selfSig})
		}

		for _, subKey := range e.Subkeys {
			if subKey.PublicKey.KeyId == id {
				keys = append(keys, openpgp.Key{Entity: e, PublicKey: subKey.PublicKey, PrivateKey: subKey.PrivateKey, SelfSignature: subKey.Sig})
			}
		}
	}
	return
}

// KeysByIdUsage returns the set of keys with the given id that also
// meet the key usage given by requiredUsage.  The requiredUsage is
// expressed as the bitwise-OR of packet.KeyFlag* values. This method is
// part of the interface for []openpgp.Entities
func (t TrustAnchors) KeysByIdUsage(id uint64, requiredUsage byte) (keys []openpgp.Key) {
	for _, key := range t.KeysById(id) {
		if len(key.Entity.Revocations) > 0 {
			continue
		}

		if key.SelfSignature.RevocationReason != nil {
			continue
		}

		if key.SelfSignature.FlagsValid && requiredUsage != 0 {
			var usage byte
			if key.SelfSignature.FlagCertify {
				usage |= packet.KeyFlagCertify
			}
			if key.SelfSignature.FlagSign {
				usage |= packet.KeyFlagSign
			}
			if key.SelfSignature.FlagEncryptCommunications {
				usage |= packet.KeyFlagEncryptCommunications
			}
			if key.SelfSignature.FlagEncryptStorage {
				usage |= packet.KeyFlagEncryptStorage
			}
			if usage&requiredUsage != requiredUsage {
				continue
			}
		}

		keys = append(keys, key)
	}
	return
}

// DecryptionKeys returns all private keys that are valid for
// decryption. No private keys are stored by the system so it is always
// a noop. This method is part of the interface for []openpgp.Entities
func (t TrustAnchors) DecryptionKeys() (keys []openpgp.Key) {
	return
}

// IsSignedBy will return true if the object is signed by one of the
// members of the TrustAnchors list
func (t TrustAnchors) IsSignedBy(sig []byte) (valid bool, signedBody []byte) {
	block, _ := clearsign.Decode(sig)
	if block == nil {
		return false, signedBody
	}

	_, sigErr := openpgp.CheckDetachedSignature(t, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)
	if sigErr == nil {
		return true, block.Bytes
	}
	return false, signedBody
}
