package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/pkg/discovery"
	"github.com/abhishek622/movieapp/pkg/discovery/consul"
	"github.com/abhishek622/movieapp/pkg/tracing"
	"github.com/abhishek622/movieapp/rating/internal/controller/rating"
	grpchandler "github.com/abhishek622/movieapp/rating/internal/handler/grpc"
	httphandler "github.com/abhishek622/movieapp/rating/internal/handler/http"
	"github.com/abhishek622/movieapp/rating/internal/repository/memory"
	"github.com/opentracing/opentracing-go"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
)

const serviceName = "rating"

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	f, err := os.Open("configs/default.yaml")
	if err != nil {
		logger.Fatal("Failed to open configuration", zap.Error(err))
	}
	var cfg config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		logger.Fatal("Failed to parse configuration", zap.Error(err))
	}
	port := cfg.API.Port
	logger.Info("Starting the rating service", zap.Int("port", port))

	registry, err := consul.NewRegistry(cfg.ServiceDiscovery.Consul.Address)
	if err != nil {
		logger.Fatal("Failed to init rating service registry", zap.Error(err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Jaeger Tracing ---
	tracer, err := tracing.NewTracerWithLogger(
		serviceName,     // "rating"
		cfg.Jaeger.Host, // Jaeger agent host
		cfg.Jaeger.Port, // Jaeger agent port
		logger,          // Custom logger
	)
	if err != nil {
		logger.Fatal("Failed to initialize Jaeger tracer", zap.Error(err))
	}
	defer func() {
		// Jaeger will handle cleanup automatically
		logger.Info("Jaeger tracer initialized successfully")
	}()
	// Set global tracer for the application
	opentracing.SetGlobalTracer(tracer)
	logger.Info("Jaeger tracer initialized successfully", zap.String("service", serviceName))

	// --- Service registration / health heartbeat ---
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("localhost:%d", port)); err != nil {
		logger.Fatal("Failed to report healthy state", zap.Error(err))
	}
	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("Failed to report healthy state: " + err.Error())
			}
			time.Sleep(1 * time.Second)
		}
	}()
	defer registry.Deregister(ctx, instanceID, serviceName)

	// --- gRPC server (mTLS) ---
	repo := memory.New()
	ctrl := rating.New(repo, nil)
	h := grpchandler.New(ctrl)
	httpHandler := httphandler.New(ctrl)
	serverCert, err := tls.LoadX509KeyPair("configs/rating-cert.pem", "configs/rating-key.pem")
	if err != nil {
		logger.Fatal("Failed to load server certificate and key", zap.Error(err))
	}
	caCert, err := os.ReadFile("configs/ca-cert.pem")
	if err != nil {
		logger.Fatal("Failed to read CA certificate", zap.Error(err))
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		logger.Fatal("Failed to append CA certificate")
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	// Start HTTP server
	go func() {
		httpMux := http.NewServeMux()
		httpMux.HandleFunc("/rating", httpHandler.Handle)
		httpServer := &http.Server{
			Addr:    fmt.Sprintf("localhost:%d", port+1000), // HTTP on port+1000
			Handler: httpMux,
		}
		logger.Info("Starting HTTP server", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", port))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.Creds(credentials.NewTLS(tlsConfig)),
	)
	reflection.Register(srv)
	gen.RegisterRatingServiceServer(srv, h)

	// Graceful shout down
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s := <-sigChan
		logger.Info("Received signal, attempting graceful shutdown", zap.Any("signal", s))

		// Shutdown Jaeger tracer
		logger.Info("Jaeger tracer shutdown completed")

		// Then cancel context and stop gRPC server
		cancel()
		srv.GracefulStop()
		logger.Info("Graceful stopped the gRPC server")
	}()
	if err := srv.Serve(lis); err != nil {
		logger.Fatal("Failed to serve gRPC server", zap.Error(err))
	}

	wg.Wait()
}
