FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o developer-platform

FROM gcr.io/distroless/base-debian12

WORKDIR /srv/app
COPY --from=builder /app/developer-platform ./server
COPY .env.example ./.env.example

ENV GIN_MODE=release
EXPOSE 8080

ENTRYPOINT ["./server"]
