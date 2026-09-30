# Proyecto 3 — M.U.M.N.K8s

Estudiante: Victor Hugo Velásquez Hernández. Carné: **202100054**.
País asignado: **CHN (China)**. Ruta principal: `/grpc-202100054`.

## Estado

El usuario aprobó el **paso 1: arquitectura, contratos y matriz de requisitos** y
autorizó el **paso 2: estructura y validaciones locales**. La planificación no
acredita implementación de servicios ni pruebas de despliegue ejecutadas.

Fuente de requisitos: `Proyecto3-M.U.M.N.md`, proporcionado por el usuario.
El documento se interpreta como enunciado académico, no como autorización para
aprovisionar recursos, publicar imágenes o modificar accesos al repositorio.

## Planificación para revisar

1. [Arquitectura, decisiones y fases](docs/planificacion/01-arquitectura.md).
2. [Contratos de datos, comunicación y métricas](docs/planificacion/02-contratos.md).
3. [Matriz de requisitos, aceptación y gancho de calidad](docs/planificacion/03-requisitos-y-validacion.md).
4. [Trabajo por ramas y publicación](docs/ramas.md).
5. [Fase 2: estructura y validaciones locales](docs/fase2-estructura.md).

Comprobación local desde la raíz del repositorio:

```bash
make -C proyecto3 check
```

Comprueba el esqueleto, los contratos y los datos de referencia; todavía no acredita
servicios implementados ni integración. `make -C proyecto3 doctor` muestra las
herramientas pendientes del entorno completo.

Las propuestas de diseño son la base aprobada para el desarrollo. Las contradicciones de
la cátedra siguen pendientes hasta contar con aclaración o adoptar una interpretación
documentada; la aprobación de esta planificación no equivale a validación de la cátedra.

## Entrega

- Fecha límite del enunciado: **22 de octubre de 2026**.
- Evaluación: **24 de octubre de 2026**.
- Mismo repositorio privado de los proyectos anteriores, carpeta `proyecto3/`.
- Manual técnico y guía de instalación en Markdown, código, Dockerfiles,
  manifiestos, resultados y evidencias funcionales.
- Acceso de los auxiliares `JoseLorenzana272` y `KINGR0X`, por verificar al cierre.
- Entrega en UEDI. El enunciado establece nota cero si falta esta entrega o el manual Markdown.

Este README es el índice de planificación; **todavía no es el manual técnico final**.
