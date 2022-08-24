package lsat

import (
	"fmt"

	macaroonutils "github.com/getAlby/lsat-middleware/macaroon"

	"github.com/lightningnetwork/lnd/lntypes"
	"gopkg.in/macaroon.v2"
)

func VerifyLSAT(mac *macaroon.Macaroon, rootKey []byte, preimage lntypes.Preimage) error {
	_, err := mac.VerifySignature(rootKey, nil)
	if err != nil {
		return err
	}
	macaroonId, err := macaroonutils.GetMacIdFromMacaroon(mac)
	if err != nil {
		return err
	}
	if macaroonId.PaymentHash != preimage.Hash() {
		return fmt.Errorf("Invalid Preimage %s for PaymentHash %s", preimage, macaroonId.PaymentHash)
	}
	return nil
}
