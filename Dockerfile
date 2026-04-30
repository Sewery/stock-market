# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /app

RUN apk add --no-cache bash curl ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV HOST=0.0.0.0
ENV PORT=8080

EXPOSE 8080

CMD ["go", "run", "./cmd/server"]