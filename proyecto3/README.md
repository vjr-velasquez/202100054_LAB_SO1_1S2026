# Proyecto 3 — M.U.M.N.K8s

Estudiante: Victor Hugo Velásquez Hernández. Carné: **202100054**.
País asignado: **CHN (China)**. Ruta principal: `/grpc-202100054`.

## Fase 3: flujo local implementado

```text
Locust/HTTP → Rust → Go API → Go cliente gRPC → Go servidor gRPC
           → Go writer → RabbitMQ → Go consumidor → Valkey
```

La rama `proyecto3-fase3-flujo` incorpora servicios reales y pruebas de integración.
El entorno Compose es de desarrollo: GKE, Gateway API, Zot, Operator, dashboards,
HPA y comparación de réplicas corresponden a las fases siguientes.

Los dos roles de entrada y los dos de publicación son contenedores separados.
Cada pareja comparte red para comunicarse por loopback, como los futuros pods.
La cantidad final de deployments publicadores y la topología de réplicas Valkey
siguen pendientes de aclaración del enunciado.

## Ejecutar

Requisitos locales: Docker Engine y Compose, Python >=3.10 y Make. Los compiladores
Go/Rust corren dentro de imágenes Docker; no necesitan instalación global.
La primera construcción descarga imágenes y dependencias.

Desde la raíz del repositorio:

```bash
make -C proyecto3 local-up
```

El constructor usa Buildx si está disponible y el constructor Docker clásico como
alternativa. La API queda en `http://127.0.0.1:18080`. Para cambiar el puerto, definir
MUMN_PORT al ejecutar los comandos. RabbitMQ y Valkey no publican puertos del host.
El proyecto Compose `mumn-p3` tiene red y volúmenes independientes del Proyecto 2.

Enviar un reporte:

```bash
curl -i http://127.0.0.1:18080/grpc-202100054 \
  -H 'Content-Type: application/json' \
  -H 'X-Run-ID: prueba-manual' \
  -d '{"country":"CHN","warplanes_in_air":20,"warships_in_water":8,"timestamp":"2026-09-30T12:00:00Z"}'
```

La respuesta 202 contiene `status`, `event_id` y `run_id`. Significa aceptación
confirmada por RabbitMQ; el consumidor almacena después en Valkey.
Para reintentar sin duplicar, enviar `Idempotency-Key` con el mismo UUID. Un UUID
nuevo representa un nuevo reporte aunque su contenido sea idéntico.

Ver el contenido almacenado y los logs:

```bash
docker compose -f proyecto3/deploy/compose.local.yaml exec valkey valkey-cli HGETALL mumn:run:prueba-manual
docker compose -f proyecto3/deploy/compose.local.yaml logs --tail=50 writer consumer
```

`GET /healthz` indica que el proceso Rust está activo; no verifica todo el flujo.
Las pruebas de integración comprueban las dependencias reales.

## Pruebas

```bash
make -C proyecto3 check-reference
make -C proyecto3 check
make -C proyecto3 check-integration
```

- `check-reference`: 14 pruebas Python y comprobaciones del esqueleto/contratos.
- `check`: lo anterior, compilación Go/Rust, pruebas unitarias y `go vet`.
- `check-integration`: inicia el entorno local, comprueba Lua/concurrencia/publicación
  y ejecuta nueve escenarios de extremo a extremo. Detiene temporalmente RabbitMQ,
  Valkey y reinicia el consumidor de **este proyecto** para comprobar recuperación.
  Deja los servicios activos y conserva las corridas para inspección.

Los resultados de la última integración se escriben en
`artifacts/integration-latest.json` (archivo local ignorado por Git).
El gancho opcional `pre-push` y `ci/check.sh` reutilizan `make check`; no se ha
activado configuración Git ni un workflow remoto automáticamente.

Carga básica con dos usuarios durante diez segundos:

```bash
RUN_ID=locust-prueba docker compose -f proyecto3/deploy/compose.local.yaml run --rm locust
```

[Locust](load/README.md) lee `load/load-profile.json`. La descarga de este archivo
como artefacto OCI se implementará junto con Zot. Esta prueba breve no sustituye
la evaluación de carga, HPA y réplicas de la fase 6.

## Datos y errores

El [contrato](contracts/README.md) limita países a USA/RUS/CHN/ESP/GTM, aviones
0–50, barcos 0–30 y timestamps con zona y hasta seis decimales. Rechaza campos
extra/repetidos, tipos incorrectos y fechas imposibles. El cuerpo máximo es 4 KiB.
Los servicios propagan un plazo máximo de cinco segundos para aceptar un reporte.

RabbitMQ usa cola/exchange durables y mensajes persistentes. El writer espera
confirmación y verifica el enrutamiento. El consumidor confirma solo al guardar o
comprobar un duplicado; los conflictos y mensajes inválidos se conservan en
`mumn.reports.failed` con un motivo.

Valkey guarda un hash por corrida (`mumn:run:<run_id>`): eventos únicos, conteos,
histogramas y último estado de cada país. Lua valida antes de una única escritura
para evitar modificaciones parciales; un evento tardío no reemplaza el estado reciente.
No se aplica TTL parcial. El entorno local tiene 128 MiB de límite y `noeviction`;
si se agota, el consumo espera y registra el fallo.

Valkey usa AOF con sincronización cada segundo y volumen persistente. La prueba de
reinicio comprueba el volumen, pero no garantiza pérdida cero ante caída abrupta
del host. La instalación local de una instancia no proporciona alta disponibilidad.

## Detener

```bash
make -C proyecto3 local-down
```

Conserva los volúmenes. No usar `down -v` si se desean conservar los reportes.

## Entrega académica

Fecha límite: **22/10/2026**. Evaluación: **24/10/2026**. Mantener código,
Dockerfiles, manifiestos, evidencias y manual Markdown en este repositorio privado.
Verificar acceso de `JoseLorenzana272` y `KINGR0X` y completar entrega en UEDI.
Este README documenta el avance local; el manual final se ampliará con el despliegue.
