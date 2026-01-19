# Multi-stage build for giggum application
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod ./
COPY go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o giggum .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    bash \
    git

# Install Go toolchain for giggum runtime
RUN apk add --no-cache go

# Install Go linters
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
RUN go install honnef.co/go/tools/cmd/staticcheck@latest
RUN go install golang.org/x/tools/cmd/goimports@latest

# Install Node.js for opencode if needed
RUN apk add --no-cache nodejs npm

# Install opencode CLI
RUN npm install -g opencode-ai

# Create app directory
WORKDIR /src

# Copy the binary from builder stage
COPY --from=builder /build/giggum /usr/local/bin/giggum

# Set up entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create non-root user
RUN adduser -D -s /bin/bash giggum
RUN chown -R giggum:giggum /src
USER giggum

# Set environment variables
ENV OPENAI_BASE_URL=http://100.83.162.29:1234
ENV GOPATH=/home/giggum/go
ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

# Volume mounting point
VOLUME ["/src"]

# Expose any needed ports (if webhook server is used)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["giggum"]