package lsat

import (
	"fmt"

	"github.com/getAlby/lsat-middleware/caveat"
	macaroonutils "github.com/getAlby/lsat-middleware/macaroon"

	"github.com/lightningnetwork/lnd/lntypes"
	"gopkg.in/macaroon.v2"
)

func VerifyLSAT(mac *macaroon.Macaroon, rootKey []byte, conditions []caveat.Caveat, preimage lntypes.Preimage) error {
	rawCaveats, err := mac.VerifySignature(rootKey, nil)
	if err != nil {
		return err
	}
	if err := caveat.VerifyCaveats(rawCaveats, conditions); err != nil {
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
