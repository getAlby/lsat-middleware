package lnurl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"proxy/utils"

	decodepay "github.com/fiatjaf/ln-decodepay"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
)

type LNURLWrapper struct {
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

func (wrapper *LNURLWrapper) AddInvoice(lnInvoice *lnrpc.Invoice) (string, lntypes.Hash, error) {
	username, domain := utils.ParseLnAddress(wrapper.Address)
	lnAddressUrl := fmt.Sprintf("https://%v/.well-known/lnurlp/%v", domain, username)
	lnAddressUrlResBody, err := MakeGetRequest(lnAddressUrl)
	if err != nil {
		return "", lntypes.Hash{}, err
	}
	lnAddressUrlResJson := &LnAddressUrlResJson{}
	if err := json.Unmarshal(lnAddressUrlResBody, &lnAddressUrlResJson); err != nil {
		return "", lntypes.Hash{}, err
	}

	callbackUrl := fmt.Sprintf("%v?amount=%v", lnAddressUrlResJson.Callback, 1000*lnInvoice.Value)
	callbackUrlResBody, err := MakeGetRequest(callbackUrl)
	if err != nil {
		return "", lntypes.Hash{}, err
	}
	callbackUrlResJson := &CallbackUrlResJson{}
	if err := json.Unmarshal(callbackUrlResBody, &callbackUrlResJson); err != nil {
		return "", lntypes.Hash{}, err
	}

	invoice := callbackUrlResJson.PR
	decoded, err := decodepay.Decodepay(invoice)
	if err != nil {
		return "", lntypes.Hash{}, err
	}

	paymentHash, err := lntypes.MakeHashFromStr(decoded.PaymentHash)
	return invoice, paymentHash, nil
}

func MakeGetRequest(Url string) ([]byte, error) {
	res, err := http.Get(Url)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}
	return resBody, nil
}
