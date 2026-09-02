#!/usr/bin/env bash

set -Eeuo pipefail

MARKER="# SO1_PROYECTO2_202100054"

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT2_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

GENERATOR="$PROJECT2_DIR/workloads/scripts/generate_containers.sh"
LOG_DIR="$PROJECT2_DIR/logs"
LOG_FILE="$LOG_DIR/cron.log"

if ! command -v crontab >/dev/null 2>&1; then
    echo "ERROR: crontab no está instalado"
    exit 1
fi

if [ ! -x "$GENERATOR" ]; then
    echo "ERROR: el generador no existe o no tiene permiso de ejecución"
    exit 1
fi

if ! systemctl is-active --quiet cron; then
    echo "ERROR: el servicio cron no está activo"
    exit 1
fi

mkdir -p "$LOG_DIR"

TEMP_FILE="$(mktemp)"
trap 'rm -f "$TEMP_FILE"' EXIT

crontab -l 2>/dev/null \
    | grep -vF "$MARKER" > "$TEMP_FILE" || true

printf '*/2 * * * * /bin/bash %s >> %s 2>&1 %s\n' \
    "$GENERATOR" \
    "$LOG_FILE" \
    "$MARKER" >> "$TEMP_FILE"

crontab "$TEMP_FILE"

echo "Cronjob instalado correctamente:"
crontab -l | grep -F "$MARKER"