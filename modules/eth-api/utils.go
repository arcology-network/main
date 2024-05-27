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

package ethapi

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	internal "github.com/arcology-network/main/modules/eth-api/backend"
	mtypes "github.com/arcology-network/main/types"
	jsonrpc "github.com/deliveroo/jsonrpc-go"
	eth "github.com/ethereum/go-ethereum"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethflt "github.com/ethereum/go-ethereum/eth/filters"
)

func AttachChainId(tx *mtypes.RPCTransaction, chainid uint64) *mtypes.RPCTransaction { //} map[string]string {
	tx.ChainID = (*hexutil.Big)(big.NewInt(0).SetUint64(chainid))
	return tx
}

func ToTransactionResponse(tx *mtypes.RPCTransaction, chainid uint64) interface{} { //} map[string]string {
	return AttachChainId(tx, chainid)
}
func ToBlockIndex(v interface{}) (int, error) {
	if str, ok := v.(string); !ok {
		return 0, errors.New("unexpected data type given")
	} else {
		if str[:2] == "0x" {
			str = str[2:]
		}
		iint, err := strconv.ParseInt(str, 10, 0)
		if err != nil {
			return 0, errors.New("unexpected data type given")
		}
		return int(iint), nil
	}
}

func ToBlockNumber(v interface{}) (int64, error) {
	if str, ok := v.(string); !ok {
		return 0, errors.New("unexpected data type given")
	} else {
		switch str {
		case "latest":
			return mtypes.BlockNumberLatest, nil
		case "earliest":
			return mtypes.BlockNumberEarliest, nil
		case "pending":
			return mtypes.BlockNumberPending, nil
		case "finalized":
			return mtypes.BlockNumberFinalized, nil
		case "safe":
			return mtypes.BlockNumberSafe, nil
		default:
			if str[:2] == "0x" {
				str = str[2:]
			}
			return strconv.ParseInt(str, 16, 0)
		}
	}
}

func ToBool(v interface{}) (bool, error) {
	if str, ok := v.(bool); !ok {
		return false, errors.New("unexpected data type given")
	} else {
		return str, nil
	}
}

func ToAddress(v interface{}) (ethcmn.Address, error) {
	if str, ok := v.(string); !ok {
		return ethcmn.Address{}, errors.New("unexpected data type given")
	} else {
		return ethcmn.HexToAddress(str), nil
	}
}

func ToHash(v interface{}) (ethcmn.Hash, error) {
	if str, ok := v.(string); !ok {
		return ethcmn.Hash{}, errors.New("unexpected data type given")
	} else {
		return ethcmn.HexToHash(str), nil
	}
}

func ParseJsonParam[T any](v interface{}, paramDesc string) (*T, error) {
	var obj T
	vData, err := json.Marshal(v)
	if err != nil {
		return nil, errors.New("cannot marshal " + paramDesc)
	}
	if err := json.Unmarshal(vData, &obj); err != nil {
		return nil, errors.New("cannot parse " + paramDesc)
	}
	return &obj, nil
}

func ToBytes(v interface{}) ([]byte, error) {
	if str, ok := v.(string); !ok {
		return []byte{}, errors.New("unexpected data type given")
	} else {
		if str[:2] == "0x" {
			str = str[2:]
		}
		bytes, err := hex.DecodeString(str)
		if err != nil {
			return []byte{}, errors.New("invalid characters included")
		}
		return bytes, nil
	}
}

