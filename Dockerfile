# Stage 1: Build the Go application
FROM golang:1.20-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Install git to allow fetching dependencies (if required)
RUN apk add --no-cache git

# Copy the source code from your local machine to the container
COPY . .

# Download dependencies and build the Go app
RUN go mod tidy
RUN go build -o boltdbweb .

# Stage 2: Create a minimal container and copy the binary
FROM alpine:3.18

RUN apk --no-cache add curl

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/boltdbweb .

# Expose port 80
#EXPOSE 80

# Run the binary with environment variables as arguments
# TODO make args configurable
# CMD ["sh", "-c", "./boltdbweb -d /data/model-store.db -p 8080"]
# docker run -p 8080:8080 -v /catalogexplorer:/data boltdbweb-app
# docker build -t boltdbweb-app . --platform linux/amd64
# docker tag boltdbweb-app catalog-explorer:latest
# docker push catalog-explorer:latest       