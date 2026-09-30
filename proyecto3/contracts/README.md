# Contratos versionados

- `report.schema.json`: estructura del JSON público. Los límites elegidos son los
  rangos sugeridos por el enunciado, aprobados como base del diseño.
- `war_report.proto`: propuesta gRPC con números del enunciado preservados.
  `go_package` fija la ruta propuesta para el módulo futuro; todavía no existe
  código generado ni se ha compilado con `protoc`.

El ID viaja en cabeceras REST y metadatos gRPC, sin agregar campos al JSON académico.
Véase [contrato completo](../docs/planificacion/02-contratos.md).

`make check` comprueba las reglas de dominio con Python estándar y una revisión
estructural acotada de estos archivos. No es un motor general JSON Schema ni un
compilador protobuf. La validación real de servicios se añadirá en la fase 3.
El formato `date-time` debe validarse explícitamente en cada implementación.

Para esta versión, la política de entrada exige números JSON escritos como enteros:
por ejemplo, `10.0` se rechaza aunque algunos validadores JSON Schema lo consideren
un entero matemático. Esta política adicional se verifica en los casos locales.
Los timestamps admiten hasta seis decimales de segundo; una precisión mayor se
rechaza, nunca se trunca. Esta restricción adicional al formato `date-time` evita
que el oráculo pierda precisión y debe conservarse en las implementaciones futuras.
Normalizar UTC antes de comparar identidades permite reentregas con zonas equivalentes.
