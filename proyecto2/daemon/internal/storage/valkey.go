package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/valkey-io/valkey-go"
)

const (
	latestSampleKey = "so1:proyecto2:telemetry:latest"
	historyKey      = "so1:proyecto2:telemetry:history"
	historyLimit    = 1000
)

type Store struct {
	client valkey.Client
}

func New(address string) (*Store, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("la dirección de Valkey está vacía")
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{address},
	})
	if err != nil {
		return nil, fmt.Errorf("crear cliente de Valkey: %w", err)
	}

	return &Store{
		client: client,
	}, nil
}

func (store *Store) Close() {
	if store == nil || store.client == nil {
		return
	}

	store.client.Close()
}

func (store *Store) Ping(ctx context.Context) error {
	if store == nil || store.client == nil {
		return fmt.Errorf("cliente de Valkey no inicializado")
	}

	if err := store.client.Do(
		ctx,
		store.client.B().Ping().Build(),
	).Error(); err != nil {
		return fmt.Errorf("comprobar conexión con Valkey: %w", err)
	}

	return nil
}

func (store *Store) SaveSample(
	ctx context.Context,
	sample any,
) error {
	if store == nil || store.client == nil {
		return fmt.Errorf("cliente de Valkey no inicializado")
	}

	payload, err := json.Marshal(sample)
	if err != nil {
		return fmt.Errorf("serializar muestra de telemetría: %w", err)
	}

	results := store.client.DoMulti(
		ctx,
		store.client.B().
			Set().
			Key(latestSampleKey).
			Value(string(payload)).
			Build(),

		store.client.B().
			Lpush().
			Key(historyKey).
			Element(string(payload)).
			Build(),

		store.client.B().
			Ltrim().
			Key(historyKey).
			Start(0).
			Stop(historyLimit-1).
			Build(),
	)

	for index, result := range results {
		if err := result.Error(); err != nil {
			return fmt.Errorf(
				"guardar muestra en Valkey, operación %d: %w",
				index+1,
				err,
			)
		}
	}

	return nil
}
