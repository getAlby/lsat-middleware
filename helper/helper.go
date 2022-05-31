package helper

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
)

func IsValidMacaroon(authField string) bool {
	// Trim leading and trailing spaces
	authField = strings.TrimSpace(authField)
	// A typical authField
	// Authorization: LSAT AGIAJEemVQUTEyNCR0exk7ek90Cg==:1234abcd1234abcd1234abcd

	if len(authField) == 0 {
		return false
	}

	// Check if macaroon is base64 encoded
	token := strings.Split(authField, " ")[1]
	macaroon := strings.TrimSpace(strings.Split(token, ":")[0])
	if len(macaroon) == 0 || !IsBase64(macaroon) {
		return false
	}

	// May need other type of validation
	return true
}

func IsValidPreimage(authField string) bool {
	// Trim leading and trailing spaces
	authField = strings.TrimSpace(authField)
	// A typical authField
	// Authorization: LSAT AGIAJEemVQUTEyNCR0exk7ek90Cg==:1234abcd1234abcd1234abcd

	if len(authField) == 0 {
		return false
	}

	// Check if preimage is hex encoded
	token := strings.Split(authField, " ")[1]
	preimage := strings.TrimSpace(strings.Split(token, ":")[1])
	if len(preimage) == 0 || !IsHex(preimage) {
		return false
	}

	// May need other type of validation
	return true
}

func IsBase64(str string) bool {
	_, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return true
	}
	return false
}

func IsHex(str string) bool {
	_, err := hex.DecodeString(str)
	if err != nil {
		return true
	}
	return false
}
