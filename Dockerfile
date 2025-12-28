FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY pkg/ ./pkg/
RUN CGO_ENABLED=0 GOOS=linux go build -o traefik-config-middleware
RUN CGO_ENABLED=0 GOOS=linux go build -o healthcheck healthcheck.go

FROM scratch

WORKDIR /app
COPY --from=builder /app/traefik-config-middleware .
COPY --from=builder /app/healthcheck .

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/healthcheck"]
CMD ["./traefik-config-middleware"]