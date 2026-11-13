FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make
WORKDIR /app
COPY go.mod go.sum ./

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/app \
    ./cmd/app/main.go




FROM alpine:latest AS runtime
RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
WORKDIR /app

COPY --from=builder /app/.env /app/.env
COPY --from=builder /app/bin/app /app/app
COPY --from=builder /app/config /app/config

RUN chown -R appuser:appuser /app
USER appuser

FROM runtime AS app
EXPOSE 8080
ENTRYPOINT ["/app/app"]
