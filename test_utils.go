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
	TEST_MACAROON_VALID = "AgEETFNBVALnAUr/gQMBARJNYWNhcm9vbklkZW50aWZpZXIB/4IAAQMBB1ZlcnNpb24BBgABC1BheW1lbnRIYXNoAf+EAAEHVG9rZW5JZAH/hgAAABT/gwEBAQRIYXNoAf+EAAEGAUAAABn/hQEBAQlbMzJddWludDgB/4YAAQYBQAAAbP+CAiBo/5j/kXn/oE0p/4z/ki8cQ13/gB0UHXX/jQ7/uP+JOv+g/+dv/8d1//QT/+hMASD/xv/8/5v/5//8/9FZ//f/nf///9IMC/+7/+n/jv+v/8X/8P+3/5j/pv/sVGhj//lY/5J7/7EMAAACJ0Jhc2VVUkw9aHR0cDovL2xvY2FsaG9zdDo4MDgwL3Byb3RlY3RlZAACFlJlcXVlc3RQYXRoPS9wcm90ZWN0ZWQAAAYgpMKU7hwkkT7HmksJIp01g9oS9XM/hxXRfjeK6ahUvX8="
	TEST_PREIMAGE_VALID = "3965343963626361373962353265376339646632623534373234343163623766"

	TEST_MACAROON_WITHOUT_CAVEATS          = "AgEETFNBVALmAUr/gQMBARJNYWNhcm9vbklkZW50aWZpZXIB/4IAAQMBB1ZlcnNpb24BBgABC1BheW1lbnRIYXNoAf+EAAEHVG9rZW5JZAH/hgAAABT/gwEBAQRIYXNoAf+EAAEGAUAAABn/hQEBAQlbMzJddWludDgB/4YAAQYBQAAAa/+CAiD/pv/jOjY1/9oC/4z/tHb/qf/2Jf+d/4H/u/+YGHj/+/+O/8D/v/+P/8X/qRL/5v/x/4r/tkIBIA1Y/8j/pR3/0P+b/7cwWP+W/87/sD18GP//Hf/f/9Aj//NcBFs2/9VhNEUF/70AAAAGIDlR1jVm5IfEJgvuSQoJLqLg4FcW4Ib1vW8sbkRHdUWX"
	TEST_MACAROON_WITHOUT_CAVEATS_PREIMAGE = "651505fae9ea341c770c6ebef207d8560d546eb3aee26985e584c15d1c987875"

	TEST_PREIMAGE_INVALID = "fbe9ac25c04e14b10177514e2d57b0e39224e70277ac1a2cd23c28e58cd4ea35"

	ROOT_KEY = "ABDEGHKLMPTC"
)

const BASE_URL = "BaseURL"

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
