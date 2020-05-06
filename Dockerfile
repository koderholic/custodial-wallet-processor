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
COPY --from=builder /build/migration /app/bin/migration
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
    echo "withdrawToHotWalletUrl: http://order-book" >> config.yaml && \
    echo "notificationServiceUrl: http://notifications" >> config.yaml && \
    echo "coldWalletEmail: akinyemi@bundle.africa" >> config.yaml && \
    echo "coldWalletEmailTemplateId: d-c2c966c47fc3405598733a6a7178b28f" >> config.yaml && \
    echo "BTC_SLIP_VALUE: 0" >> config.yaml && \
    echo "BNB_SLIP_VALUE: 714" >> config.yaml && \
    echo "expireCacheDuration: 400" >> config.yaml && \
    echo "requestTimeout: 60" >> config.yaml && \
    echo "ETH_SLIP_VALUE: 60" >> config.yaml && \
    echo "maxIdleConns : 25" >> config.yaml && \
    echo "maxOpenConns : 50" >> config.yaml && \
    echo "connMaxLifetime: 300" >> config.yaml && \
    echo "floatPercentage: 10" >> config.yaml && \
    echo "dbMigrationPath : ./migration" >> config.yaml && \
    echo "sweepCronInterval: 1/30 * * * *" >> config.yaml && \
    echo "floatCronInterval: 1/30 * * * *" >> config.yaml && \
    echo "sweepFeePercentageThreshold: 2" >> config.yaml && \
    echo "SENTRY_DSN: https://52fb6b65fcdf4fd89143d81611f7a12c@sentry.io/3640925" >> config.yaml
