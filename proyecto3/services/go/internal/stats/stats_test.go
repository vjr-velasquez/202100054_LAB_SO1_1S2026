package stats

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

//go:embed testdata/reports.json
var data []byte

func fixture(t *testing.T) map[string]string {
	var err error
	var rows []struct {
		ID     string          `json:"event_id"`
		Report json.RawMessage `json:"report"`
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"count": strconv.Itoa(len(rows)), "country:CHN:count": "3"}
	for _, r := range rows {
		values["event:"+r.ID] = string(r.Report)
	}
	return values
}
func TestFixture(t *testing.T) {
	s, err := Build(fixture(t), "fixture", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	v := s.Summary[0]
	if v["max_planes"] != 30 || v["min_planes"] != 0 || v["max_ships"] != 8 || v["min_ships"] != 0 || v["mode_planes"] != "20" || v["mode_ships"] != "5" || v["mode_planes_frequency"] != 2 || v["mode_ships_frequency"] != 2 || v["count"] != 3 || len(s.Series) != 3 || s.TopPlanes[0]["country"] != "CHN" {
		t.Fatalf("%+v", s)
	}
}
func TestFilterOnlySeries(t *testing.T) {
	from := time.Date(2026, 9, 29, 12, 3, 0, 0, time.UTC)
	s, err := Build(fixture(t), "fixture", &from, nil)
	if err != nil || len(s.Series) != 1 || s.Summary[0]["count"] != 3 {
		t.Fatalf("%+v %v", s, err)
	}
}
func TestEmptyAndCorrupt(t *testing.T) {
	s, err := Build(map[string]string{}, "empty", nil, nil)
	if err != nil || s.Summary[0]["max_planes"] != nil || s.Summary[0]["count"] != 0 {
		t.Fatal(s, err)
	}
	v := fixture(t)
	v["count"] = "900"
	if _, err = Build(v, "broken", nil, nil); err == nil {
		t.Fatal("aceptó corrupción")
	}
}
