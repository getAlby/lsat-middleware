package utils

import (
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/lightningnetwork/lnd/lntypes"
	"gopkg.in/macaroon.v2"
)

func ValidMacaroon(authField string) (*macaroon.Macaroon, bool) {
	// Trim leading and trailing spaces
	authField = strings.TrimSpace(authField)
	// A typical authField
	// Authorization: LSAT AGIAJEemVQUTEyNCR0exk7ek90Cg==:1234abcd1234abcd1234abcd

	if len(authField) == 0 {
		return nil, false
	}

	// Check if macaroon is base64 encoded
	token := strings.Split(authField, " ")[1]
	macaroonString := strings.TrimSpace(strings.Split(token, ":")[0])
	if len(macaroonString) == 0 || !IsBase64(macaroonString) {
		return nil, false
	}

	macBytes, err := base64.StdEncoding.DecodeString(macaroonString)
	if err != nil {
		return nil, false
	}
	mac := &macaroon.Macaroon{}
	if err := mac.UnmarshalBinary(macBytes); err != nil {
		return nil, false
	}

	return mac, true
}

func ValidPreimage(authField string) (lntypes.Preimage, bool) {
	// Trim leading and trailing spaces
	authField = strings.TrimSpace(authField)
	// A typical authField
	// Authorization: LSAT AGIAJEemVQUTEyNCR0exk7ek90Cg==:1234abcd1234abcd1234abcd

	if len(authField) == 0 {
		return lntypes.Preimage{}, false
	}

	// Check if preimage is hex encoded
	token := strings.Split(authField, " ")[1]
	preimageString := strings.TrimSpace(strings.Split(token, ":")[1])

	if len(preimageString) == 0 || !IsHex(preimageString) {
		return lntypes.Preimage{}, false
	}
	preimage, err := lntypes.MakePreimageFromStr(preimageString)
	if err != nil {
		return lntypes.Preimage{}, false
	}

	return preimage, true
}

func IsBase64(str string) bool {
	_, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return false
	}
	return true
}

func IsHex(str string) bool {
	_, err := hex.DecodeString(str)
	if err != nil {
		return false
	}
	return true
}
