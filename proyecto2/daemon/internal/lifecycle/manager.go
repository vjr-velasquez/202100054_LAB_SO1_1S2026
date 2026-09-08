package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Manager struct {
	projectRoot string
}

func New(projectRoot string) (*Manager, error) {
	absoluteRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf(
			"resolver ruta del proyecto: %w",
			err,
		)
	}

	requiredFiles := []string{
		"monitoring/compose.yaml",
		"kernel/Makefile",
		"cron/install_cron.sh",
		"cron/remove_cron.sh",
	}

	for _, relativePath := range requiredFiles {
		fullPath := filepath.Join(
			absoluteRoot,
			relativePath,
		)

		if _, err := os.Stat(fullPath); err != nil {
			return nil, fmt.Errorf(
				"validar %s: %w",
				fullPath,
				err,
			)
		}
	}

	return &Manager{
		projectRoot: absoluteRoot,
	}, nil
}

func (manager *Manager) Start(ctx context.Context) error {
	fmt.Println("Iniciando infraestructura de monitoreo...")

	if err := manager.run(
		ctx,
		"docker",
		"compose",
		"-f",
		filepath.Join(
			manager.projectRoot,
			"monitoring/compose.yaml",
		),
		"up",
		"-d",
		"--wait",
	); err != nil {
		return fmt.Errorf(
			"iniciar Compose: %w",
			err,
		)
	}

	fmt.Println("Cargando módulo del kernel...")

	if err := manager.run(
		ctx,
		"make",
		"-C",
		filepath.Join(manager.projectRoot, "kernel"),
		"load",
	); err != nil {
		return fmt.Errorf(
			"cargar módulo del kernel: %w",
			err,
		)
	}

	fmt.Println("Instalando cronjob del proyecto...")

	if err := manager.run(
		ctx,
		"bash",
		filepath.Join(
			manager.projectRoot,
			"cron/install_cron.sh",
		),
	); err != nil {
		rollbackContext, cancel :=
			context.WithCancel(context.Background())
		defer cancel()

		_ = manager.run(
			rollbackContext,
			"make",
			"-C",
			filepath.Join(manager.projectRoot, "kernel"),
			"unload",
		)

		return fmt.Errorf(
			"instalar cronjob: %w",
			err,
		)
	}

	fmt.Println("Ciclo de vida iniciado correctamente")

	return nil
}

func (manager *Manager) Stop(ctx context.Context) error {
	fmt.Println("Retirando cronjob del proyecto...")

	cronError := manager.run(
		ctx,
		"bash",
		filepath.Join(
			manager.projectRoot,
			"cron/remove_cron.sh",
		),
	)

	fmt.Println("Descargando módulo del kernel...")

	moduleError := manager.run(
		ctx,
		"make",
		"-C",
		filepath.Join(manager.projectRoot, "kernel"),
		"unload",
	)

	if err := errors.Join(cronError, moduleError); err != nil {
		return fmt.Errorf(
			"finalizar ciclo de vida: %w",
			err,
		)
	}

	fmt.Println("Ciclo de vida finalizado correctamente")

	return nil
}

func (manager *Manager) run(
	ctx context.Context,
	name string,
	arguments ...string,
) error {
	command := exec.CommandContext(
		ctx,
		name,
		arguments...,
	)

	command.Dir = manager.projectRoot
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf(
			"ejecutar %s: %w",
			name,
			err,
		)
	}

	return nil
}
