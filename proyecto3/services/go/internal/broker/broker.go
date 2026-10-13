package broker

import (
	"context"
	"errors"
	"net"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const Queue = "mumn.reports.v1"
const Failed = "mumn.reports.failed"
const Exchange = "mumn.reports"
const Route = "report.v1"

// guard interrupts the actual socket, including synchronous AMQP calls that ignore ctx.
func guard(ctx context.Context, socket net.Conn) func() {
	if deadline, ok := ctx.Deadline(); ok {
		_ = socket.SetDeadline(deadline)
	}
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { _ = socket.Close(); close(done) })
	return func() {
		if !stop() {
			<-done
		}
		_ = socket.SetDeadline(time.Time{})
	}
}
func dial(ctx context.Context, url string) (*amqp.Connection, net.Conn, error) {
	var socket net.Conn
	stop := func() {}
	conn, err := amqp.DialConfig(url, amqp.Config{Heartbeat: 10 * time.Second, Dial: func(network, addr string) (net.Conn, error) {
		raw, err := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, addr)
		if err == nil {
			socket = raw
			stop = guard(ctx, raw)
		}
		return raw, err
	}})
	stop()
	if err != nil && socket != nil {
		_ = socket.Close()
	}
	if ctx.Err() != nil {
		if socket != nil {
			_ = socket.Close()
		}
		return nil, socket, ctx.Err()
	}
	return conn, socket, err
}
func Dial(ctx context.Context, url string) (*amqp.Connection, error) {
	limited, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, _, err := dial(limited, url)
	return conn, err
}
func Topology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(Exchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	for _, name := range []string{Queue, Failed} {
		if _, err := ch.QueueDeclare(name, true, false, false, false, nil); err != nil {
			return err
		}
	}
	return ch.QueueBind(Queue, Route, Exchange, false, nil)
}

type session struct {
	raw      net.Conn
	conn     *amqp.Connection
	ch       *amqp.Channel
	confirms <-chan amqp.Confirmation
	returns  <-chan amqp.Return
}

func (s *session) close() {
	if s.raw != nil {
		_ = s.raw.Close()
	}
	s.conn = nil
	s.ch = nil
	s.raw = nil
}

type Publisher struct {
	url   string
	slots chan *session
}

func NewPublisher(url string, size int) *Publisher {
	p := &Publisher{url, make(chan *session, size)}
	for i := 0; i < size; i++ {
		p.slots <- &session{}
	}
	return p
}
func (p *Publisher) Publish(ctx context.Context, exchange, route, id string, body []byte, headers amqp.Table) (result error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	defer func() {
		if ctx.Err() != nil {
			result = ctx.Err()
		}
	}()
	var s *session
	select {
	case s = <-p.slots:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { p.slots <- s }()
	if s.conn != nil && !s.conn.IsClosed() {
		defer guard(ctx, s.raw)()
	}
	if s.conn == nil || s.conn.IsClosed() {
		conn, raw, err := dial(ctx, p.url)
		if err != nil {
			return err
		}
		s.conn = conn
		s.raw = raw
		defer guard(ctx, s.raw)()
		ch, err := conn.Channel()
		if err != nil {
			s.close()
			return err
		}
		s.ch = ch
		if err = Topology(ch); err != nil {
			s.close()
			return err
		}
		if err = ch.Confirm(false); err != nil {
			s.close()
			return err
		}
		s.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
		s.returns = ch.NotifyReturn(make(chan amqp.Return, 1))
	}
	err := s.ch.PublishWithContext(ctx, exchange, route, true, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: id, Body: body, Headers: headers, Timestamp: time.Now().UTC()})
	if err != nil {
		s.close()
		return err
	}
	select {
	case confirm, ok := <-s.confirms:
		if !ok || !confirm.Ack {
			s.close()
			return errors.New("publicación no confirmada")
		}
		// Return is dispatched before Ack on this single-flight channel.
		select {
		case <-s.returns:
			s.close()
			return errors.New("mensaje sin ruta")
		default:
		}
		return nil
	case <-ctx.Done():
		s.close()
		return ctx.Err()
	}
}
