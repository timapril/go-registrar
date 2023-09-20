package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidDurationError is used to indicate that the config
	// provides an invalid duration.
	ErrInvalidDurationError = errors.New("invalid Token Duration")

	// ErrInvalidToken is used to indicate that the provided CSRF token
	// is invalid for some reason.
	ErrInvalidToken = errors.New("invalid CSRF token")
)

// Config is an interface that must be implemented in order for the
// library to work correctly.
type Config interface {
	GetValidityPeriod() time.Duration
	GetHMACKey() []byte
}

// numFields represents the number of fields contained in a CSRF token.
const numFields = 4

// bitsInByte is used for shifting a value over one byte.
const bitsInByte = 8

// randomSeedLength represents the number of random bytes to include in
// the hmac nonce.
const randomSeedLength = 6

// CheckCSRF verifies a CSRF token as part of a request. A bool
// indicating if the CSRF token was valid and an error if there is an
// issue with the token.
func CheckCSRF(username string, conf Config, token string) (bool, error) {
	tokens := strings.Split(token, ":")

	if len(tokens) != numFields {
		return false, ErrInvalidToken
	}

	user := tokens[0]
	expire, expireErr := strconv.ParseInt(tokens[1], 10, 64)
	nonce, nonceErr := strconv.ParseInt(tokens[2], 10, 64)
	hmacsum := tokens[3]

	if user != username || expireErr != nil || nonceErr != nil {
		return false, ErrInvalidToken
	}

	expireTime := time.Unix(expire, 0)
	if !expireTime.After(time.Now()) {
		return false, ErrInvalidToken
	}

	hmacsumcomputed := genHmac(username, expire, nonce, conf)
	if hmacsumcomputed != hmacsum {
		return false, ErrInvalidToken
	}

	return true, nil
}

// GenerateCSRF will use the username and config provided to generate
// a new CSRF token. An error is retuend if a new token cannot be
// generated.
func GenerateCSRF(username string, conf Config) (string, error) {
	var randInt int64

	randLength := randomSeedLength
	randBytes := make([]byte, randLength)

	count, err := rand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("unable to get random seed: %w", err)
	}

	if count != randLength {
		return "", fmt.Errorf("insufficient randomness available: %w", err)
	}

	for randByte := range randBytes {
		randInt = randInt<<bitsInByte + int64(randBytes[randByte])
	}

	tokenExpire := time.Now().Add(conf.GetValidityPeriod()).Unix()

	hexSum := genHmac(username, tokenExpire, randInt, conf)

	return fmt.Sprintf("%s:%d:%d:%s", username, tokenExpire, randInt, hexSum), nil
}

func genHmac(username string, expire int64, nonce int64, conf Config) string {
	tempCSRF := fmt.Sprintf("%s:%d:%d:", username, expire, nonce)

	mac := hmac.New(sha256.New, conf.GetHMACKey())

	mac.Write([]byte(tempCSRF))

	sum := mac.Sum(nil)
	hexSum := ""

	for sumByte := range sum {
		hexSum = fmt.Sprintf("%s%X", hexSum, sum[sumByte])
	}

	return hexSum
}
