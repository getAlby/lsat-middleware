package lsat

import (
	"bytes"
	"encoding/gob"
	"fmt"
	mintMacaroon "proxy/macaroon"

	"github.com/lightningnetwork/lnd/lntypes"
	"gopkg.in/macaroon.v2"
)

func VerifyLSAT(mac *macaroon.Macaroon, rootKey []byte, preimage lntypes.Preimage) error {
	_, err := mac.VerifySignature(rootKey, nil)
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(mac.Id()))
	macaroonId := &mintMacaroon.MacaroonIdentifier{}
	if err = dec.Decode(macaroonId); err != nil {
		return err
	}
	if macaroonId.PaymentHash != preimage.Hash() {
		return fmt.Errorf("Invalid Preimage %v for PaymentHash %v", preimage, macaroonId.PaymentHash)
	}
	return err
}
