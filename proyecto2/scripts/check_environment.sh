#!/usr/bin/env bash

set -u

PASS=0
WARN=0
FAIL=0

ok() {
  PASS=$((PASS + 1))
  printf '[OK]   %s\n' "$1"
}

warn() {
  WARN=$((WARN + 1))
  printf '[WARN] %s\n' "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  printf '[FAIL] %s\n' "$1"
}

section() {
  printf '\n## %s\n' "$1"
}

check_command() {
  local command_name="$1"
  local version_args="${2:---version}"

  if command -v "$command_name" >/dev/null 2>&1; then
    ok "$command_name está instalado"
    "$command_name" $version_args 2>/dev/null | head -n 1 || true
  else
    fail "$command_name no está instalado"
  fi
}

kernel_release="$(uname -r)"

printf 'Diagnóstico de compatibilidad — Proyecto 2 SO1\n'
printf 'Fecha UTC: %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
printf 'Host: %s\n' "$(hostname)"
printf 'Usuario: %s (UID %s)\n' "$(id -un)" "$(id -u)"

section "Sistema operativo"
uname -a
if [ -r /etc/os-release ]; then
  . /etc/os-release
  printf 'Distribución: %s\n' "${PRETTY_NAME:-desconocida}"
fi

case "$(uname -m)" in
  x86_64|aarch64)
    ok "Arquitectura soportada: $(uname -m)"
    ;;
  *)
    warn "Arquitectura no validada para el proyecto: $(uname -m)"
    ;;
esac

section "Herramientas de construcción"
check_command gcc
check_command make
check_command go version
check_command clang
check_command llvm-strip
check_command git

section "Docker"
if command -v docker >/dev/null 2>&1; then
  ok "Docker CLI está instalado"
  docker --version || true

  if docker info >/dev/null 2>&1; then
    ok "El daemon de Docker responde"
    docker info --format 'Docker Server={{.ServerVersion}} Cgroup={{.CgroupVersion}} Driver={{.CgroupDriver}}' || true
  else
    fail "El daemon de Docker no responde o el usuario no tiene permisos"
  fi

  if docker compose version >/dev/null 2>&1; then
    ok "Docker Compose está disponible"
    docker compose version || true
  else
    fail "Falta el plugin docker compose"
  fi
else
  fail "Docker no está instalado"
fi

section "Kernel y módulos"
printf 'Kernel activo: %s\n' "$kernel_release"

if [ -d "/lib/modules/$kernel_release/build" ]; then
  ok "Headers del kernel disponibles"
else
  fail "Faltan headers para $kernel_release"
fi

if [ -r /proc/modules ]; then
  ok "La lista de módulos del kernel es accesible"
else
  fail "No se puede leer /proc/modules"
fi

if command -v modprobe >/dev/null 2>&1; then
  ok "modprobe está disponible"
else
  fail "modprobe no está disponible"
fi

section "eBPF"
check_command bpftool version

if [ -r /sys/kernel/btf/vmlinux ]; then
  ok "BTF del kernel disponible en /sys/kernel/btf/vmlinux"
else
  fail "BTF del kernel no está disponible"
fi

if [ -d /sys/kernel/tracing ]; then
  ok "tracefs está disponible"
else
  warn "No se encontró /sys/kernel/tracing"
fi

if [ -r /sys/kernel/tracing/events/syscalls/sys_enter_kill/format ]; then
  ok "Tracepoint sys_enter_kill disponible"
elif [ -r /sys/kernel/debug/tracing/events/syscalls/sys_enter_kill/format ]; then
  ok "Tracepoint sys_enter_kill disponible mediante debugfs"
else
  warn "No se encontró el tracepoint sys_enter_kill; puede requerir montar tracefs o usar otro punto de enlace"
fi

kernel_config=""
if [ -r "/boot/config-$kernel_release" ]; then
  kernel_config="/boot/config-$kernel_release"
elif [ -r /proc/config.gz ] && command -v zcat >/dev/null 2>&1; then
  kernel_config="/proc/config.gz"
fi

if [ -n "$kernel_config" ]; then
  ok "Configuración del kernel disponible"
  if [ "$kernel_config" = "/proc/config.gz" ]; then
    zcat "$kernel_config" | grep -E '^CONFIG_(BPF|BPF_SYSCALL|BPF_JIT|DEBUG_INFO_BTF|KPROBES|TRACEPOINTS)=' || true
  else
    grep -E '^CONFIG_(BPF|BPF_SYSCALL|BPF_JIT|DEBUG_INFO_BTF|KPROBES|TRACEPOINTS)=' "$kernel_config" || true
  fi
else
  warn "No fue posible leer la configuración del kernel"
fi

section "Cgroups"
cgroup_type="$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)"
printf 'Tipo: %s\n' "${cgroup_type:-desconocido}"

if [ "$cgroup_type" = "cgroup2fs" ]; then
  ok "cgroup v2 está activo"
  if [ -r /sys/fs/cgroup/cgroup.controllers ]; then
    printf 'Controladores: '
    tr '\n' ' ' < /sys/fs/cgroup/cgroup.controllers
    printf '\n'
  fi
else
  warn "El entorno no usa cgroup v2; la detección de contenedores deberá adaptarse"
fi

section "Recursos y puertos"
free -h 2>/dev/null || true
df -h / 2>/dev/null || true

for port_number in 3000 6379; do
  if command -v ss >/dev/null 2>&1 && ss -ltn 2>/dev/null | awk '{print $4}' | grep -Eq "[:.]$port_number$"; then
    warn "El puerto TCP $port_number ya está en uso"
  else
    ok "El puerto TCP $port_number parece disponible"
  fi
done

section "Resumen"
printf 'OK: %s | WARN: %s | FAIL: %s\n' "$PASS" "$WARN" "$FAIL"

if [ "$FAIL" -gt 0 ]; then
  printf 'Resultado: entorno incompleto. Corrige los FAIL antes de implementar.\n'
  exit 1
fi

printf 'Resultado: compatibilidad base aprobada.\n'
