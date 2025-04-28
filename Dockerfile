# Этап сборки
FROM golang:1.22-alpine AS builder

# Устанавливаем необходимые инструменты
RUN apk add --no-cache git

WORKDIR /app

# Копируем все файлы проекта
COPY . .


# Устанавливаем переменную окружения для использования локальной версии Go
ENV GOTOOLCHAIN=auto

# Генерируем Swagger документацию
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g cmd/main.go

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/main.go

# Финальный этап
FROM alpine:latest

# Устанавливаем необходимые пакеты
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Копируем бинарный файл из этапа сборки
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/.env .
# Создаем непривилегированного пользователя
RUN adduser -D -g '' appuser
USER appuser

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./main"]
