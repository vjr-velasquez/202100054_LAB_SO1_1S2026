# Verificación del proyecto

Resultado local del 30/09/2026 (America/Guatemala):

| Comprobación | Resultado |
|---|---|
| Referencia Python | 14 pruebas aprobadas. |
| Servicios Go | Pruebas de dominio, transporte y bloqueo AMQP aprobadas; go vet aprobado. |
| API Rust | 3 pruebas de validación aprobadas y binario compilado. |
| Integración Go con Valkey/RabbitMQ | Escritura concurrente/deduplicación, conflicto, corrupción sin escritura parcial y mensaje sin ruta comprobados. |
| Flujo completo | 9 escenarios aprobados, incluidos reinicios y recuperación. |
| Locust básico | 2 usuarios, 10 segundos, 20 solicitudes y 0 errores; media HTTP aproximada 7 ms. |

La medición breve de Locust es una prueba de funcionamiento local, no una conclusión
sobre capacidad o rendimiento en GKE. La aceptación HTTP se mide antes de que el
consumidor termine; los escenarios de integración comprueban también Valkey.

Ejecutar `make -C proyecto3 check-integration` desde la raíz para repetir los casos.
El script guarda la corrida y resultados en `artifacts/integration-latest.json`.
Los tests de fallo operan exclusivamente sobre el stack Compose local `mumn-p3`.

## Fase 4: Kubernetes local (06/10/2026)

- Operator SDK Helm con dos CRD: creación, actualización y eliminación comprobadas.
- Siete verificaciones de clúster aprobadas con `make check-cluster`: 11 consultas
  reales a Infinity, corridas vacías/inválidas, actualización de ambos CR,
  persistencia al cambiar UID del pod, reconstrucción del Service y retención del
  mismo PVC al eliminar/recrear ambos CR.
- `make check`: referencia Python, compilación y pruebas Go/Rust, go vet,
  lint de ambos charts y render Kustomize aprobados.
- Navegador Chromium: 11 paneles renderizados con el fixture esperado, incluyendo
  modas/frecuencia y dos series. Captura local `artifacts/phase4-dashboard.png`.
  La inspección detectó y permitió corregir el filtro de campos textuales,
  `url_options.method` requerido por Infinity y OOM con el límite anterior de 512 MiB.

Las pruebas de ciclo de vida usan exclusivamente `kind-mumn-phase4` y modifican
los CR de demostración `store`/`view`; conservan el PVC. La prueba de esta fase
publica el fixture a RabbitMQ y utiliza el consumidor real; la ruta de entrada
completa permanece validada en el entorno Compose de fase 3.

## Alcance pendiente

Gateway, HPA, Zot, artefacto OCI, comparación 1/2 réplicas y GKE todavía no están
implementados ni validados. Las pruebas locales no acreditan el despliegue en GCP.
