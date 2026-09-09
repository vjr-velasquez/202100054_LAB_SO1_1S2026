package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/ebpfwatcher"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/policy"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/storage"
)

const deletionConfirmationTimeout = 3 * time.Second

type confirmedDeletionClient interface {
	KillProjectContainer(context.Context, string) error
	RemoveProjectContainer(context.Context, string) error
}

func requestConfirmedTermination(
	ctx context.Context,
	client confirmedDeletionClient,
	candidate policy.Candidate,
	tracker *deletionTracker,
) (ebpfwatcher.Event, error) {
	if tracker == nil {
		return ebpfwatcher.Event{},
			fmt.Errorf("el rastreador eBPF no está disponible")
	}

	confirmation, cancel, err := tracker.Register(candidate.PID)
	if err != nil {
		return ebpfwatcher.Event{}, err
	}
	defer cancel()

	if err := client.KillProjectContainer(
		ctx,
		candidate.ID,
	); err != nil {
		return ebpfwatcher.Event{}, fmt.Errorf(
			"enviar SIGKILL al contenedor %.12s: %w",
			candidate.ID,
			err,
		)
	}

	timer := time.NewTimer(deletionConfirmationTimeout)
	defer timer.Stop()

	select {
	case event, open := <-confirmation:
		if !open {
			return ebpfwatcher.Event{},
				fmt.Errorf(
					"se canceló la confirmación del PID %d",
					candidate.PID,
				)
		}
		if event.TargetPID != int32(candidate.PID) ||
			event.Signal != deletionSignal ||
			event.Source != ebpfwatcher.EventSourceSignalGenerate {
			return ebpfwatcher.Event{}, fmt.Errorf(
				"confirmación eBPF inválida para el PID %d: objetivo=%d señal=%d origen=%s",
				candidate.PID,
				event.TargetPID,
				event.Signal,
				event.Source,
			)
		}

		return event, nil

	case <-timer.C:
		return ebpfwatcher.Event{},
			fmt.Errorf(
				"eBPF no confirmó SIGKILL para el PID %d",
				candidate.PID,
			)

	case <-ctx.Done():
		return ebpfwatcher.Event{}, ctx.Err()
	}
}

func removeConfirmedContainer(
	ctx context.Context,
	client confirmedDeletionClient,
	valkeyAddress string,
	candidate policy.Candidate,
	tracker *deletionTracker,
) (ebpfwatcher.Event, bool, error) {
	event, err := requestConfirmedTermination(
		ctx,
		client,
		candidate,
		tracker,
	)
	if err != nil {
		return ebpfwatcher.Event{}, false, err
	}

	if err := client.RemoveProjectContainer(
		ctx,
		candidate.ID,
	); err != nil {
		return ebpfwatcher.Event{}, false, fmt.Errorf(
			"limpiar contenedor después de SIGKILL: %w",
			err,
		)
	}

	if err := persistConfirmedDeletion(
		ctx,
		valkeyAddress,
		candidate,
		event,
	); err != nil {
		return event, true, err
	}

	return event, true, nil
}

func persistConfirmedDeletion(
	ctx context.Context,
	address string,
	candidate policy.Candidate,
	event ebpfwatcher.Event,
) error {
	store, err := storage.New(address)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		return err
	}

	deletion := map[string]any{
		"deleted_at":     time.Now().UTC(),
		"container_id":   candidate.ID,
		"container_name": candidate.Name,
		"profile":        candidate.Profile,
		"tier":           candidate.Tier,
		"container_pid":  candidate.PID,
		"cpu_percent":    candidate.CPU,
		"memory_percent": candidate.MemoryPercent,
		"vsz_kb":         candidate.VSZKB,
		"rss_kb":         candidate.RSSKB,
		"caller_pid":     event.CallerPID,
		"target_pid":     event.TargetPID,
		"signal":         event.Signal,
		"command":        event.Command,
		"observed_at":    event.ObservedAt,
		"timestamp_ns":   event.TimestampNS,
		"event_source":   event.Source,
		"status":         "container_deleted",
	}

	if err := store.SaveContainerDeletion(
		ctx,
		deletion,
	); err != nil {
		return fmt.Errorf(
			"registrar eliminación confirmada: %w",
			err,
		)
	}

	return nil
}
