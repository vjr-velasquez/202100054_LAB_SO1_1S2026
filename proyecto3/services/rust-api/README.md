# API REST Rust

`POST /grpc-202100054` recibe el JSON del contrato, valida campos y normaliza UTC.
Acepta cabeceras opcionales `Idempotency-Key` (UUID) y `X-Run-ID` (1–64 caracteres
alfanuméricos, guion o guion bajo). Si faltan, genera UUID y usa RUN_ID (default).

GO_API_URL configura el siguiente salto. LISTEN_ADDR configura la escucha
(predeterminado 0.0.0.0:8080). El cuerpo máximo es 4 KiB y hay hasta 128 solicitudes
en procesamiento; el plazo hacia Go es de cinco segundos.

Devuelve 202 únicamente cuando recibe confirmación consistente desde Go, que a su
vez espera al broker. No garantiza que el consumidor ya haya guardado el reporte.
`GET /healthz` indica salud del proceso. Los errores usan estados HTTP 400, 413,
415, 429, 502, 503 o 504 según el caso.

Se compila con Docker; Cargo.lock conserva la resolución de dependencias. Las pruebas
Rust de validación se ejecutan como parte de la construcción.

Framework: [Axum](https://docs.rs/axum/latest/axum/).
