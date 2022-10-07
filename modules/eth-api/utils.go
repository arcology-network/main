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

	"github.com/HPISTechnologies/component-lib/ethrpc"
	eth "github.com/HPISTechnologies/evm"
	ethcmn "github.com/HPISTechnologies/evm/common"
	ethflt "github.com/HPISTechnologies/evm/eth/filters"
	internal "github.com/HPISTechnologies/main/modules/eth-api/backend"
)

func ToTransactionResponse(tx *ethrpc.RPCTransaction) map[string]string {
	return map[string]string{
		"value":    NumberToHex(tx.Value),
		"gas":      NumberToHex(tx.Gas),
		"gasPrice": NumberToHex(tx.GasPrice),
		"from":     tx.From.Hex(),
	}
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
			return ethrpc.BlockNumberLatest, nil
		case "earliest":
			return ethrpc.BlockNumberEarliest, nil
		case "pending":
			return ethrpc.BlockNumberPending, nil
		default:
			if str[:2] == "0x" {
				str = str[2:]
			}
			return strconv.ParseInt(str, 16, 0)
		}
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
	callMsg, err := ToCallMsg(v, false)
	if err != nil {
		return SendTxArgs{}, err
	}

	sendTxArgs := SendTxArgs{
		To:       callMsg.To,
		Gas:      callMsg.Gas,
		GasPrice: callMsg.GasPrice,
		Value:    callMsg.Value,
		Data:     callMsg.Data,
	}
	if nonce, ok := v.(map[string]interface{})["nonce"]; ok {
		if str, ok := nonce.(string); !ok {
			return SendTxArgs{}, errors.New("unexpected data type given in nonce field")
		} else {
			if str[:2] == "0x" {
				str = str[2:]
			}
			nonce, err := strconv.ParseUint(str, 16, 0)
			if err != nil {
				return SendTxArgs{}, errors.New("invalid characters included in gas field")
			}
			sendTxArgs.Nonce = nonce
		}
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
			from, err := strconv.ParseInt(v.(string), 10, 64)
			if err != nil {
				return eth.FilterQuery{}, err
			}
			m["fromBlock"] = fmt.Sprintf("0x%x", from)
		}
		if v, ok := m["toBlock"]; ok {
			to, err := strconv.ParseInt(v.(string), 10, 64)
			if err != nil {
				return eth.FilterQuery{}, err
			}
			m["toBlock"] = fmt.Sprintf("0x%x", to)
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
