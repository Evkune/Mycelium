# Build Stage
FROM golang:1.23-alpine3.20 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies and create the vendor directory
RUN go mod vendor

# Copy the source code and vendor directory
COPY . .

# Build the Go app with the vendor directory
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o main .

# Final Stage: Use distroless as the base image
FROM gcr.io/distroless/static:nonroot

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/main /

# Expose port 8080
EXPOSE 8080

# Command to run the executable
CMD ["/main"]