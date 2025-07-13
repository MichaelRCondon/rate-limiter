# BUILD executable
FROM golang:1.24.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
COPY hello/ ./hello/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runner stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]