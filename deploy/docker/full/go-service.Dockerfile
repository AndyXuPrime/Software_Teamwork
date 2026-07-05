ARG IMAGE_REGISTRY_PREFIX=
ARG GO_IMAGE=golang:1.25-alpine
ARG ALPINE_IMAGE=alpine:3.22

FROM ${IMAGE_REGISTRY_PREFIX}${GO_IMAGE} AS build
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ARG ALPINE_MIRROR=
ARG SERVICE_DIR
ARG TARGET=./cmd/server
ARG BINARY=server
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    GOTOOLCHAIN=local
WORKDIR /src/${SERVICE_DIR}
COPY ${SERVICE_DIR}/go.mod ${SERVICE_DIR}/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY ${SERVICE_DIR}/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/${BINARY} ${TARGET}

FROM ${IMAGE_REGISTRY_PREFIX}${ALPINE_IMAGE}
ARG ALPINE_MIRROR=
RUN --mount=type=cache,target=/var/cache/apk \
    if [ -n "$ALPINE_MIRROR" ]; then sed -i "s|https://dl-cdn.alpinelinux.org/alpine|$ALPINE_MIRROR|g" /etc/apk/repositories; fi \
    && apk add --update-cache --cache-dir /var/cache/apk ca-certificates tzdata
ARG BINARY=server
RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/${BINARY} /usr/local/bin/service
USER app
ENTRYPOINT ["/usr/local/bin/service"]
