# syntax=docker/dockerfile:1.7

FROM golang:1.27.0-alpine3.24 AS build

WORKDIR /src
ARG TARGET=api

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN case "$TARGET" in api|worker) ;; *) exit 1 ;; esac \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app "./cmd/${TARGET}"

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=build /out/app /app

USER app
ENTRYPOINT ["/app"]
