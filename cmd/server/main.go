package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	pb "github.com/zgq/wallet/gen/wallet"
	grpchandler "github.com/zgq/wallet/internal/handler/grpc"
	resthandler "github.com/zgq/wallet/internal/handler/rest"
	"github.com/zgq/wallet/internal/repository"
	"github.com/zgq/wallet/internal/service"
)

func main() {
	repo, err := newRepo()
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}
	svc := service.New(repo)

	// --- REST server ---
	r := gin.Default()
	resthandler.NewHandler(svc).RegisterRoutes(r)

	httpAddr := envOr("HTTP_ADDR", ":8080")
	httpServer := &http.Server{Addr: httpAddr, Handler: r}

	// --- gRPC server ---
	grpcAddr := envOr("GRPC_ADDR", ":5505")
	grpcServer := grpc.NewServer()
	pb.RegisterWalletServiceServer(grpcServer, grpchandler.NewServer(svc))

	// Start gRPC
	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatalf("gRPC listen: %v", err)
		}
		log.Printf("gRPC listening on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC serve: %v", err)
		}
	}()

	// Start HTTP
	go func() {
		log.Printf("HTTP listening on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP serve: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx) //nolint:errcheck
	grpcServer.GracefulStop()
}

func newRepo() (repository.Repository, error) {
	switch envOr("STORAGE_TYPE", "memory") {
	case "postgres":
		dsn := envOr("POSTGRES_DSN", "postgres://localhost/wallet?sslmode=disable")
		return repository.NewPostgresRepo(dsn)
	default:
		return repository.NewMemoryRepo(), nil
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
