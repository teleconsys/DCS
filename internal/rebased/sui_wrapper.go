package rebased

import (
	"context"
	"fmt"
	"strings"

	suiclient "github.com/coming-chat/go-sui/v2/client"
	"github.com/coming-chat/go-sui/v2/lib"
	sui_types "github.com/coming-chat/go-sui/v2/sui_types"
	"github.com/coming-chat/go-sui/v2/types"

	"github.com/teleconsys/DCS/internal/iota_rpc"
)

// Wrapper wraps an already-dialed Sui/IOTA-Rebased JSON-RPC client.
type Wrapper struct {
	rpc      *suiclient.Client
	endpoint string
}

func New(rpc *suiclient.Client) *Wrapper { return &Wrapper{rpc: rpc} }

// ---- method helpers ----------------------------------------------------------

type iotaMethod string

// func (m iotaMethod) String() string { return "iota_" + string(m) }

func (m iotaMethod) String() string { return string(m) }

type suiMethod string

func (m suiMethod) String() string { return "sui_" + string(m) }

// raw (no prefix)
type rawMethod string

func (m rawMethod) String() string { return string(m) }

// Try iota_* first, then sui_*, then raw (unprefixed).
func (w *Wrapper) call(ctx context.Context, out any, name string, params ...any) error {
	fmt.Println("name: ", name)
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
	opts types.SuiObjectDataOptions,
) (*types.SuiObjectResponse, error) {
	var out types.SuiObjectResponse
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
	opts *types.SuiTransactionBlockResponseOptions, // which parts to return
	reqType types.ExecuteTransactionRequestType, // e.g., "WaitForLocalExecution"
) (*types.SuiTransactionBlockResponse, error) {
	var rsp types.SuiTransactionBlockResponse
	if err := w.call(ctx, &rsp, "executeTransactionBlock", txBytesBase64, signatures, opts, reqType); err != nil {
		return nil, err
	}
	return &rsp, nil
}

// ---- split coin ----------------------------------------------

func (w *Wrapper) SplitCoinUnsigned(
	ctx context.Context,
	signerAddress string,
	coinID string,
	amounts []uint64,
	gasObject *string,
	gasBudget uint64,
) (*types.TransactionBytes, error) {

	addrPtr, err := sui_types.NewAddressFromHex(signerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid signerAddress: %w", err)
	}
	signerAddressSui := *addrPtr
	// Convert inputs to SDK-required types
	// coinID: string -> sui_types.ObjectID (value)
	coinObjIDPtr, err := sui_types.NewObjectIdFromHex(coinID)
	if err != nil {
		return nil, fmt.Errorf("invalid coinID: %w", err)
	}
	coinObjID := *coinObjIDPtr

	// amounts: []uint64 -> []types.SafeSuiBigInt[uint64]
	splitAmounts := make([]types.SafeSuiBigInt[uint64], 0, len(amounts))
	for _, a := range amounts {
		splitAmounts = append(splitAmounts, types.NewSafeSuiBigInt(a))
	}

	// gasObject: *string -> *sui_types.ObjectID
	var gasObjID *sui_types.ObjectID
	if gasObject != nil {
		gid, err := sui_types.NewObjectIdFromHex(*gasObject)
		if err != nil {
			return nil, fmt.Errorf("invalid gasObject: %w", err)
		}
		gasObjID = gid
	}

	// gasBudget: uint64 -> types.SafeSuiBigInt[uint64]
	budget := types.NewSafeSuiBigInt(gasBudget)

	txb, err := w.rpc.SplitCoin(ctx, signerAddressSui, coinObjID, splitAmounts, gasObjID, budget)
	if err != nil {
		return nil, err
	}

	fmt.Println("txb: ", txb)

	return txb, nil
}

