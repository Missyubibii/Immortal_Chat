# --- Stage 1: Build ---
FROM golang:1.22-alpine AS builder

# Cai dat git
RUN apk add --no-cache git

WORKDIR /app

# Copy file thu vien
COPY go.mod go.sum ./
RUN go mod download

# Copy code va build
COPY . .
RUN go build -o main ./cmd/server/main.go

# --- Stage 2: Run ---
FROM alpine:latest

WORKDIR /app

# Copy binary tu Stage 1
COPY --from=builder /app/main .
COPY .env .
# Copy folder web neu can
COPY web ./web

# Mo cong 8080
EXPOSE 8080

# Lenh chay mac dinh
CMD ["./main"]
