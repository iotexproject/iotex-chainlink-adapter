package main

import (
	"crypto/tls"
	"sync"

	"github.com/iotexproject/iotex-antenna-go/v2/account"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

type AdopterConfig struct {
	PrivateKey     string
	Endpoint       string
	SecureEndpoint bool
}

type Adopter struct {
	mu       *sync.RWMutex
	grpcConn *grpc.ClientConn
	client   iotexapi.APIServiceClient
	cfg      *AdopterConfig
	account  account.Account
}

func NewAdopter(cfg *AdopterConfig) (*Adopter, error) {
	acc, err := account.HexStringToAccount(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	return &Adopter{
		mu:      &sync.RWMutex{},
		cfg:     cfg,
		account: acc,
	}, nil
}

func (a *Adopter) connect() (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Check if the existing connection is good.
	if a.grpcConn != nil && a.grpcConn.GetState() != connectivity.Shutdown {
		return
	}
	opts := []grpc.DialOption{}
	if a.cfg.SecureEndpoint {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	a.grpcConn, err = grpc.Dial(a.cfg.Endpoint, opts...)
	a.client = iotexapi.NewAPIServiceClient(a.grpcConn)
	return err
}

func (a *Adopter) Handle(req Request) (string, error) {
	if err := a.connect(); err != nil {
		return "", err
	}
	// TODO construct and make contract call here
	return "", nil
}
