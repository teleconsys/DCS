package rebased

import (
	"context"
	"fmt"
	"strings"

	suiclient "github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/types"
	suitypes "github.com/coming-chat/go-sui/v2/types"
)

// Wrapper wraps an already-dialed Sui/IOTA-Rebased JSON-RPC client.
type Wrapper struct {
	rpc *suiclient.Client
}

func New(rpc *suiclient.Client) *Wrapper { return &Wrapper{rpc: rpc} }

// ---- method helpers ----------------------------------------------------------

type iotaMethod string

func (m iotaMethod) String() string { return "iota_" + string(m) }

type suiMethod string

func (m suiMethod) String() string { return "sui_" + string(m) }

// raw (no prefix)
type rawMethod string

func (m rawMethod) String() string { return string(m) }

// Try iota_* first, then sui_*, then raw (unprefixed).
func (w *Wrapper) call(ctx context.Context, out any, name string, params ...any) error {
	if err := w.rpc.CallContext(ctx, out, iotaMethod(name), params...); err == nil {
		return nil
	}
	if err := w.rpc.CallContext(ctx, out, suiMethod(name), params...); err == nil {
		return nil
	}
	return w.rpc.CallContext(ctx, out, rawMethod(name), params...)
}

func methodNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "method not found")
}

// ---- basic reads -------------------------------------------------------------

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

// Use SDK types for options/response; works with iota_*, sui_* or raw.
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

// ---- build unsigned move call -----------------------------------------------

// Tries moveCall (iota_/sui_/raw), then unsafe_moveCall (iota_/sui_/raw).
// Note: gasBudget must be sent as a string for JSON-RPC big-int.
func (w *Wrapper) MoveCallUnsigned(
	ctx context.Context,
	signerAddress string,
	packageID string,
	module string,
	function string,
	typeArgs []string,
	args []any,
	gasObject *string,
	gasBudget uint64,
) (*types.TransactionBytes, error) {
	var txb types.TransactionBytes

	if typeArgs == nil {
		typeArgs = []string{}
	}
	gb := fmt.Sprintf("%d", gasBudget) // BigInt as string

	// 1) moveCall
	if err := w.call(ctx, &txb, "moveCall",
		signerAddress, packageID, module, function,
		typeArgs, args, gasObject, gb,
	); err == nil {
		return &txb, nil
	} else if !methodNotFound(err) {
		return nil, err
	}

	// 2) unsafe_moveCall
	if err := w.call(ctx, &txb, "unsafe_moveCall",
		signerAddress, packageID, module, function,
		typeArgs, args, gasObject, gb,
	); err == nil {
		return &txb, nil
	} else if !methodNotFound(err) {
		return nil, err
	}

	return nil, fmt.Errorf("MoveCallUnsigned: method not found on RPC (tried moveCall / unsafe_moveCall)")
}

// ---- submit signed tx --------------------------------------------------------

func (w *Wrapper) ExecuteTransactionBlock(
	ctx context.Context,
	txBytesBase64 string, // base64 tx bytes
	signatures []any, // base64 sig(s)
	opts *suitypes.SuiTransactionBlockResponseOptions, // which parts to return
	reqType suitypes.ExecuteTransactionRequestType, // e.g., "WaitForLocalExecution"
) (*suitypes.SuiTransactionBlockResponse, error) {
	var rsp suitypes.SuiTransactionBlockResponse
	if err := w.call(ctx, &rsp, "executeTransactionBlock", txBytesBase64, signatures, opts, reqType); err != nil {
		return nil, err
	}
	return &rsp, nil
}

// Dial opens a JSON-RPC connection to an IOTA Rebased (Sui-compatible) node.
func Dial(rpcURL string) (*Wrapper, error) {
	rpc, err := suiclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	return New(rpc), nil
}
