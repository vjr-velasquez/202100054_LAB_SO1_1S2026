package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

type Manager struct {
	projectRoot string
	cronUser    string
}

type userLookup func(string) (*user.User, error)

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

	cronUser, err := resolveCronUser(
		os.Geteuid(),
		os.Getenv("SUDO_USER"),
		user.Lookup,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolver usuario del cronjob: %w",
			err,
		)
	}

	return &Manager{
		projectRoot: absoluteRoot,
		cronUser:    cronUser,
	}, nil
}

func resolveCronUser(
	effectiveUID int,
	sudoUser string,
	lookup userLookup,
) (string, error) {
	if effectiveUID != 0 ||
		sudoUser == "" ||
		sudoUser == "root" {
		return "", nil
	}

	account, err := lookup(sudoUser)
	if err != nil {
		return "", fmt.Errorf(
			"buscar usuario %q: %w",
			sudoUser,
			err,
		)
	}

	if account.Uid == "0" {
		return "", nil
	}

	return account.Username, nil
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

	if err := manager.runCronScript(
		ctx,
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

	cronError := manager.runCronScript(
		ctx,
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

func (manager *Manager) runCronScript(
	ctx context.Context,
	scriptPath string,
) error {
	name, arguments := cronCommand(
		manager.cronUser,
		scriptPath,
	)

	return manager.run(
		ctx,
		name,
		arguments...,
	)
}

func cronCommand(
	cronUser string,
	scriptPath string,
) (string, []string) {
	if cronUser == "" {
		return "bash", []string{scriptPath}
	}

	return "sudo", []string{
		"-u",
		cronUser,
		"--",
		"bash",
		scriptPath,
	}
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
