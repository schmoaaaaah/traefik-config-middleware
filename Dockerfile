FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o traefik-config-middleware

FROM scratch

WORKDIR /app
COPY --from=builder /app/traefik-config-middleware .

EXPOSE 8080
CMD ["./traefik-config-middleware"]