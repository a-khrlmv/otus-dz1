# syntax=docker/dockerfile:1

# ==========================================
# Stage 1: Build
# ==========================================

FROM golang:1.26-alpine3.24 AS builder

WORKDIR /app

# Копируем зависимости отдельно для кеширования
COPY go.mod go.sum ./

RUN go mod download

# Копируем исходный код приложения
COPY . .

# Компилируем статический Go-бинарник
RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /app/bin/social-api \
    ./cmd/api

# ==========================================
# Stage 2: Runtime
# ==========================================

FROM alpine:3.24

# Сертификаты для исходящих HTTPS-соединений
RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S -G appgroup appuser

WORKDIR /app

# Копируем только бинарный файл
COPY --from=builder --chown=appuser:appgroup \
    /app/bin/social-api /app/social-api

# Запуск без root-привилегий
USER appuser

EXPOSE 8080

# Запуск Go-приложения
ENTRYPOINT ["/app/social-api"]
