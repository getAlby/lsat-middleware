package macaroon

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"

	"gopkg.in/macaroon.v2"
)

type MacaroonIdentifier struct {
	Version     uint16
	PaymentHash []byte
	TokenId     [32]byte
}

func BakeMacaroon(paymentHash []byte) (*macaroon.Macaroon, error) {
	rootKey, err := generateRootKey()
	if err != nil {
		return nil, err
	}

	identifier, err := generateMacaroonIdentifier(paymentHash)
	if err != nil {
		return nil, err
	}

	return macaroon.New(
		rootKey[:],
		identifier,
		"LSAT",
		macaroon.LatestVersion,
	)
}

func generateMacaroonIdentifier(paymentHash []byte) ([]byte, error) {
	tokenId, err := generateTokenId()
	if err != nil {
		return nil, err
	}

	id := &MacaroonIdentifier{
		Version:     0,
		PaymentHash: paymentHash,
		TokenId:     tokenId,
	}

	var identifier bytes.Buffer
	enc := gob.NewEncoder(&identifier)
	if err := enc.Encode(id); err != nil {
		return nil, err
	}
	return identifier.Bytes(), err
}

func generateTokenId() ([32]byte, error) {
	var tokenId [32]byte
	_, err := rand.Read(tokenId[:])
	return tokenId, err
}

func generateRootKey() ([32]byte, error) {
	var rootKey [32]byte
	_, err := rand.Read(rootKey[:])
	return rootKey, err
}
