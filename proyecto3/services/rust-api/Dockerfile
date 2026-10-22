FROM rust:1.90-alpine AS build
RUN apk add --no-cache musl-dev
WORKDIR /src
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo test --locked && cargo build --locked --release
FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app
USER app
COPY --from=build /src/target/release/mumn-api /usr/local/bin/mumn-api
ENTRYPOINT ["mumn-api"]
