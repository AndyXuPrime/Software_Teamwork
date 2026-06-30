FROM golang:1.25-alpine AS build

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/file ./cmd/server

FROM alpine:3.21

RUN addgroup -S file && adduser -S -G file file && mkdir -p /data/file-storage && chown -R file:file /data/file-storage
COPY --from=build /out/file /usr/local/bin/file

USER file
WORKDIR /data
EXPOSE 8082

ENTRYPOINT ["file"]
