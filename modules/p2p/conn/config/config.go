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
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		NID  string `yaml:"nid" json:"nid"`
		SID  string `yaml:"sid" json:"sid"`
		Host string `yaml:"host" json:"host"`
		Port int    `yaml:"port" json:"port"`
	} `yaml:"server" json:"server"`
	ZooKeeper struct {
		Servers        []string `yaml:"servers" json:"servers"`
		PeerConfigRoot string   `yaml:"croot" json:"croot"`
		ConnStatusRoot string   `yaml:"sroot" json:"sroot"`
	} `yaml:"zk" json:"zk"`
	Kafka struct {
		Servers  []string `yaml:"servers" json:"servers"`
		TopicIn  string   `yaml:"tin" json:"tin"`
		TopicOut string   `yaml:"tout" json:"tout"`
	} `yaml:"kafka" json:"kafka"`
}

func LoadConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
