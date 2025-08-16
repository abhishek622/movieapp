package tracing

import (
	"fmt"
	"os"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
)

// NewTracer creates a new Jaeger tracer
func NewTracer(serviceName, jaegerHost, jaegerPort string) (opentracing.Tracer, error) {
	// Set environment variables for Jaeger
	os.Setenv("JAEGER_SERVICE_NAME", serviceName)
	os.Setenv("JAEGER_AGENT_HOST", jaegerHost)
	os.Setenv("JAEGER_AGENT_PORT", jaegerPort)
	os.Setenv("JAEGER_SAMPLER_TYPE", "const")
	os.Setenv("JAEGER_SAMPLER_PARAM", "1")

	// Create Jaeger configuration
	cfg := &config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: fmt.Sprintf("%s:%s", jaegerHost, jaegerPort),
		},
	}

	// Create logger
	logger := log.StdLogger

	// Create metrics factory
	metricsFactory := metrics.NullFactory

	// Create tracer
	tracer, closer, err := cfg.NewTracer(
		config.Logger(logger),
		config.Metrics(metricsFactory),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger tracer: %w", err)
	}

	// Store closer for cleanup (you might want to return this as well)
	_ = closer

	return tracer, nil
}

// NewTracerWithLogger creates a new Jaeger tracer with custom logger
func NewTracerWithLogger(serviceName, jaegerHost, jaegerPort string, logger *zap.Logger) (opentracing.Tracer, error) {
	// Set environment variables for Jaeger
	os.Setenv("JAEGER_SERVICE_NAME", serviceName)
	os.Setenv("JAEGER_AGENT_HOST", jaegerHost)
	os.Setenv("JAEGER_AGENT_PORT", jaegerPort)
	os.Setenv("JAEGER_SAMPLER_TYPE", "const")
	os.Setenv("JAEGER_SAMPLER_PARAM", "1")

	// Create Jaeger configuration
	cfg := &config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: fmt.Sprintf("%s:%s", jaegerHost, jaegerPort),
		},
	}

	// Create custom logger adapter
	jaegerLogger := &jaegerLoggerAdapter{logger: logger}

	// Create metrics factory
	metricsFactory := metrics.NullFactory

	// Create tracer
	tracer, closer, err := cfg.NewTracer(
		config.Logger(jaegerLogger),
		config.Metrics(metricsFactory),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger tracer: %w", err)
	}

	// Store closer for cleanup
	_ = closer

	return tracer, nil
}

// jaegerLoggerAdapter adapts zap logger to Jaeger logger interface
type jaegerLoggerAdapter struct {
	logger *zap.Logger
}

func (l *jaegerLoggerAdapter) Error(msg string) {
	l.logger.Error(msg)
}

func (l *jaegerLoggerAdapter) Infof(msg string, args ...interface{}) {
	l.logger.Sugar().Infof(msg, args...)
}
