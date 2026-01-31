# Multi-stage build for small image
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest
RUN apk add --no-cache libc6-compat
WORKDIR /app
COPY --from=builder /app/server .
COPY templates/ templates/
COPY static/ static/
VOLUME /data
EXPOSE 8080
ENV DATA_DIR=/data
CMD ["./server"]
