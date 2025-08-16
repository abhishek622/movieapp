# Jaeger Tracing Setup for Movie App

This document explains how to set up and use Jaeger v2.9 for distributed tracing in the Movie App.

## Overview

The application has been updated to use Jaeger v2.9 instead of OTLP tracing. Jaeger provides:

- Distributed tracing across microservices
- Web-based UI for viewing traces
- Support for both UDP and gRPC collection
- Better performance and simpler setup

## Prerequisites

- Docker and Docker Compose
- Go 1.24.2 or later

## Quick Start

### 1. Start Jaeger

```bash
docker-compose -f docker-compose.jaeger.yml up -d
```

This will start Jaeger on the following ports:

- **16686**: Jaeger UI (http://localhost:16686)
- **6831**: Jaeger Agent UDP (used by services)
- **14268**: HTTP Collector
- **14250**: gRPC Collector

### 2. Start Your Services

Start the services in separate terminals:

```bash
# Terminal 1: Metadata Service
cd metadata/cmd
go run main.go

# Terminal 2: Rating Service
cd rating/cmd
go run main.go

# Terminal 3: Movie Service
cd movie/cmd
go run main.go
```

### 3. View Traces

Open your browser and navigate to: http://localhost:16686

You should see:

- Service names: `metadata`, `rating`, `movie`
- Traces showing the flow between services
- Performance metrics and timing information

## Configuration

### Service Configuration

Each service now uses Jaeger configuration instead of OTLP:

```yaml
# configs/default.yaml
jaeger:
  host: localhost
  port: 6831
```

### Environment Variables

The following environment variables are automatically set:

- `JAEGER_SERVICE_NAME`: Service name (e.g., "movie", "metadata", "rating")
- `JAEGER_AGENT_HOST`: Jaeger agent host (default: localhost)
- `JAEGER_AGENT_PORT`: Jaeger agent port (default: 6831)
- `JAEGER_SAMPLER_TYPE`: Sampling type (default: "const")
- `JAEGER_SAMPLER_PARAM`: Sampling parameter (default: 1)

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Movie    │    │  Metadata   │    │   Rating    │
│  Service   │    │  Service    │    │  Service    │
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                    ┌─────────────┐
                    │   Jaeger    │
                    │   Agent     │
                    │  (UDP:6831) │
                    └─────────────┘
                           │
                    ┌─────────────┐
                    │   Jaeger    │
                    │ Collector   │
                    └─────────────┘
                           │
                    ┌─────────────┐
                    │   Jaeger    │
                    │   Storage   │
                    └─────────────┘
                           │
                    ┌─────────────┐
                    │   Jaeger    │
                    │     UI      │
                    │ (Port:16686)│
                    └─────────────┘
```

## Benefits of Jaeger v2.9

1. **Simplified Setup**: No need for TLS certificates or complex OTLP configuration
2. **Better Performance**: UDP-based agent communication is faster
3. **Rich UI**: Built-in web interface for trace analysis
4. **Multiple Protocols**: Supports both UDP and gRPC collection
5. **Production Ready**: Used by many large-scale applications

## Troubleshooting

### Service Won't Start

Check that Jaeger is running:

```bash
docker ps | grep jaeger
```

### No Traces Appearing

1. Verify Jaeger agent is accessible on port 6831
2. Check service logs for tracing errors
3. Ensure services are configured with correct Jaeger host/port

### Performance Issues

1. Adjust sampling rate in configuration
2. Monitor Jaeger resource usage
3. Consider using gRPC collector for high-throughput scenarios

## Migration from OTLP

The following changes were made to migrate from OTLP to Jaeger:

1. **Dependencies**: Replaced OTLP packages with Jaeger packages
2. **Configuration**: Updated YAML configs to use Jaeger settings
3. **Tracing Package**: Rewrote `pkg/tracing/tracing.go` for Jaeger
4. **Service Updates**: Modified all service main.go files
5. **Removed TLS**: No more certificate requirements for tracing

## Next Steps

1. **Add Custom Spans**: Implement custom tracing in business logic
2. **Metrics Integration**: Add Prometheus metrics alongside traces
3. **Alerting**: Set up alerts for trace failures
4. **Storage**: Configure persistent storage for traces
