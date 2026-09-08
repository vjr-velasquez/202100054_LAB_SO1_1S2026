package main

import (
	"context"
	"fmt"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/dockerclient"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/policy"
)

func replenishContainers(
	ctx context.Context,
	client *dockerclient.Client,
	result policy.Result,
) ([]dockerclient.Container, error) {
	profiles := make(
		[]string,
		0,
		result.MissingLow+result.MissingHigh,
	)

	for index := 0; index < result.MissingLow; index++ {
		profiles = append(profiles, "low")
	}

	highProfiles := []string{
		"high-cpu",
		"high-memory",
	}

	for index := 0; index < result.MissingHigh; index++ {
		profiles = append(
			profiles,
			highProfiles[index%len(highProfiles)],
		)
	}

	if len(profiles) == 0 {
		fmt.Println("Reposición: no faltan contenedores")
		return nil, nil
	}

	fmt.Printf(
		"Reposición: se crearán %d contenedores\n",
		len(profiles),
	)

	createdContainers := make(
		[]dockerclient.Container,
		0,
		len(profiles),
	)

	for _, profile := range profiles {
		fmt.Printf(
			"Creando contenedor con perfil=%s...\n",
			profile,
		)

		container, err := client.CreateProjectContainer(
			ctx,
			profile,
		)
		if err != nil {
			return createdContainers, fmt.Errorf(
				"crear perfil %s: %w",
				profile,
				err,
			)
		}

		createdContainers = append(
			createdContainers,
			container,
		)

		fmt.Printf(
			"Creado ID=%.12s nombre=%s perfil=%s tier=%s\n",
			container.ID,
			container.Name,
			container.Profile,
			container.Tier,
		)
	}

	fmt.Printf(
		"Reposición completada: %d contenedores creados\n",
		len(createdContainers),
	)

	return createdContainers, nil
}
