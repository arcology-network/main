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
	"encoding/binary"
	"errors"
	"io"
)

const (
	PackageHeaderSize                 = 37
	PackageHeaderVersionPos           = 0
	PackageHeaderIDPos                = 1
	PackageHeaderTotalPackageCountPos = 9
	PackageHeaderPackageNoPos         = 13
	PackageHeaderDataLenPos           = 17
	PackageHeaderTotalDataLenPos      = 21
	PackageHeaderDataOffsetPos        = 29
	MaxPackageSize                    = 4 * 1024
	MaxPackageBodySize                = MaxPackageSize - PackageHeaderSize
)

type PackageHeader struct {
	Version           byte
	ID                uint64
	TotalPackageCount uint32
	PackageNo         uint32
	DataLen           uint32
	TotalDataLen      uint64
	DataOffset        uint64
}

type Package struct {
	Header PackageHeader
	Body   []byte
}

func (p Package) MarshalBinary() ([]byte, error) {
	buf := make([]byte, PackageHeaderSize+len(p.Body))
	buf[PackageHeaderVersionPos] = p.Header.Version
	binary.LittleEndian.PutUint64(buf[PackageHeaderIDPos:], p.Header.ID)
	binary.LittleEndian.PutUint32(buf[PackageHeaderTotalPackageCountPos:], p.Header.TotalPackageCount)
	binary.LittleEndian.PutUint32(buf[PackageHeaderPackageNoPos:], p.Header.PackageNo)
	binary.LittleEndian.PutUint32(buf[PackageHeaderDataLenPos:], p.Header.DataLen)
	binary.LittleEndian.PutUint64(buf[PackageHeaderTotalDataLenPos:], p.Header.TotalDataLen)
	binary.LittleEndian.PutUint64(buf[PackageHeaderDataOffsetPos:], p.Header.DataOffset)
	copy(buf[PackageHeaderSize:], p.Body)

	return buf, nil
}

func (p *Package) UnmarshalBinary(b []byte) error {
	p.Header.Version = b[PackageHeaderVersionPos]
	p.Header.ID = binary.LittleEndian.Uint64(b[PackageHeaderIDPos:])
	p.Header.TotalPackageCount = binary.LittleEndian.Uint32(b[PackageHeaderTotalPackageCountPos:])
	p.Header.PackageNo = binary.LittleEndian.Uint32(b[PackageHeaderPackageNoPos:])
	p.Header.DataLen = binary.LittleEndian.Uint32(b[PackageHeaderDataLenPos:])
	p.Header.TotalDataLen = binary.LittleEndian.Uint64(b[PackageHeaderTotalDataLenPos:])
	p.Header.DataOffset = binary.LittleEndian.Uint64(b[PackageHeaderDataOffsetPos:])

	return nil
}

func ReadPackage(reader *bufio.Reader) (*Package, error) {
	var h [PackageHeaderSize]byte
	n, err := io.ReadFull(reader, h[:])
	if err != nil {
		return nil, err
	}
	if n != PackageHeaderSize {
		return nil, errors.New("package header is broken")
	}

	var p Package
	p.UnmarshalBinary(h[:])

	b := make([]byte, p.Header.DataLen)
	n, err = io.ReadFull(reader, b[:])
	if err != nil {
		return nil, err
	}
	if uint32(n) != p.Header.DataLen {
		return nil, errors.New("insufficient package data")
	}
	p.Body = b
	return &p, nil
}

func WritePackage(writer io.Writer, p *Package) error {
	b, _ := p.MarshalBinary()
	_, err := writer.Write(b)
	return err
}
