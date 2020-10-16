FROM golang:latest AS builder

WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download && mkdir /build

COPY ./ /src
RUN go build -o /build/service
RUN go build -o /build/float_manager cronjobs/float_manager/entry.go
RUN go build -o /build/sweep_job cronjobs/sweep_job/entry.go
RUN go get -u github.com/kisielk/errcheck && go get github.com/golangci/govet
RUN /go/bin/errcheck -verbose -exclude /src/checkIgnore ./... && go vet ./...


FROM debian:latest
COPY --from=builder /build /app/bin
COPY --from=builder /src/migration /app/bin/migration
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
    echo "coldWalletEmail: finance@bundle.africa" >> config.yaml && \
    echo "rateServiceUrl: http://rates" >> config.yaml && \
    echo "TWServiceURL: https://raw.githubusercontent.com/trustwallet" >> config.yaml && \
    echo "coldWalletEmailTemplateId: d-c2c966c47fc3405598733a6a7178b28f" >> config.yaml && \
    echo "expireCacheDuration: 400" >> config.yaml && \
    echo "requestTimeout: 60" >> config.yaml && \
    echo "maxIdleConns : 25" >> config.yaml && \
    echo "maxOpenConns : 50" >> config.yaml && \
    echo "connMaxLifetime: 300" >> config.yaml && \
    echo "floatPercentage: 10" >> config.yaml && \
    echo "enableFloatManager : true"  >> config.yaml && \
    echo "dbMigrationPath : ./migration" >> config.yaml && \
    echo "sweepCronInterval: 1/15 * * * *" >> config.yaml && \
    echo "floatCronInterval: 10 */3 * * *" >> config.yaml && \
    echo "coldWalletSmsNumber: +2348178500655" >> config.yaml && \
    echo "SENTRY_DSN: https://52fb6b65fcdf4fd89143d81611f7a12c@sentry.io/3640925" >> config.yaml
