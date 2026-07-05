ARG IMAGE_REGISTRY_PREFIX=
ARG GO_IMAGE=golang:1.25-alpine
ARG ALPINE_IMAGE=alpine:3.22

FROM ${IMAGE_REGISTRY_PREFIX}${GO_IMAGE} AS build
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ARG ALPINE_MIRROR=
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    GOTOOLCHAIN=local
WORKDIR /src/services/ai-gateway
COPY services/ai-gateway/go.mod services/ai-gateway/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY services/ai-gateway/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ai-gateway-local-seed ./cmd/local-seed
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/pressly/goose/v3/cmd/goose@v3.27.0

FROM ${IMAGE_REGISTRY_PREFIX}${ALPINE_IMAGE}
ARG ALPINE_MIRROR=
RUN --mount=type=cache,target=/var/cache/apk \
    if [ -n "$ALPINE_MIRROR" ]; then sed -i "s|https://dl-cdn.alpinelinux.org/alpine|$ALPINE_MIRROR|g" /etc/apk/repositories; fi \
    && apk add --update-cache --cache-dir /var/cache/apk ca-certificates postgresql-client tzdata
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY --from=build /out/ai-gateway-local-seed /usr/local/bin/ai-gateway-local-seed
COPY services/auth/migrations /workspace/services/auth/migrations
COPY services/file/migrations /workspace/services/file/migrations
COPY services/knowledge/migrations /workspace/services/knowledge/migrations
COPY services/qa/migrations /workspace/services/qa/migrations
COPY services/document/migrations /workspace/services/document/migrations
COPY services/ai-gateway/migrations /workspace/services/ai-gateway/migrations
COPY deploy/seeds /workspace/deploy/seeds
COPY deploy/docker/full/migrate.sh /usr/local/bin/docker-migrate
COPY deploy/docker/full/seed.sh /usr/local/bin/docker-seed
RUN chmod +x /usr/local/bin/docker-migrate /usr/local/bin/docker-seed
WORKDIR /workspace
