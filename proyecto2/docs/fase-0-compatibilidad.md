# Fase 0 — Compatibilidad del entorno

## Objetivo

Confirmar que la máquina Linux donde se ejecutará el Proyecto 2 puede:

1. Compilar y cargar un módulo de kernel.
2. Compilar programas eBPF.
3. Ejecutar el Daemon en Go.
4. Administrar contenedores Docker.
5. Ejecutar Grafana y Valkey mediante Docker Compose.

Esta fase no implementa todavía ningún componente funcional.

## Decisión de despliegue

El módulo de kernel, la sonda eBPF, el Daemon y los contenedores de carga deben ejecutarse sobre la misma máquina Linux. Esto es necesario porque las métricas de `/proc`, los procesos observados y los eventos eBPF pertenecen al kernel del anfitrión.

El Daemon se ejecutará inicialmente en el anfitrión y utilizará Docker para administrar:

- Contenedores de carga.
- Grafana.
- Valkey.

## Línea base esperada

| Recurso | Condición |
|---|---|
| Sistema | Ubuntu 24.04 LTS o distribución Linux compatible |
| Kernel | Headers correspondientes al kernel activo |
| Módulos | Capacidad de usar `insmod` y `rmmod` con privilegios |
| eBPF | Clang/LLVM, bpftool, BTF y tracepoints |
| Cgroups | cgroup v2 recomendado |
| Go | Go 1.22 o superior recomendado |
| Docker | Engine activo y acceso al daemon |
| Compose | Comando `docker compose` disponible |
| Puertos | TCP 3000 para Grafana y 6379 para Valkey |
| Permisos | Usuario con `sudo` para kernel y eBPF |

No se fijará una versión exacta del kernel hasta revisar la salida real del diagnóstico.

## Ejecución del diagnóstico

Desde la raíz del repositorio:

```bash
git switch proyecto2-fase0
cd proyecto2
bash scripts/check_environment.sh | tee fase0-entorno.txt
```

El script es de solo lectura: no instala paquetes ni cambia configuración.

## Cómo interpretar el resultado

- `OK`: requisito disponible.
- `WARN`: no bloquea inmediatamente, pero debe revisarse.
- `FAIL`: impide compilar o ejecutar una parte obligatoria.

La fase 0 se considera aprobada cuando no quedan resultados `FAIL` relacionados con:

- Docker.
- Docker Compose.
- Go.
- GCC y Make.
- Headers del kernel.
- Clang y LLVM.
- bpftool.
- BTF.

## Comprobaciones manuales adicionales

### Permisos del kernel

```bash
sudo -v
sudo lsmod | head
```

### Docker

```bash
sudo docker run --rm hello-world
docker compose version
```

### Tracepoint de kill

```bash
sudo test -r /sys/kernel/tracing/events/syscalls/sys_enter_kill/format \
  && echo "sys_enter_kill disponible"
```

Si `tracefs` no está montado, se debe revisar la configuración antes de continuar. No se montará automáticamente desde el diagnóstico.

## Resultado que debe conservarse

Guardar `fase0-entorno.txt` como evidencia local. Antes de incorporarlo al repositorio se debe revisar que no contenga nombres de host, direcciones o datos que no se deseen publicar.

## Siguiente fase

Cuando el entorno sea compatible se iniciará la fase 1:

- Automatización de los perfiles `go-client` y `alpine`.
- Etiquetas de identificación del proyecto.
- Imagen intrusa controlada.
- Script de creación aleatoria de cinco contenedores.
