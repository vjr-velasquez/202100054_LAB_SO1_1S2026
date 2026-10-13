# Servicios Go

Un módulo con cinco roles ejecutables desde `cmd/service`:

| Rol | Interfaz | Destino configurable |
|---|---|---|
| `api` | HTTP :8081, POST /reports | GRPC_BRIDGE_URL |
| `grpc-client` | HTTP :8082, POST /reports | GRPC_TARGET |
| `grpc-server` | gRPC :50051, SendReport | WRITER_URL |
| `writer` | HTTP :8083, POST /publish | RABBITMQ_URL |
| `consumer` | Consumo AMQP | RABBITMQ_URL y VALKEY_ADDR |

Los roles HTTP exponen `/healthz` como salud del proceso. No acredita disponibilidad
de dependencias. Todas las direcciones de escucha se pueden ajustar con LISTEN_ADDR.
La API Rust genera identidad cuando falta; los servicios internos exigen identidad
válida y conservan el plazo recibido.

El writer mantiene cuatro sesiones independientes, una publicación en vuelo por
sesión, con confirmación y comprobación de retorno AMQP. La cancelación cierra el
socket, incluso si una operación de la biblioteca no atiende el contexto.

El consumidor usa ACK manual, prefetch 32 y procesamiento secuencial por instancia.
Se puede aumentar la cantidad de consumidores. Si Valkey falla, conserva el mensaje
sin ACK y espera un segundo antes de reintentar. Las identidades conflictivas y los
mensajes inválidos se transfieren a la cola de fallos con confirmación antes del ACK.

Los reportes y sus estadísticas se guardan en un hash por corrida, mediante Lua
con una única escritura final después de validar. Un mismo ID y contenido normalizado
no incrementa estadísticas; el mismo ID con contenido distinto es un conflicto.

Pruebas de dominio/transporte se ejecutan durante la construcción Docker. Las pruebas
con servicios reales llevan el build tag `integration` y requieren las variables
VALKEY_ADDR y RABBITMQ_URL; no omiten silenciosamente la verificación si faltan.

Fuentes: [gRPC Go](https://grpc.io/docs/languages/go/quickstart/) y
[cliente RabbitMQ](https://pkg.go.dev/github.com/rabbitmq/amqp091-go).
