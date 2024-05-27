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

package consensus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/spf13/cobra"

	"github.com/arcology-network/consensus-engine/privval"

	"github.com/arcology-network/consensus-engine/p2p"

	"github.com/arcology-network/consensus-engine/cmd/tendermint/commands"
	cfg "github.com/arcology-network/consensus-engine/config"
	"github.com/arcology-network/consensus-engine/libs/log"
	tmos "github.com/arcology-network/consensus-engine/libs/os"
	tmrand "github.com/arcology-network/consensus-engine/libs/rand"
	"github.com/arcology-network/consensus-engine/types"
	tmtime "github.com/arcology-network/consensus-engine/types/time"
)

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	// af     = ""
	//aft         = ""
	// addressfile = ""
)

// func init() {
// 	flags := InitCmd.Flags()

// 	// flags.String("af", "", "address file for create genesis")
// 	//flags.String("aft", "", "address templete file ")
// 	// flags.String("addressfile", "", "address  file ")
// }

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "init consensus Daemon",
	RunE:  initCmd,
}

func initCmd(cmd *cobra.Command, args []string) error {
	// af = viper.GetString("af")
	//aft = viper.GetString("aft")
	// addressfile = viper.GetString("addressfile")
	initCfg()
	return nil
}

func AddToAF(addressfile, af, address string) error {
	affile, err := os.OpenFile(af, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	if err != nil {
		fmt.Printf("open file err=%v\n", err)
		return err
	}
	defer affile.Close()

	data, err := ioutil.ReadFile(addressfile)
	if err != nil {
		fmt.Printf("read file addressfile  err=%v\n", err)
		return err
	}
	_, err = affile.Write(data)
	if err != nil {
		fmt.Printf("write address file into filec err=%v\n", err)
		return err
	}

	basebys := []byte("0000,0x" + address + ",0\n")
	_, err = affile.Write(basebys)
	if err != nil {
		fmt.Printf("write current node address into file err=%v\n", err)
		return err
	}

	return nil
}

// if not init ,so init
func initCfg() {
	balance := big.NewInt(0)
	// this will ensure that config.toml is there if not yet created, and create dir
	config, err := commands.ParseConfig()
	if err != nil {
		fmt.Printf("read file templete  err=%v\n", err)
		return
	}

	pv, userAddr, err := GetAddress(config)
	if err != nil {
		return
	}

	// if len(af) > 0 {

	// 	err := AddToAF(addressfile, af, userAddr)
	// 	if err != nil {
	// 		fmt.Printf("AddToAF  err=%v\n", err)
	// 		return
	// 	}

	// }

	var DEFAULT_DENOM = "mycoin"

	// Now, we want to add the custom app_state
	appState, err := DefaultGenAppState(userAddr, DEFAULT_DENOM, balance)
	if err != nil {
		fmt.Printf("DefaultGenAppState  err=%v\n", err)
		return
	}

	err = initFilesWithConfig(config, userAddr, pv, appState)
	if err != nil {
		return
	}

	return
}

func GetAddress(config *cfg.Config) (*privval.FilePV, string, error) {
	privValKeyFile := config.PrivValidatorKeyFile()
	privValStateFile := config.PrivValidatorStateFile()
	var pv *privval.FilePV
	if tmos.FileExists(privValKeyFile) {
		pv = privval.LoadFilePV(privValKeyFile, privValStateFile)
		logger.Info("Found private validator", "keyFile", privValKeyFile,
			"stateFile", privValStateFile)
	} else {
		pv = privval.GenFilePV(privValKeyFile, privValStateFile)
		pv.Save()
		logger.Info("Generated private validator", "keyFile", privValKeyFile,
			"stateFile", privValStateFile)
	}
	pk, err := pv.GetPubKey()
	if err != nil {
		return nil, "", err
	}

	return pv, pk.Address().String(), nil

}
func initFilesWithConfig(config *cfg.Config, addr string, pv *privval.FilePV, appState json.RawMessage) error {

	nodeKeyFile := config.NodeKeyFile()
	if tmos.FileExists(nodeKeyFile) {
		logger.Info("Found node key", "path", nodeKeyFile)
	} else {
		if _, err := p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			return err
		}
		logger.Info("Generated node key", "path", nodeKeyFile)
	}

	// genesis file
	genFile := config.GenesisFile()
	if tmos.FileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		genDoc := types.GenesisDoc{
			ChainID:         fmt.Sprintf("test-chain-%v", tmrand.Str(6)),
			GenesisTime:     tmtime.Now(),
			ConsensusParams: types.DefaultConsensusParams(),
		}
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return fmt.Errorf("can't get pubkey: %w", err)
		}
		genDoc.Validators = []types.GenesisValidator{{
			Address: pubKey.Address(),
			PubKey:  pubKey,
			Power:   10,
		}}
		genDoc.AppState = appState
		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	return nil
}

// DefaultGenAppState expects two args: an account address
// and a coin denomination, and gives lots of coins to that address.
func DefaultGenAppState(address string, coinDenom string, balance *big.Int) (json.RawMessage, error) {
	opts := fmt.Sprintf(`{
      "accounts": [{
        "address": "%s",
        "coins": [
          {
            "denom": "%s",
            "amount": %d
          }
        ]
      }]
    }`, address, coinDenom, balance)
	return json.RawMessage(opts), nil
}
