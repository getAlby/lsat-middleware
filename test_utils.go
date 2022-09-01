package test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

const SATS_PER_BTC = 100000000

const MIN_SATS_TO_BE_PAID = 1

const (
	TEST_MACAROON_VALID = "AgEETFNBVALgAUr/gQMBARJNYWNhcm9vbklkZW50aWZpZXIB/4IAAQMBB1ZlcnNpb24BBgABC1BheW1lbnRIYXNoAf+EAAEHVG9rZW5JZAH/hgAAABT/gwEBAQRIYXNoAf+EAAEGAUAAABn/hQEBAQlbMzJddWludDgB/4YAAQYBQAAAZf+CAiD/xkv/wP/aSkFQ/8//6f/lEWcMcE0NEAlSRGIkbX//6f+cAQr/2P+tSP+1ASD/0GQ0/5wELh//xP/gEv+O/65W//T/5f/5/5z/qP+/NBgcGf+W/+D/gf/iBf+iRv/W//MAAAIkUGF0aD1odHRwOi8vbG9jYWxob3N0OjgwODAvcHJvdGVjdGVkAAAGIM/NOETVID4fIo7y+mh6sutgLWf3GmtOb6is2rmfW5J5"
	TEST_PREIMAGE_VALID = "3532373238323137333333303162316638633138616331393866656333326332"

	TEST_MACAROON_WITHOUT_CAVEATS          = "AgEETFNBVALmAUr/gQMBARJNYWNhcm9vbklkZW50aWZpZXIB/4IAAQMBB1ZlcnNpb24BBgABC1BheW1lbnRIYXNoAf+EAAEHVG9rZW5JZAH/hgAAABT/gwEBAQRIYXNoAf+EAAEGAUAAABn/hQEBAQlbMzJddWludDgB/4YAAQYBQAAAa/+CAiD/pv/jOjY1/9oC/4z/tHb/qf/2Jf+d/4H/u/+YGHj/+/+O/8D/v/+P/8X/qRL/5v/x/4r/tkIBIA1Y/8j/pR3/0P+b/7cwWP+W/87/sD18GP//Hf/f/9Aj//NcBFs2/9VhNEUF/70AAAAGIDlR1jVm5IfEJgvuSQoJLqLg4FcW4Ib1vW8sbkRHdUWX"
	TEST_MACAROON_WITHOUT_CAVEATS_PREIMAGE = "651505fae9ea341c770c6ebef207d8560d546eb3aee26985e584c15d1c987875"

	TEST_PREIMAGE_INVALID = "fbe9ac25c04e14b10177514e2d57b0e39224e70277ac1a2cd23c28e58cd4ea35"

	ROOT_KEY = "ABDEGHKLMPTC"
)

const Path = "Path"

func FiatToBTC(currency string, value float64) *http.Request {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://blockchain.info/tobtc?currency=%s&value=%f", currency, value), nil)
	if err != nil {
		return nil
	}
	return req
}

type FiatRateConfig struct {
	Currency string
	Amount   float64
}

func (fr *FiatRateConfig) FiatToBTCAmountFunc(req *http.Request) (amount int64) {
	if req == nil {
		return MIN_SATS_TO_BE_PAID
	}
	res, err := http.Get(fmt.Sprintf("https://blockchain.info/tobtc?currency=%s&value=%f", fr.Currency, fr.Amount))
	if err != nil {
		return MIN_SATS_TO_BE_PAID
	}
	defer res.Body.Close()

	amountBits, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return MIN_SATS_TO_BE_PAID
	}
	amountInBTC, err := strconv.ParseFloat(string(amountBits), 32)
	if err != nil {
		return MIN_SATS_TO_BE_PAID
	}
	amountInSats := SATS_PER_BTC * amountInBTC
	return int64(amountInSats)
}
