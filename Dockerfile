FROM golang:1.22.4-alpine3.20 AS builder

ARG GO_IN=/app \
    GO_LD_FLAGS="-s -w" \
    APP_UID=1000 \
    APP_GID=1000 \
    GROUP_NAME=bingrp \
    USER_NAME=binusr

WORKDIR /app
COPY . .

RUN addgroup -g ${APP_GID} ${GROUP_NAME} && \
    adduser -D -g '' -u ${APP_UID} -G ${GROUP_NAME} ${USER_NAME} && \
    chown -R ${USER_NAME}:${GROUP_NAME} /app

USER ${USER_NAME}

RUN \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=$(if [ "${TARGETPLATFORM}" = "linux/amd64" ]; then echo "amd64"; elif [ "${TARGETPLATFORM}" = "linux/arm64" ]; then echo "arm64"; elif [ "${TARGETPLATFORM}" = "linux/arm/v7" ]; then echo "arm"; elif [ "${TARGETPLATFORM}" = "linux/386" ]; then echo "386"; fi) \
    go build \
        -a \
        -installsuffix cgo \
        -ldflags="${GO_LD_FLAGS}" \
        -o /app/build/compiled.bin \
        ${GO_IN}

FROM scratch

ARG USER_NAME=binusr

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /app/build/compiled.bin /opt/dotp

USER ${USER_NAME}

CMD ["/opt/dotp"]
