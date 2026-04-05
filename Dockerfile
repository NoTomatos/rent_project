FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код
COPY . .

RUN go build -o gorrent cmd/api/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/gorrent .

EXPOSE 8080

CMD ["./gorrent"]