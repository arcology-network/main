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

package config

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
)

type GlobalConfig struct {
	ChainId       *big.Int          `json:"chain_id"`
	Kafka         map[string]string `json:"kafka"`
	Rpc           map[string]string `json:"rpc"`
	Zookeeper     string            `json:"zookeeper"`
	Concurrency   map[string]int    `json:"concurrency"`
	ClusterName   string            `json:"cluster_name"`
	ClusterId     int               `json:"cluster_id"`
	LogConfigFile string            `json:"log_config_file"`
	// Coinbase        string                   `json:"coinbase"`
	PersistentPeers string                   `json:"persistent_peers"`
	RemoteCaches    string                   `json:"remote_caches"`
	P2pPeers        []map[string]interface{} `json:"p2p.peers"`
	P2pGateway      map[string]interface{}   `json:"p2p.gateway"`
	P2pConn         map[string]interface{}   `json:"p2p.conn"`
}

func LoadGlobalConfig(file string) GlobalConfig {
	jsonFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	bytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}

	var config GlobalConfig
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		panic(err)
	}

	return config
}

func (global GlobalConfig) GetConcurrency(service string) int {
	if c, ok := global.Concurrency[service]; ok {
		return c
	}
	return global.Concurrency["default"]
}
