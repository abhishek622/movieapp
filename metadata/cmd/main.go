package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/abhishek622/movieapp/gen"
	"github.com/abhishek622/movieapp/metadata/internal/controller/metadata"
	grpchandler "github.com/abhishek622/movieapp/metadata/internal/handler/grpc"
	httphandler "github.com/abhishek622/movieapp/metadata/internal/handler/http"
	"github.com/abhishek622/movieapp/metadata/internal/repository/memory"
	"github.com/abhishek622/movieapp/pkg/discovery"
	"github.com/abhishek622/movieapp/pkg/discovery/consul"
	"github.com/abhishek622/movieapp/pkg/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
)

const serviceName = "metadata"

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// simulateCPULoad := flag.Bool("simulatecpuload", false, "simulate CPU load for profiling")
	// flag.Parse()
	// if *simulateCPULoad {
	// 	go heavyOperation()
	// }

	// go func() {
	// 	if err := http.ListenAndServe("localhost:6060", nil); err != nil {
	// 		logger.Fatal("Failed to start profiler handler", zap.Error(err))
	// 	}
	// }()

	f, err := os.Open("configs/default.yaml")
	if err != nil {
		logger.Fatal("Failed to open configuration", zap.Error(err))
	}

	var cfg config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		logger.Fatal("Failed to parse configuration", zap.Error(err))
	}

	port := cfg.API.Port
	logger.Info("Starting the metadata service", zap.Int("port", port))

	registry, err := consul.NewRegistry(cfg.ServiceDiscovery.Consul.Address)
	if err != nil {
		logger.Fatal("Failed to init metadata service registry", zap.Error(err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Jaeger Tracing ---
	tracer, err := tracing.NewTracerWithLogger(
		serviceName,     // "metadata"
		cfg.Jaeger.Host, // Jaeger agent host
		cfg.Jaeger.Port, // Jaeger agent port
		logger,          // Custom logger
	)
	if err != nil {
		logger.Fatal("Failed to initialize Jaeger tracer", zap.Error(err))
	}
	defer func() {
		// Shutdown Jaeger tracer
		logger.Info("Jaeger tracer shutdown completed")
	}()
	// Set global tracer for the application
	opentracing.SetGlobalTracer(tracer)
	logger.Info("Jaeger tracer initialized successfully", zap.String("service", serviceName))

	// metrics reporting
	reporter := prometheus.NewReporter(prometheus.Options{})
	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Tags:           map[string]string{"service": serviceName},
		CachedReporter: reporter,
		Separator:      prometheus.DefaultSeparator,
		SanitizeOptions: &tally.SanitizeOptions{
			NameCharacters: tally.ValidCharacters{
				Ranges:     tally.AlphanumericRange,
				Characters: []rune{'_', ':'},
			},
			KeyCharacters: tally.ValidCharacters{
				Ranges:     tally.AlphanumericRange,
				Characters: []rune{'_'},
			},
			ValueCharacters: tally.ValidCharacters{
				Ranges:     tally.AlphanumericRange,
				Characters: []rune{'_', ':', '.', '-'},
			},
			ReplacementCharacter: '_',
		},
	}, 10*time.Second)
	defer closer.Close()
	http.Handle("/metrics", reporter.HTTPHandler())
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Prometheus.MetricsPort), nil); err != nil {
			logger.Fatal("Failed to start the metrics handler", zap.Error(err))
		}
	}()

	counter := scope.Tagged(map[string]string{
		"service": serviceName,
	}).Counter("service_started")
	counter.Inc(1)

	// --- Service registration / health heartbeat ---
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("localhost:%d", port)); err != nil {
		logger.Fatal("Failed to register service", zap.Error(err))
	}
	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				logger.Error("Failed to report healthy state", zap.Error(err))
			}
			time.Sleep(1 * time.Second)
		}
	}()
	defer registry.Deregister(ctx, instanceID, serviceName)

	// --- gRPC server (mTLS) ---
	repo := memory.New()
	ctrl := metadata.New(repo)
	h := grpchandler.New(ctrl, scope)
	httpHandler := httphandler.New(ctrl)
	serverCert, err := tls.LoadX509KeyPair("configs/metadata-cert.pem", "configs/metadata-key.pem")
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
		httpMux.HandleFunc("/metadata", httpHandler.GetMetadata)
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
		logger.Fatal("Failed to listen", zap.Error(err))
	}
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.Creds(credentials.NewTLS(tlsConfig)),
	)
	reflection.Register(srv)
	gen.RegisterMetadataServiceServer(srv, h)

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

// func heavyOperation() {
// 	for {
// 		token := make([]byte, 1024)
// 		rand.Read(token)
// 		md5.New().Write(token)
// 	}
// }
