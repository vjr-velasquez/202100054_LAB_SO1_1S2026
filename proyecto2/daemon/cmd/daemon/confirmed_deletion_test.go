package main

import (
	"context"
	"testing"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/policy"
)

type cancellationDockerClient struct {
	cancel      context.CancelFunc
	killCalls   int
	removeCalls int
}

func (client *cancellationDockerClient) KillProjectContainer(
	context.Context,
	string,
) error {
	client.killCalls++
	client.cancel()
	return nil
}

func (client *cancellationDockerClient) RemoveProjectContainer(
	context.Context,
	string,
) error {
	client.removeCalls++
	return nil
}

func TestRemoveConfirmedContainerDoesNotDeleteWithoutConfirmation(
	t *testing.T,
) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &cancellationDockerClient{cancel: cancel}

	_, removed, err := removeConfirmedContainer(
		ctx,
		client,
		"127.0.0.1:6379",
		policy.Candidate{
			ID:   "container-id",
			Name: "test-container",
			PID:  1234,
		},
		newDeletionTracker(),
	)
	if err == nil {
		t.Fatal("se esperaba error sin confirmación eBPF")
	}
	if removed {
		t.Fatal("el contenedor no debía marcarse como eliminado")
	}
	if client.killCalls != 1 {
		t.Errorf("solicitudes SIGKILL=%d; se esperaba 1", client.killCalls)
	}
	if client.removeCalls != 0 {
		t.Errorf("eliminaciones=%d; se esperaba 0", client.removeCalls)
	}
}
