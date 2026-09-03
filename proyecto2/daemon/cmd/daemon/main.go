package main

import (
	"flag"
	"fmt"
	"log"
	"sort"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/telemetry"
)

const defaultProcPath = "/proc/continfo_pr2_so1_202100054"

func main() {
	procPath := flag.String(
		"proc",
		defaultProcPath,
		"ruta del archivo de telemetría del módulo",
	)

	topCount := flag.Int(
		"top",
		5,
		"cantidad de procesos con mayor uso de CPU",
	)

	flag.Parse()

	snapshot, err := telemetry.ReadSnapshot(*procPath)
	if err != nil {
		log.Fatalf("no se pudo obtener la telemetría: %v", err)
	}

	sort.SliceStable(snapshot.Processes, func(i, j int) bool {
		return snapshot.Processes[i].CPUPercent >
			snapshot.Processes[j].CPUPercent
	})

	limit := *topCount
	if limit < 0 {
		limit = 0
	}
	if limit > len(snapshot.Processes) {
		limit = len(snapshot.Processes)
	}

	memoryPercent :=
		float64(snapshot.Memory.UsedKB) * 100 /
			float64(snapshot.Memory.TotalKB)

	fmt.Println("Daemon de telemetría — Proyecto 2")
	fmt.Printf(
		"Memoria: %d KB usados de %d KB (%.2f%%)\n",
		snapshot.Memory.UsedKB,
		snapshot.Memory.TotalKB,
		memoryPercent,
	)
	fmt.Printf("Procesos encontrados: %d\n", len(snapshot.Processes))
	fmt.Printf("Top %d procesos por CPU:\n", limit)

	for _, process := range snapshot.Processes[:limit] {
		fmt.Printf(
			"PID=%d nombre=%s CPU=%.2f%% RSS=%d KB VSZ=%d KB\n",
			process.PID,
			process.Name,
			process.CPUPercent,
			process.ResidentSizeKB,
			process.VirtualSizeKB,
		)
	}
}
