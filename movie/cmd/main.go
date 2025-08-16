package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/movie/internal/controller/movie"
	metadatagateway "github.com/abhishek622/movieapp/movie/internal/gateway/metadata/http"
	ratinggateway "github.com/abhishek622/movieapp/movie/internal/gateway/rating/http"
	grpchandler "github.com/abhishek622/movieapp/movie/internal/handler/grpc"
	"github.com/abhishek622/movieapp/pkg/discovery"
	"github.com/abhishek622/movieapp/pkg/discovery/consul"
	"github.com/abhishek622/movieapp/pkg/tracing"
	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	"github.com/opentracing/opentracing-go"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
)

const serviceName = "movie"

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
	logger.Info("Starting the movie service", zap.Int("port", port))

	registry, err := consul.NewRegistry(cfg.ServiceDiscovery.Consul.Address)
	if err != nil {
		logger.Fatal("Failed to init movie service registry", zap.Error(err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Jaeger Tracing ---
	tracer, err := tracing.NewTracerWithLogger(
		serviceName,     // "movie"
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

	// --- Service registration ---
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("localhost:%d", port)); err != nil {
		logger.Fatal("Failed to register service", zap.Error(err))
	}
	defer registry.Deregister(ctx, instanceID, serviceName)

	metadataGateway := metadatagateway.New(registry)
	ratingGateway := ratinggateway.New(registry)
	ctrl := movie.New(ratingGateway, metadataGateway)
	h := grpchandler.New(ctrl)
	serverCert, err := tls.LoadX509KeyPair("configs/movie-cert.pem", "configs/movie-key.pem")
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
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	const limit = 1000 // 1000 requests per second
	const burst = 1000 // 1000 burst capacity
	// create a limiter instance
	l := newLimiter(limit, burst)

	srv := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(ratelimit.UnaryServerInterceptor(l)),
	)

	reflection.Register(srv)
	gen.RegisterMovieServiceServer(srv, h)

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

type limiter struct {
	l *rate.Limiter
}

func newLimiter(limit int, burst int) *limiter {
	return &limiter{rate.NewLimiter(rate.Limit(limit), burst)}
}

func (l *limiter) Limit() bool {
	return !l.l.Allow() // Return true if rate limit exceeded, false if allowed
}
