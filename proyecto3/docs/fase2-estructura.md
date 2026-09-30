# Fase 2 — Estructura y validación local

Rama: `proyecto3-fase2-estructura`, creada desde la fase 1 aceptada.

## Estructura

```text
proyecto3/
├── contracts/            JSON Schema y contrato protobuf
├── services/rust-api/    espacio para la API Rust de la fase 3
├── services/go/          espacio para los roles Go de la fase 3
├── operator/             espacio para Operator y charts de la fase 4
├── deploy/               espacio para manifiestos Kubernetes
├── infra/                espacio para GCP y Zot
├── load/                 espacio para Locust
├── scripts/              validación y diagnóstico
├── tests/fixtures/       reportes y resultados esperados
├── ci/                   entrada reutilizable para CI
├── .githooks/            plantilla opcional pre-push
└── docs/                 planificación y evidencias por fase
```

Las carpetas de servicios contienen README para conservarlas en Git sin simular
una implementación. Se decidirán los deployments después de aclarar D02.

## Comandos

Desde la raíz del repositorio:

```bash
make -C proyecto3 check
make -C proyecto3 doctor
sh proyecto3/ci/check.sh
```

`check` necesita Python 3.10 o posterior y Make; no instala paquetes, no levanta
contenedores ni accede a la nube. Comprueba estructura/enlaces, declaraciones del
contrato y el dataset, y ejecuta las pruebas del gancho y del oráculo estadístico.
El oráculo es una referencia de pruebas; no sustituye al consumidor ni demuestra
atomicidad, ACK, persistencia o deduplicación entre procesos.

`doctor` comprueba presencia de herramientas en PATH. Devuelve error si falta alguna
del inventario del proyecto; eso no invalida las pruebas locales disponibles.
Todavía deben elegirse versiones compatibles antes de instalar o compilar servicios.

`check-integration` y `check-cluster` devuelven error explícito de pendiente. Se
implementarán en sus fases, sin convertir un chequeo omitido en éxito.

## Gancho opcional

La plantilla `.githooks/pre-push` llama al mismo `make check`. No está instalada en
la configuración Git. Se puede invocar manualmente desde el repositorio con:

```bash
sh proyecto3/.githooks/pre-push
```

Por ahora el gancho valida el árbol de trabajo actual, no una instantánea de todos
los commits que se publican. Debe evolucionar junto con los servicios. El script
`ci/check.sh` está listo para conectarse al proveedor CI; no hay workflow remoto
configurado en esta fase.

## Criterios cubiertos

- Reportes válidos y valores límite; rechazo de tipos, campos y fechas incorrectos.
- Dataset de cinco eventos con resultados explícitos y verificables.
- Reentrega frente a nuevo evento con igual contenido; conflicto de identidad.
- Llegadas tardías inéditas, empates, mismo timestamp y corrida vacía.
- Alteración deliberada del conteo esperado en archivo temporal: el comando falla.

Estos casos son pruebas locales de referencia. Las futuras implementaciones Rust,
Go y Valkey deberán pasar casos equivalentes mediante sus interfaces reales.

## Verificación realizada el 29/09/2026

- `make -C proyecto3 check`: 14 pruebas locales aprobadas, incluida detección de
  un conteo alterado deliberadamente en un archivo temporal.
- Sintaxis de los scripts shell verificada con `sh -n`.
- Targets de integración y clúster: comprobado que devuelven error de pendiente.
- Diagnóstico: presentes Python, Make, Git, Go y Docker en PATH; faltan `rustc`,
  `cargo`, `protoc`, `kubectl`, `helm`, `operator-sdk`, `gcloud`, `oras` y `locust`.
- No se ejecutaron pruebas de servicios, Docker, Operator, CI remoto ni GKE.
- Revisión con subagente: se fijó precisión máxima de seis decimales en timestamps
  y normalización UTC antes de deduplicar; ambas condiciones tienen pruebas locales.

## Pendientes de fases posteriores

- Servicios, Dockerfiles, dependencias y compilación: fase 3.
- Compilación real del protobuf y generación de clientes/servidores: fase 3.
- Operator, manifiestos y dashboards: fase 4.
- Aprovisionamiento, registro HTTPS y publicación: fase 5.
- Carga y validación distribuida: fases 3–6.
- Aclaraciones D02/D03 y datos GCP: continúan pendientes.
