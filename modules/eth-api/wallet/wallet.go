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
	"errors"
	"fmt"
	"math/big"

	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtyp "github.com/ethereum/go-ethereum/core/types"
	ethcrp "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

type Wallet struct {
	accounts []string
	keys     []*ecdsa.PrivateKey
	signer   ethtyp.Signer
	findkeys map[ethcmn.Address]*ecdsa.PrivateKey
}

func NewWallet(chainId *big.Int, privateKeys []string) *Wallet {
	accounts := make([]string, 0, len(privateKeys))
	keys := make([]*ecdsa.PrivateKey, 0, len(privateKeys))
	findkeys := make(map[ethcmn.Address]*ecdsa.PrivateKey, len(privateKeys))
	for _, pk := range privateKeys {
		privateKey, _ := ethcrp.HexToECDSA(pk)
		account := ethcrp.PubkeyToAddress(*(privateKey.Public().(*ecdsa.PublicKey))).Hex()
		accounts = append(accounts, account)
		keys = append(keys, privateKey)
		findkeys[ethcmn.HexToAddress(pk)] = privateKey
	}
	return &Wallet{
		accounts: accounts,
		keys:     keys,
		signer:   ethtyp.NewLondonSigner(chainId),
		findkeys: findkeys,
	}
}

func (w *Wallet) Accounts() []string {
	return w.accounts
}
func TextAndHash(data []byte) ([]byte, string) {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), string(data))
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(msg))
	return hasher.Sum(nil), msg
}
func TextHash(data []byte) []byte {
	hash, _ := TextAndHash(data)
	return hash
}
func (w *Wallet) Sign(acct ethcmn.Address, data []byte) ([]byte, error) {
	key, ok := w.findkeys[acct]
	if !ok {
		return []byte{}, errors.New("address not found")
	}

	signature, err := ethcrp.Sign(TextHash(data), key)
	if err != nil {
		return nil, err
	}
	signature[ethcrp.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return signature, nil
}

func (w *Wallet) SignTx(index int, nonce uint64, to *ethcmn.Address, value *big.Int, gas uint64, gasPrice *big.Int, data []byte) ([]byte, error) {
	if index >= len(w.keys) {
		return nil, errors.New("account index out of bounds")
	}

	tx := ethtyp.NewTx(
		&ethtyp.LegacyTx{
			Nonce:    nonce,
			To:       to,
			Value:    value,
			Gas:      gas,
			GasPrice: gasPrice,
			Data:     data,
		})
	hash := w.signer.Hash(tx)
	signature, err := ethcrp.Sign(hash[:], w.keys[index])
	if err != nil {
		return nil, err
	}
	signedTx, err := tx.WithSignature(w.signer, signature)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	ethtyp.Transactions{signedTx}.EncodeIndex(0, buf)
	return buf.Bytes(), nil
}
