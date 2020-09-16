package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"math/big"
	"strings"
	"sync"

	"github.com/iotexproject/iotex-antenna-go/v2/account"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
)

type AdapterConfig struct {
	PrivateKey     string
	Endpoint       string
	SecureEndpoint bool
}

type Adapter struct {
	mu       *sync.RWMutex
	grpcConn *grpc.ClientConn
	api      iotexapi.APIServiceClient
	cfg      *AdapterConfig
	account  account.Account
}

func NewAdapter(cfg *AdapterConfig) (*Adapter, error) {
	acc, err := account.HexStringToAccount(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		mu:      &sync.RWMutex{},
		cfg:     cfg,
		account: acc,
	}, nil
}

func (a *Adapter) connect() (err error) {
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
	a.api = iotexapi.NewAPIServiceClient(a.grpcConn)
	return err
}

func (a *Adapter) Handle(ctx context.Context, req Request) (string, error) {
	if err := a.connect(); err != nil {
		return "", err
	}

	data, err := composeExecData(req.Data)
	if err != nil {
		return "", err
	}

	exec := &iotextypes.Execution{
		Contract: req.Data.ContractAddress,
		Data:     data,
		Amount:   "0",
	}
	return a.callContract(ctx, exec)
}

func (a *Adapter) callContract(ctx context.Context, exec *iotextypes.Execution) (string, error) {
	// get nonce
	nonceResp, err := a.api.GetAccount(ctx, &iotexapi.GetAccountRequest{Address: a.account.Address().String()})
	if err != nil {
		return "", err
	}

	core := &iotextypes.ActionCore{
		Nonce: nonceResp.GetAccountMeta().GetPendingNonce(),
		Action: &iotextypes.ActionCore_Execution{
			Execution: exec,
		},
		GasLimit: 15000000,
	}

	// get gaslimit
	sealed, err := sign(a.account, core)
	if err != nil {
		return "", err
	}
	gasLimitResp, err := a.api.EstimateGasForAction(ctx, &iotexapi.EstimateGasForActionRequest{Action: sealed})
	if err != nil {
		return "", err
	}
	core.GasLimit = gasLimitResp.GetGas()

	// get gasprice
	gasPriceResp, err := a.api.SuggestGasPrice(ctx, &iotexapi.SuggestGasPriceRequest{})
	if err != nil {
		return "", err
	}
	core.GasPrice = big.NewInt(0).SetUint64(gasPriceResp.GetGasPrice()).String()

	// call execution
	sealed, err = sign(a.account, core)
	if err != nil {
		return "", err
	}
	resp, err := a.api.SendAction(ctx, &iotexapi.SendActionRequest{Action: sealed})
	if err != nil {
		return "", err
	}
	return resp.GetActionHash(), nil
}

func composeExecData(data Data) ([]byte, error) {

	d, err := hexToBytes(data.Function)
	if err != nil {
		return d, err
	}
	prefix, err := hexToBytes(data.DataPrefix)
	if err != nil {
		return d, err
	}
	d = append(d, prefix...)
	res, err := hexToBytes(data.Result)
	if err != nil {
		return d, err
	}
	return append(d, res...), nil
}

func hexToBytes(x string) ([]byte, error) {
	if strings.HasPrefix(x, "0x") {
		x = x[2:]
	}
	return hex.DecodeString(x)
}

func sign(a account.Account, act *iotextypes.ActionCore) (*iotextypes.Action, error) {
	msg, err := proto.Marshal(act)
	if err != nil {
		return nil, err
	}
	sig, err := a.Sign(msg)
	if err != nil {
		return nil, err
	}
	return &iotextypes.Action{
		Core:         act,
		SenderPubKey: a.PublicKey().Bytes(),
		Signature:    sig,
	}, nil
}