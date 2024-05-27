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

package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtyp "github.com/ethereum/go-ethereum/core/types"
	ethcrp "github.com/ethereum/go-ethereum/crypto"
	ethrlp "github.com/ethereum/go-ethereum/rlp"
	bip32 "github.com/tyler-smith/go-bip32"
	bip39 "github.com/tyler-smith/go-bip39"
)

func TestWallet(t *testing.T) {
	mnemonic := "elevator ridge panic maid response dragon pony ghost annual insect crime auction chaos scrap brother"
	seed := bip39.NewSeed(mnemonic, "")
	masterKey, _ := bip32.NewMasterKey(seed)
	publicKey := masterKey.PublicKey()

	// Display mnemonic and keys
	fmt.Println("Mnemonic: ", mnemonic)
	fmt.Println("Master private key: ", masterKey)
	fmt.Println("Master public key: ", publicKey)
}

func TestSign(t *testing.T) {
	privateKey, _ := ethcrp.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	publicKey, _ := privateKey.Public().(*ecdsa.PublicKey)
	from := ethcrp.PubkeyToAddress(*publicKey)
	t.Log(from)
	tx := ethtyp.NewTransaction(
		1,
		ethcmn.HexToAddress("0x57De3b28C55095E5cA67a8e20fA9D7D5d9aEf891"),
		new(big.Int).SetUint64(0),
		10000,
		new(big.Int).SetUint64(1),
		[]byte{1},
	)
	signer := ethtyp.NewLondonSigner(new(big.Int).SetUint64(1))
	h := signer.Hash(tx)
	signature, _ := ethcrp.Sign(h[:], privateKey)
	signedTx, _ := tx.WithSignature(signer, signature)
	ts := ethtyp.Transactions{signedTx}
	buf := new(bytes.Buffer)
	ts.EncodeIndex(0, buf)
	rawTransaction := fmt.Sprintf("%x", buf.Bytes())
	t.Log(rawTransaction)

	tx2 := new(ethtyp.Transaction)
	ethrlp.DecodeBytes(buf.Bytes(), tx2)
	msg, _ := core.TransactionToMessage(tx2, ethtyp.NewLondonSigner(new(big.Int).SetUint64(1)), nil)
	t.Log(
		msg.From.Hex(),
		msg.To.Hex(),
		msg.Nonce,
		msg.GasLimit,
		msg.GasPrice,
		msg.Value,
	)
}
