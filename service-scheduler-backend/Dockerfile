# multi-stage build for Go backend
FROM golang:1.25-alpine AS builder

WORKDIR /app
ENV CGO_ENABLED=0
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# install git for module fetches
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# build binary
RUN go build -o /srv/api ./cmd/api

# runtime image
FROM alpine:3.18
RUN apk add --no-cache ca-certificates postgresql-client

COPY --from=builder /srv/api /usr/local/bin/api
COPY --from=builder /app/internal/database/migrations /migrations
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]
