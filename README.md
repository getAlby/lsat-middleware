# Gin-LSAT

A middleware for [Gin](https://github.com/gin-gonic/gin) framework that uses [LSAT](https://lsat.tech/) (a protocol standard for authentication and paid APIs) and provides handler functions to accept microtransactions before serving ad-free content or any paid APIs.

## Installation

Assuming you've installed Go and Gin 

1. Run this:

```
go get github.com/DhananjayPurohit/gin-lsat
```

2. Create `.env` file (refer `.env_example`) and configure `LND_ADDRESS` and `MACAROON_HEX` for LND client or `LNURL_ADDRESS` for LNURL client, `LN_CLIENT_TYPE` (out of LND, LNURL) and `ROOT_KEY` (for minting macaroons).  

## Usage

[This example](https://github.com/DhananjayPurohit/gin-lsat/blob/middleware-test/examples/main.go) shows how to use Gin-LSAT.

## Testing

Run `go test` to run tests.
