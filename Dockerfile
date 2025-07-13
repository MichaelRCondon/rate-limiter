#BUILDER 
FROM golang:1.24.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY application_config.json .
COPY config/ ./config/
COPY ratelimiter/ ./ratelimiter
COPY types/ ./types/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rate-limiter .

#RUNNER
FROM alpine:latest

 # We'll want HTTPS eventually
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/rate-limiter .
COPY --from=builder /app/application_config.json .

EXPOSE 8080

CMD ["./rate-limiter"]