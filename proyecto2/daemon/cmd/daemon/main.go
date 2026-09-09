package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"
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

	valkeyAddress := flag.String(
		"valkey-address",
		"127.0.0.1:6379",
		"dirección del servidor Valkey",
	)

	projectRoot := flag.String(
		"project-root",
		"..",
		"ruta de la carpeta proyecto2",
	)

	manageLifecycle := flag.Bool(
		"manage-lifecycle",
		true,
		"levantar infraestructura, módulo y cronjob automáticamente",
	)

	enableEBPF := flag.Bool(
		"ebpf",
		true,
		"capturar y almacenar señales kill mediante eBPF",
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

	execute := flag.Bool(
		"execute",
		false,
		"aplicar las eliminaciones propuestas por la politica",
	)

	interval := flag.Duration(
		"interval",
		30*time.Second,
		"intervalo entre lecturas; 0 ejecuta solamente una vez",
	)

	flag.Parse()

	if *interval < 0 {
		log.Fatal("el intervalo no puede ser negativo")
	}

	if *interval > 0 && *interval < 20*time.Second {
		log.Fatal(
			"el intervalo periódico debe ser de al menos 20 segundos",
		)
	}

	if *interval > 60*time.Second {
		log.Fatal(
			"el intervalo periódico no puede superar 60 segundos",
		)
	}

	if *manageLifecycle {
		cleanupLifecycle, err :=
			startManagedLifecycle(*projectRoot)
		if err != nil {
			log.Fatalf(
				"no se pudo iniciar el servicio: %v",
				err,
			)
		}

		defer cleanupLifecycle()
	}

	if *enableEBPF {
		cleanupEBPF, err := startEBPFMonitor(
			*valkeyAddress,
		)
		if err != nil {
			log.Printf(
				"no se pudo iniciar eBPF: %v",
				err,
			)
			return
		}

		defer cleanupEBPF()
	}

	dockerClient := dockerclient.New(*dockerSocket)

	config := cycleConfig{
		procPath:      *procPath,
		valkeyAddress: *valkeyAddress,
		topCount:      *topCount,
		lowTarget:     *lowTarget,
		highTarget:    *highTarget,
		execute:       *execute,
		dockerClient:  dockerClient,
	}

	if *interval == 0 {
		if err := runCycle(config); err != nil {
			log.Printf(
				"falló el ciclo de telemetría: %v",
				err,
			)
		}

		return
	}

	stopSignals := make(chan os.Signal, 1)

	signal.Notify(
		stopSignals,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer signal.Stop(stopSignals)

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	fmt.Printf(
		"Daemon periódico iniciado: intervalo=%s\n",
		interval.String(),
	)

	run := func() {
		fmt.Printf(
			"\nCiclo iniciado: %s\n",
			time.Now().UTC().Format(time.RFC3339),
		)

		if err := runCycle(config); err != nil {
			log.Printf("el ciclo terminó con error: %v", err)
		}
	}

	run()

	for {
		select {
		case <-ticker.C:
			run()

		case receivedSignal := <-stopSignals:
			fmt.Printf(
				"\nSeñal %s recibida; cerrando el daemon\n",
				receivedSignal,
			)
			return
		}
	}

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

func printPolicyPlan(result policy.Result, execute bool) {
	mode := "DRY-RUN"
	if execute {
		mode = "EXECUTE"
	}

	fmt.Printf("\nPlan de administración — %s\n", mode)

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

	if !execute {
		fmt.Println("DRY-RUN: no se modificó ningún contenedor")
	}
}

func executePolicy(
	ctx context.Context,
	client *dockerclient.Client,
	result policy.Result,
) error {
	removedCount := 0

	for _, decision := range result.Decisions {
		if decision.Action != policy.ActionRemove {
			continue
		}

		fmt.Printf(
			"Eliminando ID=%.12s nombre=%s...\n",
			decision.Candidate.ID,
			decision.Candidate.Name,
		)

		if err := client.RemoveProjectContainer(
			ctx,
			decision.Candidate.ID,
		); err != nil {
			return fmt.Errorf(
				"eliminar %s: %w",
				decision.Candidate.Name,
				err,
			)
		}

		removedCount++
	}

	fmt.Printf(
		"Ejecución completada: %d contenedores eliminados\n",
		removedCount,
	)

	return nil
}
