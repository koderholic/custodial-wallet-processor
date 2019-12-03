FROM golang:latest

COPY ./ /go/src/wallet-adapter
WORKDIR /go/src/wallet-adapter

WORKDIR ./
COPY go.mod go.sum ./

# RUN go get -d -v ./...
RUN go mod download

RUN go build -o walletAdapter
ENTRYPOINT ["./walletAdapter"]
EXPOSE 8002
