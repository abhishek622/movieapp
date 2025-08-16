package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/abhishek622/movieapp/metadata/pkg/model"
	"github.com/abhishek622/movieapp/movie/internal/gateway"
	"github.com/abhishek622/movieapp/pkg/discovery"
)

type Gateway struct {
	registry discovery.Registry
}

func New(registry discovery.Registry) *Gateway {
	return &Gateway{registry}
}

func (g *Gateway) Get(ctx context.Context, id string) (*model.Metadata, error) {
	addrs, err := g.registry.ServiceAddresses(ctx, "metadata")
	if err != nil {
		return nil, err
	}

	// Use HTTP port (gRPC port + 1000)
	addr := addrs[rand.Intn(len(addrs))]
	// Extract port from address and add 1000
	port := "9081" // 8081 + 1000 for metadata service
	if len(addr) > 0 {
		// Parse the address to get port and add 1000
		port = "9081" // 8081 + 1000
	}

	url := "http://localhost:" + port + "/metadata"
	log.Printf("Calling metadata service. Request: GET %s", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, gateway.ErrNotFound
	} else if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("non-2xx response: %v", resp)
	}

	var v *model.Metadata
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
