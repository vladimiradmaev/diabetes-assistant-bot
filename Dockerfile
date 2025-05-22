FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o diabetes-helper

# Use a smaller image for the final container
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/diabetes-helper .

# Copy .env file
COPY .env .

# Expose port if needed
# EXPOSE 8080

# Run the application
CMD ["./diabetes-helper"] 