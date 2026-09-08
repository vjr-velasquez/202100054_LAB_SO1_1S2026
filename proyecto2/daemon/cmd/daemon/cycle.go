package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/dockerclient"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/policy"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/telemetry"
)

type cycleConfig struct {
	procPath      string
	valkeyAddress string
	topCount      int
	lowTarget     int
	highTarget    int
	execute       bool
	dockerClient  *dockerclient.Client
}

func runCycle(config cycleConfig) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	snapshot, err := telemetry.ReadSnapshot(config.procPath)
	if err != nil {
		return fmt.Errorf("obtener telemetría: %w", err)
	}

	processByPID := make(map[int]telemetry.ProcessStats)

	for _, process := range snapshot.Processes {
		processByPID[process.PID] = process
	}

	containers, err :=
		config.dockerClient.ListProjectContainers(ctx)
	if err != nil {
		return fmt.Errorf("consultar Docker: %w", err)
	}

	printMemory(snapshot)
	printTopProcesses(snapshot.Processes, config.topCount)
	printContainers(containers, processByPID)

	candidates := buildPolicyCandidates(
		containers,
		processByPID,
	)

	policyResult := policy.Evaluate(
		candidates,
		config.lowTarget,
		config.highTarget,
	)

	printPolicyPlan(policyResult, config.execute)

	if config.execute {
		if err := executePolicy(
			ctx,
			config.dockerClient,
			policyResult,
		); err != nil {
			return fmt.Errorf("aplicar política: %w", err)
		}
	}

	if err := persistTelemetry(
		ctx,
		config.valkeyAddress,
		snapshot,
		containers,
		processByPID,
		policyResult,
		config.execute,
	); err != nil {
		return fmt.Errorf("persistir telemetría: %w", err)
	}

	fmt.Println("Telemetría almacenada correctamente en Valkey")

	return nil
}
