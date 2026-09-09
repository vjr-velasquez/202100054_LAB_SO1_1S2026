package main

import (
	"fmt"
	"testing"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/ebpfwatcher"
)

func TestDeletionTrackerMatchesTargetPID(t *testing.T) {
	tracker := newDeletionTracker()

	confirmation, cancel, err := tracker.Register(1234)
	if err != nil {
		t.Fatalf("registrar PID: %v", err)
	}
	defer cancel()

	matched := tracker.Observe(
		ebpfwatcher.Event{
			TargetPID: 1234,
			Signal:    9,
			Command:   "daemon",
			Source:    ebpfwatcher.EventSourceSignalGenerate,
		},
	)

	if !matched {
		t.Fatal("el evento debía coincidir")
	}

	event, available := <-confirmation
	if !available {
		t.Fatal("no se recibió la confirmación")
	}

	if event.TargetPID != 1234 {
		t.Errorf(
			"PID recibido=%d",
			event.TargetPID,
		)
	}
}

func TestDeletionTrackerIgnoresUnknownPID(t *testing.T) {
	tracker := newDeletionTracker()

	matched := tracker.Observe(
		ebpfwatcher.Event{
			TargetPID: 9999,
			Signal:    15,
			Source:    ebpfwatcher.EventSourceSignalGenerate,
		},
	)

	if matched {
		t.Fatal("un PID no registrado no debe coincidir")
	}
}

func TestDeletionTrackerIgnoresOtherSignalForPendingPID(t *testing.T) {
	tracker := newDeletionTracker()

	confirmation, cancel, err := tracker.Register(1234)
	if err != nil {
		t.Fatalf("registrar PID: %v", err)
	}
	defer cancel()

	matched := tracker.Observe(
		ebpfwatcher.Event{
			TargetPID: 1234,
			Signal:    15,
			Source:    ebpfwatcher.EventSourceSignalGenerate,
		},
	)
	if matched {
		t.Fatal("una señal distinta de SIGKILL no debe confirmar la eliminación")
	}

	matched = tracker.Observe(
		ebpfwatcher.Event{
			TargetPID: 1234,
			Signal:    deletionSignal,
			Source:    ebpfwatcher.EventSourceSignalGenerate,
		},
	)
	if !matched {
		t.Fatal("SIGKILL debía confirmar el PID que seguía pendiente")
	}

	event, available := <-confirmation
	if !available {
		t.Fatal("no se recibió la confirmación SIGKILL")
	}
	if event.Signal != deletionSignal {
		t.Errorf("señal recibida=%d", event.Signal)
	}
}

func TestDeletionTrackerIgnoresSysKillEntry(t *testing.T) {
	tracker := newDeletionTracker()

	confirmation, cancel, err := tracker.Register(1234)
	if err != nil {
		t.Fatalf("registrar PID: %v", err)
	}
	defer cancel()

	if tracker.Observe(ebpfwatcher.Event{
		TargetPID: 1234,
		Signal:    deletionSignal,
		Source:    ebpfwatcher.EventSourceSysKill,
	}) {
		t.Fatal("sys_enter no debe confirmar una señal antes de su generación")
	}

	if !tracker.Observe(ebpfwatcher.Event{
		TargetPID: 1234,
		Signal:    deletionSignal,
		Source:    ebpfwatcher.EventSourceSignalGenerate,
	}) {
		t.Fatal("signal_generate debía confirmar el PID pendiente")
	}

	if _, available := <-confirmation; !available {
		t.Fatal("no se recibió la confirmación")
	}
}

func TestDeletionTrackerMatchesPendingSysKillForAudit(t *testing.T) {
	tracker := newDeletionTracker()

	_, cancel, err := tracker.Register(1234)
	if err != nil {
		t.Fatalf("registrar PID: %v", err)
	}
	defer cancel()

	if !tracker.MatchesPendingSysKill(ebpfwatcher.Event{
		TargetPID: 1234,
		Signal:    deletionSignal,
		Source:    ebpfwatcher.EventSourceSysKill,
	}) {
		t.Fatal("sys_kill del PID pendiente debía conservarse para auditoría")
	}

	if tracker.MatchesPendingSysKill(ebpfwatcher.Event{
		TargetPID: 9999,
		Signal:    deletionSignal,
		Source:    ebpfwatcher.EventSourceSysKill,
	}) {
		t.Fatal("sys_kill de un PID ajeno no debía conservarse")
	}

	if tracker.MatchesPendingSysKill(ebpfwatcher.Event{
		TargetPID: 1234,
		Signal:    15,
		Source:    ebpfwatcher.EventSourceSysKill,
	}) {
		t.Fatal("una señal distinta de SIGKILL no debía conservarse")
	}
}

func TestDeletionTrackerDisableClosesPendingAndRejectsNew(t *testing.T) {
	tracker := newDeletionTracker()

	confirmation, cancel, err := tracker.Register(1234)
	if err != nil {
		t.Fatalf("registrar PID: %v", err)
	}
	defer cancel()

	tracker.Disable(fmt.Errorf("lector finalizado"))

	if _, available := <-confirmation; available {
		t.Fatal("la espera pendiente debía cerrarse")
	}

	if _, _, err := tracker.Register(5678); err == nil {
		t.Fatal("un tracker deshabilitado debía rechazar registros")
	}
}

func TestDeletionTrackerRejectsInvalidPID(t *testing.T) {
	tracker := newDeletionTracker()

	_, _, err := tracker.Register(0)
	if err == nil {
		t.Fatal("se esperaba error para PID inválido")
	}
}
