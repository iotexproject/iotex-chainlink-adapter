package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/iotexproject/iotex-antenna-go/v2/account"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
)

const _minimumConsumerGasLimit = uint64(600000)

type AdapterConfig struct {
	PrivateKey     string
	Endpoint       string
	SecureEndpoint bool
}

type nonceKeeper struct {
	sync.Mutex

	address string

	nextPending uint64
	lastSync    time.Time
}

func (n *nonceKeeper) NextPending(ctx context.Context, api iotexapi.APIServiceClient) (uint64, error) {
	n.Lock()
	defer n.Unlock()
	if time.Now().After(n.lastSync.Add(15 * time.Second)) {
		nonceResp, err := api.GetAccount(ctx, &iotexapi.GetAccountRequest{Address: n.address})
		if err != nil {
			return 0, err
		}
		n.nextPending = nonceResp.GetAccountMeta().GetPendingNonce()
		n.lastSync = time.Now()
	}
	ret := n.nextPending
	n.nextPending += 1
	return ret, nil
}

type Adapter struct {
	mu          *sync.RWMutex
	grpcConn    *grpc.ClientConn
	api         iotexapi.APIServiceClient
	cfg         *AdapterConfig
	account     account.Account
	nonceKeeper *nonceKeeper
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
		nonceKeeper: &nonceKeeper{
			address: acc.Address().String(),
		},
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

	nonce, err := a.nonceKeeper.NextPending(ctx, a.api)
	if err != nil {
		return "", err
	}

	core := &iotextypes.ActionCore{
		Nonce: nonce,
		Action: &iotextypes.ActionCore_Execution{
			Execution: exec,
		},
		GasLimit: 10000000,
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
	core.GasLimit = gasLimitResp.GetGas() + _minimumConsumerGasLimit

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
