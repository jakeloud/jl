# syntax=docker/dockerfile:1
FROM oven/bun:1.2.15-alpine AS frontend-stage

WORKDIR /app
COPY . .
RUN cd vite-app && bun i && bun run build

FROM golang:1.25.1 AS build-stage

WORKDIR /app

COPY . .
RUN go mod download

COPY --from=frontend-stage /app/server/dist /app/server/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/jl

FROM scratch
COPY --from=build-stage /app/jl /app/jl

CMD ["/app/jl --dry -d"]
