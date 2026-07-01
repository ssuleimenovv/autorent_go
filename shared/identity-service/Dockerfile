FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build
RUN go build -o /app/shared/identity-service/identity-service ./shared/identity-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/shared/identity-service/identity-service .

EXPOSE 8080

CMD ["./identity-service"]
