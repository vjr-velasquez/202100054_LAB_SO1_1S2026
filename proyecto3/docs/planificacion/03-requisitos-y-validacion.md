# Matriz de requisitos y validación

Todos los requisitos están **pendientes de implementación y verificación**.
Una propuesta documentada no se marca como requisito técnico cumplido.
O = obligatorio del enunciado; P = decisión de diseño propuesta; E = extra opcional;
C = alcance sujeto a aclaración.

## Trazabilidad

| ID | Tipo | Requisito/componente | Paso | Prueba o evidencia de aceptación |
|---|---|---|---|---|
| R01 | O | Mismo repositorio privado, carpeta `proyecto3`. | 7 | Revisar árbol y privacidad real del repositorio. |
| R02 | O | GKE funcional y máquinas N1. | 5 | Inventario de clúster, nodos y VM Zot, tipos de máquina y estado Ready. |
| R03 | O | Locust genera JSON del enunciado. | 3,6 | Países, rangos y timestamps válidos; configuración y reporte exportado. |
| R04 | O | Gateway API y ruta `/grpc-202100054`. | 5 | Gateway/HTTPRoute aceptados, backend saludable y POST externo llega al flujo completo. |
| R05 | O | API REST Rust. | 3 | Código/imagen propios; casos válidos e inválidos y propagación correcta de errores. |
| R06 | O | HPA Rust 1–3, objetivo CPU 30%. | 6 | Manifiesto, requests, métricas disponibles y serie de réplicas/CPU durante carga y recuperación. |
| R07 | O | Go entrada: REST y cliente gRPC, dos contenedores. | 3 | Pod con ambos contenedores y evidencia del salto gRPC real. |
| R08 | O,C | Go publicador: servidor gRPC y writer, dos contenedores; cantidad de deployments D02. | 3 | Topología aclarada, RPC recibido y publicación correlacionada; ambos contenedores por pod requerido. |
| R09 | O | Concurrencia y comunicación REST/gRPC. | 3,6 | Errores y cancelación propagados; prueba concurrente sin pérdidas ni bloqueo indefinido. |
| R10 | O | RabbitMQ como broker principal. | 3 | Publicación, cola y consumo reales; no sustitución por otro broker. |
| R11 | O | Consumidor Go guarda en Valkey. | 3,4 | Dataset conocido almacenado con valores y conteos exactos. |
| R12 | O | Operator SDK Helm dentro de GKE. | 4,5 | Proyecto generado, charts, watches, RBAC e imagen ejecutándose. |
| R13 | O | CRD/CR independiente Valkey. | 4 | Crear/actualizar CR genera StatefulSet, Service y PVC; comprobar reconciliación y eliminación documentada. |
| R14 | O | Persistencia Valkey y conexión del consumidor. | 4 | Reiniciar pod con datos de ensayo conserva resultados; Consumer usa Service del Operator. |
| R15 | O | CRD/CR independiente Grafana. | 4 | CR genera Deployment/Service y provisioning; cambio declarativo se refleja. |
| R16 | O | Datasource Grafana y visualización. | 4 | Consulta real y todos los paneles contrastados con dataset, no solo conexión exitosa. |
| R17 | O | Zot externo en VM GCP, HTTPS. | 5 | Push/pull con TLS validado desde clientes y nodos; configuración y persistencia documentadas. |
| R18 | O,C | Todas las imágenes del proyecto desde Zot. | 5,7 | Inventario y referencias reales de pods/init containers; excluir solo componentes administrados cuyo alcance se aclare. |
| R19 | O | Descarga y uso de archivo OCI. | 5 | Digest y descarga de `load-profile.json`; evidencia de que Locust usa ese archivo. |
| R20 | O | Máximo general aviones. | 4 | Dataset: 30. |
| R21 | O | Mínimo general aviones. | 4 | Dataset: 0. |
| R22 | O | Máximo general barcos. | 4 | Dataset: 8. |
| R23 | O | Mínimo general barcos. | 4 | Dataset: 0. |
| R24 | O,C | Top países por aviones, semántica D05. | 4 | Dataset: CHN 30, USA 20, ESP 0. |
| R25 | O,C | Top países por barcos, semántica D05. | 4 | Dataset: CHN 8, USA 5, ESP 0. |
| R26 | O,C | Moda aviones, política de empate D05. | 4 | Dataset: 20, frecuencia 2; caso separado de empate. |
| R27 | O,C | Moda barcos, política de empate D05. | 4 | Dataset: 5, frecuencia 2; caso separado de empate. |
| R28 | O | Serie temporal CHN de ambos recursos. | 4 | Tres puntos en orden, aviones 10/20/30, barcos 2/5/8. |
| R29 | O | Nombre del país asignado. | 4 | Panel China (CHN), consistente con último dígito 4. |
| R30 | O | Total de reportes CHN. | 4 | Dataset: 3; reentrega no lo incrementa. |
| R31 | O,C | Comparar 1 y 2 réplicas Go Writers. | 6 | Misma carga y recursos por pod; comparar throughput, p95, errores, cola y latencia hasta almacenamiento. |
| R32 | O,C | Comparar Valkey según rúbrica, topología D03. | 6 | Topología explícita; medir escritura/lectura y retraso de réplica, sin confundir réplica con almacén independiente. |
| R33 | O | Manual y guía exclusivamente Markdown. | 7 | Arquitectura, flujo, Gateway, REST/gRPC, RabbitMQ, Operator/charts/reconciliación, HPA, Zot/OCI, instalación, pruebas y conclusiones. |
| R34 | O | Evidencias funcionales y capturas. | 3–7 | Archivos legibles vinculados al manual; versiones, corrida y fecha identificables. |
| R35 | O | Acceso de auxiliares. | 7 | Verificar `JoseLorenzana272` y `KINGR0X`; registrar estado real del acceso. |
| R36 | O | Entrega UEDI antes del 22/10/2026. | 7 | Comprobante de envío por el estudiante; no confundir Git push con entrega UEDI. |
| R37 | O | Defensa práctica el 24/10/2026. | 7 | Ensayar flujo, fallos, Operator y explicación de resultados; preguntas de evaluación aún no conocidas. |
| R38 | P | Idempotencia y consistencia de métricas. | 3,4 | Reentrega, concurrencia y conflicto de identidad comprobados. |
| R39 | P | Gancho de calidad funcional. | 2–7 | Detecta fallo deliberado en copia/datos de prueba; no informa éxito si faltan dependencias. |
| R40 | E | Dapr `/dapr-202100054`. | Extra | SDK, pub/sub sobre RabbitMQ, suscripción y comparación con flujo base. |
| R41 | E | Falco y regla personalizada. | Extra | Regla YAML, simulación en pod Rust/Go usando directorio de prueba y captura de alerta incluida en manual. |

