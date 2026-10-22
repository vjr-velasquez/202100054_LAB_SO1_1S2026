package broker

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestBlockedHandshakeReleasesSlots(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				buffer := make([]byte, 64)
				for {
					if _, err := conn.Read(buffer); err != nil {
						return
					}
				}
			}()
		}
	}()
	p := NewPublisher("amqp://guest:guest@"+listener.Addr().String()+"/", 1)
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		start := time.Now()
		err = p.Publish(ctx, Exchange, Route, "test", []byte(`{}`), nil)
		cancel()
		if err == nil || time.Since(start) > time.Second {
			t.Fatalf("publicación bloqueada: %v", err)
		}
		if len(p.slots) != 1 {
			t.Fatal("slot perdido")
		}
	}
}
func TestGuardInterruptsBlockedWrite(t *testing.T) {
	writer, reader := net.Pipe()
	defer reader.Close()
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	stop := guard(ctx, writer)
	defer stop()
	start := time.Now()
	_, err := writer.Write([]byte("blocked"))
	if err == nil || time.Since(start) > time.Second {
		t.Fatal("write no interrumpido")
	}
}
