# Stage 1: Build the Go application
FROM golang:1.24.3 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Download the dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the source code into the container
COPY ./src .

# Build the Go application with cache mounts
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o automate-home .

FROM chromedp/headless-shell:131.0.6724.0 AS final

WORKDIR /
COPY --from=builder /app/automate-home .

EXPOSE 8080
ENTRYPOINT ["/automate-home"]
