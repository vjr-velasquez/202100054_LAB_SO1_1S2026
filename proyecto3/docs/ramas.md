# Trabajo por ramas

Cada fase se desarrolla en una rama propia, siguiendo el patrón usado en el Proyecto 2.
Las ramas son acumulativas: la siguiente nace del último commit aceptado de la anterior.
No se requiere crear ni cambiar una rama `main` para comenzar.

| Fase | Rama | Contenido |
|---|---|---|
| 1 | `proyecto3-fase1-planificacion` | Arquitectura, contratos documentados y matriz. |
| 2 | `proyecto3-fase2-estructura` | Carpetas, contratos versionados y validación local. |
| 3 | `proyecto3-fase3-flujo` | Rust, Go, gRPC, RabbitMQ y consumidor. |
| 4 | `proyecto3-fase4-operator-grafana` | Operator, Valkey y dashboards. |
| 5 | `proyecto3-fase5-gcp-zot` | GKE, Zot, OCI y Gateway. |
| 6 | `proyecto3-fase6-carga` | Locust, HPA, réplicas y recuperación. |
| 7 | `proyecto3-fase7-entrega` | Manual, evidencias y revisión final. |

La rama inicial parte de `f991e23` de `proyecto2-fase4-ebpf`, el HEAD existente al
empezar el Proyecto 3. Conserva el historial del mismo repositorio. Los cambios
sin commit del Proyecto 2 no forman parte de los commits del Proyecto 3.

## Ciclo de una fase

1. Crear la rama desde la fase anterior aceptada.
2. Modificar únicamente el alcance autorizado.
3. Ejecutar las comprobaciones disponibles y registrar las pendientes.
4. Revisar el diff y añadir rutas explícitas; evitar `git add .` mientras existan
   cambios de otro proyecto en el directorio de trabajo.
5. Crear un commit descriptivo y publicar la rama.
6. Presentar resultados para aceptación antes de iniciar la siguiente fase.

Ejemplo para publicar la planificación desde la raíz del repositorio:

```bash
git push -u origin proyecto3-fase1-planificacion
```

Ejemplo para publicar la estructura:

```bash
git push -u origin proyecto3-fase2-estructura
```

Los comandos son instrucciones de publicación, no evidencia de un push realizado.
La publicación requiere acceso al remoto. No incluir secretos, archivos `.env`,
binarios ni credenciales en los commits. No crear anticipadamente las ramas de
fases futuras: deben partir del resultado aceptado de su predecesora.
