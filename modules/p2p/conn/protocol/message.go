package protocol

import (
	"bufio"
	"io"
	"math"
)

type Message struct {
	ID   uint64
	Type byte
	Data []byte
}

const (
	ProtocolVersion            byte = 1
	MessageTypeHandshake       byte = 1
	MessageTypeRouting         byte = 2
	MessageTypeClientBroadcast byte = 3
	MessageTypeClientSend      byte = 4
	MessageTypeTestData        byte = 5
)

func (m Message) ToPackages() []*Package {
	totalDataLen := len(m.Data)
	totalPackageCount := uint32(math.Ceil(float64(totalDataLen) / MaxPackageBodySize))
	packages := make([]*Package, totalPackageCount)
	for i := range packages {
		packages[i] = &Package{
			Header: PackageHeader{
				Version:           ProtocolVersion,
				ID:                m.ID,
				TotalPackageCount: totalPackageCount,
				PackageNo:         uint32(i),
				DataLen:           MaxPackageBodySize,
				TotalDataLen:      uint64(totalDataLen),
				DataOffset:        uint64(i * MaxPackageBodySize),
			},
		}
		if i == len(packages)-1 {
			packages[i].Header.DataLen = uint32(totalDataLen - i*(MaxPackageBodySize))
		}
		packages[i].Body = m.Data[packages[i].Header.DataOffset : packages[i].Header.DataOffset+uint64(packages[i].Header.DataLen)]
		if i == 0 {
			packages[i].Header.DataOffset = uint64(m.Type)
		}
	}
	return packages
}

func (m *Message) FromPackages(packages []*Package) *Message {
	m.Data = make([]byte, packages[0].Header.TotalDataLen)
	for i := range packages {
		if packages[i].Header.PackageNo == 0 {
			m.ID = packages[i].Header.ID
			m.Type = byte(packages[i].Header.DataOffset)
			packages[i].Header.DataOffset = 0
		}
		copy(m.Data[packages[i].Header.DataOffset:], packages[i].Body)
	}
	return m
}

func ReadMessage(reader *bufio.Reader) (*Message, error) {
	p, err := ReadPackage(reader)
	if err != nil {
		return nil, err
	}

	var m Message
	m.FromPackages([]*Package{p})
	return &m, nil
}

func WriteMessage(writer io.Writer, m *Message) error {
	packages := m.ToPackages()
	for _, p := range packages {
		err := WritePackage(writer, p)
		if err != nil {
			return err
		}
	}

	return nil
}
