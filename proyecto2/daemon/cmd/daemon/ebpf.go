package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/ebpfwatcher"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/storage"
)

func startEBPFMonitor(
	valkeyAddress string,
	tracker *deletionTracker,
) (func(), error) {
	store, err := storage.New(valkeyAddress)
	if err != nil {
		return nil, fmt.Errorf(
			"crear almacenamiento eBPF: %w",
			err,
		)
	}

	pingContext, cancelPing := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	err = store.Ping(pingContext)
	cancelPing()

	if err != nil {
		store.Close()

		return nil, fmt.Errorf(
			"conectar eBPF con Valkey: %w",
			err,
		)
	}

	watcher, err := ebpfwatcher.New()
	if err != nil {
		store.Close()

		return nil, fmt.Errorf(
			"iniciar lector eBPF: %w",
			err,
		)
	}

	monitorContext, cancelMonitor :=
		context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			event, err := watcher.Read()
			if err != nil {
				if monitorContext.Err() != nil {
					return
				}

				log.Printf(
					"el monitor eBPF terminó con error: %v",
					err,
				)
				if tracker != nil {
					tracker.Disable(err)
				}
				return
			}

			relevant := false
			if tracker != nil {
				switch event.Source {
				case ebpfwatcher.EventSourceSysKill:
					relevant = tracker.MatchesPendingSysKill(event)
				case ebpfwatcher.EventSourceSignalGenerate:
					relevant = tracker.Observe(event)
				}
			}

			if !relevant {
				continue
			}

			fmt.Printf(
				"eBPF kill: origen=%s emisor=%d objetivo=%d señal=%d comando=%s\n",
				event.Source,
				event.CallerPID,
				event.TargetPID,
				event.Signal,
				event.Command,
			)

			saveContext, cancelSave := context.WithTimeout(
				monitorContext,
				5*time.Second,
			)

			err = store.SaveEBPFEvent(
				saveContext,
				event,
			)
			cancelSave()

			if err != nil {
				log.Printf(
					"no se pudo guardar el evento eBPF: %v",
					err,
				)
			}
		}
	}()

	cleanup := func() {
		fmt.Println("Deteniendo monitor eBPF...")

		cancelMonitor()

		if err := watcher.Close(); err != nil {
			log.Printf(
				"cerrar monitor eBPF: %v",
				err,
			)
		}

		<-done
		store.Close()

		fmt.Println("Monitor eBPF finalizado correctamente")
	}

	fmt.Println("Monitor eBPF conectado con Valkey")

	return cleanup, nil
}
