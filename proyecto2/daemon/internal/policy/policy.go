package policy

import "sort"

type Action string

const (
	ActionKeep   Action = "keep"
	ActionRemove Action = "remove"
	ActionSkip   Action = "skip"
)

type Candidate struct {
	ID            string
	Name          string
	PID           int
	Profile       string
	Tier          string
	Protected     bool
	CPU           float64
	MemoryPercent float64
	VSZKB         uint64
	RSSKB         uint64
}

type Decision struct {
	Candidate Candidate
	Action    Action
	Reason    string
}

type Result struct {
	Decisions   []Decision
	LowKept     int
	HighKept    int
	MissingLow  int
	MissingHigh int
}

func Evaluate(
	candidates []Candidate,
	lowTarget int,
	highTarget int,
) Result {
	if lowTarget < 0 {
		lowTarget = 0
	}
	if highTarget < 0 {
		highTarget = 0
	}

	result := Result{}
	lowCandidates := make([]Candidate, 0)
	highCandidates := make([]Candidate, 0)

	for _, candidate := range candidates {
		switch {
		case candidate.Protected:
			result.Decisions = append(result.Decisions, Decision{
				Candidate: candidate,
				Action:    ActionKeep,
				Reason:    "contenedor protegido",
			})

		case candidate.Profile == "intruder" ||
			candidate.Tier == "intruder":
			result.Decisions = append(result.Decisions, Decision{
				Candidate: candidate,
				Action:    ActionRemove,
				Reason:    "perfil intruso",
			})

		case candidate.Tier == "low":
			lowCandidates = append(lowCandidates, candidate)

		case candidate.Tier == "high":
			highCandidates = append(highCandidates, candidate)

		default:
			result.Decisions = append(result.Decisions, Decision{
				Candidate: candidate,
				Action:    ActionSkip,
				Reason:    "perfil o tier desconocido",
			})
		}
	}

	sort.SliceStable(lowCandidates, func(i, j int) bool {
		if lowCandidates[i].MemoryPercent != lowCandidates[j].MemoryPercent {
			return lowCandidates[i].MemoryPercent < lowCandidates[j].MemoryPercent
		}
		if lowCandidates[i].VSZKB != lowCandidates[j].VSZKB {
			return lowCandidates[i].VSZKB < lowCandidates[j].VSZKB
		}
		if lowCandidates[i].RSSKB != lowCandidates[j].RSSKB {
			return lowCandidates[i].RSSKB < lowCandidates[j].RSSKB
		}
		if lowCandidates[i].CPU != lowCandidates[j].CPU {
			return lowCandidates[i].CPU < lowCandidates[j].CPU
		}
		return lowCandidates[i].ID < lowCandidates[j].ID
	})

	sort.SliceStable(highCandidates, func(i, j int) bool {
		if highCandidates[i].MemoryPercent != highCandidates[j].MemoryPercent {
			return highCandidates[i].MemoryPercent > highCandidates[j].MemoryPercent
		}
		if highCandidates[i].VSZKB != highCandidates[j].VSZKB {
			return highCandidates[i].VSZKB > highCandidates[j].VSZKB
		}
		if highCandidates[i].RSSKB != highCandidates[j].RSSKB {
			return highCandidates[i].RSSKB > highCandidates[j].RSSKB
		}
		if highCandidates[i].CPU != highCandidates[j].CPU {
			return highCandidates[i].CPU > highCandidates[j].CPU
		}
		return highCandidates[i].ID < highCandidates[j].ID
	})

	for index, candidate := range lowCandidates {
		decision := Decision{
			Candidate: candidate,
			Action:    ActionKeep,
			Reason:    "dentro del objetivo de bajo consumo",
		}

		if index >= lowTarget {
			decision.Action = ActionRemove
			decision.Reason = "excede el objetivo de bajo consumo"
		} else {
			result.LowKept++
		}

		result.Decisions = append(result.Decisions, decision)
	}

	for index, candidate := range highCandidates {
		decision := Decision{
			Candidate: candidate,
			Action:    ActionKeep,
			Reason:    "dentro del objetivo de alto consumo",
		}

		if index >= highTarget {
			decision.Action = ActionRemove
			decision.Reason = "excede el objetivo de alto consumo"
		} else {
			result.HighKept++
		}

		result.Decisions = append(result.Decisions, decision)
	}

	result.MissingLow = lowTarget - result.LowKept
	result.MissingHigh = highTarget - result.HighKept

	return result
}
