# Build stage
FROM golang:1.24.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Copy .env to container
COPY .env ./


# Statically build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main .

# Final slim image
FROM scratch

WORKDIR /app

# Copy only the static binary
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]

