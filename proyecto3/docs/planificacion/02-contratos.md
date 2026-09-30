# Contratos propuestos — Datos, comunicación y métricas

Estado: propuesta del paso 1; pendiente de aceptación. Los límites y semánticas que
no están definidos por el enunciado son decisiones de diseño, no requisitos atribuidos a la cátedra.

## Entrada pública

`POST /grpc-202100054`, `Content-Type: application/json`.

```json
{
  "country": "ESP",
  "warplanes_in_air": 42,
  "warships_in_water": 14,
  "timestamp": "2026-03-12T20:15:30Z"
}
```

| Campo | Contrato propuesto |
|---|---|
| `country` | String exacto en mayúsculas: `USA`, `RUS`, `CHN`, `ESP`, `GTM`. |
| `warplanes_in_air` | Entero JSON, inclusivo 0–50; rechazar nulos, strings y fracciones. |
| `warships_in_water` | Entero JSON, inclusivo 0–30; rechazar nulos, strings y fracciones. |
| `timestamp` | Fecha RFC 3339 con zona; normalizar UTC. Admitir hasta seis decimales de segundo, sin truncamiento silencioso. Rechazar fechas imposibles o sin zona. |

Los cuatro campos son obligatorios; propuesta: rechazar campos extra para detectar
errores de contrato. No rechazar reportes históricos válidos por antigüedad: el
ejemplo académico y las pruebas controladas pueden usar fechas anteriores.
Límite inicial propuesto del cuerpo: 4 KiB, configurable y documentado.

## Identidad y corridas

- Cabecera opcional `Idempotency-Key`: UUID generado por Locust para cada evento
  lógico y reutilizado al reintentar. Si falta, Rust genera un UUID nuevo.
- Cabecera opcional `X-Run-ID`: identifica el conjunto de prueba; si falta, usar la
  corrida activa configurada. Todos los componentes deben conservar ese valor.
- El JSON externo permanece igual al del enunciado. La identidad viaja en cabeceras,
  metadatos gRPC y propiedades/envoltura interna de mensajería.
- La deduplicación es por `(run_id, event_id)`. Dos reportes con campos iguales y
  UUID diferentes representan dos reportes legítimos; no deduplicar por timestamp.
- Reutilizar el mismo UUID con contenido diferente es un conflicto: no sobrescribir
  el original; registrar y enviar a la cola de fallos. La detección puede ser posterior
  al HTTP 202, porque se realiza al consumir. La respuesta de aceptación no promete
  que ese contenido vaya a contabilizarse.
- Sin una clave conservada por el cliente no se garantiza deduplicar reintentos HTTP.

## Semántica de respuesta

Éxito propuesto:

```json
{"status":"accepted","event_id":"<uuid>","run_id":"<corrida>"}
```

`202 Accepted` solo después de confirmación del broker y comprobación de enrutamiento
a la cola esperada. Significa **aceptado por RabbitMQ**, no persistido todavía en Valkey.

| HTTP | Situación |
|---|---|
| 400 | JSON, campos, cabeceras o valores inválidos. |
| 413 | Cuerpo por encima del límite. |
| 415 | Tipo de contenido no admitido. |
| 429 | Límite de concurrencia local alcanzado; reintentar con la misma identidad. |
| 503 | Dependencia no disponible o publicación rechazada. |
| 504 | Plazo agotado; resultado de publicación puede ser incierto. |
| 500 | Error interno no clasificado; registrar correlación sin exponer credenciales. |

Error propuesto: `{"error":{"code":"...","message":"..."},"event_id":"..."}`.
La identidad se omite si el error ocurrió antes de asignarla.
Un timeout no prueba que el mensaje no haya llegado: conservar la clave al reintentar.

## Saltos REST y gRPC

| Salto | Contrato propuesto |
|---|---|
| Rust → API Go | `POST /reports` por Service interno, mismo JSON y cabeceras. |
| API Go → cliente gRPC compañero | `POST /reports` por loopback; transforma al protobuf. |
| Cliente → servidor gRPC | `WarReportService.SendReport`; campos y números del proto académico preservados. |
| Servidor → writer compañero | `POST /publish` por loopback; propagar datos e identidad. |
| Writer → RabbitMQ | Publicación persistente; esperar confirmación y comprobar retorno por falta de ruta. |

Mantener los números del enum del enunciado: `unknown=0`, `usa=1`, `rus=2`,
`chn=3`, `esp=4`, `gtm=5`. Rechazar 0 y valores desconocidos; conversión explícita
desde los códigos JSON, sin depender del orden de una lista.
El nombre de paquete es `wartweets`; definir el `go_package` real al crear el módulo,
evitando usar el import relativo del ejemplo como decisión definitiva.
La respuesta protobuf conserva `status`; `accepted` solo tras confirmación del writer.
Los metadatos `event-id` y `run-id` conservan identidad sin cambiar los cuatro campos.

Mapear `InvalidArgument` a 400, `ResourceExhausted` a 429, `Unavailable` a 503,
`DeadlineExceeded` a 504 e `Internal` a 500. No convertir errores internos en éxitos.
Propagar un presupuesto temporal finito entre saltos y cancelar trabajo cuando venza;
los valores concretos se fijarán en configuración y se medirán en la fase de carga.

## Mensajería y almacenamiento

Envoltura interna propuesta, versión 1:

```json
{
  "schema_version": 1,
  "event_id": "<uuid>",
  "run_id": "<corrida>",
  "received_at": "2026-09-29T18:00:00Z",
  "report": {
    "country": "CHN",
    "warplanes_in_air": 20,
    "warships_in_water": 8,
    "timestamp": "2026-09-29T18:00:00Z"
  }
}
```

