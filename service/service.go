package service

import (
	"context"
	ln "proxy/lnd"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
)

type Service struct {
	LnClient ln.LNClient
}

func (svc *Service) GenerateInvoice(ctx context.Context, lnInvoice lnrpc.Invoice) (string, lntypes.Hash, error) {
	lndInvoice, err := svc.LnClient.AddInvoice(ctx, &lnInvoice)
	if err != nil {
		return "", lntypes.Hash{}, err
	}

	invoice := lndInvoice.PaymentRequest
	paymentHash, err := lntypes.MakeHash(lndInvoice.RHash)
	if err != nil {
		return invoice, lntypes.Hash{}, err
	}
	return invoice, paymentHash, nil
}
