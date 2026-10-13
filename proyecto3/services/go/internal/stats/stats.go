package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/report"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/storage"
)

type Row map[string]any
type Snapshot struct {
	Summary   []Row `json:"summary"`
	TopPlanes []Row `json:"top_planes"`
	TopShips  []Row `json:"top_ships"`
	Series    []Row `json:"series"`
}
type event struct {
	ID     string
	Report report.Report
}

func Build(values map[string]string, run string, from, to *time.Time) (Snapshot, error) {
	out := Snapshot{Summary: []Row{}, TopPlanes: []Row{}, TopShips: []Row{}, Series: []Row{}}
	events := []event{}
	planes := map[int]int{}
	ships := map[int]int{}
	latest := map[string]event{}
	count := 0
	for key, body := range values {
		if !strings.HasPrefix(key, "event:") {
			continue
		}
		id := strings.TrimPrefix(key, "event:")
		if _, err := report.Identity(id, run); err != nil {
			return out, fmt.Errorf("identidad almacenada inválida")
		}
		r, err := report.Parse([]byte(body))
		if err != nil {
			return out, fmt.Errorf("reporte almacenado inválido")
		}
		e := event{id, r}
		events = append(events, e)
		planes[r.Planes]++
		ships[r.Ships]++
		if previous, ok := latest[r.Country]; !ok || r.SortKey(id) > previous.Report.SortKey(previous.ID) {
			latest[r.Country] = e
		}
		if r.Country == "CHN" {
			count++
		}
	}
	if len(values) > 0 {
		if values["count"] != strconv.Itoa(len(events)) {
			return out, fmt.Errorf("contador global inconsistente")
		}
		actual := values["country:CHN:count"]
		if actual == "" {
			actual = "0"
		}
		if actual != strconv.Itoa(count) {
			return out, fmt.Errorf("contador CHN inconsistente")
		}
	}
	extrema := func(hist map[int]int) (any, any, string, int) {
		if len(hist) == 0 {
			return nil, nil, "Sin datos", 0
		}
		keys := []int{}
		maxFreq := 0
		for v, n := range hist {
			keys = append(keys, v)
			if n > maxFreq {
				maxFreq = n
			}
		}
		sort.Ints(keys)
		modes := []string{}
		for _, v := range keys {
			if hist[v] == maxFreq {
				modes = append(modes, strconv.Itoa(v))
			}
		}
		return keys[0], keys[len(keys)-1], strings.Join(modes, ", "), maxFreq
	}
	minP, maxP, modeP, freqP := extrema(planes)
	minS, maxS, modeS, freqS := extrema(ships)
	out.Summary = append(out.Summary, Row{"run": run, "country": "China (CHN)", "count": count, "total": len(events), "min_planes": minP, "max_planes": maxP, "min_ships": minS, "max_ships": maxS, "mode_planes": modeP, "mode_ships": modeS, "mode_planes_frequency": freqP, "mode_ships_frequency": freqS})
	for country, e := range latest {
		out.TopPlanes = append(out.TopPlanes, Row{"country": country, "value": e.Report.Planes})
		out.TopShips = append(out.TopShips, Row{"country": country, "value": e.Report.Ships})
	}
	for _, rows := range [][]Row{out.TopPlanes, out.TopShips} {
		sort.Slice(rows, func(i, j int) bool {
			a, b := rows[i]["value"].(int), rows[j]["value"].(int)
			if a == b {
				return rows[i]["country"].(string) < rows[j]["country"].(string)
			}
			return a > b
		})
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Report.SortKey(events[i].ID) < events[j].Report.SortKey(events[j].ID)
	})
	for _, e := range events {
		if e.Report.Country != "CHN" {
			continue
		}
		t, _ := time.Parse(time.RFC3339Nano, e.Report.Timestamp)
		if from != nil && t.Before(*from) || to != nil && t.After(*to) {
			continue
		}
		out.Series = append(out.Series, Row{"time": e.Report.Timestamp, "planes": e.Report.Planes, "ships": e.Report.Ships})
	}
	return out, nil
}
func Handler(s *storage.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		run := r.URL.Query().Get("run")
		if _, err := report.Identity("00000000-0000-4000-8000-000000000000", run); err != nil {
			http.Error(w, "corrida inválida", 400)
			return
		}
		limits := [2]*time.Time{}
		for i, name := range []string{"from", "to"} {
			if value := r.URL.Query().Get(name); value != "" {
				ms, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					http.Error(w, "rango inválido", 400)
					return
				}
				t := time.UnixMilli(ms)
				limits[i] = &t
			}
		}
		if limits[0] != nil && limits[1] != nil && limits[0].After(*limits[1]) {
			http.Error(w, "rango invertido", 400)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		values, err := s.Client.HGetAll(ctx, storage.Key(run)).Result()
		if err != nil {
			http.Error(w, "Valkey no disponible", 503)
			return
		}
		snapshot, err := Build(values, run, limits[0], limits[1])
		if err != nil {
			http.Error(w, "datos inconsistentes", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snapshot)
	})
	return mux
}
