// +build NOCI

package config

import (
	"testing"
	"time"

	"github.com/go-zookeeper/zk"
)

func TestPeerConfigWatcher(t *testing.T) {
	conn, _, err := zk.Connect([]string{"localhost:2181"}, 60*time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	watcher, err := NewPeerConfigWatcher(conn, "/p2p/peer/config", nil)
	if err != nil {
		t.Error(err)
		return
	}
	watcher.Serve()
}
