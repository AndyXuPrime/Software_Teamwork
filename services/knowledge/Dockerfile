FROM golang:1.25-alpine AS build

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/knowledge ./cmd/server

FROM alpine:3.21

RUN addgroup -S knowledge && adduser -S -G knowledge knowledge
COPY --from=build /out/knowledge /usr/local/bin/knowledge

USER knowledge
EXPOSE 8083

ENTRYPOINT ["knowledge"]
