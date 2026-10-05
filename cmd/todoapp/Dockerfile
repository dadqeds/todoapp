FROM golang:1.25.1-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/cmd/todoapp/exe /app/cmd/todoapp

FROM alpine:3.23
# Постоянный uid: на сервере папка логов монтируется с хоста, и make server-up
# отдаёт её этому пользователю (на Linux права проверяются, в Docker Desktop — нет).
RUN addgroup -S -g 10001 todoapp && adduser -S -u 10001 -G todoapp todoapp \
    && mkdir -p /app/out/logs && chown -R todoapp:todoapp /app
WORKDIR /app
COPY --from=builder /app/cmd/todoapp/exe /app/exe
USER todoapp
CMD ["/app/exe"]
