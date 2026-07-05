ARG IMAGE_REGISTRY_PREFIX=
ARG BUN_IMAGE=oven/bun:1.3.12-alpine
ARG NGINX_IMAGE=nginx:1.27-alpine

FROM ${IMAGE_REGISTRY_PREFIX}${BUN_IMAGE} AS build
ARG ALPINE_MIRROR=
WORKDIR /app
COPY package.json bun.lock ./
COPY apps/web/package.json apps/web/package.json
RUN --mount=type=cache,target=/root/.bun bun install --frozen-lockfile
COPY apps/web apps/web
RUN bun run --cwd apps/web build

FROM ${IMAGE_REGISTRY_PREFIX}${NGINX_IMAGE}
COPY deploy/docker/full/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/apps/web/dist /usr/share/nginx/html
EXPOSE 80
