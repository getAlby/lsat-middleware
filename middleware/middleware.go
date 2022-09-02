package middleware

import (
	"net/http"

	"github.com/getAlby/lsat-middleware/caveat"
	"github.com/getAlby/lsat-middleware/ln"
)

type amountFunc func(*http.Request) int64
type caveatFunc func(*http.Request) []caveat.Caveat
type LsatMiddleware struct {
	AmountFunc amountFunc
	LNClient   ln.LNClient
	CaveatFunc caveatFunc
	RootKey    []byte
}

func NewLsatMiddleware(lnClientConfig *ln.LNClientConfig,
	amountF amountFunc, caveatF caveatFunc) (*LsatMiddleware, error) {
	lnClient, err := ln.InitLnClient(lnClientConfig)
	if err != nil {
		return nil, err
	}
	middleware := &LsatMiddleware{
		AmountFunc: amountF,
		LNClient:   lnClient,
		CaveatFunc: caveatF,
		RootKey:    lnClientConfig.RootKey,
	}
	return middleware, nil
}
