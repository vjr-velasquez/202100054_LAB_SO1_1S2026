# Verificación de fase 3

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

## Alcance pendiente

Operator, reconciliación de CR, dashboards, Gateway, HPA, Zot, artefacto OCI,
comparación 1/2 réplicas y GKE todavía no están implementados ni validados.
