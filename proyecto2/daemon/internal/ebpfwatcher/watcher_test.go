package ebpfwatcher

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDecodeKillEvent(t *testing.T) {
	rawEvent := rawKillEvent{
		TimestampNS: 123456,
		CallerPID:   1000,
		TargetPID:   2000,
		Signal:      15,
	}

	copy(rawEvent.Command[:], "so1-test")

	var buffer bytes.Buffer

	if err := binary.Write(
		&buffer,
		binary.LittleEndian,
		rawEvent,
	); err != nil {
		t.Fatalf("codificar evento de prueba: %v", err)
	}

	event, err := decodeKillEvent(buffer.Bytes())
	if err != nil {
		t.Fatalf("decodificar evento: %v", err)
	}

	if event.TimestampNS != 123456 {
		t.Errorf("timestamp=%d", event.TimestampNS)
	}

	if event.CallerPID != 1000 {
		t.Errorf("caller PID=%d", event.CallerPID)
	}

	if event.TargetPID != 2000 {
		t.Errorf("target PID=%d", event.TargetPID)
	}

	if event.Signal != 15 {
		t.Errorf("signal=%d", event.Signal)
	}

	if event.Command != "so1-test" {
		t.Errorf("command=%q", event.Command)
	}
}

func TestDecodeKillEventRejectsShortSample(t *testing.T) {
	_, err := decodeKillEvent(make([]byte, 10))
	if err == nil {
		t.Fatal("se esperaba error para un evento incompleto")
	}
}
