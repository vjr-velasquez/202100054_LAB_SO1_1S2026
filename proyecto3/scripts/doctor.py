#!/usr/bin/env python3
"""Diagnóstico sin instalaciones ni conexión a Docker, GCP o Kubernetes."""

import shutil
import sys

TOOLS = {
    "python3": "validación local (Python 3.10 o posterior)",
    "make": "comandos de validación",
    "git": "ramas y commits",
    "go": "servicios Go, fase 3",
    "rustc": "API Rust, fase 3",
    "cargo": "compilación Rust, fase 3",
    "protoc": "compilación protobuf, fase 3",
    "docker": "construcción/contenedores, fases 3–5",
    "kubectl": "Kubernetes, fases 4–5",
    "helm": "charts, fase 4",
    "operator-sdk": "Operator Helm, fase 4",
    "gcloud": "GCP, fase 5",
    "oras": "artefacto OCI (herramienta propuesta), fase 5",
    "locust": "generación de carga, fases 3 y 6",
}

missing = []
for tool, purpose in TOOLS.items():
    found = shutil.which(tool)
    print(f"{'PRESENTE' if found else 'FALTA':8} {tool:14} {purpose}")
    if not found:
        missing.append(tool)
if sys.version_info < (3, 10):
    missing.append("Python >= 3.10")
print("Solo se comprueba PATH; no versiones compatibles, autenticación ni servicios activos.")
print("Entorno completo pendiente." if missing else "Herramientas presentes; falta verificar compatibilidad.")
sys.exit(1 if missing else 0)
