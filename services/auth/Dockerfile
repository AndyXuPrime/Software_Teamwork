FROM golang:1.25-alpine AS build

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth ./cmd/server

FROM alpine:3.22

RUN addgroup -S auth && adduser -S -G auth auth
COPY --from=build /out/auth /usr/local/bin/auth

USER auth
EXPOSE 8001

ENTRYPOINT ["auth"]
