//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/broker"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/report"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/storage"
)

func TestAtomicStorage(t *testing.T) {
	addr := os.Getenv("VALKEY_ADDR")
	if addr == "" {
		t.Fatal("VALKEY_ADDR obligatorio para integración")
	}
	s := storage.New(addr)
	defer s.Client.Close()
	ctx := context.Background()
	e := report.Event{Version: 1, ID: uuid.NewString(), Run: "test-" + uuid.NewString(), ReceivedAt: time.Now().UTC().Format(time.RFC3339Nano), Report: report.Report{Country: "CHN", Planes: 10, Ships: 2, Timestamp: "2026-09-29T12:00:00Z"}}
	key := storage.Key(e.Run)
	defer s.Client.Del(ctx, key)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Save(ctx, e); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if count := s.Client.HGet(ctx, key, "count").Val(); count != "1" {
		t.Fatalf("duplicados: %s", count)
	}
	conflict := e
	conflict.Report.Planes = 30
	if _, err := s.Save(ctx, conflict); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("conflicto: %v", err)
	}
	if count := s.Client.HGet(ctx, key, "count").Val(); count != "1" {
		t.Fatal("conflicto alteró contador")
	}
	second := e
	second.ID = uuid.NewString()
	if _, err := s.Save(ctx, second); err != nil {
		t.Fatal(err)
	}
	if count := s.Client.HGet(ctx, key, "count").Val(); count != "2" {
		t.Fatal("evento legítimo descartado")
	}
	before := s.Client.HGetAll(ctx, key).Val()
	s.Client.HSet(ctx, key, "planes:10", "broken")
	third := e
	third.ID = uuid.NewString()
	if _, err := s.Save(ctx, third); err == nil {
		t.Fatal("aceptó estado corrupto")
	}
	if s.Client.HExists(ctx, key, "event:"+third.ID).Val() {
		t.Fatal("escritura parcial")
	}
	if s.Client.HGet(ctx, key, "count").Val() != before["count"] {
		t.Fatal("modificó contador antes de fallar")
	}
}
func TestPublisherRequiresRouting(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Fatal("RABBITMQ_URL obligatorio")
	}
	pub := broker.NewPublisher(url, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	body, _ := json.Marshal(map[string]string{"probe": "unroutable"})
	if err := pub.Publish(ctx, broker.Exchange, "no-such-route-"+uuid.NewString(), uuid.NewString(), body, nil); err == nil {
		t.Fatal("aceptó mensaje sin ruta")
	}
}
