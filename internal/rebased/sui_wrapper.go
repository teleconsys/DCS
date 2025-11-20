package rebased

import (
	"context"
	"fmt"
	"os"
	"strings"

	suiclient "github.com/coming-chat/go-sui/v2/client"
	suitypes "github.com/coming-chat/go-sui/v2/types"
)

// Wrapper wraps an already-dialed Sui/IOTA-Rebased JSON-RPC client.
type Wrapper struct {
	rpc *suiclient.Client
}

func New(rpc *suiclient.Client) *Wrapper { return &Wrapper{rpc: rpc} }

// ---- method helpers ----------------------------------------------------------

type rawMethod string

func (m rawMethod) String() string { return string(m) }

// Call the RPC method
func (w *Wrapper) call(ctx context.Context, out any, name string, params ...any) error {
	return w.rpc.CallContext(ctx, out, rawMethod(name), params...)
}

func methodNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "method not found")
}

// ---- basic reads -------------------------------------------------------------

func (w *Wrapper) Ping(ctx context.Context) (uint64, error) {
	var seqStr string
	if err := w.call(ctx, &seqStr, "iota_getLatestCheckpointSequenceNumber"); err != nil {
		return 0, err
	}
	var seq uint64
	_, err := fmt.Sscan(seqStr, &seq)
	return seq, err
}

func (w *Wrapper) ChainIdentifier(ctx context.Context) (string, error) {
	var id string
	if err := w.call(ctx, &id, "iota_getChainIdentifier"); err != nil {
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
	if err := w.call(ctx, &out, "iota_getObject", objectID, &opts); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- build unsigned move call -----------------------------------------------

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
) (*suitypes.TransactionBytes, error) {
	var txb suitypes.TransactionBytes

	if typeArgs == nil {
		typeArgs = []string{}
	}
	gb := fmt.Sprintf("%d", gasBudget) // BigInt as string

	// moveCall
	if err := w.call(ctx, &txb, "moveCall",
		signerAddress, packageID, module, function,
		typeArgs, args, gasObject, gb,
	); err == nil {
		return &txb, nil
	} else if !methodNotFound(err) {
		return nil, err
	}

	return nil, fmt.Errorf("MoveCallUnsigned: method moveCall not found on RPC")
}

// Note: gasBudget must be sent as a string for JSON-RPC big-int.
func (w *Wrapper) UnsafeMoveCallUnsigned(
	ctx context.Context,
	signerAddress string,
	packageID string,
	module string,
	function string,
	typeArgs []string,
	args []any,
	gasObject *string,
	gasBudget uint64,
) (*suitypes.TransactionBytes, error) {
	var txb suitypes.TransactionBytes

	if typeArgs == nil {
		typeArgs = []string{}
	}
	gb := fmt.Sprintf("%d", gasBudget) // BigInt as string

	// unsafe_moveCall
	if err := w.call(ctx, &txb, "unsafe_moveCall",
		signerAddress, packageID, module, function,
		typeArgs, args, gasObject, gb,
	); err == nil {
		return &txb, nil
	} else if !methodNotFound(err) {
		return nil, err
	}

	return nil, fmt.Errorf("UnsafeMoveCallUnsigned: method unsafe_moveCall not found on RPC")
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
	if err := w.call(ctx, &rsp, "iota_executeTransactionBlock", txBytesBase64, signatures, opts, reqType); err != nil {
		return nil, err
	}
	return &rsp, nil
}

// ---- pay iota ----------------------------------------------

// Build an unsigned payIota transaction
func (w *Wrapper) PayIotaUnsigned(
	ctx context.Context,
	signerAddress string,
	inputCoins []string,
	recipients []string,
	amounts []string,
	gasBudget uint64,
) (*suitypes.TransactionBytes, error) {
	fmt.Println("Building transaction pay iota...")
	var txb suitypes.TransactionBytes

	gasBudgetStr := fmt.Sprintf("%d", gasBudget)

	err := w.call(ctx, &txb, "unsafe_payIota",
		signerAddress, // signer
		inputCoins,    // input_coins
		recipients,    // recipients (pay to yourself)
		amounts,       // amounts
		gasBudgetStr,  // gas_budget
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building the transaction: %v\n", err)
		return nil, err
	}

	return &txb, nil
}

// Dial opens a JSON-RPC connection to an IOTA Rebased (Sui-compatible) node.
func Dial(rpcURL string) (*Wrapper, error) {
	rpc, err := suiclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	return New(rpc), nil
}
