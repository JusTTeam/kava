package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/snapshots"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
	db "github.com/tendermint/tm-db"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/app/params"
)

const (
	flagMempoolEnableAuth    = "mempool.enable-authentication"
	flagMempoolAuthAddresses = "mempool.authorized-addresses"
)

type appCreator struct {
	encodingConfig params.EncodingConfig
}

// newApp loads config from AppOptions and returns a new app.
func (ac appCreator) newApp(
	logger log.Logger,
	db db.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {

	var cache sdk.MultiStorePersistentCache
	if cast.ToBool(appOpts.Get(server.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	pruningOpts, err := server.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	snapshotDir := filepath.Join(cast.ToString(appOpts.Get(flags.FlagHome)), "data", "snapshots") // TODO can this not be hard coded
	snapshotDB, err := sdk.NewLevelDB("metadata", snapshotDir)
	if err != nil {
		panic(err)
	}
	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	mempoolEnableAuth := cast.ToBool(appOpts.Get(flagMempoolEnableAuth))
	mempoolAuthAddresses, err := accAddressesFromBech32(
		cast.ToStringSlice(appOpts.Get(flagMempoolAuthAddresses))...,
	)
	if err != nil {
		panic(fmt.Sprintf("could not get authorized address from config: %v", err))
	}

	return app.NewApp(
		logger, db, traceStore, ac.encodingConfig,
		app.Options{
			// TODO home dir - needed for upgrade keeper
			// TODO crisis.FlagSkipGenesisInvariants - needed for crisis module
			SkipLoadLatest:       false,
			SkipUpgradeHeights:   skipUpgradeHeights,
			InvariantCheckPeriod: cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
			MempoolEnableAuth:    mempoolEnableAuth,
			MempoolAuthAddresses: mempoolAuthAddresses,
		},
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(server.FlagMinGasPrices))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(server.FlagHaltHeight))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(server.FlagHaltTime))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(server.FlagMinRetainBlocks))), // TODO what is this?
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get(server.FlagTrace))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(server.FlagIndexEvents))),
		baseapp.SetSnapshotStore(snapshotStore),
		baseapp.SetSnapshotInterval(cast.ToUint64(appOpts.Get(server.FlagStateSyncSnapshotInterval))),
		baseapp.SetSnapshotKeepRecent(cast.ToUint32(appOpts.Get(server.FlagStateSyncSnapshotKeepRecent))),
	)
}

// appExport writes out an app's state to json.
func (ac appCreator) appExport(
	logger log.Logger,
	db db.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
) (servertypes.ExportedApp, error) {
	panic("TODO") // TODO
	return servertypes.ExportedApp{}, nil
}

func (ac appCreator) addStartCmdFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
}

// accAddressesFromBech32 converts a slice of bech32 encoded addresses into a slice of address tyeps
func accAddressesFromBech32(addresses ...string) ([]sdk.AccAddress, error) {
	var decodedAddresses []sdk.AccAddress
	for _, s := range addresses {
		a, err := sdk.AccAddressFromBech32(s)
		if err != nil {
			return nil, err
		}
		decodedAddresses = append(decodedAddresses, a)
	}
	return decodedAddresses, nil
}
