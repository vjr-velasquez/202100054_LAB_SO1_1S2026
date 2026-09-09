package dockerclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	return function(request)
}

func TestKillProjectContainerUsesDockerSIGKILL(t *testing.T) {
	var requests []*http.Request

	client := &Client{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(
				request *http.Request,
			) (*http.Response, error) {
				requests = append(requests, request)

				if request.Method == http.MethodGet {
					return dockerResponse(
						http.StatusOK,
						`{"State":{"Running":true},"Config":{"Labels":{"so1.project":"proyecto2","so1.protected":"false"}}}`,
					), nil
				}

				return dockerResponse(http.StatusNoContent, ""), nil
			}),
		},
	}

	if err := client.KillProjectContainer(
		context.Background(),
		"container-id",
	); err != nil {
		t.Fatalf("enviar SIGKILL: %v", err)
	}

	if len(requests) != 2 {
		t.Fatalf("solicitudes=%d; se esperaban 2", len(requests))
	}

	killRequest := requests[1]
	if killRequest.Method != http.MethodPost {
		t.Errorf("método=%s; se esperaba POST", killRequest.Method)
	}
	if killRequest.URL.Path != "/containers/container-id/kill" {
		t.Errorf("ruta=%s", killRequest.URL.Path)
	}
	if killRequest.URL.Query().Get("signal") != "SIGKILL" {
		t.Errorf("signal=%q", killRequest.URL.Query().Get("signal"))
	}
}

func TestKillProjectContainerRejectsUnsafeTargets(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "foreign container",
			body: `{"State":{"Running":true},"Config":{"Labels":{"so1.project":"other"}}}`,
		},
		{
			name: "protected container",
			body: `{"State":{"Running":true},"Config":{"Labels":{"so1.project":"proyecto2","so1.protected":"true"}}}`,
		},
		{
			name: "stopped container",
			body: `{"State":{"Running":false},"Config":{"Labels":{"so1.project":"proyecto2","so1.protected":"false"}}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestCount := 0
			client := &Client{
				httpClient: &http.Client{
					Transport: roundTripFunc(func(
						request *http.Request,
					) (*http.Response, error) {
						requestCount++
						return dockerResponse(http.StatusOK, test.body), nil
					}),
				},
			}

			if err := client.KillProjectContainer(
				context.Background(),
				"container-id",
			); err == nil {
				t.Fatal("se esperaba rechazo del contenedor")
			}

			if requestCount != 1 {
				t.Fatalf("solicitudes=%d; no debía enviarse POST", requestCount)
			}
		})
	}
}

func TestKillProjectContainerRejectsEmptyID(t *testing.T) {
	client := &Client{}

	if err := client.KillProjectContainer(
		context.Background(),
		"  ",
	); err == nil {
		t.Fatal("se esperaba error para ID vacío")
	}
}

func dockerResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
