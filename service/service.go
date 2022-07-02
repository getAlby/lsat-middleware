package service

import (
	"proxy/lnd"
	"proxy/lnurl"
)

type Service struct {
	LndClient   *lnd.LNDWrapper
	LnurlClient *lnurl.LNURLWrapper
}
