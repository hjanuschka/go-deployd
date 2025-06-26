# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.23-alpine AS build

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/deployd cmd/deployd/main.go

# Production stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /app/deployd .

# Copy the resources and .deployd directories
COPY resources ./resources
COPY .deployd ./.deployd

# Expose the port
EXPOSE 2403

# Run the application
CMD ["./deployd"]
