# Stage 1: Builder
FROM golang:1.24.2-alpine AS builder

# Аргумент для inline кэша (передаётся через --build-arg)
ARG BUILDKIT_INLINE_CACHE=0

# Устанавливаем рабочую директорию
WORKDIR /app

# Сначала копируем файлы зависимостей, чтобы не пересобирать их при изменении исходного кода
COPY go.mod go.sum ./
RUN go mod download


# Копируем исходный код приложения и статические файлы
COPY cmd cmd

# Сборка приложения с использованием кэша Go (mount типа cache доступен при использовании BuildKit)
RUN --mount=type=cache,target=/gocache \
GOCACHE=/gocache \
GOOS=linux GOARCH=amd64 \
go build -ldflags="-w -s" -o ntfy2tg ./cmd/main/


# Stage 2: Final image
FROM bash:4.4.23

# Копируем собранный бинарник и статические файлы из стадии builder
COPY --from=builder /app/ntfy2tg /app/ntfy2tg

CMD ["./ntfy2tg"]
