package dockerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	projectLabel = "proyecto2"
	carnetLabel  = "202100054"
)

type containerCreateRequest struct {
	Image      string            `json:"Image"`
	Cmd        []string          `json:"Cmd,omitempty"`
	Labels     map[string]string `json:"Labels"`
	HostConfig hostConfig        `json:"HostConfig"`
}

type hostConfig struct {
	Memory   int64 `json:"Memory"`
	NanoCPUs int64 `json:"NanoCpus"`
}

type containerCreateResponse struct {
	ID       string   `json:"Id"`
	Warnings []string `json:"Warnings"`
}

func (client *Client) CreateProjectContainer(
	ctx context.Context,
	profile string,
) (Container, error) {
	requestBody, tier, err := createRequestForProfile(profile)
	if err != nil {
		return Container{}, err
	}

	name := fmt.Sprintf(
		"so1-%s-daemon-%d",
		profile,
		time.Now().UTC().UnixNano(),
	)

	query := url.Values{}
	query.Set("name", name)

	var created containerCreateResponse

	if err := client.postJSON(
		ctx,
		"/containers/create?"+query.Encode(),
		requestBody,
		http.StatusCreated,
		&created,
	); err != nil {
		return Container{}, fmt.Errorf(
			"crear contenedor %s: %w",
			profile,
			err,
		)
	}

	startPath := "/containers/" +
		url.PathEscape(created.ID) +
		"/start"

	if err := client.postJSON(
		ctx,
		startPath,
		nil,
		http.StatusNoContent,
		nil,
	); err != nil {
		_ = client.RemoveProjectContainer(ctx, created.ID)

		return Container{}, fmt.Errorf(
			"iniciar contenedor %s: %w",
			profile,
			err,
		)
	}

	return Container{
		ID:        created.ID,
		Name:      name,
		Image:     requestBody.Image,
		Profile:   profile,
		Tier:      tier,
		Protected: false,
		Running:   true,
	}, nil
}

func createRequestForProfile(
	profile string,
) (containerCreateRequest, string, error) {
	var request containerCreateRequest
	var tier string

	switch profile {
	case "low":
		tier = "low"
		request = containerCreateRequest{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "240"},
			HostConfig: hostConfig{
				Memory:   64 * 1024 * 1024,
				NanoCPUs: 100_000_000,
			},
		}

	case "high-cpu":
		tier = "high"
		request = containerCreateRequest{
			Image: "alpine:latest",
			Cmd: []string{
				"sh",
				"-c",
				"while true; do :; done",
			},
			HostConfig: hostConfig{
				Memory:   128 * 1024 * 1024,
				NanoCPUs: 500_000_000,
			},
		}

	case "high-memory":
		tier = "high"
		request = containerCreateRequest{
			Image: "roldyoran/go-client:latest",
			HostConfig: hostConfig{
				Memory:   384 * 1024 * 1024,
				NanoCPUs: 250_000_000,
			},
		}

	default:
		return containerCreateRequest{}, "", fmt.Errorf(
			"perfil de creación desconocido: %s",
			profile,
		)
	}

	request.Labels = map[string]string{
		"so1.project":    projectLabel,
		"so1.carnet":     carnetLabel,
		"so1.profile":    profile,
		"so1.tier":       tier,
		"so1.protected":  "false",
		"so1.created-by": "daemon",
	}

	return request, tier, nil
}

func (client *Client) postJSON(
	ctx context.Context,
	path string,
	body any,
	expectedStatus int,
	target any,
) error {
	var reader io.Reader

	if body != nil {
		encodedBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("codificar solicitud: %w", err)
		}

		reader = bytes.NewReader(encodedBody)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://docker"+path,
		reader,
	)
	if err != nil {
		return fmt.Errorf("crear solicitud: %w", err)
	}

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("consultar Docker: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		responseBody, _ := io.ReadAll(
			io.LimitReader(response.Body, 4096),
		)

		return fmt.Errorf(
			"Docker respondió %s: %s",
			response.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	if target == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decodificar respuesta Docker: %w", err)
	}

	return nil
}
