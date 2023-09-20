package superclient

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// AuthInfoDictionary is a list of valid characters that may be used in
// an authInfo password.
const AuthInfoDictionary = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_[]{}\\|;:,<.>/?`~"

// targetPasswordLength indicates the target password length to generate.
const targetPasswordLength = 30

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	randBytes := make([]byte, n)
	_, err := rand.Read(randBytes)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, fmt.Errorf("unable to gnereate random bytes: %w", err)
	}

	return randBytes, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) (string, error) {
	randBytes, err := GenerateRandomBytes(s)

	return strings.Replace(base64.URLEncoding.EncodeToString(randBytes), "=", "", -1), err
}

// GetAuthInfo will gerneate an AuthInfo password that will be used for
// a host or domain object. If a password cannot be generated, an error
// wiil be returned.
func GetAuthInfo() (ai string, err error) {
	for {
		genPassword := make([]byte, targetPasswordLength)
		_, err = rand.Read(genPassword)

		if err != nil {
			return
		}

		for k, v := range genPassword {
			genPassword[k] = AuthInfoDictionary[int(v)%len(AuthInfoDictionary)]
		}

		if IsValidAuthInfo(string(genPassword)) {
			return string(genPassword), nil
		}
	}
}

// IsValidAuthInfo takes a proposed AuthInfo password and will return
// true if matches the password requirements, otherwise false is
// returned.
func IsValidAuthInfo(authInfo string) bool {
	if len(authInfo) < 8 && len(authInfo) > 32 {
		return false
	}

	reDigit := regexp.MustCompile(`.*[0-9].*`)
	hasDigit := reDigit.MatchString(authInfo)
	reSpecial := regexp.MustCompile(`.*[\x21-\x2F\x3A-\x40\x5B-\x60\x7B-\x7E].*`)
	hasSpecial := reSpecial.MatchString(authInfo)
	reLetterLower := regexp.MustCompile(`.*[a-z].*`)
	hasLetterLower := reLetterLower.MatchString(authInfo)
	reLetterUpper := regexp.MustCompile(`.*[A-Z].*`)
	hasLetterUpper := reLetterUpper.MatchString(authInfo)

	return hasDigit && hasSpecial && hasLetterLower && hasLetterUpper
}
