package invoice

import (
	"context"
	"errors"
	"proxy/service"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
)

func GenerateInvoice(svc *service.Service, ctx context.Context, lnInvoice lnrpc.Invoice) (string, lntypes.Hash, error) {
	if svc.LndClient != nil {
		lndInvoice, err := svc.LndClient.AddInvoice(ctx, &lnInvoice)
		if err != nil {
			return "", lntypes.Hash{}, err
		}

		invoice := lndInvoice.PaymentRequest
		paymentHash, err := lntypes.MakeHash(lndInvoice.RHash)
		if err != nil {
			return invoice, lntypes.Hash{}, err
		}
		return invoice, paymentHash, nil
	} else if svc.LnurlClient != nil {
		invoice, paymentHash, err := svc.LnurlClient.AddInvoice(&lnInvoice)
		if err != nil {
			return "", lntypes.Hash{}, err
		}
		return invoice, paymentHash, nil
	}
	return "", lntypes.Hash{}, errors.New("Client not configured to generate invoice")
}
