package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	model "github.com/fairwix/resource-booking-service/internal/model"
	"github.com/fairwix/resource-booking-service/internal/model/desk"
	"github.com/fairwix/resource-booking-service/internal/model/room"
	"github.com/fairwix/resource-booking-service/internal/service"
	grpctransport "github.com/fairwix/resource-booking-service/internal/transport/grpc"
	httptransport "github.com/fairwix/resource-booking-service/internal/transport/http"
	pb "github.com/fairwix/resource-booking-service/proto"
)

func main() {
	shutdownTimeout := 5 * time.Second
	if value := os.Getenv("SHUTDOWN_TIMEOUT"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			log.Fatalf("SHUTDOWN_TIMEOUT must be a positive duration (for example, 5s): %q", value)
		}
		shutdownTimeout = parsed
	}

	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		pool, err := pgxpool.New(context.Background(), databaseURL)
		if err != nil {
			log.Fatalf("failed to configure PostgreSQL connection: %v", err)
		}
		defer pool.Close()

		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = pool.Ping(pingCtx)
		cancel()
		if err != nil {
			log.Fatalf("failed to connect to PostgreSQL: %v", err)
		}
		log.Println("connected to PostgreSQL")
	}

	meetingRoom, err := room.NewRoom(1, "Meeting Room", 10)
	if err != nil {
		log.Printf("failed to create room: %v", err)
		return
	}

	deskResource, err := desk.NewDesk(2, 15)
	if err != nil {
		log.Printf("failed to create desk: %v", err)
		return
	}

	slot, err := model.NewTimeSlot(time.Now(), time.Now().Add(time.Hour))
	if err != nil {
		log.Printf("failed to create time slot: %v", err)
		return
	}
	log.Printf("created slot: %s", slot)

	model.ShiftSlot(&slot, 30*time.Minute)
	log.Printf("shifted slot: %s", slot)

	bookingService := service.NewBookingService(map[int]model.Reservable{
		1: &meetingRoom,
		2: &deskResource,
	})

	httpHandler := httptransport.NewHandler(bookingService)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", httpHandler.Health)
	mux.HandleFunc("/resources", httpHandler.GetState)
	mux.HandleFunc("/resources/book", httpHandler.Book)

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: mux,
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Printf("failed to create gRPC listener: %v", err)
		return
	}
	defer grpcListener.Close()

	grpcServer := grpc.NewServer()

	grpcHandler := grpctransport.NewServer(bookingService)

	pb.RegisterBookingServiceServer(
		grpcServer,
		grpcHandler,
	)

	healthServer := health.NewServer()

	grpc_health_v1.RegisterHealthServer(
		grpcServer,
		healthServer,
	)

	reflection.Register(grpcServer)

	healthServer.SetServingStatus(
		"",
		grpc_health_v1.HealthCheckResponse_SERVING,
	)

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	g, ctx := errgroup.WithContext(signalCtx)

	g.Go(func() error {
		log.Printf("HTTP server started on :%s", httpPort)

		err := httpServer.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server failed: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		log.Printf("gRPC server started on :%s", grpcPort)

		err := grpcServer.Serve(grpcListener)

		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("gRPC server failed: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		<-ctx.Done()

		log.Println("shutting down servers...")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		healthServer.SetServingStatus(
			"",
			grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		)

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown failed: %v", err)
		}

		grpcStopped := make(chan struct{})

		go func() {
			grpcServer.GracefulStop()
			close(grpcStopped)
		}()

		select {
		case <-grpcStopped:
			log.Println("gRPC server stopped gracefully")

		case <-shutdownCtx.Done():
			log.Println("gRPC graceful shutdown timed out")
			grpcServer.Stop()
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		log.Printf("server stopped with error: %v", err)
		return
	}

	log.Println("servers stopped")
}

// errGroup все пишут через него без go

// планировшик горутин
//свитч контекст
