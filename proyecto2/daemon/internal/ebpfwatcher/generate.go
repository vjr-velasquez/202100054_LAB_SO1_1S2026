package ebpfwatcher

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go@v0.17.3 -target bpfel -cc clang killMonitor ./bpf/kill_monitor.bpf.c -- -I./bpf -O2 -g -D__TARGET_ARCH_x86 -Wno-missing-declarations
