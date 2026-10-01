#!/bin/sh
set -eu
project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
docker run --rm -v "$project_dir:/work" -w /work/services/go \
    -e OUTPUT_OWNER="$(id -u):$(id -g)" golang:1.24-alpine sh -c '
    set -eu
    apk add --no-cache protobuf
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
    protoc -I /work/contracts \
      --go_out=. --go_opt=module=github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go \
      --go-grpc_out=. --go-grpc_opt=module=github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go \
      /work/contracts/war_report.proto
    chown -R "$OUTPUT_OWNER" gen
    '
