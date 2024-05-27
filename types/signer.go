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

package types

import (
	"math/big"

	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

const (
	Signer_London = iota
	Signer_Cancun
)

func DefaultSignerType() uint8 {
	return Signer_London
}

func MakeSigner(signerType uint8, chainId *big.Int) evmTypes.Signer {
	switch signerType {
	case Signer_London:
		return evmTypes.NewLondonSigner(chainId)
	case Signer_Cancun:
		return evmTypes.NewCancunSigner(chainId)
	}
	return nil
}

func GetSignerType(height *big.Int, config *params.ChainConfig) uint8 {
	var signerType uint8

	switch {
	case isCancun():
		signerType = Signer_Cancun
	default:
		signerType = Signer_London
	}
	return signerType
}

func isCancun() bool {
	return false
}
