package main

import (
	"context"
	"fmt"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto2/daemon/internal/ebpfwatcher"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	watcher, err := ebpfwatcher.New()
	if err != nil {
		log.Fatalf("iniciar monitor eBPF: %v", err)
	}

	stopContext, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	go func() {
		<-stopContext.Done()
		_ = watcher.Close()
	}()

	fmt.Println(
		"Monitor eBPF activo; esperando llamadas kill...",
	)

	for {
		event, err := watcher.Read()
		if err != nil {
			if stopContext.Err() != nil {
				fmt.Println("Monitor eBPF finalizado")
				return
			}

			_ = watcher.Close()
			log.Fatalf("leer evento eBPF: %v", err)
		}

		fmt.Printf(
			"Evento kill: emisor=%d objetivo=%d señal=%d comando=%s observado=%s\n",
			event.CallerPID,
			event.TargetPID,
			event.Signal,
			event.Command,
			event.ObservedAt.Format("2006-01-02T15:04:05Z07:00"),
		)
	}
}
