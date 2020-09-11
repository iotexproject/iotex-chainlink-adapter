package main

import (
	"log"
	"os"
)

func main() {
	cfg := &AdopterConfig{
		PrivateKey: os.Getenv("IO_PRIVATE_KEY"),
		Endpoint:   os.Getenv("IO_ENDPOINT"),
	}
	if os.Getenv("IO_SECURE_ENDPOINT") == "true" {
		cfg.SecureEndpoint = true
	}

	port := os.Getenv("IO_ADOPTER_PORT")
	if port == "" {
		port = "80"
	}

	adopter, err := NewAdopter(cfg)
	if err != nil {
		log.Fatalln("Failed to new adopter: ", err)
	}

	r := NewServerRouter(adopter.Handle)
	if err := r.Run(":" + port); err != nil {
		log.Println("HTTP server shutdown: ", err)
	}
}
