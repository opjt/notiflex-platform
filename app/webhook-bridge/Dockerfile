FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY app/webhook-bridge/main.go ./
RUN CGO_ENABLED=0 go build -o bridge .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/bridge /bridge
ENTRYPOINT ["/bridge"]
