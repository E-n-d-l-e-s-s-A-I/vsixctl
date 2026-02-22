package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/registry/marketplace"
)

func main() {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxConnsPerHost:     10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
	registry := marketplace.NewRegistry("https://marketplace.visualstudio.com/_apis/public/gallery", client)
	ctx, _ := context.WithCancel(context.Background())
	res, _ := registry.Search(ctx, "go")
	fmt.Println(res)
}
