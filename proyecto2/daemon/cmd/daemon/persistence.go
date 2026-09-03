package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/dockerclient"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/policy"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/storage"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/telemetry"
)

type persistedSample struct {
	Timestamp  time.Time             `json:"timestamp"`
	Mode       string                `json:"mode"`
	Memory     telemetry.MemoryStats `json:"memory"`
	Containers []persistedContainer  `json:"containers"`
	Decisions  []persistedDecision   `json:"decisions"`
	Summary    persistedSummary      `json:"summary"`
}

type persistedContainer struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Image          string  `json:"image"`
	PID            int     `json:"pid"`
	Profile        string  `json:"profile"`
	Tier           string  `json:"tier"`
	Protected      bool    `json:"protected"`
	TelemetryFound bool    `json:"telemetry_found"`
	CPUPercent     float64 `json:"cpu_percent"`
	RSSKB          uint64  `json:"rss_kb"`
	VSZKB          uint64  `json:"vsz_kb"`
}

type persistedDecision struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Profile   string        `json:"profile"`
	Tier      string        `json:"tier"`
	Protected bool          `json:"protected"`
	Action    policy.Action `json:"action"`
	Reason    string        `json:"reason"`
	CPU       float64       `json:"cpu_percent"`
	RSSKB     uint64        `json:"rss_kb"`
}

type persistedSummary struct {
	LowKept     int `json:"low_kept"`
	HighKept    int `json:"high_kept"`
	MissingLow  int `json:"missing_low"`
	MissingHigh int `json:"missing_high"`
}

func persistTelemetry(
	ctx context.Context,
	address string,
	snapshot telemetry.Snapshot,
	containers []dockerclient.Container,
	processByPID map[int]telemetry.ProcessStats,
	result policy.Result,
	execute bool,
) error {
	store, err := storage.New(address)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		return err
	}

	mode := "dry-run"
	if execute {
		mode = "execute"
	}

	sample := persistedSample{
		Timestamp: time.Now().UTC(),
		Mode:      mode,
		Memory:    snapshot.Memory,
		Summary: persistedSummary{
			LowKept:     result.LowKept,
			HighKept:    result.HighKept,
			MissingLow:  result.MissingLow,
			MissingHigh: result.MissingHigh,
		},
	}

	for _, container := range containers {
		process, found := processByPID[container.PID]

		sample.Containers = append(
			sample.Containers,
			persistedContainer{
				ID:             container.ID,
				Name:           container.Name,
				Image:          container.Image,
				PID:            container.PID,
				Profile:        container.Profile,
				Tier:           container.Tier,
				Protected:      container.Protected,
				TelemetryFound: found,
				CPUPercent:     process.CPUPercent,
				RSSKB:          process.ResidentSizeKB,
				VSZKB:          process.VirtualSizeKB,
			},
		)
	}

	for _, decision := range result.Decisions {
		sample.Decisions = append(
			sample.Decisions,
			persistedDecision{
				ID:        decision.Candidate.ID,
				Name:      decision.Candidate.Name,
				Profile:   decision.Candidate.Profile,
				Tier:      decision.Candidate.Tier,
				Protected: decision.Candidate.Protected,
				Action:    decision.Action,
				Reason:    decision.Reason,
				CPU:       decision.Candidate.CPU,
				RSSKB:     decision.Candidate.RSSKB,
			},
		)
	}

	if err := store.SaveSample(ctx, sample); err != nil {
		return fmt.Errorf("guardar telemetría: %w", err)
	}

	return nil
}
