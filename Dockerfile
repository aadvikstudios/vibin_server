# Use the official Go image for building
FROM golang:1.20 as builder

# Set the working directory
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o vibin-server main.go

# Use a minimal base image for running the application
FROM alpine:latest

# Set working directory
WORKDIR /root/

# Copy the compiled binary
COPY --from=builder /app/vibin-server .

# Expose the port the application will run on
EXPOSE 8080

# Command to run the executable
CMD ["./vibin-server"]