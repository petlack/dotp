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
    makepkg --dir .pkgs/archlinux/pkgbuild-src --noconfirm

package-arch-docker:
    docker build -t dotp-build-arch -f .pkgs/archlinux.Dockerfile .
package-arch-docker-export:
    docker create --name tmp-dotp-build-arch dotp-build-arch
    docker export tmp-dotp-build-arch > dotp-build-arch.tar
    docker rm tmp-dotp-build-arch
    # docker rmi dotp-build-arch

package-alpine-docker:
    docker build -t dotp-build-alpine -f .pkgs/alpine.Dockerfile .
package-alpine-docker-export:
    docker create --name tmp-dotp-build-alpine dotp-build-alpine
    docker export tmp-dotp-build-alpine > dotp-build-alpine.tar
    docker rm tmp-dotp-build-alpine

package-debian-docker:
    docker build -t dotp-build-debian -f .pkgs/debian.Dockerfile .
package-debian-docker-export:
    docker create --name tmp-dotp-build-debian dotp-build-debian
    docker export tmp-dotp-build-debian > dotp-build-debian.tar
    docker rm tmp-dotp-build-debian

# Clean package build artifacts
package-clean:
    rm -rf .pkgs/archlinux/pkgbuild-{src,git}/{src,pkg,{{ BIN_NAME }}}
    rm -rf .pkgs/archlinux/pkgbuild-{src,git}/*.tar.{zst,gz}

publish: release git-push release-arch-git

# Bump the version, create a git tag and release branch
release: release-src release-arch-src release-alpine
    just git-tag; \
    just git-release-branch; \
    echo "Updated version to $(cat {{ VERSION_FILE }})"

# Bump the version, create a git tag and release branch
release-src:
    @echo "Bumping version in {{ VERSION_FILE }}"; \
    prev_version=$(cat {{ VERSION_FILE }}); \
    echo "Current version is $prev_version"; \
    base_version=$(echo $prev_version | grep -oE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-z]+)?' || echo ""); \
    echo "Base version is $base_version"; \
    date_stamp=$(date +%Y%m%d); \
    sequence=$(grep -o "$date_stamp\.[0-9][0-9]" {{ VERSION_FILE }} | tail -1 | grep -o '[0-9][0-9]$' || echo "00"); \
    let "next_sequence=10#$sequence+1"; \
    next_sequence=$(printf "%02d" $next_sequence); \
    next_version="$base_version.$date_stamp.$next_sequence"; \
    echo $next_version > {{ VERSION_FILE }}; \
    echo "Replacing $prev_version with $next_version in README.md"; \
    sed -i "s/$prev_version/$next_version/" README.md; \
    git add {{ VERSION_FILE }} README.md && git commit -m "chore: bump $prev_version -> $next_version"

# Bump the version in the Alpine package
release-alpine:
    sed -i "s/^pkgver=.*$/pkgver=$(cat {{ VERSION_FILE }})/" .pkgs/alpine/APKBUILD
    git add .pkgs/alpine && git commit -m "chore(alpine): bump version to $(cat {{ VERSION_FILE }})"

# Bump the version in the Arch *-src package
release-arch-src:
    sed -i "s/^pkgver=.*$/pkgver=$(cat {{ VERSION_FILE }})/" .pkgs/archlinux/pkgbuild-src/PKGBUILD
    git add .pkgs/archlinux/pkgbuild-src && git commit -m "chore(arch-src): bump version to $(cat {{ VERSION_FILE }})"

# Bump the version in the Arch *-git package
release-arch-git:
    makepkg --dir .pkgs/archlinux/pkgbuild-git --noconfirm
    makepkg --dir .pkgs/archlinux/pkgbuild-git --printsrcinfo > .pkgs/archlinux/pkgbuild-git/.SRCINFO
    git add .pkgs/archlinux/pkgbuild-git && git commit -m "chore(arch-git): bump version to $(cat {{ VERSION_FILE }})"

# Run the binary
run *args:
    ./build/{{ BIN_NAME }} "${@}"

# Create source tarball
tarball:
    tar -czf .pkgs/archlinux/pkgbuild-src/dotp-$(cat version.txt).tar.gz \
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
