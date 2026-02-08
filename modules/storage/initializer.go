/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package storage

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/arcology-network/common-lib/storage/transactional"
	"github.com/arcology-network/consensus-engine/state"
	adaptorcommon "github.com/arcology-network/eu/eth"
	"github.com/arcology-network/main/modules/core"
	interfaces "github.com/arcology-network/storage-committer/common"
	"github.com/arcology-network/storage-committer/type/commutative"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/logger"
	"github.com/ethereum/go-ethereum/cmd/utils"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	apihandler "github.com/arcology-network/eu/apihandler"
	cache "github.com/arcology-network/storage-committer/storage/cache"
	evmcore "github.com/ethereum/go-ethereum/core"

	"github.com/arcology-network/common-lib/exp/mempool"
	"github.com/arcology-network/common-lib/exp/slice"
	mtypes "github.com/arcology-network/main/types"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"

	statestore "github.com/arcology-network/storage-committer"
	stgproxy "github.com/arcology-network/storage-committer/storage/proxy"

	scommon "github.com/arcology-network/streamer/common"
)

type Initializer struct {
	genesisFile     string
	storage_db_path string
	sender          actor.OutboundSender

	from string
}

func NewInitializer() actor.Business {
	return &Initializer{
		from: "storage.intializer",
	}
}

func (i *Initializer) Inputs() ([]string, bool) {
	return []string{}, false
}

func (i *Initializer) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgLocalParentInfo:  1,
		scommon.MsgInitialization:   1,
		scommon.MsgInitScheduletate: 1,
	}
}

func (i *Initializer) SetSender(sender actor.OutboundSender) {
	i.sender = sender
}

// Config implements Configurable interface.
func (i *Initializer) Config(params map[string]interface{}) {
	i.genesisFile = params["genesis_file"].(string)
	i.storage_db_path = params["dbpath"].(string)
}

func (i *Initializer) InitMsgs() []*scommon.Message {
	ret, err := i.sender.SendSync("tmstatestore", "Load", "", 0, i.from)
	if err != nil {
		panic(err)
	}
	state := ret.(state.State)

	height := state.LastBlockHeight

	genesis := ReadGenesis(i.genesisFile)
	blockStart := &actor.BlockStart{
		Timestamp: big.NewInt(int64(genesis.Timestamp)),
		Coinbase:  genesis.Coinbase,
		Extra:     genesis.ExtraData,
	}

	parentinfo := &mtypes.ParentInfo{}
	var store *statestore.StateStore
	var rootHash evmCommon.Hash
	if height == 0 {
		// Make place holder for recover functions.
		transactional.RegisterRecoverFunc("urlupdate", func(interface{}, []byte) error {
			return nil
		})
		transactional.RegisterRecoverFunc("parentinfo", func(interface{}, []byte) error {
			return nil
		})
		transactional.RegisterRecoverFunc("schdstate", func(interface{}, []byte) error {
			return nil
		})

		store, rootHash, _ = InitGenesisAccounts(i.storage_db_path, genesis, uint64(height))

		evmblock := genesis.ToBlock()

		block, err := core.CreateBlock(evmblock.Header(), [][]byte{}, mtypes.GetSignerType(big.NewInt(height), genesis.Config))
		if err != nil {
			panic("Create genesis block err!")
		}
		i.sender.SendSync("blockstore", "Save", block, block.Height, i.from)
		hash := evmblock.Hash()
		// var na int
		i.sender.SendSync("statestore", "Save", &State{
			Height:     0,
			ParentHash: hash,
			ParentRoot: rootHash,
		}, block.Height, i.from)
		excessBlobGas := uint64(0)
		if evmblock.Header().ExcessBlobGas != nil {
			excessBlobGas = *evmblock.Header().ExcessBlobGas
		}
		blobGasUsed := uint64(0)
		if evmblock.Header().BlobGasUsed != nil {
			blobGasUsed = *evmblock.Header().BlobGasUsed
		}
		parentinfo = &mtypes.ParentInfo{
			ParentHash:    hash,
			ParentRoot:    rootHash,
			ExcessBlobGas: excessBlobGas,
			BlobGasUsed:   blobGasUsed,
		}
	} else {

		// db = ccdb.NewLevelDBDataStore(i.storage_db_path)

		db := stgproxy.NewLevelDBStoreProxy(i.storage_db_path) //.EnableCache()
		db.Inject(RootPrefix, commutative.NewPath())
		store = statestore.NewStateStore(db)

		// Register recover function.
		transactional.RegisterRecoverFunc("urlupdate", func(_ interface{}, bs []byte) error {
			// var updates storage.UrlUpdate
			// if err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&updates); err != nil {
			// 	fmt.Printf("Error decoding UrlUpdate, err = %v\n", err)
			// 	return err
			// }

			// values := make([]interface{}, len(updates.EncodedValues))
			// for i, v := range updates.EncodedValues {
			// 	values[i] = ccdb.Codec{}.Decode(v, nil) //urltyp.FromBytes(v)
			// }

			// db.BatchInject(updates.Keys, values)
			// fmt.Printf("[storage.Initializer] Recover urlupdate.\n")
			return nil
		})
		transactional.RegisterRecoverFunc("parentinfo", func(_ interface{}, bs []byte) error {
			var pi mtypes.ParentInfo
			if err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&pi); err != nil {
				fmt.Printf("Error decoding ParentInfo, err = %v\n", err)
				return err
			}

			// var na int
			i.sender.SendSync("statestore", "Save", &State{
				Height:        uint64(height),
				ParentHash:    pi.ParentHash,
				ParentRoot:    pi.ParentRoot,
				ExcessBlobGas: pi.ExcessBlobGas,
				BlobGasUsed:   pi.BlobGasUsed,
			}, uint64(height), i.from)
			logger.Log.Debug(context.Background(), i.from, "[storage.Initializer] Recover parentinfo", logger.F("pi", pi))
			return nil
		})
		transactional.RegisterRecoverFunc("schdstate", func(_ interface{}, bs []byte) error {
			var state mtypes.SchdState
			if err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&state); err != nil {
				logger.Log.Error(context.Background(), i.from, "Error decoding SchdState", logger.F("err", err))
				return err
			}

			// var na int
			i.sender.SendSync("schdstore", "DirectWrite", &state, uint64(height), i.from)
			logger.Log.Debug(context.Background(), i.from, "[storage.Initializer] Recover schdstate.")
			return nil
		})
		// Recover.
		txID := fmt.Sprintf("%d", height)
		// var na int
		// fmt.Printf("[storage.Initializer] Recover transactional store to height: %s\n", txID)
		logger.Log.Debug(context.Background(), i.from, "[storage.Initializer] Recover transactional store", logger.F("height", txID))
		_, err := i.sender.SendSync("transactionalstore", "Recover", txID, uint64(height), i.from)
		if err != nil {
			panic(fmt.Sprintf("[storage.Initializer] Error occurred while recovering transactional store: %v\n", err))
		}

		ret, err = i.sender.SendSync("statestore", "GetParentInfo", "", uint64(height), i.from)
		if err != nil {
			panic(err)
		}
		parentinfo = ret.(*mtypes.ParentInfo)
	}

	i.sender.SendSync("urlstore", "Init", store.ReadOnlyStore(), uint64(height), i.from)
	i.sender.SendSync("storage", "InitHeight", uint64(height), uint64(height), i.from)

	ret, err = i.sender.SendSync("schdstore", "Load", "", uint64(height), i.from)
	if err != nil {
		panic(fmt.Sprintf("load conflication err : %v\n", err))
	}
	states := ret.([]mtypes.SchdState)

	return []*scommon.Message{
		{
			Name:   scommon.MsgLocalParentInfo,
			Height: uint64(height),
			Data:   parentinfo,
			From:   i.from,
		},
		{
			Name:   scommon.MsgInitScheduletate,
			Height: uint64(height),
			Data:   states,
			From:   i.from,
		},
		{
			Name:   scommon.MsgInitialization,
			Height: uint64(height),
			Data: &mtypes.Initialization{
				Store:             store,
				BlockStart:        blockStart,
				ChainConfig:       genesis.Config,
				ParentInformation: parentinfo,
			},
			From: i.from,
		},
	}
}
func (i *Initializer) RegisterActions(reg actor.ActionRegistrar) {

}
func InitGenesisAccounts(dbpath string, genesis *evmcore.Genesis, height uint64) (*statestore.StateStore, evmCommon.Hash, []*univaluepk.Univalue) {
	db := stgproxy.NewLevelDBStoreProxy(dbpath)
	stateStore := statestore.NewStateStore(db)
	db.Inject(RootPrefix, commutative.NewPath())

	transitions := createTransitions(db, genesis.Alloc)

	stateStore.Import(slice.Clone(transitions))
	stateStore.Precommit([]uint64{0})
	stateStore.Commit(height)

	return stateStore, evmCommon.Hash{}, transitions
}

