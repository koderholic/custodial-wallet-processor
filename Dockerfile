FROM golang:latest AS builder

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY ./ /build
RUN go build -o /build/service

FROM debian:latest
RUN mkdir -p /app/bin
COPY --from=builder /build/service /app/bin/service
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /app/bin/

ENTRYPOINT ["/app/bin/service"]
EXPOSE 8002

## Build Config
RUN echo "appPort: 8200" >> config.yaml && \
    echo "serviceName : crypto-wallet-adapter" >> config.yaml && \
    echo "purgeCacheInterval: 5" >> config.yaml && \
    echo "authenticationServiceURL: http://authentication" >> config.yaml && \
    echo "cryptoAdapterServiceURL: http://crypto-adapter" >> config.yaml && \
    echo "keyManagementServiceURL: http://key-management" >> config.yaml && \
    echo "lockerServiceURL: http://locker" >> config.yaml && \
    echo "lockerServicePrefix : Wallet-Adapter-Lock-" >> config.yaml && \
    echo "depositWebhookURL: http://crypto-adapter/incoming-deposit" >> config.yaml && \
    echo "BTC_SLIP_VALUE: 0" >> config.yaml && \
    echo "BNB_SLIP_VALUE: 714" >> config.yaml && \
    echo "expireCacheDuration: 400" >> config.yaml && \
    echo "ETH_SLIP_VALUE: 60" >> config.yaml
