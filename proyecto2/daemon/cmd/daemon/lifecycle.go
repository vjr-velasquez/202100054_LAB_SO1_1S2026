package main

import (
	"context"
	"fmt"
	"log"
	"time"

	projectlifecycle "github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/lifecycle"
)

func startManagedLifecycle(
	projectRoot string,
) (func(), error) {
	manager, err := projectlifecycle.New(projectRoot)
	if err != nil {
		return nil, fmt.Errorf(
			"preparar ciclo de vida: %w",
			err,
		)
	}

	startContext, cancelStart := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)

	err = manager.Start(startContext)
	cancelStart()

	if err != nil {
		return nil, fmt.Errorf(
			"iniciar ciclo de vida: %w",
			err,
		)
	}

	cleanup := func() {
		stopContext, cancelStop := context.WithTimeout(
			context.Background(),
			2*time.Minute,
		)
		defer cancelStop()

		if err := manager.Stop(stopContext); err != nil {
			log.Printf(
				"no se pudo finalizar completamente el ciclo de vida: %v",
				err,
			)
		}
	}

	return cleanup, nil
}
