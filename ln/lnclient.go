package ln

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getAlby/lsat-middleware/caveat"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
	"google.golang.org/grpc"
)

const (
	LND_CLIENT_TYPE   = "LND"
	LNURL_CLIENT_TYPE = "LNURL"
)

type LNClientConfig struct {
	LNClientType string
	LNDConfig    LNDoptions
	LNURLConfig  LNURLoptions
	Caveats      []caveat.Caveat
	RootKey      []byte
}
type LNClient interface {
	AddInvoice(ctx context.Context, lnReq *lnrpc.Invoice, httpReq *http.Request, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error)
}

type LNClientConn struct {
	LNClient LNClient
}

func InitLnClient(lnClientConfig *LNClientConfig) (lnClient LNClient, err error) {
	switch lnClientConfig.LNClientType {
	case LND_CLIENT_TYPE:
		lnClient, err = NewLNDclient(lnClientConfig.LNDConfig)
		if err != nil {
			return lnClient, fmt.Errorf("Error initializing LN client: %s", err.Error())
		}
	case LNURL_CLIENT_TYPE:
		lnClient, err = NewLNURLClient(lnClientConfig.LNURLConfig)
		if err != nil {
			return lnClient, fmt.Errorf("Error initializing LN client: %s", err.Error())
		}
	default:
		return lnClient, fmt.Errorf("LN Client type not recognized: %s", lnClientConfig.LNClientType)
	}

	return lnClient, nil
}

func (lnClientConn *LNClientConn) GenerateInvoice(ctx context.Context, lnInvoice lnrpc.Invoice, httpReq *http.Request) (string, lntypes.Hash, error) {
	lnClientInvoice, err := lnClientConn.LNClient.AddInvoice(ctx, &lnInvoice, httpReq)
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
