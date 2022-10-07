package config

import "testing"

func TestToJson(t *testing.T) {
	cfg := PeerConfig{
		ID:              "peerid",
		Host:            "remote",
		Port:            5555,
		ConnectionCount: 4,
		AssignTo:        "conn-svc1",
		Mode:            1,
	}
	jsonStr := `{"id":"peerid","host":"remote","port":5555,"connections":4,"assign_to":"conn-svc1","mode":1}`

	json, err := cfg.ToJsonStr()
	if err != nil {
		t.Error(err)
		return
	}

	if json != jsonStr {
		t.Error("Fail")
	}
}

func TestFromJson(t *testing.T) {
	jsonStr := `{"id":"peerid","host":"remote","port":5555,"connections":4,"assign_to":"conn-svc1","mode":1}`

	var cfg PeerConfig
	err := cfg.FromJsonStr(jsonStr)
	if err != nil {
		t.Error(err)
		return
	}

	if cfg.ID != "peerid" ||
		cfg.Host != "remote" ||
		cfg.Port != 5555 ||
		cfg.ConnectionCount != 4 ||
		cfg.AssignTo != "conn-svc1" ||
		cfg.Mode != 1 {
		t.Error("Fail")
	}
}
