package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var Countries = map[string]int32{"USA": 1, "RUS": 2, "CHN": 3, "ESP": 4, "GTM": 5}
var stamp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:[0-5]\d(?:\.\d{1,6})?(?:Z|[+-]\d{2}:\d{2})$`)
var runPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

type Report struct {
	Country   string `json:"country"`
	Planes    int    `json:"warplanes_in_air"`
	Ships     int    `json:"warships_in_water"`
	Timestamp string `json:"timestamp"`
}

type Event struct {
	Version    int    `json:"schema_version"`
	ID         string `json:"event_id"`
	Run        string `json:"run_id"`
	ReceivedAt string `json:"received_at"`
	Report     Report `json:"report"`
}

func Decode(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("contenido JSON adicional")
	}
	return nil
}

func exactObject(data []byte, allowed map[string]bool) error {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return errors.New("se requiere objeto JSON")
	}
	seen := map[string]bool{}
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok || !allowed[key] || seen[key] {
			return errors.New("campo desconocido o repetido")
		}
		seen[key] = true
		var value json.RawMessage
		if err = d.Decode(&value); err != nil {
			return err
		}
	}
	if _, err = d.Token(); err != nil {
		return err
	}
	if err = d.Decode(new(any)); err != io.EOF {
		return errors.New("contenido adicional")
	}
	return nil
}
func Parse(data []byte) (Report, error) {
	if err := exactObject(data, map[string]bool{"country": true, "warplanes_in_air": true, "warships_in_water": true, "timestamp": true}); err != nil {
		return Report{}, err
	}
	// Pointers distinguish absent/null quantities from valid zero values.
	var wire struct {
		Country   *string `json:"country"`
		Planes    *int    `json:"warplanes_in_air"`
		Ships     *int    `json:"warships_in_water"`
		Timestamp *string `json:"timestamp"`
	}
	if err := Decode(data, &wire); err != nil {
		return Report{}, err
	}
	if wire.Country == nil || wire.Planes == nil || wire.Ships == nil || wire.Timestamp == nil {
		return Report{}, errors.New("faltan campos obligatorios")
	}
	r := Report{*wire.Country, *wire.Planes, *wire.Ships, *wire.Timestamp}
	err := r.Normalize()
	return r, err
}

func (r *Report) Normalize() error {
	if _, ok := Countries[r.Country]; !ok {
		return errors.New("país inválido")
	}
	if r.Planes < 0 || r.Planes > 50 || r.Ships < 0 || r.Ships > 30 {
		return errors.New("cantidades fuera de rango")
	}
	if !stamp.MatchString(r.Timestamp) {
		return errors.New("timestamp inválido o precisión mayor a seis decimales")
	}
	// time.Parse accepts offset hour 24 and minute 60; reject those explicitly.
	if r.Timestamp[len(r.Timestamp)-1] != 'Z' {
		offset := r.Timestamp[len(r.Timestamp)-5:]
		if offset[:2] > "23" || offset[3:] > "59" {
			return errors.New("zona horaria inválida")
		}
	}
	t, err := time.Parse(time.RFC3339Nano, r.Timestamp)
	if err != nil || t.UTC().Year() < 1 || t.UTC().Year() > 9999 {
		return errors.New("fecha inválida")
	}
	r.Timestamp = t.UTC().Format(time.RFC3339Nano)
	return nil
}

func Identity(id, run string) (string, error) {
	if !runPattern.MatchString(run) {
		return "", errors.New("run_id inválido")
	}
	parsed, err := uuid.Parse(id)
	if err != nil || len(id) != 36 {
		return "", errors.New("event_id debe ser UUID canónico")
	}
	return parsed.String(), nil
}

func ParseEvent(data []byte) (Event, error) {
	var e Event
	if err := exactObject(data, map[string]bool{"schema_version": true, "event_id": true, "run_id": true, "received_at": true, "report": true}); err != nil {
		return e, err
	}
	if err := Decode(data, &e); err != nil {
		return e, err
	}
	if e.Version != 1 {
		return e, errors.New("versión de evento no soportada")
	}
	var raw struct {
		Report json.RawMessage `json:"report"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return e, err
	}
	r, err := Parse(raw.Report)
	if err != nil {
		return e, err
	}
	e.Report = r
	e.ID, err = Identity(e.ID, e.Run)
	if err != nil {
		return e, err
	}
	if _, err = time.Parse(time.RFC3339Nano, e.ReceivedAt); err != nil {
		return e, fmt.Errorf("received_at inválido")
	}
	return e, nil
}

func (r Report) SortKey(id string) string {
	t, _ := time.Parse(time.RFC3339Nano, r.Timestamp)
	return t.UTC().Format("2006-01-02T15:04:05.000000Z") + ":" + id
}
