#!/bin/sh
# Entrada reutilizable por CI; requiere Python >= 3.10 y make.
set -eu
project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
exec make -C "$project_dir" check
