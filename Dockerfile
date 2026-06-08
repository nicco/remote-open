FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /remote-open-server ./cmd/server/

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /remote-open-server /usr/local/bin/remote-open-server
EXPOSE 20080
ENTRYPOINT ["remote-open-server"]
