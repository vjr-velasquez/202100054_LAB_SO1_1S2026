package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/dockerclient"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/policy"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/telemetry"
)

const (
	defaultProcPath     = "/proc/continfo_pr2_so1_202100054"
	defaultDockerSocket = "/var/run/docker.sock"
)

func main() {
	procPath := flag.String(
		"proc",
		defaultProcPath,
		"ruta del archivo de telemetría del módulo",
	)

	dockerSocket := flag.String(
		"docker-socket",
		defaultDockerSocket,
		"ruta del socket de Docker",
	)

	topCount := flag.Int(
		"top",
		5,
		"cantidad de procesos con mayor uso de CPU",
	)

	lowTarget := flag.Int(
		"low-target",
		3,
		"cantidad de contenedores de bajo consumo que deben conservarse",
	)

	highTarget := flag.Int(
		"high-target",
		2,
		"cantidad de contenedores de alto consumo que deben conservarse",
	)

	flag.Parse()

	snapshot, err := telemetry.ReadSnapshot(*procPath)
	if err != nil {
		log.Fatalf("no se pudo obtener la telemetría: %v", err)
	}

	processByPID := make(map[int]telemetry.ProcessStats)
	for _, process := range snapshot.Processes {
		processByPID[process.PID] = process
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dockerClient := dockerclient.New(*dockerSocket)

	containers, err := dockerClient.ListProjectContainers(ctx)
	if err != nil {
		log.Fatalf("no se pudo consultar Docker: %v", err)
	}

	printMemory(snapshot)
	printTopProcesses(snapshot.Processes, *topCount)
	printContainers(containers, processByPID)

	candidates := buildPolicyCandidates(containers, processByPID)

	policyResult := policy.Evaluate(
		candidates,
		*lowTarget,
		*highTarget,
	)

	printPolicyPlan(policyResult)
}

func printMemory(snapshot telemetry.Snapshot) {
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
}

func printTopProcesses(
	processes []telemetry.ProcessStats,
	topCount int,
) {
	sortedProcesses := append([]telemetry.ProcessStats(nil), processes...)

	sort.SliceStable(sortedProcesses, func(i, j int) bool {
		return sortedProcesses[i].CPUPercent >
			sortedProcesses[j].CPUPercent
	})

	limit := topCount
	if limit < 0 {
		limit = 0
	}
	if limit > len(sortedProcesses) {
		limit = len(sortedProcesses)
	}

	fmt.Printf("\nTop %d procesos por CPU:\n", limit)

	for _, process := range sortedProcesses[:limit] {
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

func printContainers(
	containers []dockerclient.Container,
	processByPID map[int]telemetry.ProcessStats,
) {
	fmt.Printf(
		"\nContenedores del proyecto encontrados: %d\n",
		len(containers),
	)

	for _, container := range containers {
		process, found := processByPID[container.PID]

		if !found {
			fmt.Printf(
				"ID=%.12s nombre=%s perfil=%s PID=%d telemetría=no encontrada\n",
				container.ID,
				container.Name,
				container.Profile,
				container.PID,
			)
			continue
		}

		fmt.Printf(
			"ID=%.12s nombre=%s perfil=%s tier=%s protegido=%t "+
				"PID=%d CPU=%.2f%% RSS=%d KB VSZ=%d KB\n",
			container.ID,
			container.Name,
			container.Profile,
			container.Tier,
			container.Protected,
			container.PID,
			process.CPUPercent,
			process.ResidentSizeKB,
			process.VirtualSizeKB,
		)
	}
}

func buildPolicyCandidates(
	containers []dockerclient.Container,
	processByPID map[int]telemetry.ProcessStats,
) []policy.Candidate {
	candidates := make([]policy.Candidate, 0, len(containers))

	for _, container := range containers {
		process := processByPID[container.PID]

		candidates = append(candidates, policy.Candidate{
			ID:        container.ID,
			Name:      container.Name,
			Profile:   container.Profile,
			Tier:      container.Tier,
			Protected: container.Protected,
			CPU:       process.CPUPercent,
			RSSKB:     process.ResidentSizeKB,
		})
	}

	return candidates
}

func printPolicyPlan(result policy.Result) {
	fmt.Println("\nPlan de administración — DRY-RUN")

	for _, decision := range result.Decisions {
		fmt.Printf(
			"[%s] ID=%.12s nombre=%s perfil=%s tier=%s "+
				"CPU=%.2f%% RSS=%d KB motivo=%s\n",
			decision.Action,
			decision.Candidate.ID,
			decision.Candidate.Name,
			decision.Candidate.Profile,
			decision.Candidate.Tier,
			decision.Candidate.CPU,
			decision.Candidate.RSSKB,
			decision.Reason,
		)
	}

	fmt.Printf(
		"Resumen: low conservados=%d faltantes=%d | "+
			"high conservados=%d faltantes=%d\n",
		result.LowKept,
		result.MissingLow,
		result.HighKept,
		result.MissingHigh,
	)

	fmt.Println("DRY-RUN: no se modificó ningún contenedor")
}
