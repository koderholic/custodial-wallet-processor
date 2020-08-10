FROM golang:latest AS builder

WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download && mkdir /build

COPY ./ /src
RUN go build -o /build/service
RUN go build -o /build/float_manager cronjobs/float_manager/entry.go
RUN go build -o /build/sweep_job cronjobs/sweep_job/entry.go


FROM debian:latest
COPY --from=builder /build /app/bin
COPY --from=builder /build/config.yml /app/bin/config.yml
COPY --from=builder /src/migration /app/bin/migration
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /app/bin/

ENTRYPOINT ["/app/bin/service"]
EXPOSE 8002