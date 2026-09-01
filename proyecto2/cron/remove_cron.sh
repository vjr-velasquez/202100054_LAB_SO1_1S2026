#!/usr/bin/env bash

set -Eeuo pipefail

MARKER="# SO1_PROYECTO2_202100054"

if ! command -v crontab >/dev/null 2>&1; then
    echo "ERROR: crontab no está instalado"
    exit 1
fi

CURRENT_FILE="$(mktemp)"
FILTERED_FILE="$(mktemp)"

trap 'rm -f "$CURRENT_FILE" "$FILTERED_FILE"' EXIT

if ! crontab -l > "$CURRENT_FILE" 2>/dev/null; then
    echo "No existe un crontab para el usuario actual"
    exit 0
fi

grep -vF "$MARKER" "$CURRENT_FILE" > "$FILTERED_FILE" || true

crontab "$FILTERED_FILE"

echo "Cronjob del Proyecto 2 eliminado correctamente"