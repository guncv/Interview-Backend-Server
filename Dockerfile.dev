# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache curl git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz \
    | tar xvz -C /app

# Build the application, excluding test files
RUN go build -o main cmd/server/main.go


# Dev runtime with Air
FROM golang:1.24-alpine AS dev

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY . .
COPY --from=builder /app/migrate ./migrate
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

COPY migration.sh .

RUN chmod +x /app/migration.sh


EXPOSE 8080

ENTRYPOINT ["/app/migration.sh"]
CMD ["/app/main"]
