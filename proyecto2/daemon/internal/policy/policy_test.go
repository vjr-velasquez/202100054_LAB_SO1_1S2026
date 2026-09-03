package policy

import "testing"

func TestEvaluateSelectsExpectedContainers(t *testing.T) {
	candidates := []Candidate{
		{ID: "low-1", Tier: "low", CPU: 1, RSSKB: 100},
		{ID: "low-2", Tier: "low", CPU: 2, RSSKB: 100},
		{ID: "low-3", Tier: "low", CPU: 0.5, RSSKB: 100},
		{ID: "low-4", Tier: "low", CPU: 3, RSSKB: 100},
		{ID: "high-1", Tier: "high", CPU: 80, RSSKB: 200},
		{ID: "high-2", Tier: "high", CPU: 50, RSSKB: 200},
		{ID: "high-3", Tier: "high", CPU: 20, RSSKB: 200},
		{ID: "intruder", Tier: "intruder", Profile: "intruder"},
		{ID: "grafana", Protected: true},
	}

	result := Evaluate(candidates, 3, 2)

	if result.LowKept != 3 {
		t.Fatalf("LowKept=%d; se esperaba 3", result.LowKept)
	}

	if result.HighKept != 2 {
		t.Fatalf("HighKept=%d; se esperaba 2", result.HighKept)
	}

	if result.MissingLow != 0 || result.MissingHigh != 0 {
		t.Fatalf(
			"déficit inesperado: low=%d high=%d",
			result.MissingLow,
			result.MissingHigh,
		)
	}

	expected := map[string]Action{
		"low-1":    ActionKeep,
		"low-2":    ActionKeep,
		"low-3":    ActionKeep,
		"low-4":    ActionRemove,
		"high-1":   ActionKeep,
		"high-2":   ActionKeep,
		"high-3":   ActionRemove,
		"intruder": ActionRemove,
		"grafana":  ActionKeep,
	}

	for _, decision := range result.Decisions {
		expectedAction, exists := expected[decision.Candidate.ID]
		if !exists {
			t.Fatalf("decisión inesperada para %s", decision.Candidate.ID)
		}

		if decision.Action != expectedAction {
			t.Errorf(
				"%s: acción=%s; se esperaba %s",
				decision.Candidate.ID,
				decision.Action,
				expectedAction,
			)
		}
	}
}

func TestEvaluateReportsDeficitsAndSkipsUnknown(t *testing.T) {
	result := Evaluate([]Candidate{
		{ID: "low-1", Tier: "low"},
		{ID: "unknown", Tier: "other"},
	}, 3, 2)

	if result.MissingLow != 2 {
		t.Errorf("MissingLow=%d; se esperaba 2", result.MissingLow)
	}

	if result.MissingHigh != 2 {
		t.Errorf("MissingHigh=%d; se esperaba 2", result.MissingHigh)
	}

	for _, decision := range result.Decisions {
		if decision.Candidate.ID == "unknown" &&
			decision.Action != ActionSkip {
			t.Errorf(
				"perfil desconocido recibió acción insegura: %s",
				decision.Action,
			)
		}
	}
}
