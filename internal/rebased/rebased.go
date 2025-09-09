package rebased

import (
	"context"
	"errors"
	"strconv"

	suiclient "github.com/coming-chat/go-sui/v2/client"
)

// RPC method prefix helpers

type IotaMethod string

func (m IotaMethod) String() string { return string(m) }

var _ suiclient.Method = IotaMethod("")

// Builds "iota_<name>" (full-node RPC).
func iotaMethod(name string) suiclient.Method { return IotaMethod("iota_" + name) }

// Builds "sui_<name>" (Sui negative check).
func suiMethod(name string) suiclient.Method { return IotaMethod("sui_" + name) }

// Client
type Client struct{ rpc *suiclient.Client }

// Dial opens an HTTPS JSON-RPC connection to an IOTA Rebased node.
// Example: Dial("https://api.testnet.iota.cafe:443")
func Dial(rpcURL string) (*Client, error) {
	cli, err := suiclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	return &Client{rpc: cli}, nil
}

// Generic passthrough for RPC calls.
func (c *Client) Call(
	ctx context.Context,
	out interface{},
	method suiclient.Method,
	params ...any,
) error {
	return c.rpc.CallContext(ctx, out, method, params...)
}

// Check node liveness by retourning the latest checkpoint sequence number
func (c *Client) Ping(ctx context.Context) (uint64, error) {
	var raw string
	if err := c.Call(ctx, &raw,
		iotaMethod("getLatestCheckpointSequenceNumber")); err != nil {
		return 0, err
	}
	return strconv.ParseUint(raw, 10, 64)
}

// Returns the chain ID string, e.g. "iota:rebase-testnet".
func (c *Client) ChainIdentifier(ctx context.Context) (string, error) {
	var id string
	if err := c.Call(ctx, &id, iotaMethod("getChainIdentifier")); err != nil {
		return "", err
	}
	return id, nil
}

// Returns true if the endpoint is an IOTA-Rebased node
func (c *Client) IsIota(ctx context.Context) (bool, error) {
	// Positive check
	id, err := c.ChainIdentifier(ctx)
	if err != nil || id == "" {
		return false, err
	}

	// Negative check – Sui namespace must be disabled
	var dummy any
	err = c.Call(ctx, &dummy, suiMethod("getChainIdentifier"))
	if err == nil {
		return false, errors.New("server also accepted Sui namespace")
	}
	return true, nil
}
