# Proyecto 2 — Telemetría de contenedores

Sistema de observabilidad y gestión autónoma de contenedores para Sistemas Operativos 1.

**Estudiante:** Victor Hugo Velásquez Hernández  
**Carné:** 202100054

## Componentes previstos

- Módulo de kernel en C con interfaz `/proc/continfo_pr2_so1_202100054`.
- Daemon en Go para lectura, análisis y gestión de contenedores.
- Sonda eBPF para confirmar señales de terminación.
- Cronjob cada dos minutos para generar cargas.
- Valkey para almacenamiento histórico.
- Grafana para visualización.

## Decisiones aprobadas

- Se usarán directamente `roldyoran/go-client` y `alpine`.
- Siempre se conservarán al menos tres contenedores de bajo consumo y dos de alto consumo.
- Grafana y Valkey quedarán protegidos de la política de eliminación.
- Todo el proyecto residirá en esta carpeta `proyecto2`.

## Estado

Fase 0 — validación de compatibilidad del entorno.

Ejecutar el diagnóstico en la máquina Linux donde se desarrollará y calificará el proyecto:

```bash
cd proyecto2
bash scripts/check_environment.sh
```

La explicación de cada comprobación se encuentra en [docs/fase-0-compatibilidad.md](docs/fase-0-compatibilidad.md).
