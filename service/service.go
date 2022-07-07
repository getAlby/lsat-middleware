package service

import (
	"context"
	"proxy/ln"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
)

type Service struct {
	LnClient ln.LNClient
}

func (svc *Service) GenerateInvoice(ctx context.Context, lnInvoice lnrpc.Invoice) (string, lntypes.Hash, error) {
	lnClientInvoice, err := svc.LnClient.AddInvoice(ctx, &lnInvoice)
	if err != nil {
		return "", lntypes.Hash{}, err
	}

	invoice := lnClientInvoice.PaymentRequest
	paymentHash, err := lntypes.MakeHash(lnClientInvoice.RHash)
	if err != nil {
		return invoice, lntypes.Hash{}, err
	}
	return invoice, paymentHash, nil
}
