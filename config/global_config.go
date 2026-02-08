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
	"io/ioutil"
	"math/big"

	"gopkg.in/yaml.v2"
)

type GlobalConfig struct {
	ChainId           *big.Int                 `yaml:"chain_id"`
	Concurrency       int                      `yaml:"concurrency"`
	Executors         []map[string]interface{} `yaml:"executors"`
	ClusterName       string                   `yaml:"cluster_name"`
	ClusterId         int                      `yaml:"cluster_id"`
	LogConfigFile     string                   `yaml:"log_config_file"`
	Coinbase          string                   `yaml:"coinbase"`
	PersistentPeers   string                   `json:"persistent_peers"`
	RpcConcurrent     int                      `yaml:"rpcConcurrent"`
	RpcTimeoutSeconds int                      `yaml:"rpcTimeoutSeconds"`
}

func LoadGlobalConfig(path string) (*GlobalConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &GlobalConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// func LoadGlobalConfig(file string) GlobalConfig {
// 	jsonFile, err := os.Open(file)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer jsonFile.Close()

// 	bytes, err := ioutil.ReadAll(jsonFile)
// 	if err != nil {
// 		panic(err)
// 	}

// 	var config GlobalConfig
// 	err = json.Unmarshal(bytes, &config)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return config
// }

// func (global GlobalConfig) GetConcurrency(service string) int {
// 	if c, ok := global.Concurrency[service]; ok {
// 		return c
// 	}
// 	return global.Concurrency["default"]
// }
