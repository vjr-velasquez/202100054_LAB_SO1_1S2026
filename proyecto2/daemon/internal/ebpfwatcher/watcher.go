package ebpfwatcher

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

const rawKillEventSize = 40

type Event struct {
	TimestampNS uint64    `json:"timestamp_ns"`
	ObservedAt  time.Time `json:"observed_at"`
	CallerPID   uint32    `json:"caller_pid"`
	TargetPID   int32     `json:"target_pid"`
	Signal      int32     `json:"signal"`
	Command     string    `json:"command"`
}

type rawKillEvent struct {
	TimestampNS uint64
	CallerPID   uint32
	TargetPID   int32
	Signal      int32
	Command     [16]byte
	Padding     [4]byte
}

type Watcher struct {
	objects    killMonitorObjects
	tracepoint link.Link
	reader     *ringbuf.Reader
}

func New() (*Watcher, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf(
			"ajustar límite de memoria eBPF: %w",
			err,
		)
	}

	watcher := &Watcher{}

	if err := loadKillMonitorObjects(
		&watcher.objects,
		nil,
	); err != nil {
		return nil, fmt.Errorf(
			"cargar objetos eBPF: %w",
			err,
		)
	}

	tracepoint, err := link.Tracepoint(
		"raw_syscalls",
		"sys_enter",
		watcher.objects.TraceKill,
		nil,
	)
	if err != nil {
		_ = watcher.objects.Close()

		return nil, fmt.Errorf(
			"conectar tracepoint raw_syscalls/sys_enter: %w",
			err,
		)
	}

	reader, err := ringbuf.NewReader(watcher.objects.Events)
	if err != nil {
		_ = tracepoint.Close()
		_ = watcher.objects.Close()

		return nil, fmt.Errorf(
			"abrir ring buffer eBPF: %w",
			err,
		)
	}

	watcher.tracepoint = tracepoint
	watcher.reader = reader

	return watcher, nil
}

func (watcher *Watcher) Read() (Event, error) {
	record, err := watcher.reader.Read()
	if err != nil {
		return Event{}, fmt.Errorf(
			"leer ring buffer eBPF: %w",
			err,
		)
	}

	event, err := decodeKillEvent(record.RawSample)
	if err != nil {
		return Event{}, err
	}

	event.ObservedAt = time.Now().UTC()

	return event, nil
}

func (watcher *Watcher) Close() error {
	var readerError error
	var tracepointError error

	if watcher.reader != nil {
		readerError = watcher.reader.Close()
	}

	if watcher.tracepoint != nil {
		tracepointError = watcher.tracepoint.Close()
	}

	return errors.Join(
		readerError,
		tracepointError,
		watcher.objects.Close(),
	)
}

func decodeKillEvent(sample []byte) (Event, error) {
	if len(sample) < rawKillEventSize {
		return Event{}, fmt.Errorf(
			"evento eBPF incompleto: recibido=%d esperado=%d",
			len(sample),
			rawKillEventSize,
		)
	}

	var rawEvent rawKillEvent

	if err := binary.Read(
		bytes.NewReader(sample[:rawKillEventSize]),
		binary.LittleEndian,
		&rawEvent,
	); err != nil {
		return Event{}, fmt.Errorf(
			"decodificar evento eBPF: %w",
			err,
		)
	}

	return Event{
		TimestampNS: rawEvent.TimestampNS,
		CallerPID:   rawEvent.CallerPID,
		TargetPID:   rawEvent.TargetPID,
		Signal:      rawEvent.Signal,
		Command: strings.TrimRight(
			string(rawEvent.Command[:]),
			"\x00",
		),
	}, nil
}
