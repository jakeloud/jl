# syntax=docker/dockerfile:1
FROM node:20.12.0-alpine3.19 AS frontend-stage

WORKDIR /app
COPY . .
RUN npm i && npm run build

FROM golang:1.23.3 AS build-stage

WORKDIR /app

COPY . .
RUN go mod download

COPY --from=frontend-stage /app/server/dist /app/server/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/jl

FROM alpine:3.19.4

WORKDIR /

COPY --from=build-stage /app/jl /app/jl

ENTRYPOINT ["/bin/sh", "-c", "cd /app && ./jl"]
