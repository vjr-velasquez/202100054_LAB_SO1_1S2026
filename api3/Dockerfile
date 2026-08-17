FROM docker.io/library/golang:1.24-alpine AS builder

WORKDIR /src
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o api3 main.go

FROM scratch

COPY --from=builder /src/api3 /api3

EXPOSE 8083

ENTRYPOINT ["/api3"]
