# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /go/src/github.com/moov-io/paygate

# No gcc needed - using modernc.org/sqlite (pure Go)
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GONOSUMDB=*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/paygate ./cmd/server/

# Final stage - minimal image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/paygate /app/paygate
COPY --from=builder /go/src/github.com/moov-io/paygate/examples/config.yaml /app/examples/config.yaml

EXPOSE 8082 9092

ENTRYPOINT ["/app/paygate"]