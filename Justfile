set dotenv-load := true
set positional-arguments := true
set shell := ["bash", "-c"]

BIN_NAME := "dotp"
BIN_ENTRYPOINT := "."
GIT_REMOTE := "github"
DOCKER_IMAGE_NAME := "dotp"
DOCKER_REGISTRY := "docker.io/petlack/dotp"
VERSION_FILE := "./version.txt"

# Show this help message
help:
    just --list

# Clean up build artifacts
build-clean:
    rm -rf build

# Clean all artifacts
clean: build-clean package-clean

# Compile the binary (for local development) - fast compile time
compile: build-clean
    CGO_ENABLED=0 \
    GOOS=linux \
        go build \
            -installsuffix cgo \
            -o build/{{ BIN_NAME }} \
            {{ BIN_ENTRYPOINT }}

# Compile the binary (for release) - slow compile time
compile-release: build-clean
    CGO_ENABLED=0 \
    GOOS=linux \
        go build -a \
            -ldflags="-s -w" \
            -installsuffix cgo \
            -o build/{{ BIN_NAME }} \
            {{ BIN_ENTRYPOINT }}

# Watch for changes and recompile
dev *args:
    find . -name "*.go" | entr -cr just compile run "${@}"

# Build the docker image
docker-build:
    docker build -t {{ DOCKER_IMAGE_NAME }} .

# Dump the docker image to a tarball
docker-dump:
    docker export {{ DOCKER_IMAGE_NAME }} > {{ DOCKER_IMAGE_NAME }}.tar

# Push the docker image to the registry
docker-push: docker-build
    @echo "Pushing image to {{ DOCKER_REGISTRY }}/{{ DOCKER_IMAGE_NAME }}"; \
    tag=$(cat version.txt); \
    docker tag {{ DOCKER_IMAGE_NAME }} {{ DOCKER_REGISTRY }}/{{ DOCKER_IMAGE_NAME }}:$tag; \
    docker tag {{ DOCKER_IMAGE_NAME }} {{ DOCKER_REGISTRY }}/{{ DOCKER_IMAGE_NAME }}:latest; \
    docker push {{ DOCKER_REGISTRY }}/{{ DOCKER_IMAGE_NAME }}:$tag; \
    docker push {{ DOCKER_REGISTRY }}/{{ DOCKER_IMAGE_NAME }}:latest;

# Run the docker image
docker-run *args:
    docker run --rm \
        --name {{ DOCKER_IMAGE_NAME }} \
        --cpus 0.5 \
        --memory 8m \
        "$@" \
        {{ DOCKER_IMAGE_NAME }}

docs-watch:
    find . -type f -name "*.md" | entr -r dotmd README.md

# Create a git tag
git-tag:
    git tag -a v$(cat {{ VERSION_FILE }}) -m "release: v$(cat {{ VERSION_FILE }})"

# Create a release branch
git-push:
    version=$(cat {{ VERSION_FILE }}); \
    git push {{ GIT_REMOTE }} main v$version release/v$version

# Create a release branch
git-release-branch:
    git branch release/v$(cat {{ VERSION_FILE }})

# Build Arch package
package-arch: package-clean tarball
    makepkg --dir archlinux/pkgbuild-src --noconfirm

# Clean package build artifacts
package-clean:
    rm -rf archlinux/pkgbuild-*/{src,pkg,{{ BIN_NAME }}}
    rm -rf archlinux/pkgbuild-*/*.zst

# Bump the version, create a git tag and release branch
release-src:
    @echo "Bumping version in {{ VERSION_FILE }}"; \
    version=$(cat {{ VERSION_FILE }}); \
    echo "Current version is $version"; \
    base_version=$(echo $version | grep -oE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-z]+)?' || echo ""); \
    echo "Base version is $base_version"; \
    date_stamp=$(date +%Y%m%d); \
    sequence=$(grep -o "$date_stamp\.[0-9][0-9]" {{ VERSION_FILE }} | tail -1 | grep -o '[0-9][0-9]$' || echo "00"); \
    let "new_sequence=10#$sequence+1"; \
    new_sequence=$(printf "%02d" $new_sequence); \
    new_version="$base_version.$date_stamp.$new_sequence"; \
    echo $new_version > {{ VERSION_FILE }}; \
    git add {{ VERSION_FILE }} && git commit -m "chore: bump $version -> $new_version"; \

# Bump the version in the Arch package
release-arch:
    sed -i "s/^pkgver=.*$/pkgver=$(cat {{ VERSION_FILE }})/" archlinux/pkgbuild-src/PKGBUILD
    sed -i "s/^pkgver=.*$/pkgver=$(cat {{ VERSION_FILE }})/" archlinux/pkgbuild-git/PKGBUILD
    git add archlinux && git commit -m "chore(arch): bump version to $(cat {{ VERSION_FILE }})"

# Bump the version, create a git tag and release branch
release: release-src release-arch
    just git-tag; \
    just git-release-branch; \
    version=$(cat {{ VERSION_FILE }}); \
    echo "Updated version to $version"

# Run the binary
run *args:
    ./build/{{ BIN_NAME }} "${@}"

# Create source tarball
tarball:
    tar -czf archlinux/pkgbuild-src/dotp-$(cat version.txt).tar.gz \
        *.go go.mod version.txt LICENSE README.md

# Run tests
test arg=".":
    go clean -testcache && go test -v "{{ arg }}"

# Watch for changes and run tests
test-watch arg=".":
    find . -name "*.go" | entr -cr just test "{{ arg }}"

# Simple test
test-env:
    @binary=./build/{{ BIN_NAME }}; \
    totp_secret=$("$binary" new); \
    totp=$("$binary" get --secret-unsafe-value=$totp_secret); \
    TOTP_SECRET=$totp_secret "$binary" validate --secret-env=TOTP_SECRET $totp
