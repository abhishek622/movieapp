# movieapp

Microservices with Go 2nd edition

### command to run hashicorp consul

```bash
docker run -d -p 8500:8500 -p 8600:8600/udp --name=dev-consul hashicorp/consul agent -server -ui -node=server-1 -bootstrap-expect=1 -client="0.0.0.0"
```

### To start docker service

```bash
docker start dev-consul
```

### To stop docker service

```bash
docker stop dev-consul
```

### To make request in movie service

```bash
grpcurl -cacert configs/ca-cert.pem -cert configs/movie-cert.pem -key configs/movie-key.pem -d '{"movie_id":"1"}' localhost:8083 MovieService.GetMovieDetails

```

### To run prometheus

```bash
docker run -d --name prometheus -p 9090:9090 -v "$(pwd)/configs/prometheus.yaml:/etc/prometheus/prometheus.yml" -v "$(pwd)/configs/alerts.rules:/etc/prometheus/alerts.rules" prom/prometheus:latest --config.file=/etc/prometheus/prometheus.yml --web.enable-lifecycle
```

### To run alert manager

```bash
docker run -d --name alertmanager -p 9093:9093 -v "$(pwd)/configs/alertmanager.yml:/etc/alertmanager/alertmanager.yml" prom/alertmanager:latest --config.file=/etc/alertmanager/alertmanager.yml --web.external-url=http://localhost:9093
```

### To run jaeger

```bash
 docker run --rm --name jaeger -e COLLECTOR_OTLP_ENABLED=true -p 16686:16686 -p 4317:4317 -p 4318:4318 -p 5778:5778 -p 9411:9411 cr.jaegertracing.io/jaegertracing/jaeger:2.9.0
```

### To run grafana

```bash
docker run -d -p 3000:3000 grafana/grafana
```

