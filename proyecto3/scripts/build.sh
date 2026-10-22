#!/bin/sh
set -eu
project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
if ! docker buildx version >/dev/null 2>&1; then
    export DOCKER_BUILDKIT=0
fi
docker build -t mumn-go:phase3 "$project_dir/services/go"
docker build -t mumn-rust:phase3 "$project_dir/services/rust-api"
