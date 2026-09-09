package storage

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	latestContainerDeletionKey  = "so1:proyecto2:containers:deleted:latest"
	containerDeletionHistoryKey = "so1:proyecto2:containers:deleted:history"
	containerDeletionLimit      = 1000
)

func (store *Store) SaveContainerDeletion(
	ctx context.Context,
	deletion any,
) error {
	if store == nil || store.client == nil {
		return fmt.Errorf("cliente de Valkey no inicializado")
	}

	payload, err := json.Marshal(deletion)
	if err != nil {
		return fmt.Errorf(
			"serializar eliminación de contenedor: %w",
			err,
		)
	}

	results := store.client.DoMulti(
		ctx,
		store.client.B().
			Set().
			Key(latestContainerDeletionKey).
			Value(string(payload)).
			Build(),

		store.client.B().
			Lpush().
			Key(containerDeletionHistoryKey).
			Element(string(payload)).
			Build(),

		store.client.B().
			Ltrim().
			Key(containerDeletionHistoryKey).
			Start(0).
			Stop(containerDeletionLimit-1).
			Build(),
	)

	for index, result := range results {
		if err := result.Error(); err != nil {
			return fmt.Errorf(
				"guardar eliminación, operación %d: %w",
				index+1,
				err,
			)
		}
	}

	return nil
}
