package app

import (
	storetypes "cosmossdk.io/store/types"
	circuittypes "cosmossdk.io/x/circuit/types"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	hypercoretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	hyperwarptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
	epochstypes "github.com/osmosis-labs/osmosis/v15/x/epochs/types"
	gammtypes "github.com/osmosis-labs/osmosis/v15/x/gamm/types"
	poolmanagertypes "github.com/osmosis-labs/osmosis/v15/x/poolmanager/types"
	txfeestypes "github.com/osmosis-labs/osmosis/v15/x/txfees/types"

	irotypes "github.com/dymensionxyz/dymension/v3/x/iro/types"
	kastypes "github.com/dymensionxyz/dymension/v3/x/kas/types"

	dymnstypes "github.com/dymensionxyz/dymension/v3/x/dymns/types"

	delayedacktypes "github.com/dymensionxyz/dymension/v3/x/delayedack/types"
	eibcmoduletypes "github.com/dymensionxyz/dymension/v3/x/eibc/types"
	incentivestypes "github.com/dymensionxyz/dymension/v3/x/incentives/types"
	lightcliendmoduletypes "github.com/dymensionxyz/dymension/v3/x/lightclient/types"
	lockuptypes "github.com/dymensionxyz/dymension/v3/x/lockup/types"
	rollappmoduletypes "github.com/dymensionxyz/dymension/v3/x/rollapp/types"
	sequencermoduletypes "github.com/dymensionxyz/dymension/v3/x/sequencer/types"
	sponsorshiptypes "github.com/dymensionxyz/dymension/v3/x/sponsorship/types"
	streamermoduletypes "github.com/dymensionxyz/dymension/v3/x/streamer/types"
)

// GenerateKeys generates new keys (KV Store, Transient store, and memory store).
func (a *AppKeepers) GenerateKeys() {
	// Define what keys will be used in the cosmos-sdk key/value store.
	// Cosmos-SDK modules each have a "key" that allows the application to reference what they've stored on the chain.
	a.keys = KVStoreKeys

	// Define transient store keys
	a.tkeys = storetypes.NewTransientStoreKeys(paramstypes.TStoreKey, evmtypes.TransientKey, feemarkettypes.TransientKey)

	// MemKeys are for information that is stored only in RAM.
	a.memKeys = storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
}

// GetSubspace gets existing substore from keeper.
func (a *AppKeepers) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := a.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// GetKVStoreKeys gets KV Store keys.
func (a *AppKeepers) GetKVStoreKeys() map[string]*storetypes.KVStoreKey {
	return a.keys
}

// GetTransientStoreKey gets Transient Store keys.
func (a *AppKeepers) GetTransientStoreKey() map[string]*storetypes.TransientStoreKey {
	return a.tkeys
}

// GetMemoryStoreKey get memory Store keys.
func (a *AppKeepers) GetMemoryStoreKey() map[string]*storetypes.MemoryStoreKey {
	return a.memKeys
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (a *AppKeepers) GetKey(storeKey string) *storetypes.KVStoreKey {
	return a.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (a *AppKeepers) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return a.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (a *AppKeepers) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return a.memKeys[storeKey]
}

var KVStoreKeys = storetypes.NewKVStoreKeys(
	authtypes.StoreKey,
	authzkeeper.StoreKey,
	banktypes.StoreKey,
	stakingtypes.StoreKey,
	minttypes.StoreKey,
	distrtypes.StoreKey,
	slashingtypes.StoreKey,
	govtypes.StoreKey,
	paramstypes.StoreKey,
	ibcexported.StoreKey,
	upgradetypes.StoreKey,
	feegrant.StoreKey,
	evidencetypes.StoreKey,
	ibctransfertypes.StoreKey,
	capabilitytypes.StoreKey,
	circuittypes.StoreKey,
	crisistypes.StoreKey,
	consensusparamtypes.StoreKey,
	irotypes.StoreKey,
	rollappmoduletypes.StoreKey,
	sequencermoduletypes.StoreKey,
	sponsorshiptypes.StoreKey,
	streamermoduletypes.StoreKey,
	packetforwardtypes.StoreKey,
	delayedacktypes.StoreKey,
	eibcmoduletypes.StoreKey,
	dymnstypes.StoreKey,
	lightcliendmoduletypes.StoreKey,
	grouptypes.StoreKey,
	hypercoretypes.ModuleName,
	hyperwarptypes.ModuleName,
	kastypes.ModuleName,

	// ethermint keys
	evmtypes.StoreKey,
	feemarkettypes.StoreKey,

	// osmosis keys
	lockuptypes.StoreKey,
	epochstypes.StoreKey,
	gammtypes.StoreKey,
	poolmanagertypes.StoreKey,
	incentivestypes.StoreKey,
	txfeestypes.StoreKey,
	ratelimittypes.StoreKey,
)