Propuesta: exchange directo `mumn.reports`, routing key `report.v1`, cola
`mumn.reports.v1` y cola de fallos `mumn.reports.failed`. Fijar también `message_id`
al UUID. Versionar el contrato y rechazar versiones no soportadas.

El consumidor actualiza identidad procesada, reporte e índices en una operación
atómica de Valkey, mediante script con validaciones previas a cualquier escritura.
No asumir rollback de un script que falla después de escribir. Diseñar claves de tipos
conocidos y probar errores de esquema/tipo; detener consumo si se detecta corrupción.
Un duplicado con igual contenido no incrementa ningún contador. ACK manual después
de confirmar almacenamiento, o después de comprobar que ya fue procesado.
Ante Valkey indisponible, pausar/reintentar con espera acotada sin ACK ni bucle intensivo;
ante datos inválidos, enviar a cola de fallos con motivo y verificar la transferencia
antes de retirar el original. Evitar descarte silencioso.

Confirmación del publicador y ACK del consumidor cubren etapas distintas; no equivalen
a entrega exactamente una vez. Se propone entrega al menos una vez con procesamiento
idempotente. Fuente: [confirmaciones de RabbitMQ](https://www.rabbitmq.com/docs/confirms).

Modelo lógico por corrida:

| Información | Estructura propuesta |
|---|---|
| Identidad y reporte | Registro por UUID, incluyendo contenido canónico para detectar conflictos. |
| Serie temporal | Índice ordenado por timestamp de evento y desempate por UUID; puntos separados aunque compartan tiempo. |
| Frecuencias globales | Histogramas 0–50 y 0–30 para calcular moda y extremos exactos. |
| Último estado de país | Reporte con máximo `(timestamp, event_id)`; llegadas tardías no sustituyen un estado más reciente. |
| Conteos | Total global y por país de reportes únicos procesados. |

Usar namespace de claves por corrida. No expirar claves de deduplicación antes que sus
reportes y agregados. Propuesta inicial: corrida activa sin TTL parcial, memoria limitada
con política sin eviction y limpieza explícita de corridas cerradas después de conservar
evidencia. Si se agrega TTL, aplicarlo coherentemente a la corrida completa y probarlo.
La duración/carga debe caber en memoria; agotamiento es fallo visible, no descarte automático.
Configurar y probar persistencia en disco; documentar la ventana de pérdida según la
política AOF elegida. Reiniciar un pod no demuestra tolerancia a pérdida del disco.

## Definición verificable de los paneles

Propuesta D05: las estadísticas globales representan todos los reportes únicos
procesados de la **corrida seleccionada**. Los filtros temporales afectan solo a las
series y se rotulan como tales; cambiar el rango visual no cambia silenciosamente
el universo de los otros paneles. El dashboard indica corrida y alcance.

| Panel | Definición propuesta |
|---|---|
| Máximo/mínimo de aviones | Extremos del valor `warplanes_in_air` de todos los reportes de la corrida. |
| Máximo/mínimo de barcos | Extremos del valor `warships_in_water` de todos los reportes de la corrida. |
| Top de países: aviones | Orden descendente del último valor de aviones por país; desempate por código ascendente. |
| Top de países: barcos | Orden descendente del último valor de barcos por país; desempate por código ascendente. |
| Moda de aviones/barcos | Valores con frecuencia máxima global; mostrar todos los empates ordenados y su frecuencia. |
| Serie CHN | Dos series, una por recurso; usar timestamp del evento, conservar llegadas fuera de orden. |
| País asignado | Texto `China (CHN)`. |
| Total CHN | Cantidad de reportes únicos CHN procesados de la corrida, no intentos HTTP. |

Sin datos: mostrar “Sin datos” para extremos, moda y series; total CHN = 0.
Países sin reportes no tienen un último estado: no inventar valores cero en rankings.
Los rankings no suman observaciones sucesivas del mismo estado; si la cátedra solicita
otra definición, ajustar contrato y resultados esperados antes de implementar.
Refresco propuesto: 5 segundos. Medir retraso de extremo a extremo separadamente de
la latencia HTTP; no usar la etiqueta “tiempo real” como garantía de latencia no medida.

## Conjunto de aceptación pequeño

Datos dentro de una corrida nueva; `e1`–`e5` son alias de UUID distintos.

| Evento | País | Aviones | Barcos | Hora UTC del mismo día |
|---|---|---:|---:|---|
| e1 | CHN | 10 | 2 | 12:00:00 |
| e2 | USA | 20 | 5 | 12:01:00 |
| e3 | CHN | 20 | 5 | 12:02:00 |
| e4 | ESP | 0 | 0 | 12:03:00 |
| e5 | CHN | 30 | 8 | 12:04:00 |

Resultados esperados calculados para revisión:

- Aviones: mínimo 0, máximo 30, moda 20 con frecuencia 2.
- Barcos: mínimo 0, máximo 8, moda 5 con frecuencia 2.
- Ranking aviones: CHN 30, USA 20, ESP 0.
- Ranking barcos: CHN 8, USA 5, ESP 0.
- CHN: 3 reportes; serie aviones `[10,20,30]`, barcos `[2,5,8]`.
- Total global procesado: 5. Reentregar e3 no cambia ningún resultado.
- En otra corrida vacía, entregar primero e5 y luego e1, ambos inéditos: CHN mantiene
  último estado 30/8, su conteo es 2 y la serie queda ordenada e1 → e5. Este ensayo
  comprueba llegada tardía independientemente de la deduplicación.
- Repetir el cuerpo de e3 con un UUID nuevo sí agrega un reporte.

Agregar casos independientes para corrida vacía, empate de modas, empate de rankings,
timestamp idéntico, extremos permitidos, datos inválidos y UUID con contenido conflictivo.
Estos son criterios planificados, **no resultados de pruebas ejecutadas**.
