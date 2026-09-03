package telemetry

type Snapshot struct {
	Memory    MemoryStats    `json:"memory"`
	Processes []ProcessStats `json:"processes"`
}

type MemoryStats struct {
	TotalKB uint64 `json:"total_kb"`
	FreeKB  uint64 `json:"free_kb"`
	UsedKB  uint64 `json:"used_kb"`
}

type ProcessStats struct {
	PID            int     `json:"pid"`
	Name           string  `json:"name"`
	Command        string  `json:"command"`
	VirtualSizeKB  uint64  `json:"vsz_kb"`
	ResidentSizeKB uint64  `json:"rss_kb"`
	MemoryPercent  float64 `json:"memory_percent"`
	CPUPercent     float64 `json:"cpu_percent"`
}
