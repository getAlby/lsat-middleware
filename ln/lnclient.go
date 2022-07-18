package ln

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
	"google.golang.org/grpc"
)

type LNClientConfig struct {
	LNClientType string
	LNDConfig    LNDoptions
	LNURLConfig  LNURLoptions
}
type LNClient interface {
	AddInvoice(ctx context.Context, req *lnrpc.Invoice, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error)
}

type LNClientConn struct {
	LNClient LNClient
}

func (lnClientConn *LNClientConn) GenerateInvoice(ctx context.Context, lnInvoice lnrpc.Invoice) (string, lntypes.Hash, error) {
	lnClientInvoice, err := lnClientConn.LNClient.AddInvoice(ctx, &lnInvoice)
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
