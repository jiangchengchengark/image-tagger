# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY config.yaml .
COPY web/auto-label/build ./web

# Install required tools
RUN apk add --no-cache bash

EXPOSE 6060
CMD ["./main"]
