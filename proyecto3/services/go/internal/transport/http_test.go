package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestForwardPreservesErrors(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer s.Close()
	if Status(Forward(context.Background(), s.URL, Input{})) != 503 {
		t.Fatal("se perdió error")
	}
}
func TestNoFalseAccepted(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`{"status":"accepted","event_id":"wrong"}`))
	}))
	defer s.Close()
	if Status(Forward(context.Background(), s.URL, Input{ID: "expected"})) != 502 {
		t.Fatal("aceptó identidad incorrecta")
	}
}
func TestDeadline(t *testing.T) {
	r := httptest.NewRequest("POST", "/reports", nil)
	r.Header.Set(DeadlineHeader, strconv.FormatInt(time.Now().Add(-time.Second).UnixMilli(), 10))
	ctx, cancel, err := Context(r)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatal("reinició plazo")
	}
}

func TestTimeoutAfterHeaders(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if Status(Forward(ctx, s.URL, Input{})) != 504 {
		t.Fatal("timeout convertido en error de datos")
	}
}
