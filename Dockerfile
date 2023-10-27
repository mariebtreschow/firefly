# Use the Go 1.20 image to create a build artifact.
FROM golang:1.20 AS builder

# Set working directory inside the container
WORKDIR /go/src/firefly.ai/word-count

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/word-count

# Use a minimal alpine image for the final stage
FROM alpine:3.18.4


# Copy only the compiled go binary from the build stage
COPY --from=builder /go/src/firefly.ai/word-count/main .

# Make the binary executable
RUN chmod +x main

# Run the binary
ENTRYPOINT ["./main"]