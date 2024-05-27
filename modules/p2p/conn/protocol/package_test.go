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

package protocol

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/arcology-network/main/modules/p2p/conn/mock"
)

func TestPackageMarshalUnmarshal(t *testing.T) {
	p := Package{
		Header: PackageHeader{
			Version:           0x11,
			ID:                0x22,
			TotalPackageCount: 0x33,
			PackageNo:         0x44,
			DataLen:           0x55,
			TotalDataLen:      0x66,
			DataOffset:        0x77,
		},
		Body: []byte{0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
	}

	b, err := p.MarshalBinary()
	if err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(p.Body, b[PackageHeaderSize:]) {
		t.Error("Fail")
		return
	}

	var p2 Package
	err = p2.UnmarshalBinary(b)
	if err != nil {
		t.Error(err)
		return
	}

	if p.Header.Version != p2.Header.Version ||
		p.Header.ID != p2.Header.ID ||
		p.Header.TotalPackageCount != p2.Header.TotalPackageCount ||
		p.Header.PackageNo != p2.Header.PackageNo ||
		p.Header.DataLen != p2.Header.DataLen ||
		p.Header.TotalDataLen != p2.Header.TotalDataLen ||
		p.Header.DataOffset != p2.Header.DataOffset {
		t.Error("Fail")
		return
	}
}

func TestPackageReadWrite(t *testing.T) {
	p := Package{
		Header: PackageHeader{
			Version:           0x11,
			ID:                0x22,
			TotalPackageCount: 0x33,
			PackageNo:         0x44,
			DataLen:           8,
			TotalDataLen:      0x66,
			DataOffset:        0x77,
		},
		Body: []byte{0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
	}

	conn := mock.NewTCPConnection()
	if err := WritePackage(conn, &p); err != nil {
		t.Error(err)
		return
	}

	reader := bufio.NewReader(conn)
	p2, err := ReadPackage(reader)
	if err != nil {
		t.Error(err)
		return
	}

	if p.Header.Version != p2.Header.Version ||
		p.Header.ID != p2.Header.ID ||
		p.Header.TotalPackageCount != p2.Header.TotalPackageCount ||
		p.Header.PackageNo != p2.Header.PackageNo ||
		p.Header.DataLen != p2.Header.DataLen ||
		p.Header.TotalDataLen != p2.Header.TotalDataLen ||
		p.Header.DataOffset != p2.Header.DataOffset {
		t.Error("Fail")
		return
	}
}
