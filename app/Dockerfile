FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY app/main.go ./
RUN CGO_ENABLED=0 go build -o api .

FROM scratch
COPY --from=builder /app/api /api
ENTRYPOINT ["/api"]
