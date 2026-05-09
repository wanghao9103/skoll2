FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM alpine:3.20

RUN adduser -D -u 10001 appuser
WORKDIR /app

COPY --from=builder /out/server /app/server
RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

ENV SERVER_ADDR=:8080
ENV DB_DRIVER=sqlite
ENV DB_DSN=/app/data/skoll2.db

ENTRYPOINT ["/app/server"]
