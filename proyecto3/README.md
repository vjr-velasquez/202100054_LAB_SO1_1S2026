# Proyecto 3 — M.U.M.N.K8s

Estudiante: Victor Hugo Velásquez Hernández. Carné: **202100054**.
País asignado: **CHN (China)**. Ruta principal: `/grpc-202100054`.

## Estado

Estructura inicial, contratos JSON/protobuf y validaciones locales disponibles.
Los servicios Rust/Go, Operator, despliegue y pruebas de carga están pendientes.

## Comprobaciones locales

Desde la raíz del repositorio, con Python 3.10 o posterior y Make:

```bash
make -C proyecto3 check
make -C proyecto3 doctor
```

`check` valida la estructura, los contratos y los datos de referencia. No acredita
compilación ni funcionamiento de servicios. `doctor` informa herramientas ausentes
sin instalar paquetes ni conectarse a la nube.

Los contratos están en [contracts/](contracts/README.md); los resultados de referencia,
en `tests/fixtures/`. Los scripts para CI están en [ci/](ci/README.md).

## Trabajo por fases

Cada fase se desarrolla en una rama propia y nace del último commit aceptado de
la anterior. Ramas iniciales: `proyecto3-fase1-planificacion` y
`proyecto3-fase2-estructura`.

## Entrega

- Fecha límite: **22 de octubre de 2026**. Evaluación: **24 de octubre de 2026**.
- Código, Dockerfiles, manifiestos, evidencias y manual técnico Markdown en el
  mismo repositorio privado, dentro de `proyecto3/`.
- Acceso de los auxiliares `JoseLorenzana272` y `KINGR0X` y entrega en UEDI.
- El manual técnico y la guía de instalación se incorporarán en una ruta versionada
  al avanzar la implementación. Este README todavía no es el manual final.
