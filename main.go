package main

import (
	"log"
	"os"
)

func main() {
	cfg := &AdapterConfig{
		PrivateKey: os.Getenv("IO_PRIVATE_KEY"),
		Endpoint:   os.Getenv("IO_ENDPOINT"),
	}
	if os.Getenv("IO_SECURE_ENDPOINT") == "true" {
		cfg.SecureEndpoint = true
	}

	port := os.Getenv("IO_ADAPTER_PORT")
	if port == "" {
		port = "80"
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		log.Fatalln("Failed to new adapter: ", err)
	}

	r := NewServerRouter(adapter.Handle)
	if err := r.Run(":" + port); err != nil {
		log.Println("HTTP server shutdown: ", err)
	}
}
