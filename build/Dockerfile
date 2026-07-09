# --- Сборка ---
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Кэшируем зависимости отдельным слоем.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/subscriptions ./cmd/subscriptions

# --- Рантайм ---
FROM alpine:3.20

RUN adduser -D -u 10001 appuser
USER appuser

WORKDIR /app
COPY --from=builder /out/subscriptions .
COPY config.yaml .
COPY env/migrations ./env/migrations

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["./subscriptions"]
