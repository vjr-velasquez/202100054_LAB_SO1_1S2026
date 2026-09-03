package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadSnapshot(path string) (Snapshot, error) {
	var snapshot Snapshot

	content, err := os.ReadFile(path)
	if err != nil {
		return snapshot, fmt.Errorf("leer %s: %w", path, err)
	}

	if err := json.Unmarshal(content, &snapshot); err != nil {
		return snapshot, fmt.Errorf("decodificar telemetría: %w", err)
	}

	if snapshot.Memory.TotalKB == 0 {
		return snapshot, fmt.Errorf("telemetría inválida: memoria total igual a cero")
	}

	if snapshot.Processes == nil {
		snapshot.Processes = []ProcessStats{}
	}

	return snapshot, nil
}
