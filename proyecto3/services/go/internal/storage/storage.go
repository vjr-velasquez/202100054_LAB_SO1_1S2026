package storage

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/report"
)

//go:embed store.lua
var script string
var ErrConflict = errors.New("event_id con contenido diferente")

type Store struct{ Client *redis.Client }

func New(addr string) *Store {
	return &Store{redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second, ContextTimeoutEnabled: true})}
}
func Key(run string) string { return "mumn:run:" + run }
func (s *Store) Save(ctx context.Context, e report.Event) (bool, error) {
	body, err := json.Marshal(e.Report)
	if err != nil {
		return false, err
	}
	result, err := s.Client.Eval(ctx, script, []string{Key(e.Run)}, e.ID, string(body), e.Report.Country,
		strconv.Itoa(e.Report.Planes), strconv.Itoa(e.Report.Ships), e.Report.SortKey(e.ID)).Int()
	if err != nil {
		return false, fmt.Errorf("almacenamiento: %w", err)
	}
	if result == -1 {
		return false, ErrConflict
	}
	return result == 1, nil
}
