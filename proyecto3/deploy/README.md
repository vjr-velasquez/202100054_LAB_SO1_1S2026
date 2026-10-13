# Entorno local

`compose.local.yaml` crea el proyecto Docker `mumn-p3`, con red y volúmenes propios.
Solo la API Rust publica puerto: `127.0.0.1:18080` (MUMN_PORT permite cambiarlo).
RabbitMQ y Valkey no tienen puertos publicados en el host.

`grpc-client` comparte red con `go-api`; `writer` comparte red con `grpc-server`.
Esto comprueba la comunicación por loopback de los futuros pods de dos contenedores.
No representa todavía los deployments de GKE. La cantidad final de deployments
publicadores y la topología de réplica Valkey siguen pendientes de aclaración.

Las credenciales incluidas son exclusivas del entorno local de desarrollo. La fase
GKE usará Secrets, Zot y la configuración de infraestructura correspondiente.
`down` conserva los volúmenes. No usar `down -v` si se quieren conservar los datos.
