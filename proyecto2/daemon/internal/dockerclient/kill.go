package dockerclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (client *Client) KillProjectContainer(
	ctx context.Context,
	containerID string,
) error {
	if strings.TrimSpace(containerID) == "" {
		return fmt.Errorf("ID de contenedor vacío")
	}

	var inspected containerInspect

	inspectPath :=
		"/containers/" + url.PathEscape(containerID) + "/json"

	if err := client.getJSON(
		ctx,
		inspectPath,
		&inspected,
	); err != nil {
		return fmt.Errorf("verificar contenedor: %w", err)
	}

	labels := inspected.Config.Labels

	if labels["so1.project"] != "proyecto2" {
		return fmt.Errorf(
			"contenedor %.12s no pertenece al proyecto",
			containerID,
		)
	}

	if strings.EqualFold(
		labels["so1.protected"],
		"true",
	) {
		return fmt.Errorf(
			"contenedor %.12s está protegido",
			containerID,
		)
	}

	if !inspected.State.Running {
		return fmt.Errorf(
			"contenedor %.12s no está ejecutándose",
			containerID,
		)
	}

	query := url.Values{}
	query.Set("signal", "SIGKILL")

	killPath :=
		"/containers/" +
			url.PathEscape(containerID) +
			"/kill?" +
			query.Encode()

	if err := client.postJSON(
		ctx,
		killPath,
		nil,
		http.StatusNoContent,
		nil,
	); err != nil {
		return fmt.Errorf(
			"enviar SIGKILL mediante Docker: %w",
			err,
		)
	}

	return nil
}
