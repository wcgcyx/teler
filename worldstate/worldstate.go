package worldstate

/*
 * Licensed under LGPL-3.0.
 *
 * You can get a copy of the LGPL-3.0 License at
 *
 * https://www.gnu.org/licenses/lgpl-3.0.en.html
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
	logging "github.com/ipfs/go-log"
	itypes "github.com/wcgcyx/teler/types"
)

// Logger
var log = logging.Logger("worldstate")

type WorldState interface {
	vm.StateDB

	// SetTxContext sets the current transaction hash and index which are
	// used when the EVM emits new state logs. It should be invoked before
	// transaction execution.
	SetTxContext(thash common.Hash, ti int)

	// TxIndex returns the current transaction index set by Prepare.
	TxIndex() int

	// SetBalance sets the balance of the account associated with addr.
	SetBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason)

	// GetLogs returns the logs matching the specified transaction hash, and annotates
	// them with the given blockNumber and blockHash.
	GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash) []*types.Log

	// Finalise finalises the state by removing the destructed objects and clears
	// the journal as well as the refunds. Finalise, however, will not push any updates
	// into the tries just yet. Only IntermediateRoot or Commit will do that.
	Finalise(deleteEmptyObjects bool)

	// IntermediateRoot computes the current root hash of the state trie.
	// It is called in between transactions to get the root hash that
	// goes into transaction receipts.
	IntermediateRoot(deleteEmptyObjects bool) common.Hash
}

type ReadOnlyWorldState interface {
	// Exist reports whether the given account address exists in the state.
	// Notably this also returns true for self-destructed accounts.
	Exist(addr common.Address) bool

	// Empty returns whether the state object is either non-existent
	// or empty according to the EIP161 specification (balance = nonce = code = 0).
	Empty(addr common.Address) bool

	// GetBalance retrieves the balance from the given address or 0 if object not found.
	GetBalance(addr common.Address) *uint256.Int

	// GetNonce retrieves the nonce from the given address or 0 if object not found.
	GetNonce(addr common.Address) uint64

	// GetCodeHash gets the code hash of the given address.
	GetCodeHash(addr common.Address) common.Hash

	// GetCode gets the code of the given address.
	GetCode(addr common.Address) []byte

	// GetCodeSize gets the size of the code of the given address.
	GetCodeSize(addr common.Address) int

	// GetStorageRoot retrieves the storage root from the given address or empty
	// if object not found.
	GetStorageRoot(addr common.Address) common.Hash

	// GetState retrieves a value from the given account's storage trie.
	GetState(addr common.Address, key common.Hash) common.Hash
}

type MutableWorldState interface {
	ReadOnlyWorldState

	// SetLogger sets the logger for account update hooks.
	SetLogger(l *tracing.Hooks)

	// SetTxContext sets the current transaction hash and index which are
	// used when the EVM emits new state logs. It should be invoked before
	// transaction execution.
	SetTxContext(thash common.Hash, ti int)

	// TxIndex returns the current transaction index set by SetTxContext.
	TxIndex() int

	// Prepare handles the preparatory steps for executing a state transition with.
	// This method must be invoked before state transition.
	//
	// Berlin fork:
	// - Add sender to access list (2929)
	// - Add destination to access list (2929)
	// - Add precompiles to access list (2929)
	// - Add the contents of the optional tx access list (2930)
	//
	// Potential EIPs:
	// - Reset access list (Berlin)
	// - Add coinbase to access list (EIP-3651)
	// - Reset transient storage (EIP-1153)
	Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList)

	// CreateAccount explicitly creates a new state object, assuming that the
	// account did not previously exist in the state. If the account already
	// exists, this function will silently overwrite it which might lead to a
	// consensus bug eventually.
	CreateAccount(addr common.Address)

	// CreateContract is used whenever a contract is created. This may be preceded
	// by CreateAccount, but that is not required if it already existed in the
	// state due to funds sent beforehand.
	// This operation sets the 'newContract'-flag, which is required in order to
	// correctly handle EIP-6780 'delete-in-same-transaction' logic.
	CreateContract(common.Address)

	// SubBalance subtracts amount from the account associated with addr.
	SubBalance(addr common.Address, amt *uint256.Int, reason tracing.BalanceChangeReason)

	// AddBalance adds amount to the account associated with addr.
	AddBalance(addr common.Address, amt *uint256.Int, reason tracing.BalanceChangeReason)

	// SetBalance sets the balance of the account associated with addr.
	SetBalance(addr common.Address, amt *uint256.Int, reason tracing.BalanceChangeReason)

	// SetNonce sets the given nonce to the given address.
	SetNonce(addr common.Address, nonce uint64)

	// SetCode sets the code to the given address.
	SetCode(addr common.Address, code []byte)

	// SetState sets the value associated with the specific key.
	SetState(addr common.Address, key common.Hash, val common.Hash)

	// GetCommittedState retrieves the value associated with the specific key
	// without any mutations caused in the current execution.
	GetCommittedState(addr common.Address, key common.Hash) common.Hash

	// GetTransientState gets transient storage for a given account.
	GetTransientState(addr common.Address, key common.Hash) common.Hash

	// SetTransientState sets transient storage for a given account. It
	// adds the change to the journal so that it can be rolled back
	// to its previous value if there is a revert.
	SetTransientState(addr common.Address, key common.Hash, val common.Hash)

	// GetRefund returns the current value of the refund counter.
	GetRefund() uint64

	// AddRefund adds gas to the refund counter.
	AddRefund(gas uint64)

	// SubRefund removes gas from the refund counter.
	// This method will panic if the refund counter goes below zero.
	SubRefund(gas uint64)

	// SelfDestruct marks the given account as selfdestructed.
	// This clears the account balance.
	//
	// The account's state object is still available until the state is committed,
	// getStateObject will return a non-nil account after SelfDestruct.
	SelfDestruct(addr common.Address)

	// SelfDestruct given account according to EIP-6780.
	Selfdestruct6780(addr common.Address)

	// Check if given account was marked as self-destructed.
	HasSelfDestructed(addr common.Address) bool

	// AddressInAccessList returns true if the given address is in the access list.
	AddressInAccessList(addr common.Address) bool

	// AddAddressToAccessList adds the given address to the access list.
	AddAddressToAccessList(addr common.Address)

	// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
	SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool)

	// AddSlotToAccessList adds the given (address, slot)-tuple to the access list.
	AddSlotToAccessList(addr common.Address, slot common.Hash)

	// PointCache returns the point cache used in computations.
	PointCache() *utils.PointCache

	// GetLogs returns the logs matching the specified transaction hash, and annotates
	// them with the given blockNumber and blockHash.
	GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash) []*types.Log

	// Add log adds a log to the log list.
	AddLog(log *types.Log)

	// AddPreimage records a SHA3 preimage seen by the VM.
	AddPreimage(key common.Hash, val []byte)

	// Witness retrieves the current state witness being collected.
	Witness() *stateless.Witness

	// Snapshot returns an identifier for the current revision of the state.
	Snapshot() int

	// RevertToSnapshot reverts all state changes made since the given revision.
	RevertToSnapshot(ti int)

	// Finalise finalises the state by removing the destructed objects and clears
	// the journal as well as the refunds. Finalise, however, will not push any updates
	// into the tries just yet. Only IntermediateRoot or Commit will do that.
	Finalise(deleteEmptyObjects bool)

	// IntermediateRoot computes the current root hash of the state trie.
	// It is called in between transactions to get the root hash that
	// goes into transaction receipts.
	IntermediateRoot(deleteEmptyObjects bool) common.Hash

	// Commit writes the state mutations into the configured data stores.
	//
	// Once the state is committed, tries cached in stateDB (including account
	// trie, storage tries) will no longer be functional. A new state instance
	// must be created with new root and updated database for accessing post-
	// commit states.
	//
	// The associated block number of the state transition is also provided
	// for more chain context.
	Commit(block uint64, rootHash common.Hash, deleteEmptyObjects bool) (common.Hash, error)

	// Copy creates a deep copy of this state.
	Copy() MutableWorldState
}

type LayeredWorldState interface {
	// GetAccountValue gets the account value of given address.
	GetAccountValue(addr common.Address, requireCache bool) itypes.AccountValue

	// GetCodeByHash gets the code by code hash.
	GetCodeByHash(codeHash common.Hash, requireCache bool) []byte

	// GetStorageByVersion gets storage by addr and version.
	GetStorageByVersion(addr common.Address, version uint64, key common.Hash, requireCache bool) common.Hash

	// GetReadOnly gets the read only world state from the layered world state.
	GetReadOnly() ReadOnlyWorldState

	// GetMutable gets the mutable world state from the layered world state that supports state mutation.
	GetMutable() MutableWorldState

	// HandlePrune handles the pruning of the state in the state layer.
	HandlePrune(pruningRoute []common.Hash)

	// Destruct destructs this layer by removing data associated with this layer from the underlying db.
	Destruct()

	// DestructChild destructs given child by removing data associated with this child from the underlying db.
	DestructChild(child common.Hash)

	// GetRootHash gets the root hash of this layer.
	GetRootHash() common.Hash

	// GetHeight gets the current block height of this layer.
	GetHeight() uint64

	// GetLayerLog gets the log representing this layer.
	GetLayerLog() itypes.LayerLog

	// GetChildren gets the list of children.
	GetChildren() []LayeredWorldState

	// UpdateParent updates the parent of this layer to given state.
	UpdateParent(parent LayeredWorldState)

	// AddChild adds a child to the layer.
	AddChild(child LayeredWorldState)
}

type LayeredWorldStateArchive interface {
	// GetChainConfig gets the chain configuration.
	GetChainConfig() *params.ChainConfig

	// SetMaxLayerToRetain sets the configured max layer to retain in memory.
	SetMaxLayerToRetain(maxLayerToRetain uint64)

	// GetMaxLayerToRetain gets the configured max layer to retain in memory.
	GetMaxLayerToRetain() uint64

	// SetPruningFrequency sets the configured pruning frequency.
	SetPruningFrequency(pruningFrequency uint64)

	// GetPruningFrequency gets the configured pruning frequency.
	GetPruningFrequency() uint64

	// Register registers a state in the archive.
	Register(height uint64, root common.Hash, worldState LayeredWorldState)

	// Deregister deregisters a state in the archive.
	Deregister(height uint64, root common.Hash)

	// GetLayared gets the layered world state with given root hash.
	GetLayared(height uint64, root common.Hash) (LayeredWorldState, error)

	// GetMutable gets the mutable world state with given root hash from the layered world state that supports state mutation.
	GetMutable(height uint64, root common.Hash) (MutableWorldState, error)

	// Has checks if given state exists.
	Has(height uint64, root common.Hash) bool
}
