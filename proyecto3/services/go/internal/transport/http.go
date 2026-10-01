package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/report"
)

var Client = &http.Client{Timeout: 6 * time.Second}

const DeadlineHeader = "X-Request-Deadline"

type Input struct {
	Report  report.Report
	ID, Run string
}
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }
func Fail(w http.ResponseWriter, status int, message, id string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": http.StatusText(status), "message": message}, "event_id": id})
}
func Accept(w http.ResponseWriter, in Input) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(202)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted", "event_id": in.ID, "run_id": in.Run})
}
func Read(w http.ResponseWriter, r *http.Request) (Input, error) {
	in := Input{ID: r.Header.Get("Idempotency-Key"), Run: r.Header.Get("X-Run-ID")}
	content, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || content != "application/json" {
		return in, &Error{415, "se requiere application/json"}
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		var max *http.MaxBytesError
		if errors.As(err, &max) {
			return in, &Error{413, "reporte demasiado grande"}
		}
		return in, &Error{400, "cuerpo inválido"}
	}
	in.Report, err = report.Parse(data)
	if err != nil {
		return in, &Error{400, err.Error()}
	}
	in.ID, err = report.Identity(in.ID, in.Run)
	if err != nil {
		return in, &Error{400, err.Error()}
	}
	return in, nil
}
func Context(r *http.Request) (context.Context, context.CancelFunc, error) {
	deadline := time.Now().Add(5 * time.Second)
	if raw := r.Header.Get(DeadlineHeader); raw != "" {
		ms, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		provided := time.UnixMilli(ms)
		if provided.Before(deadline) {
			deadline = provided
		}
	}
	ctx, cancel := context.WithDeadline(r.Context(), deadline)
	return ctx, cancel, nil
}
func Forward(ctx context.Context, url string, in Input) error {
	body, _ := json.Marshal(in.Report)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", in.ID)
	req.Header.Set("X-Run-ID", in.Run)
	if deadline, ok := ctx.Deadline(); ok {
		req.Header.Set(DeadlineHeader, strconv.FormatInt(deadline.UnixMilli(), 10))
	}
	res, err := Client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &Error{503, "servicio no disponible"}
	}
	defer res.Body.Close()
	if res.StatusCode != 202 {
		code := res.StatusCode
		switch code {
		case 400, 413, 415, 429, 503, 504, 500:
		default:
			code = 502
		}
		return &Error{code, "dependencia rechazó la solicitud"}
	}
	var reply struct{ Status, EventID, RunID string }
	var raw map[string]string
	if err = json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&raw); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return &Error{502, "respuesta inválida de dependencia"}
	}
	reply.Status = raw["status"]
	reply.EventID = raw["event_id"]
	reply.RunID = raw["run_id"]
	if reply.Status != "accepted" || reply.EventID != in.ID || reply.RunID != in.Run {
		return &Error{502, "confirmación inconsistente"}
	}
	return nil
}
func Status(err error) int {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return 504
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Status
	}
	return 503
}
func Handler(fn func(context.Context, Input) error) http.HandlerFunc {
	slots := make(chan struct{}, 128)
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			Fail(w, 429, "concurrencia máxima alcanzada", "")
			return
		}
		in, err := Read(w, r)
		if err != nil {
			Fail(w, Status(err), err.Error(), in.ID)
			return
		}
		ctx, cancel, err := Context(r)
		if err != nil {
			Fail(w, 400, "deadline inválido", in.ID)
			return
		}
		defer cancel()
		if ctx.Err() != nil {
			Fail(w, 504, "plazo agotado", in.ID)
			return
		}
		if err = fn(ctx, in); err != nil {
			Fail(w, Status(err), http.StatusText(Status(err)), in.ID)
			return
		}
		Accept(w, in)
	}
}