El documento contiene condiciones de acceso a calificación además de penalizaciones.
No interpretar las penalizaciones como permiso para omitir un obligatorio. UEDI y manual
Markdown figuran como causas de nota cero. RabbitMQ, Locust, API Gateway, Zot y defensa
práctica tienen penalizaciones explícitas en el enunciado. Almacenamiento y visualización
suman 45/100 puntos; deben comprobarse temprano.

## Gancho de calidad propuesto

El paso 2 implementará un comando único, por ejemplo `make check`, con validaciones
rápidas locales. CI reutilizará ese comando. Integración opcional en `pre-push` después
de revisar su comportamiento; todavía no se ha instalado ningún hook.

Separar validaciones que necesitan servicios (`make check-integration`) y clúster
(`make check-cluster`). Los nombres son propuestas. Nunca iniciar aprovisionamiento
de GCP, carga masiva o borrado desde un hook local. La comprobación de clúster deberá
confirmar contexto/namespace esperado antes de usar recursos de ensayo.

| Nivel | Comprobaciones previstas | Resultado |
|---|---|---|
| Local | Compilación Rust/Go, contrato y conversiones, formato y renderizado/lint de manifiestos/charts. | Falla si hay defecto o dependencia imprescindible ausente. |
| Integración | Dataset con valores esperados, errores de dependencias, publicación, consumo, deduplicación y estadísticas. | Falla ante éxito engañoso, pérdida, duplicación o cálculo incorrecto. |
| Operator/clúster | CR separados, reconciliación, persistencia, imágenes, Gateway y paneles. | Evidencia del comportamiento real, no solo validación YAML. |
| Carga explícita | HPA, 1/2 réplicas, recuperación y comparación reproducible. | Informe medido con limitaciones y recursos utilizados. |

Cada hallazgo incluirá: ID, requisito afectado, pasos de reproducción, esperado,
observado, impacto, corrección propuesta y resultado tras repetir la prueba.

- **Bloqueante funcional:** rompe contrato, requisito obligatorio, datos o resultado
  observable, aunque no sea peligroso. No se acepta la fase con ese fallo abierto.
- **Pendiente de verificación:** comprobación sin ejecutar, servicio ausente o decisión
  sin resolver. No cuenta como aprobado.
- **Mejora:** no afecta requisito ni resultado acordado; puede pasar al backlog con motivo.

Para verificar el propio gancho se hará fallar un caso controlado, por ejemplo una
respuesta estadística con conteo incorrecto en un fixture temporal. No introducir
deliberadamente fallos en despliegues compartidos ni dejar código roto en el repositorio.

## Pruebas de carga y recuperación

En cada corrida conservar: commit/digests, configuración del generador, semilla,
usuarios, ritmo de llegada, duración, recursos, réplicas, dataset/corrida y timestamps.
Hacer calentamiento y medición con iguales condiciones; repetir al menos tres veces
cada escenario para identificar variación. Los valores de carga se elegirán dentro del
presupuesto y se registrarán antes del ensayo, sin inventar un RPS mínimo académico.

Comparar Go y Valkey por separado, cambiando una variable por ensayo. Durante la
comparación aislar el efecto del HPA manteniendo condiciones equivalentes; ejecutar
la demostración de HPA como escenario propio. Propuesta: throughput aceptado y procesado,
p50/p95/p99 HTTP, tasa de errores, retraso hasta Valkey, profundidad de cola, CPU y memoria.

Después de drenar la cola, contrastar IDs únicos generados/aceptados y resultados
persistidos. Registrar publicaciones inciertas por timeout, conflictos y fallos; un
conteo de HTTP 202 por sí solo no demuestra que todos los reportes se almacenaron.

Ensayos de recuperación: consumidor reiniciado tras guardar antes del ACK, Valkey
temporalmente inaccesible, writer/broker no disponible, reinicio de pod Valkey con PVC,
y recurso gestionado eliminado en namespace de prueba. Verificar datos y recuperación,
no solo que el pod vuelva a Ready.

## Cierre del paso 1

- [x] Enunciado leído y revisado con subagente.
- [x] Arquitectura, interfaces, modelo estadístico y matriz documentados.
- [x] Contradicciones distinguidas de decisiones de diseño.
- [x] Criterios de aceptación y gancho definidos sin implementarlos.
- [x] Usuario acepta la planificación y autoriza el paso 2 (29/09/2026); las consultas de cátedra siguen pendientes.
- [ ] D02 y D03 se aclaran antes de implementar sus topologías dependientes.
- [ ] D08 se valida técnicamente durante el desarrollo; D09 antes de aprovisionar.

La aprobación del usuario permite continuar la fase acordada; no convierte decisiones
pendientes de cátedra o pruebas futuras en resultados ya confirmados.
