package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/dockerclient"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/policy"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/telemetry"
)

const (
	cycleTimeout           = 2 * time.Minute
	dockerOperationTimeout = 10 * time.Second
	valkeyOperationTimeout = 10 * time.Second
)

type cycleConfig struct {
	procPath        string
	valkeyAddress   string
	topCount        int
	lowTarget       int
	highTarget      int
	execute         bool
	dockerClient    *dockerclient.Client
	deletionTracker *deletionTracker
}

func runCycle(config cycleConfig) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		cycleTimeout,
	)
	defer cancel()

	snapshot, containers, processByPID, err := readCurrentState(
		ctx,
		config,
	)
	if err != nil {
		return err
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
	var cycleErrors []error
	persistenceResult := policyResult
	stateIsCurrent := true

	if config.execute {
		if err := executePolicy(
			ctx,
			config.dockerClient,
			policyResult,
			config.deletionTracker,
			config.valkeyAddress,
		); err != nil {
			cycleErrors = append(
				cycleErrors,
				fmt.Errorf("aplicar política: %w", err),
			)
		}

		_, err := replenishContainers(
			ctx,
			config.dockerClient,
			policyResult,
		)
		if err != nil {
			cycleErrors = append(
				cycleErrors,
				fmt.Errorf("reponer contenedores: %w", err),
			)
		}

		refreshedSnapshot,
			refreshedContainers,
			refreshedProcessByPID,
			refreshErr := readCurrentState(ctx, config)
		if refreshErr != nil {
			stateIsCurrent = false
			cycleErrors = append(
				cycleErrors,
				fmt.Errorf("refrescar estado final: %w", refreshErr),
			)
		} else {
			snapshot = refreshedSnapshot
			containers = refreshedContainers
			processByPID = refreshedProcessByPID

			finalResult := policy.Evaluate(
				buildPolicyCandidates(
					containers,
					processByPID,
				),
				config.lowTarget,
				config.highTarget,
			)
			persistenceResult.LowKept = finalResult.LowKept
			persistenceResult.HighKept = finalResult.HighKept
			persistenceResult.MissingLow = finalResult.MissingLow
			persistenceResult.MissingHigh = finalResult.MissingHigh
		}
	}

	if stateIsCurrent {
		persistContext, cancelPersist := context.WithTimeout(
			ctx,
			valkeyOperationTimeout,
		)
		err = persistTelemetry(
			persistContext,
			config.valkeyAddress,
			snapshot,
			containers,
			processByPID,
			persistenceResult,
			config.execute,
		)
		cancelPersist()

		if err != nil {
			cycleErrors = append(
				cycleErrors,
				fmt.Errorf("persistir telemetría: %w", err),
			)
		} else {
			fmt.Println("Telemetría almacenada correctamente en Valkey")
		}
	}

	return errors.Join(cycleErrors...)
}

func readCurrentState(
	ctx context.Context,
	config cycleConfig,
) (
	telemetry.Snapshot,
	[]dockerclient.Container,
	map[int]telemetry.ProcessStats,
	error,
) {
	snapshot, err := telemetry.ReadSnapshot(config.procPath)
	if err != nil {
		return telemetry.Snapshot{}, nil, nil,
			fmt.Errorf("obtener telemetría: %w", err)
	}

	processByPID := make(map[int]telemetry.ProcessStats)
	for _, process := range snapshot.Processes {
		processByPID[process.PID] = process
	}

	dockerContext, cancelDocker := context.WithTimeout(
		ctx,
		dockerOperationTimeout,
	)
	defer cancelDocker()

	containers, err :=
		config.dockerClient.ListProjectContainers(dockerContext)
	if err != nil {
		return telemetry.Snapshot{}, nil, nil,
			fmt.Errorf("consultar Docker: %w", err)
	}

	return snapshot, containers, processByPID, nil
}
