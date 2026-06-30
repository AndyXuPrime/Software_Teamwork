FROM golang:1.25-alpine AS build

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/server

FROM alpine:3.22

RUN addgroup -S gateway && adduser -S -G gateway gateway
COPY --from=build /out/gateway /usr/local/bin/gateway

USER gateway
EXPOSE 8080

ENTRYPOINT ["gateway"]
