package middleware

import (
	"net/http"

	"github.com/getAlby/lsat-middleware/caveat"
	"github.com/getAlby/lsat-middleware/ln"
)

const REQUEST_PATH = "RequestPath"

type LsatMiddleware struct {
	AmountFunc func(req *http.Request) (amount int64)
	LNClient   ln.LNClient
	Caveats    []caveat.Caveat
	RootKey    []byte
}

func NewLsatMiddleware(lnClientConfig *ln.LNClientConfig,
	amountFunc func(req *http.Request) (amount int64)) (*LsatMiddleware, error) {
	lnClient, err := ln.InitLnClient(lnClientConfig)
	if err != nil {
		return nil, err
	}
	middleware := &LsatMiddleware{
		AmountFunc: amountFunc,
		LNClient:   lnClient,
		Caveats:    lnClientConfig.Caveats,
		RootKey:    lnClientConfig.RootKey,
	}
	return middleware, nil
}
