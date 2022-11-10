package protocol

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/arcology-network/main/modules/p2p/conn/mock"
)

func TestToPackages(t *testing.T) {
	size := MaxPackageSize * 8
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i / MaxPackageBodySize)
	}

	m := Message{
		Type: 0xcc,
		Data: data,
	}
	packages := m.ToPackages()

	for i, p := range packages {
		if i == 0 {
			if p.Header.DataOffset != 0xcc {
				t.Error("Fail")
				return
			}
		} else {
			if p.Header.DataOffset != uint64(i*MaxPackageBodySize) {
				t.Error("Fail")
				return
			}
		}

		if p.Header.Version != ProtocolVersion ||
			p.Header.TotalPackageCount != uint32(size/MaxPackageBodySize+1) ||
			p.Header.PackageNo != uint32(i) ||
			p.Header.TotalDataLen != uint64(size) {
			t.Error("Fail")
			return
		}

		if i == len(packages)-1 {
			if p.Header.DataLen != uint32(size-size/MaxPackageBodySize*MaxPackageBodySize) {
				t.Error("Fail")
				return
			}
		} else {
			if p.Header.DataLen != MaxPackageBodySize {
				t.Error("Fail")
				return
			}
		}

		if uint32(len(p.Body)) != p.Header.DataLen {
			t.Error("Fail")
			return
		}

		for _, b := range p.Body {
			if b != byte(i) {
				t.Errorf("b = %v, i = %v\n", b, i)
				return
			}
		}
	}
}

func TestFromPackages(t *testing.T) {
	size := MaxPackageSize * 8
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i / MaxPackageBodySize)
	}

	m := Message{
		Type: 0xcc,
		Data: data,
	}
	packages := m.ToPackages()

	var m2 Message
	m2.FromPackages(packages)

	if m2.Type != 0xcc {
		t.Error("Fail")
		return
	}

	for i := 0; i < size; i++ {
		if m2.Data[i] != byte(i/MaxPackageBodySize) {
			t.Error("Fail")
			return
		}
	}
}

func TestMessageReadWrite(t *testing.T) {
	m := Message{
		ID:   0xcc,
		Type: 0xdd,
		Data: []byte{0x11, 0x22, 0x33},
	}

	conn := mock.NewTCPConnection()
	if err := WriteMessage(conn, &m); err != nil {
		t.Error(err)
		return
	}

	reader := bufio.NewReader(conn)
	m2, err := ReadMessage(reader)
	if err != nil {
		t.Error(err)
		return
	}

	if m.ID != m2.ID ||
		m.Type != m2.Type ||
		!bytes.Equal(m.Data, m2.Data) {
		t.Error("Fail")
		return
	}
}
