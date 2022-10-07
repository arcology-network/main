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
