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
grpcurl -cacert temp-certs/ca-cert.pem -cert temp-certs/movie-cert.pem -key temp-certs/movie-key.pem -d '{"movie_id":"1"}' localhost:8083 MovieService.GetMovieDetails

```

### To install prometheus

```bash
docker run -p 9090:9090 -v configs:/etc/prometheus prom/prometheus
```