func ToCallMsg(v interface{}, needFrom bool) (eth.CallMsg, error) {
	if m, ok := v.(map[string]interface{}); !ok {
		return eth.CallMsg{}, errors.New("unexpected data type given")
	} else {
		msg := eth.CallMsg{}

		if data, ok := m["data"]; ok {
			if str, ok := data.(string); !ok {
				return eth.CallMsg{}, errors.New("unexpected data type given in data field")
			} else {
				if str[:2] == "0x" {
					str = str[2:]
				}
				bytes, err := hex.DecodeString(str)
				if err != nil {
					return eth.CallMsg{}, errors.New("invalid characters included in data field")
				}
				msg.Data = bytes
			}
		}

		if needFrom {
			if from, ok := m["from"]; !ok {
				return eth.CallMsg{}, errors.New("from field missing")
			} else if str, ok := from.(string); !ok {
				return eth.CallMsg{}, errors.New("unexpected data type given in from field")
			} else {
				msg.From = ethcmn.HexToAddress(str)
			}
		}

		if to, ok := m["to"]; ok {
			if str, ok := to.(string); !ok {
				return eth.CallMsg{}, errors.New("unexpected data type given in to field")
			} else {
				to := ethcmn.HexToAddress(str)
				msg.To = &to
			}
		}

		if gas, ok := m["gas"]; ok {
			if str, ok := gas.(string); !ok {
				return eth.CallMsg{}, errors.New("unexpected data type given in gas field")
			} else {
				if str[:2] == "0x" {
					str = str[2:]
				}
				gas, err := strconv.ParseUint(str, 16, 0)
				if err != nil {
					return eth.CallMsg{}, errors.New("invalid characters included in gas field")
				}
				msg.Gas = gas
			}
		}

		if gasPrice, ok := m["gasPrice"]; ok {
			if str, ok := gasPrice.(string); !ok {
				return eth.CallMsg{}, errors.New("unexpected data type given in gasPrice field")
			} else {
				if str[:2] == "0x" {
					str = str[2:]
				}
				msg.GasPrice, _ = new(big.Int).SetString(str, 16)
			}
		}

		if value, ok := m["value"]; ok {
			if str, ok := value.(string); !ok {
				return eth.CallMsg{}, errors.New("unexpected data type given in value field")
			} else {
				if str[:2] == "0x" {
					str = str[2:]
				}
				msg.Value, _ = new(big.Int).SetString(str, 16)
			}
		}

		return msg, nil
	}
}

type SendTxArgs struct {
	To       *ethcmn.Address
	Gas      uint64
	GasPrice *big.Int
	Value    *big.Int
	Nonce    uint64
	Data     []byte
}

func ToSendTxArgs(v interface{}) (SendTxArgs, error) {
	callMsg, err := ToCallMsg(v, true)
	if err != nil {
		return SendTxArgs{}, err
	}
	nonce := uint64(0)

	if noncestr, ok := v.(map[string]interface{})["nonce"]; ok {
		if str, ok := noncestr.(string); !ok {
			return SendTxArgs{}, errors.New("unexpected data type given in nonce field")
		} else {
			if str[:2] == "0x" {
				str = str[2:]
			}
			nonce, err = strconv.ParseUint(str, 16, 0)
			if err != nil {
				return SendTxArgs{}, errors.New("invalid characters included in gas field")
			}
		}
	} else {
		number, _ := ToBlockNumber("latest")
		nonce, err = backend.GetTransactionCount(callMsg.From, number)
		if err != nil {
			return SendTxArgs{}, jsonrpc.InternalError(err)
		}
	}

	if callMsg.Value == nil {
		callMsg.Value = big.NewInt(0)
	}

	sendTxArgs := SendTxArgs{
		To:       callMsg.To,
		Gas:      callMsg.Gas,
		GasPrice: callMsg.GasPrice,
		Value:    callMsg.Value,
		Data:     callMsg.Data,
		Nonce:    nonce,
	}

	return sendTxArgs, nil
}
func ToID(v interface{}) (internal.ID, error) {
	if id, ok := v.(string); !ok {
		return "", errors.New("unexpected data type given")
	} else {
		return internal.ID(id), nil
	}
}
func ToFilter(v interface{}) (eth.FilterQuery, error) {
	if m, ok := v.(map[string]interface{}); !ok {
		return eth.FilterQuery{}, errors.New("unexpected data type given")
	} else {
		if v, ok := m["fromBlock"]; ok {
			from := v.(string)
			if strings.Index(from, "0x") != 0 {
				from, err := strconv.ParseInt(v.(string), 10, 64)
				if err != nil {
					return eth.FilterQuery{}, err
				}
				m["fromBlock"] = fmt.Sprintf("0x%x", from)
			}
		}
		if v, ok := m["toBlock"]; ok {
			to := v.(string)
			if strings.Index(to, "0x") != 0 {
				to, err := strconv.ParseInt(v.(string), 10, 64)
				if err != nil {
					return eth.FilterQuery{}, err
				}
				m["toBlock"] = fmt.Sprintf("0x%x", to)
			}
		}
		bytes, _ := json.Marshal(m)
		var filter ethflt.FilterCriteria
		if err := filter.UnmarshalJSON(bytes); err != nil {
			return eth.FilterQuery{}, err
		}

		return eth.FilterQuery(filter), nil
	}
}

func LoadKeys(keyFile string) []string {
	file, err := os.Open(keyFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var privateKeys []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		segments := strings.Split(line, ",")
		privateKeys = append(privateKeys, segments[0])
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return privateKeys
}

func NumberToHex(n interface{}) string {
	return fmt.Sprintf("0x%x", n)
}
