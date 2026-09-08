package dockerclient

import "testing"

func TestCreateRequestForProfile(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		image     string
		tier      string
		memory    int64
		nanoCPUs  int64
		commanded bool
	}{
		{
			name:      "low",
			profile:   "low",
			image:     "alpine:latest",
			tier:      "low",
			memory:    64 * 1024 * 1024,
			nanoCPUs:  100_000_000,
			commanded: true,
		},
		{
			name:      "high cpu",
			profile:   "high-cpu",
			image:     "alpine:latest",
			tier:      "high",
			memory:    128 * 1024 * 1024,
			nanoCPUs:  500_000_000,
			commanded: true,
		},
		{
			name:     "high memory",
			profile:  "high-memory",
			image:    "roldyoran/go-client:latest",
			tier:     "high",
			memory:   384 * 1024 * 1024,
			nanoCPUs: 250_000_000,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, tier, err :=
				createRequestForProfile(test.profile)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}

			if request.Image != test.image {
				t.Errorf(
					"imagen=%q; se esperaba %q",
					request.Image,
					test.image,
				)
			}

			if tier != test.tier {
				t.Errorf(
					"tier=%q; se esperaba %q",
					tier,
					test.tier,
				)
			}

			if request.HostConfig.Memory != test.memory {
				t.Errorf(
					"memoria=%d; se esperaba %d",
					request.HostConfig.Memory,
					test.memory,
				)
			}

			if request.HostConfig.NanoCPUs != test.nanoCPUs {
				t.Errorf(
					"NanoCPUs=%d; se esperaba %d",
					request.HostConfig.NanoCPUs,
					test.nanoCPUs,
				)
			}

			if test.commanded && len(request.Cmd) == 0 {
				t.Error("se esperaba un comando para el contenedor")
			}

			expectedLabels := map[string]string{
				"so1.project":    "proyecto2",
				"so1.carnet":     "202100054",
				"so1.profile":    test.profile,
				"so1.tier":       test.tier,
				"so1.protected":  "false",
				"so1.created-by": "daemon",
			}

			for key, expectedValue := range expectedLabels {
				if request.Labels[key] != expectedValue {
					t.Errorf(
						"etiqueta %s=%q; se esperaba %q",
						key,
						request.Labels[key],
						expectedValue,
					)
				}
			}
		})
	}
}

func TestCreateRequestRejectsUnknownProfile(t *testing.T) {
	_, _, err := createRequestForProfile("unknown")
	if err == nil {
		t.Fatal("se esperaba error para un perfil desconocido")
	}
}
