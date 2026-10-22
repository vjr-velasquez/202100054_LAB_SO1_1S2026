package main

import (
	"testing"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/ebpfwatcher"
)

func TestFormatEBPFEvent(t *testing.T) {
	location := time.FixedZone("Guatemala", -6*60*60)
	event := ebpfwatcher.Event{
		ObservedAt: time.Date(
			2026,
			time.September,
			19,
			10,
			11,
			12,
			123456789,
			location,
		),
		Source:    ebpfwatcher.EventSourceSignalGenerate,
		CallerPID: 451,
		TargetPID: 982,
		Signal:    9,
		Command:   "daemon",
	}

	const expected = "eBPF kill: timestamp=2026-09-19T16:11:12.123456789Z origen=signal_generate emisor=451 objetivo=982 señal=9 comando=daemon"

	actual := formatEBPFEvent(event)
	if actual != expected {
		t.Fatalf(
			"formatEBPFEvent() = %q; esperado %q",
			actual,
			expected,
		)
	}
}
