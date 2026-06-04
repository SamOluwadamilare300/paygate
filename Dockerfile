# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /go/src/github.com/moov-io/paygate

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GONOSUMDB=*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/paygate ./cmd/server/

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/paygate /app/paygate
COPY --from=builder /go/src/github.com/moov-io/paygate/examples/config.yaml /app/examples/config.yaml

EXPOSE 8082 9092

ENTRYPOINT ["/app/paygate"]