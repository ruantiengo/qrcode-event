# Use the official Golang image as the build environment
FROM golang:1.18 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o main .

# Start a new stage from scratch
FROM debian:bullseye-slim

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main /app/main

# Expose port 4050 to the outside world
EXPOSE 4050

# Command to run the executable
ENTRYPOINT ["/app/main"]
