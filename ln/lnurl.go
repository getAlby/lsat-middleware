package ln

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/getAlby/lsat-middleware/utils"

	decodepay "github.com/fiatjaf/ln-decodepay"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
	"google.golang.org/grpc"
)

const MSAT_PER_SAT = 1000

type LNURLoptions struct {
	Address string
}

type LnAddressUrlResJson struct {
	Callback       string `json:"callback"`
	MaxSendable    uint64 `json:"maxSendable"`
	MinSendable    uint64 `json:"minSendable"`
	Metadata       string `json:"metadata"`
	CommentAllowed uint   `json:"commentAllowed"`
	Tag            string `json:"tag"`
}

type CallbackUrlResJson struct {
	PR string `json:"pr"`
}

type DecodedPR struct {
	Currency           string `json:"currency"`
	CreatedAt          int    `json:"created_at"`
	Expiry             int    `json:"expiry"`
	Payee              string `json:"payee"`
	MSatoshi           int64  `json:"msatoshi"`
	Description        string `json:"description,omitempty"`
	DescriptionHash    string `json:"description_hash,omitempty"`
	PaymentHash        string `json:"payment_hash"`
	MinFinalCLTVExpiry int    `json:"min_final_cltv_expiry"`
}

func NewLNURLClient(lnurlOptions LNURLoptions) (*LnAddressUrlResJson, error) {
	username, domain, err := utils.ParseLnAddress(lnurlOptions.Address)
	if err != nil {
		return nil, err
	}
	lnAddressUrl := fmt.Sprintf("https://%s/.well-known/lnurlp/%s", domain, username)
	lnAddressUrlResBody, err := DoGetRequest(lnAddressUrl)
	if err != nil {
		return nil, err
	}
	lnAddressUrlRes := &LnAddressUrlResJson{}
	if err := json.Unmarshal(lnAddressUrlResBody, lnAddressUrlRes); err != nil {
		return nil, err
	}
	return lnAddressUrlRes, nil
}

func (lnAddressUrlResJson *LnAddressUrlResJson) AddInvoice(ctx context.Context, lnInvoice *lnrpc.Invoice, httpReq *http.Request, options ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	callbackUrl := fmt.Sprintf("%s?amount=%d", lnAddressUrlResJson.Callback, MSAT_PER_SAT*lnInvoice.Value)
	callbackUrlResBody, err := DoGetRequest(callbackUrl)
	if err != nil {
		return nil, err
	}
	callbackUrlResJson := &CallbackUrlResJson{}
	if err := json.Unmarshal(callbackUrlResBody, callbackUrlResJson); err != nil {
		return nil, err
	}

	invoice := callbackUrlResJson.PR
	decoded, err := decodepay.Decodepay(invoice)
	if err != nil {
		return nil, err
	}
	paymentHash, err := lntypes.MakeHashFromStr(decoded.PaymentHash)
	if err != nil {
		return nil, err
	}
	return &lnrpc.AddInvoiceResponse{
		RHash:          paymentHash[:],
		PaymentRequest: invoice,
	}, nil
}

func DoGetRequest(Url string) ([]byte, error) {
	res, err := http.Get(Url)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()

	return ioutil.ReadAll(res.Body)
}