//--------------------------------------------------------------------------------------------------------------------------------

func createTransitions(db interfaces.ReadOnlyStore, genesisAlloc evmcore.GenesisAlloc) []*univaluepk.Univalue {
	batch := 10
	addresses := make([]evmCommon.Address, 0, batch)
	index := 0
	transitions := make([]*univaluepk.Univalue, 0, len(genesisAlloc)*10)
	for addr, _ := range genesisAlloc {
		if index%batch == 0 && index > 0 {
			transitions = append(transitions, getTransition(db, addresses, genesisAlloc)...)
			addresses = make([]evmCommon.Address, 0, batch)
		}
		addresses = append(addresses, addr)
		index++
	}
	if len(addresses) > 0 {
		transitions = append(transitions, getTransition(db, addresses, genesisAlloc)...)
	}
	return transitions
}

func getTransition(db interfaces.ReadOnlyStore, addresses []evmCommon.Address, genesisAlloc evmcore.GenesisAlloc) []*univaluepk.Univalue {
	api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
		return cache.NewWriteCache(db, 32, 1)
	}, func(cache *cache.WriteCache) { cache.Clear() }))

	stateDB := adaptorcommon.NewImplStateDB(api)
	stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for _, addr := range addresses {
		acct := genesisAlloc[addr]
		stateDB.CreateAccount(addr)
		bl, ok := uint256.FromBig(acct.Balance)
		if ok {
			bl = uint256.NewInt(0)
		}
		stateDB.SetBalance(addr, bl)
		stateDB.SetNonce(addr, uint64(1))
		code := acct.Code
		if len(code) > 0 {
			stateDB.SetCode(addr, code)
		}
		for k, v := range acct.Storage {
			stateDB.SetState(addr, k, v)
		}

	}
	_, transitions := api.WriteCache().(*cache.WriteCache).ExportAll()

	return transitions
}

// readGenesis will read the given JSON format genesis file and return
// the initialized Genesis structure
func ReadGenesis(genesisPath string) *evmcore.Genesis {
	// Make sure we have a valid genesis JSON
	//genesisPath := ctx.Args().First()
	if len(genesisPath) == 0 {
		utils.Fatalf("Must supply path to genesis JSON file")
	}
	file, err := os.Open(genesisPath)
	if err != nil {
		utils.Fatalf("Failed to read genesis file: %v", err)
	}
	defer file.Close()

	genesis := new(evmcore.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		utils.Fatalf("invalid genesis file: %v", err)
	}
	return genesis
}
