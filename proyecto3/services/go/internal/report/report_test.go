package report

import (
	"strings"
	"testing"
)

const valid = `{"country":"CHN","warplanes_in_air":10,"warships_in_water":2,"timestamp":"2026-09-29T06:00:00-06:00"}`

func TestParse(t *testing.T) {
	r, err := Parse([]byte(valid))
	if err != nil || r.Timestamp != "2026-09-29T12:00:00Z" {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestInvalid(t *testing.T) {
	for _, body := range []string{
		strings.Replace(valid, `"warplanes_in_air":10`, `"warplanes_in_air":true`, 1),
		strings.Replace(valid, `"warplanes_in_air":10`, `"warplanes_in_air":10.0`, 1),
		strings.Replace(valid, `"warplanes_in_air":10`, `"warplanes_in_air":null`, 1),
		strings.Replace(valid, `"warplanes_in_air":10,`, ``, 1),
		strings.Replace(valid, `"CHN"`, `"chn"`, 1),
		strings.Replace(valid, `"warships_in_water":2`, `"warships_in_water":31`, 1),
		strings.Replace(valid, `2026-09-29T06:00:00-06:00`, `2026-09-29T06:00:00.1234567Z`, 1),
		strings.Replace(valid, `2026-09-29T06:00:00-06:00`, `2026-02-30T06:00:00Z`, 1),
		strings.Replace(valid, `2026-09-29T06:00:00-06:00`, `2026-09-29T06:00:00+00:60`, 1),
		valid + `{}`, strings.Replace(valid, `"country"`, `"extra"`, 1),
	} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Errorf("aceptó %s", body)
		}
	}
}
func TestOrdering(t *testing.T) {
	a := Report{Timestamp: "2026-09-29T12:00:00Z"}
	b := Report{Timestamp: "2026-09-29T12:00:00.000001Z"}
	if a.SortKey("z") >= b.SortKey("a") {
		t.Fatal("orden de microsegundos incorrecto")
	}
}

func TestExactUniqueKeys(t *testing.T) {
	for _, body := range []string{strings.Replace(valid, `"country"`, `"Country"`, 1), strings.Replace(valid, `"country":`, `"warplanes_in_air":20,"country":`, 1)} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("aceptó campo ambiguo: %s", body)
		}
	}
}
