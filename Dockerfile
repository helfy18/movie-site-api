# Step 1: Build stage
FROM golang:1.26-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the working directory inside the container
COPY . .

# Build the Go app as a static binary so it runs on plain alpine
RUN CGO_ENABLED=0 go build -o main .

# Step 2: Run stage
FROM alpine:latest

# Run as an unprivileged user
RUN adduser -D -H apiuser
USER apiuser

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the build stage
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
