// Fixture publisher used only by the local Kubernetes validation job.
package main

import (
	"context"
	"encoding/json"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/broker"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/report"
	"log"
	"os"
	"time"
)

func main() {
	data, err := os.ReadFile("/fixtures/reports.json")
	if err != nil {
		log.Fatal(err)
	}
	var rows []struct {
		ID     string        `json:"event_id"`
		Report report.Report `json:"report"`
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		log.Fatal(err)
	}
	p := broker.NewPublisher(os.Getenv("RABBITMQ_URL"), 1)
	for _, r := range rows {
		e := report.Event{Version: 1, ID: r.ID, Run: "fixture", Report: r.Report, ReceivedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		body, _ := json.Marshal(e)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = p.Publish(ctx, broker.Exchange, broker.Route, e.ID, body, nil)
		cancel()
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Print("5 fixture events published")
}
