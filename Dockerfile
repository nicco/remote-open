FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /remote-open-server ./cmd/server/

FROM alpine:3.20 AS chisel
RUN apk --no-cache add curl && \
    curl -sL https://github.com/jpillora/chisel/releases/download/v1.10.1/chisel_1.10.1_linux_amd64.gz | gunzip > /chisel && \
    chmod +x /chisel

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /remote-open-server /usr/local/bin/remote-open-server
COPY --from=chisel /chisel /usr/local/bin/chisel

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 20080 20081
ENTRYPOINT ["/entrypoint.sh"]
