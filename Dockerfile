# Stage 1: Build the Go binary
FROM --platform=$BUILDPLATFORM golang:latest AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the source code
COPY . .

# List everything in the working directory
RUN ls -laR

# Install the dependencies
RUN go mod tidy

# Set build arguments for target architecture
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o simpleton-server ./cmd/simpleton

# Stage 2: Create the final minimal image
FROM busybox:latest

# Copy necessary system files for timezone and DNS resolution
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/resolv.conf /etc/resolv.conf

# Copy the built Go binary
COPY --from=builder /app/simpleton-server /simpleton-server

# Set the entrypoint
ENTRYPOINT ["/simpleton-server"]

# Set the content directory
CMD ["/www"]