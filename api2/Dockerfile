FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o api2 main.go

FROM scratch

COPY --from=builder /src/api2 /api2

EXPOSE 8082

ENTRYPOINT ["/api2"]
