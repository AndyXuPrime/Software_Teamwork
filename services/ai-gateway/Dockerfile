FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/ai-gateway ./cmd/server

FROM alpine:3.22
RUN adduser -D -H app
USER app
WORKDIR /app
COPY --from=build /out/ai-gateway /app/ai-gateway
EXPOSE 8086
ENTRYPOINT ["/app/ai-gateway"]
