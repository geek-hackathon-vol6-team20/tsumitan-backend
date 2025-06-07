# WARNING: This Dockerfile is intended for Cloud Run deployment only.
# It is not used in the development environment.

FROM golang:alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /main /src/cmd/api/main.go

FROM alpine:latest
RUN apk add --no-cache postgresql-client && rm -rf /var/cache/apk/*

WORKDIR /app
COPY --from=builder /main ./main

EXPOSE 8080

CMD ["./main"]