# syntax=docker/dockerfile:1

FROM golang:1.23.3 AS build-stage

WORKDIR /app

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server

FROM alpine:3.19.4

WORKDIR /

COPY --from=build-stage /app/server /app/server

ENTRYPOINT ["/bin/sh", "-c", "cd /app && ./server"]
