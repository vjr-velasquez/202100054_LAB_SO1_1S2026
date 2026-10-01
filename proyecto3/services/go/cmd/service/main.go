package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	pb "github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/gen/wartweets"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/broker"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/report"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/storage"
	"github.com/vjr-velasquez/202100054_LAB_SO1_1S2026/proyecto3/services/go/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if len(os.Args) != 2 {
		slog.Error("uso: service api|grpc-client|grpc-server|writer|consumer")
		os.Exit(2)
	}
	if err := run(ctx, os.Args[1]); err != nil {
		slog.Error("servicio detenido", "role", os.Args[1], "error", err)
		os.Exit(1)
	}
}
func run(ctx context.Context, role string) error {
	switch role {
	case "consumer":
		return consume(ctx)
	case "grpc-server":
		listener, err := net.Listen("tcp", env("LISTEN_ADDR", ":50051"))
		if err != nil {
			return err
		}
		server := grpc.NewServer(grpc.MaxConcurrentStreams(128), grpc.MaxRecvMsgSize(4096))
		pb.RegisterWarReportServiceServer(server, &rpcServer{writer: env("WRITER_URL", "http://127.0.0.1:8083/publish")})
		go func() { <-ctx.Done(); server.Stop() }()
		return server.Serve(listener)
	}
	var fn func(context.Context, transport.Input) error
	path := "/reports"
	address := ":8081"
	switch role {
	case "api":
		fn = func(ctx context.Context, in transport.Input) error {
			return transport.Forward(ctx, env("GRPC_BRIDGE_URL", "http://127.0.0.1:8082/reports"), in)
		}
	case "grpc-client":
		address = ":8082"
		conn, err := grpc.NewClient(env("GRPC_TARGET", "grpc-server:50051"), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		defer conn.Close()
		client := pb.NewWarReportServiceClient(conn)
		fn = func(ctx context.Context, in transport.Input) error {
			ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("event-id", in.ID, "run-id", in.Run))
			reply, err := client.SendReport(ctx, &pb.WarReportRequest{Country: pb.Countries(report.Countries[in.Report.Country]), WarplanesInAir: int32(in.Report.Planes), WarshipsInWater: int32(in.Report.Ships), Timestamp: in.Report.Timestamp})
			if err != nil {
				return &transport.Error{Status: rpcHTTP(status.Code(err)), Message: "gRPC no aceptó el evento"}
			}
			if reply.Status != "accepted" {
				return &transport.Error{Status: 502, Message: "respuesta gRPC inválida"}
			}
			return nil
		}
	case "writer":
		address = ":8083"
		path = "/publish"
		pub := broker.NewPublisher(env("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"), 4)
		fn = func(ctx context.Context, in transport.Input) error {
			e := report.Event{Version: 1, ID: in.ID, Run: in.Run, ReceivedAt: time.Now().UTC().Format(time.RFC3339Nano), Report: in.Report}
			body, _ := json.Marshal(e)
			err := pub.Publish(ctx, broker.Exchange, broker.Route, in.ID, body, nil)
			if err == nil {
				slog.Info("published", "event_id", in.ID, "run_id", in.Run)
			}
			return err
		}
	default:
		return fmt.Errorf("rol desconocido: %s", role)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+path, transport.Handler(fn))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) })
	server := &http.Server{Addr: env("LISTEN_ADDR", address), Handler: mux, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 6 * time.Second, WriteTimeout: 7 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		<-ctx.Done()
		stop, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(stop)
	}()
	slog.Info("listening", "role", role, "address", server.Addr)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func rpcHTTP(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return 400
	case codes.ResourceExhausted:
		return 429
	case codes.DeadlineExceeded, codes.Canceled:
		return 504
	case codes.Unavailable:
		return 503
	default:
		return 500
	}
}
func httpRPC(code int) codes.Code {
	switch code {
	case 400:
		return codes.InvalidArgument
	case 429:
		return codes.ResourceExhausted
	case 504:
		return codes.DeadlineExceeded
	case 503:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

type rpcServer struct {
	pb.UnimplementedWarReportServiceServer
	writer string
}

func (s *rpcServer) SendReport(ctx context.Context, req *pb.WarReportRequest) (*pb.WarReportResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	country := ""
	for name, value := range report.Countries {
		if int32(req.Country) == value {
			country = name
		}
	}
	r := report.Report{Country: country, Planes: int(req.WarplanesInAir), Ships: int(req.WarshipsInWater), Timestamp: req.Timestamp}
	if err := r.Normalize(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	md, _ := metadata.FromIncomingContext(ctx)
	idValues, runValues := md.Get("event-id"), md.Get("run-id")
	if len(idValues) != 1 || len(runValues) != 1 {
		return nil, status.Error(codes.InvalidArgument, "identidad ausente o ambigua")
	}
	id, err := report.Identity(idValues[0], runValues[0])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err = transport.Forward(ctx, s.writer, transport.Input{Report: r, ID: id, Run: runValues[0]}); err != nil {
		return nil, status.Error(httpRPC(transport.Status(err)), "writer no aceptó evento")
	}
	return &pb.WarReportResponse{Status: "accepted"}, nil
}
func consume(ctx context.Context) error {
	url := env("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	store := storage.New(env("VALKEY_ADDR", "valkey:6379"))
	defer store.Client.Close()
	failed := broker.NewPublisher(url, 1)
	for ctx.Err() == nil {
		err := consumeSession(ctx, url, store, failed)
		if ctx.Err() != nil {
			return nil
		}
		slog.Warn("consumer_reconnecting", "error", err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
	return nil
}
func consumeSession(ctx context.Context, url string, store *storage.Store, failed *broker.Publisher) error {
	conn, err := broker.Dial(ctx, url)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	if err = broker.Topology(ch); err != nil {
		return err
	}
	if err = ch.Qos(32, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.ConsumeWithContext(ctx, broker.Queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("canal de consumo cerrado")
			}
			e, parseErr := report.ParseEvent(d.Body)
			for {
				saveErr := parseErr
				if saveErr == nil {
					op, cancel := context.WithTimeout(ctx, 3*time.Second)
					_, saveErr = store.Save(op, e)
					cancel()
				}
				if saveErr == nil {
					if err = d.Ack(false); err != nil {
						return err
					}
					slog.Info("stored", "event_id", e.ID, "run_id", e.Run)
					break
				}
				if parseErr != nil || errors.Is(saveErr, storage.ErrConflict) {
					op, cancel := context.WithTimeout(ctx, 5*time.Second)
					err = failed.Publish(op, "", broker.Failed, d.MessageId, d.Body, amqp.Table{"failure_reason": saveErr.Error()})
					cancel()
					if err == nil {
						if err = d.Ack(false); err != nil {
							return err
						}
						slog.Warn("event_failed", "event_id", e.ID, "error", saveErr)
						break
					}
					return err // Closing the channel requeues the unacknowledged original.
				}
				slog.Warn("storage_retry", "event_id", e.ID, "error", saveErr)
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(time.Second):
				}
			}
		}
	}
}
