package rebased

import (
	"context"
	"fmt"

	suiclient "github.com/coming-chat/go-sui/v2/client"
	suitypes "github.com/coming-chat/go-sui/v2/types"
)

// Wrapper wraps an already-dialed Sui/IOTA-Rebased JSON-RPC client.
type Wrapper struct {
	rpc *suiclient.Client
}

func New(rpc *suiclient.Client) *Wrapper { return &Wrapper{rpc: rpc} }

// These implement client.Method
type iotaMethod string

func (m iotaMethod) String() string { return "iota_" + string(m) }

type suiMethod string

func (m suiMethod) String() string { return "sui_" + string(m) }

// Try iota_* first (Rebased), then fallback to sui_* (vanilla Sui).
func (w *Wrapper) call(ctx context.Context, out any, name string, params ...any) error {
	if err := w.rpc.CallContext(ctx, out, iotaMethod(name), params...); err == nil {
		return nil
	}
	return w.rpc.CallContext(ctx, out, suiMethod(name), params...)
}

func (w *Wrapper) Ping(ctx context.Context) (uint64, error) {
	var seqStr string
	if err := w.call(ctx, &seqStr, "getLatestCheckpointSequenceNumber"); err != nil {
		return 0, err
	}
	var seq uint64
	_, err := fmt.Sscan(seqStr, &seq)
	return seq, err
}

func (w *Wrapper) ChainIdentifier(ctx context.Context) (string, error) {
	var id string
	if err := w.call(ctx, &id, "getChainIdentifier"); err != nil {
		return "", err
	}
	return id, nil
}

// Use SDK types for options/response; Works on iota_* or sui_*.
func (w *Wrapper) GetObject(
	ctx context.Context,
	objectID string,
	opts suitypes.SuiObjectDataOptions,
) (*suitypes.SuiObjectResponse, error) {
	var out suitypes.SuiObjectResponse
	if err := w.call(ctx, &out, "getObject", objectID, &opts); err != nil {
		return nil, err
	}
	return &out, nil
}

// Build an unsigned transaction for a Move function call via RPC.
func (w *Wrapper) MoveCallUnsigned(
	ctx context.Context,
	signerAddress string, // 0x...
	packageID string, // 0x...
	module string,
	function string,
	typeArgs []string,
	args []any,
	gasObject *string, // optional gas coin object id
	gasBudget uint64, // plain uint64 keeps JSON shape correct
) (*suitypes.TransactionBytes, error) {
	var tx suitypes.TransactionBytes
	if err := w.call(ctx, &tx, "moveCall",
		signerAddress, packageID, module, function,
		typeArgs, args, gasObject, fmt.Sprintf("%d", gasBudget),
	); err != nil {
		return nil, err
	}
	return &tx, nil
}

// Submit the signed transaction bytes and get the execution response.
func (w *Wrapper) ExecuteTransactionBlock(
	ctx context.Context,
	txBytesBase64 string, // base64 tx bytes (from TransactionBytes.TxBytes)
	signatures []any, // base64 signer(s), e.g. []string{"BASE64SIG"}
	opts *suitypes.SuiTransactionBlockResponseOptions, // which parts to return
	reqType suitypes.ExecuteTransactionRequestType, // e.g. types.WaitForLocalExecution
) (*suitypes.SuiTransactionBlockResponse, error) {
	var rsp suitypes.SuiTransactionBlockResponse
	if err := w.call(ctx, &rsp, "executeTransactionBlock", txBytesBase64, signatures, opts, reqType); err != nil {
		return nil, err
	}
	return &rsp, nil
}

// Dial opens a JSON-RPC connection to an IOTA Rebased (Sui-compatible) node
// and returns a Wrapper around the RPC client.
// Example: Dial("https://api.testnet.iota.cafe:443")
func Dial(rpcURL string) (*Wrapper, error) {
	rpc, err := suiclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	return New(rpc), nil
}
