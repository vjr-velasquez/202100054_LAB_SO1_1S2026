FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o api1 main.go

FROM scratch

COPY --from=builder /src/api1 /api1

EXPOSE 8081

ENTRYPOINT ["/api1"]
