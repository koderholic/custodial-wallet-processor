FROM golang:1.13.4

COPY ./ /go/src/wallet-adapter
WORKDIR /go/src/wallet-adapter

COPY go.mod go.sum ./
RUN echo "appPort: 8200" >> config.yaml && \
    echo "serviceName : wallet-adapter" >> config.yaml && \
    echo "purgeCacheInterval: 5" >> config.yaml 


# RUN go get -d -v ./...
RUN go mod download

RUN go build -o walletAdapter
ENTRYPOINT ["./walletAdapter"]
EXPOSE 8002
