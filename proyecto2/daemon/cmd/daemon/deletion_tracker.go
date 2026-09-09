package main

import (
	"fmt"
	"sync"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/ebpfwatcher"
)

const deletionSignal int32 = 9

type deletionTracker struct {
	mutex        sync.Mutex
	pending      map[int]chan ebpfwatcher.Event
	available    bool
	disableError error
}

func newDeletionTracker() *deletionTracker {
	return &deletionTracker{
		pending:   make(map[int]chan ebpfwatcher.Event),
		available: true,
	}
}

func (tracker *deletionTracker) Register(
	targetPID int,
) (<-chan ebpfwatcher.Event, func(), error) {
	if targetPID <= 0 {
		return nil, nil, fmt.Errorf(
			"PID objetivo inválido: %d",
			targetPID,
		)
	}

	confirmation := make(
		chan ebpfwatcher.Event,
		1,
	)

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	if !tracker.available {
		return nil, nil, fmt.Errorf(
			"el monitor eBPF no está disponible: %w",
			tracker.disableError,
		)
	}

	if _, exists := tracker.pending[targetPID]; exists {
		return nil, nil, fmt.Errorf(
			"el PID %d ya espera confirmación eBPF",
			targetPID,
		)
	}

	tracker.pending[targetPID] = confirmation

	cancel := func() {
		tracker.mutex.Lock()
		defer tracker.mutex.Unlock()

		current, exists := tracker.pending[targetPID]
		if !exists || current != confirmation {
			return
		}

		delete(tracker.pending, targetPID)
		close(confirmation)
	}

	return confirmation, cancel, nil
}

func (tracker *deletionTracker) Observe(
	event ebpfwatcher.Event,
) bool {
	if event.Signal != deletionSignal {
		return false
	}
	if event.Source != ebpfwatcher.EventSourceSignalGenerate {
		return false
	}

	targetPID := int(event.TargetPID)

	tracker.mutex.Lock()

	confirmation, exists :=
		tracker.pending[targetPID]

	if exists {
		delete(tracker.pending, targetPID)
	}

	tracker.mutex.Unlock()

	if !exists {
		return false
	}

	confirmation <- event
	close(confirmation)

	return true
}

func (tracker *deletionTracker) MatchesPendingSysKill(
	event ebpfwatcher.Event,
) bool {
	if event.Signal != deletionSignal ||
		event.Source != ebpfwatcher.EventSourceSysKill {
		return false
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	if !tracker.available {
		return false
	}

	_, exists := tracker.pending[int(event.TargetPID)]
	return exists
}

func (tracker *deletionTracker) Disable(reason error) {
	if reason == nil {
		reason = fmt.Errorf("causa desconocida")
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	if !tracker.available {
		return
	}

	tracker.available = false
	tracker.disableError = reason

	for targetPID, confirmation := range tracker.pending {
		delete(tracker.pending, targetPID)
		close(confirmation)
	}
}
