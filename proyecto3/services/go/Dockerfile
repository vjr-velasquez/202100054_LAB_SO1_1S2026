FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... && go vet ./... && CGO_ENABLED=0 go build -trimpath -o /service ./cmd/service && CGO_ENABLED=0 go build -trimpath -o /fixture-publisher ./cmd/fixture-publisher
FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app
USER app
COPY --from=build /service /usr/local/bin/service
ENTRYPOINT ["service"]
COPY --from=build /fixture-publisher /usr/local/bin/fixture-publisher
