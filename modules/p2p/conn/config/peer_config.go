package config

import "encoding/json"

const (
	WorkModePassive = 1
	WorkModeActive  = 2
)

type PeerConfig struct {
	ID              string `json:"id"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	ConnectionCount byte   `json:"connections"`
	AssignTo        string `json:"assign_to"`
	Mode            int    `json:"mode"`
}

func (cfg PeerConfig) ToJsonStr() (string, error) {
	b, err := json.Marshal(cfg)
	return string(b), err
}

func (cfg *PeerConfig) FromJsonStr(str string) error {
	return json.Unmarshal([]byte(str), cfg)
}
