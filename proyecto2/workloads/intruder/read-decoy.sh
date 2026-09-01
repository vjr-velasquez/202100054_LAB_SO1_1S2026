#!/bin/sh

set -eu

TARGET_FILE="${TARGET_FILE:-/host-sensitive/sensitive.txt}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-15}"

while true; do
    timestamp="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

    if [ -r "$TARGET_FILE" ]; then
        echo "[$timestamp] [INTRUDER] Lectura exitosa del archivo señuelo:"
        cat "$TARGET_FILE"
    else
        echo "[$timestamp] [INTRUDER] No fue posible leer: $TARGET_FILE"
    fi

    sleep "$INTERVAL_SECONDS"
done