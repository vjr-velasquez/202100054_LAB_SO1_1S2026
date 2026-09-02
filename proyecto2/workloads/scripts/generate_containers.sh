#!/usr/bin/env bash

set -Eeuo pipefail

CARNET="202100054"
PROJECT="proyecto2"
CONTAINER_COUNT=5
DOCKER_BIN="/usr/bin/docker"

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT2_DIR="$(cd -- "$SCRIPT_DIR/../.." && pwd)"

INTRUDER_IMAGE="so1-intruder:${CARNET}"
DECOY_FILE="$PROJECT2_DIR/workloads/intruder/decoy/sensitive.txt"

log() {
    printf '[%s] %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$1"
}

ensure_image() {
    local image="$1"

    if ! "$DOCKER_BIN" image inspect "$image" >/dev/null 2>&1; then
        log "Descargando imagen $image"
        "$DOCKER_BIN" pull "$image" >/dev/null
    fi
}

if ! "$DOCKER_BIN" info >/dev/null 2>&1; then
    log "ERROR: el daemon de Docker no está disponible"
    exit 1
fi

if ! "$DOCKER_BIN" image inspect "$INTRUDER_IMAGE" >/dev/null 2>&1; then
    log "ERROR: falta la imagen $INTRUDER_IMAGE"
    exit 1
fi

if [ ! -f "$DECOY_FILE" ]; then
    log "ERROR: no existe el archivo señuelo $DECOY_FILE"
    exit 1
fi

ensure_image "alpine:latest"
ensure_image "roldyoran/go-client:latest"

profiles=(
    "low"
    "high-cpu"
    "high-memory"
    "intruder"
)

timestamp="$(date -u '+%Y%m%d%H%M%S')"

for index in $(seq 1 "$CONTAINER_COUNT"); do
    profile="${profiles[$((RANDOM % ${#profiles[@]}))]}"
    name="so1-${profile}-${timestamp}-${index}-${RANDOM}"

    common_labels=(
        --label "so1.project=$PROJECT"
        --label "so1.carnet=$CARNET"
        --label "so1.profile=$profile"
        --label "so1.protected=false"
        --label "so1.created-by=cron"
    )

    case "$profile" in
        low)
            container_id="$(
                "$DOCKER_BIN" run -d \
                    --name "$name" \
                    "${common_labels[@]}" \
                    --label "so1.tier=low" \
                    --memory 64m \
                    --cpus 0.10 \
                    alpine:latest sleep 240
            )"
            ;;

        high-cpu)
            container_id="$(
                "$DOCKER_BIN" run -d \
                    --name "$name" \
                    "${common_labels[@]}" \
                    --label "so1.tier=high" \
                    --memory 128m \
                    --cpus 0.50 \
                    alpine:latest \
                    sh -c 'while true; do :; done'
            )"
            ;;

        high-memory)
            container_id="$(
                "$DOCKER_BIN" run -d \
                    --name "$name" \
                    "${common_labels[@]}" \
                    --label "so1.tier=high" \
                    --memory 384m \
                    --cpus 0.25 \
                    roldyoran/go-client:latest
            )"
            ;;

        intruder)
            container_id="$(
                "$DOCKER_BIN" run -d \
                    --name "$name" \
                    "${common_labels[@]}" \
                    --label "so1.tier=intruder" \
                    --memory 64m \
                    --cpus 0.10 \
                    --mount "type=bind,src=$DECOY_FILE,dst=/host-sensitive/sensitive.txt,readonly" \
                    "$INTRUDER_IMAGE"
            )"
            ;;

        *)
            log "ERROR: perfil desconocido $profile"
            exit 1
            ;;
    esac

    log "Creado $name (${container_id:0:12}) con perfil $profile"
done

log "Generación completada: $CONTAINER_COUNT contenedores"