// SplitCoinUnsignedRPC is an alternative implementation that uses the UnsafeSplitCoin function
// from iota_rpc package instead of the SDK's SplitCoin method.
// This may be useful when the SDK method doesn't work or you need direct RPC control.
func (w *Wrapper) SplitCoinUnsignedRPC(
	ctx context.Context,
	signerAddress string,
	coinID string,
	amounts []uint64,
	gasObject *string,
	gasBudget uint64,
) (*types.TransactionBytes, error) {
	if w.endpoint == "" {
		return nil, fmt.Errorf("endpoint not set in wrapper (use Dial() to create wrapper with endpoint)")
	}

	// Convert amounts: []uint64 -> []string (BigInt format for JSON-RPC)
	splitAmounts := make([]string, 0, len(amounts))
	for _, a := range amounts {
		splitAmounts = append(splitAmounts, fmt.Sprintf("%d", a))
	}

	// Convert gasBudget: uint64 -> string (BigInt format)
	gb := fmt.Sprintf("%d", gasBudget)

	// Prepare parameters for UnsafeSplitCoin
	params := iota_rpc.SplitCoinParams{
		Signer:       signerAddress,
		CoinObjectID: coinID,
		SplitAmounts: splitAmounts,
		Gas:          gasObject, // Can be nil
		GasBudget:    gb,
	}

	// Call UnsafeSplitCoin to get txBytes string
	txBytesStr, err := iota_rpc.UnsafeSplitCoin(ctx, w.endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("UnsafeSplitCoin call failed: %w", err)
	}

	// Construct TransactionBytes with the returned txBytes
	// TxBytes field expects lib.Base64Data which is a string type alias
	var txb types.TransactionBytes
	// Use JSON unmarshaling to properly set the Base64Data field
	txB64 := lib.Base64Data(txBytesStr)
	txb.TxBytes = txB64

	return &txb, nil
}

// ---- pay sui ----------------------------------------------

func (w *Wrapper) PaySuiUnsigned(
	ctx context.Context,
	signerAddress string,
	inputCoins []string,
	recipients []string,
	amounts []uint64,
	gasBudget uint64,
) (*types.TransactionBytes, error) {
	// Convert signerAddress: string -> sui_types.SuiAddress
	addrPtr, err := sui_types.NewAddressFromHex(signerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid signerAddress: %w", err)
	}
	signerAddressSui := *addrPtr

	// Convert inputCoins: []string -> []sui_types.ObjectID
	inputCoinIDs := make([]sui_types.ObjectID, 0, len(inputCoins))
	for _, coinStr := range inputCoins {
		coinObjIDPtr, err := sui_types.NewObjectIdFromHex(coinStr)
		if err != nil {
			return nil, fmt.Errorf("invalid inputCoin %q: %w", coinStr, err)
		}
		inputCoinIDs = append(inputCoinIDs, *coinObjIDPtr)
	}

	// Convert recipients: []string -> []sui_types.SuiAddress
	recipientAddresses := make([]sui_types.SuiAddress, 0, len(recipients))
	for _, recipientStr := range recipients {
		recipientPtr, err := sui_types.NewAddressFromHex(recipientStr)
		if err != nil {
			return nil, fmt.Errorf("invalid recipient %q: %w", recipientStr, err)
		}
		recipientAddresses = append(recipientAddresses, *recipientPtr)
	}

	// Convert amounts: []uint64 -> []types.SafeSuiBigInt[uint64]
	splitAmounts := make([]types.SafeSuiBigInt[uint64], 0, len(amounts))
	for _, a := range amounts {
		splitAmounts = append(splitAmounts, types.NewSafeSuiBigInt(a))
	}

	// gasBudget: uint64 -> types.SafeSuiBigInt[uint64]
	budget := types.NewSafeSuiBigInt(gasBudget)

	fmt.Println("signerAddressSui: ", signerAddressSui)
	fmt.Println("inputCoinIDs: ", inputCoinIDs)
	fmt.Println("recipientAddresses: ", recipientAddresses)
	fmt.Println("splitAmounts: ", splitAmounts)
	fmt.Println("budget: ", budget)

	txb, err := w.rpc.PaySui(ctx, signerAddressSui, inputCoinIDs, recipientAddresses, splitAmounts, budget)
	if err != nil {
		return nil, err
	}

	return txb, nil
}

	
// Dial opens a JSON-RPC connection to an IOTA Rebased (Sui-compatible) node.
func Dial(rpcURL string) (*Wrapper, error) {
	rpc, err := suiclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	w := New(rpc)
	w.endpoint = rpcURL
	return w, nil
}
