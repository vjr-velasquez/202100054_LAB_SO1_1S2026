package dockerclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

type Container struct {
	ID        string
	Name      string
	Image     string
	PID       int
	Profile   string
	Tier      string
	Protected bool
	Running   bool
}

type containerSummary struct {
	ID string `json:"Id"`
}

type containerInspect struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`

	State struct {
		PID     int  `json:"Pid"`
		Running bool `json:"Running"`
	} `json:"State"`

	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

func New(socketPath string) *Client {
	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(
			ctx context.Context,
			network string,
			address string,
		) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		},
	}
}

func (client *Client) ListProjectContainers(
	ctx context.Context,
) ([]Container, error) {
	filterJSON, err := json.Marshal(map[string][]string{
		"label": {"so1.project=proyecto2"},
	})
	if err != nil {
		return nil, fmt.Errorf("crear filtro Docker: %w", err)
	}

	query := url.Values{}
	query.Set("all", "false")
	query.Set("filters", string(filterJSON))

	var summaries []containerSummary

	if err := client.getJSON(
		ctx,
		"/containers/json?"+query.Encode(),
		&summaries,
	); err != nil {
		return nil, fmt.Errorf("listar contenedores: %w", err)
	}

	containers := make([]Container, 0, len(summaries))

	for _, summary := range summaries {
		var inspected containerInspect

		path := "/containers/" + url.PathEscape(summary.ID) + "/json"

		if err := client.getJSON(ctx, path, &inspected); err != nil {
			return nil, fmt.Errorf(
				"inspeccionar contenedor %.12s: %w",
				summary.ID,
				err,
			)
		}

		labels := inspected.Config.Labels
		if labels == nil {
			labels = map[string]string{}
		}

		containers = append(containers, Container{
			ID:        inspected.ID,
			Name:      strings.TrimPrefix(inspected.Name, "/"),
			Image:     inspected.Config.Image,
			PID:       inspected.State.PID,
			Profile:   labels["so1.profile"],
			Tier:      labels["so1.tier"],
			Protected: strings.EqualFold(labels["so1.protected"], "true"),
			Running:   inspected.State.Running,
		})
	}

	return containers, nil
}

func (client *Client) getJSON(
	ctx context.Context,
	path string,
	target any,
) error {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"http://docker"+path,
		nil,
	)
	if err != nil {
		return fmt.Errorf("crear solicitud: %w", err)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("consultar Docker: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))

		return fmt.Errorf(
			"Docker respondió %s: %s",
			response.Status,
			strings.TrimSpace(string(body)),
		)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decodificar respuesta Docker: %w", err)
	}

	return nil
}

func (client *Client) RemoveProjectContainer(
	ctx context.Context,
	containerID string,
) error {
	if strings.TrimSpace(containerID) == "" {
		return fmt.Errorf("ID de contenedor vacío")
	}

	var inspected containerInspect

	inspectPath :=
		"/containers/" + url.PathEscape(containerID) + "/json"

	if err := client.getJSON(ctx, inspectPath, &inspected); err != nil {
		return fmt.Errorf("verificar contenedor: %w", err)
	}

	labels := inspected.Config.Labels

	if labels["so1.project"] != "proyecto2" {
		return fmt.Errorf(
			"contenedor %.12s no pertenece al proyecto",
			containerID,
		)
	}

	if strings.EqualFold(labels["so1.protected"], "true") {
		return fmt.Errorf(
			"contenedor %.12s está protegido",
			containerID,
		)
	}

	query := url.Values{}
	query.Set("force", "true")
	query.Set("v", "true")

	deletePath :=
		"/containers/" +
			url.PathEscape(containerID) +
			"?" +
			query.Encode()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		"http://docker"+deletePath,
		nil,
	)
	if err != nil {
		return fmt.Errorf("crear solicitud de eliminación: %w", err)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("eliminar contenedor: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))

		return fmt.Errorf(
			"Docker respondió %s: %s",
			response.Status,
			strings.TrimSpace(string(body)),
		)
	}

	return nil
}
