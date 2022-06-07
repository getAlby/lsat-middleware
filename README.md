# LSAT-Go-proxy

## A proxy server implementation in Go to support LSAT protocol.

### Steps to run:-

1. Clone the repo.
2. In the repo folder, run the command 
    ```shell
    $ go get .
    ```
3. Create `.env` file (refer `.env_example`) and configure `LND_ADDRESS` and `MACAROON_HEX`.
4. To start the server, run
    ```shell
    $ go run .
    ```
5. To test the endpoint send a GET request to `http://localhost:8080/protected` through browser, you will observe a response ```402 Payment Required```.
