
# Run the server locally
run:
  @go run cmd/simpleton/main.go test

# Sudo run the server locally using chroot
srun:
  @sudo go run cmd/simpleton/main.go -c test 

# Define default target architecture and operating system
default_target := "linux/amd64"
# all_targets := "linux/amd64,linux/arm64,linux/arm/v7"
all_targets := "linux/amd64,linux/arm64,linux/arm/v7,linux/arm/v6,linux/386,linux/ppc64le,linux/s390x,linux/riscv64"
tag := "kluzz/simpleton:latest"

# Rule to build the Docker image for a specific architecture
build target="linux/amd64":
    docker buildx create --use --name multiarch-builder || true
    docker buildx build \
        --platform {{target}} \
        --tag {{tag}} \
        --output type=docker \
        --progress plain \
        .

# Rule to build for multiple platforms at once
build-all:
    docker buildx create --use --name multiarch-builder || true
    docker buildx build \
        --platform {{all_targets}} \
        --tag {{tag}} \
        --push \
        --progress plain \
        .

# Push the locally built image manually
push:
    docker push {{tag}}

# Clean up buildx instance
clean:
    docker buildx rm multiarch-builder || true

# Default rule to build for the default target
default:
    just build target={{default_target}}
