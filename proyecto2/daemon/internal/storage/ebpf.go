package storage

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	latestEBPFEventKey = "so1:proyecto2:ebpf:latest"
	ebpfHistoryKey     = "so1:proyecto2:ebpf:history"
	ebpfHistoryLimit   = 1000
)

func (store *Store) SaveEBPFEvent(
	ctx context.Context,
	event any,
) error {
	if store == nil || store.client == nil {
		return fmt.Errorf(
			"cliente de Valkey no inicializado",
		)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf(
			"serializar evento eBPF: %w",
			err,
		)
	}

	results := store.client.DoMulti(
		ctx,
		store.client.B().
			Set().
			Key(latestEBPFEventKey).
			Value(string(payload)).
			Build(),

		store.client.B().
			Lpush().
			Key(ebpfHistoryKey).
			Element(string(payload)).
			Build(),

		store.client.B().
			Ltrim().
			Key(ebpfHistoryKey).
			Start(0).
			Stop(ebpfHistoryLimit-1).
			Build(),
	)

	for index, result := range results {
		if err := result.Error(); err != nil {
			return fmt.Errorf(
				"guardar evento eBPF, operación %d: %w",
				index+1,
				err,
			)
		}
	}

	return nil
}
