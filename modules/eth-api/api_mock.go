package ethapi

import (
	"context"
	"fmt"

	jsonrpc "github.com/deliveroo/jsonrpc-go"
)

func mock_chainId(ctx context.Context) (interface{}, error) {
	return "0x539", nil
}

func mock_accounts(ctx context.Context) (interface{}, error) {
	accounts := make([]string, 2)
	accounts[0] = "0x77ddd8c0fd54f68d871355d62b6ee79ea7551e1c"
	accounts[1] = "0x0047c4e79a35632d7c9cc3fafe79c554512f9c6c"
	return accounts, nil
}

func mock_estimateGas(ctx context.Context, params []interface{}) (interface{}, error) {
	data := getVal(params[0], "data")
	if len(data) > 5000 {
		return "0xa34c4", nil
	} else {
		return "0x3946a", nil
	}

}

func mock_getBlockByNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	transactions := make([]string, 0)
	uncles := make([]string, 0)
	return map[string]interface{}{
		"difficulty":       "0x20580",
		"extraData":        "0xd883010b00846765746888676f312e31392e34856c696e7578",
		"gasLimit":         "0x311293",
		"gasUsed":          "0x0",
		"hash":             "0xc9be02db34b1bbe7f66277593302b75a44b7790a3ceb0f94360f901e7dcad33b",
		"logsBloom":        "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"miner":            "0xa75cd05bf16bbea1759de2a66c0472131bc5bd8d",
		"mixHash":          "0x3b04af80eef9d52ac3028b6491fee685c65cd2a2058b2548ca4a6565f92fea30",
		"nonce":            "0x19421bb5234d9e16",
		"number":           "0x18",
		"parentHash":       "0x91abfe9ae9682314fea98a79f5fbfaa806e8d651d3013c89ae2df777d083c2f5",
		"receiptsRoot":     "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
		"sha3Uncles":       "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
		"size":             "0x218",
		"stateRoot":        "0x975669804a6477ac9b85ea2803cad3f0844c37741f90bb0f2bb4d4a623b18849",
		"timestamp":        "0x63ac40a3",
		"totalDifficulty":  "0x3042c2",
		"transactions":     transactions,
		"transactionsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
		"uncles":           uncles,
	}, nil
}

func mock_sendTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	tx, err := ToSendTxArgs(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid transaction given %v", params[0])
	}
	if len(tx.Data) > 5000 {
		return "0x800dd969fe653f821c976fff5ba73827a3bbea6f7b90218e15dc436e44653f96", nil
	} else {
		return "0x3daf1e00bc68459ff63927ea9080bd3b61505e1a61158bfdab9fd4259911201a", nil
	}
}

func mock_getTransactionByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	if fmt.Sprintf("0x%x", hash.Bytes()) == "0x800dd969fe653f821c976fff5ba73827a3bbea6f7b90218e15dc436e44653f96" {
		return map[string]interface{}{
			"blockHash":        nil,
			"blockNumber":      nil,
			"from":             "0x77ddd8c0fd54f68d871355d62b6ee79ea7551e1c",
			"gas":              "0xa34c4",
			"gasPrice":         "0x3b9aca00",
			"hash":             "0x800dd969fe653f821c976fff5ba73827a3bbea6f7b90218e15dc436e44653f96",
			"input":            "0x608060405234801561001057600080fd5b5060008080526020527fad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb580546001600160a01b03191633179055610ac3806100596000396000f3fe608060405234801561001057600080fd5b50600436106100d45760003560e01c80635b0fc9c311610081578063cf4088231161005b578063cf40882314610215578063e985e9c514610228578063f79fe5381461027457600080fd5b80635b0fc9c3146101dc5780635ef2c7f0146101ef578063a22cb4651461020257600080fd5b806314ab9038116100b257806314ab90381461015657806316a25cbd1461016b5780631896f70a146101c957600080fd5b80630178b8bf146100d957806302571be31461012257806306ab592314610135575b600080fd5b6101056100e73660046108b2565b6000908152602081905260409020600101546001600160a01b031690565b6040516001600160a01b0390911681526020015b60405180910390f35b6101056101303660046108b2565b61029f565b6101486101433660046108e7565b6102cd565b604051908152602001610119565b610169610164366004610934565b6103eb565b005b6101b06101793660046108b2565b60009081526020819052604090206001015474010000000000000000000000000000000000000000900467ffffffffffffffff1690565b60405167ffffffffffffffff9091168152602001610119565b6101696101d7366004610960565b6104e3565b6101696101ea366004610960565b6105b5565b6101696101fd366004610983565b610681565b6101696102103660046109da565b6106a3565b610169610223366004610a16565b61072d565b610264610236366004610a63565b6001600160a01b03918216600090815260016020908152604080832093909416825291909152205460ff1690565b6040519015158152602001610119565b6102646102823660046108b2565b6000908152602081905260409020546001600160a01b0316151590565b6000818152602081905260408120546001600160a01b03163081036102c75750600092915050565b92915050565b60008381526020819052604081205484906001600160a01b03163381148061031857506001600160a01b038116600090815260016020908152604080832033845290915290205460ff165b61032157600080fd5b6040805160208101889052908101869052600090606001604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291815281516020928301206000818152928390529120805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03881617905590506040516001600160a01b0386168152869088907fce0457fe73731f824cc272376169235128c118b49d344817417c6d108d155e829060200160405180910390a39695505050505050565b60008281526020819052604090205482906001600160a01b03163381148061043657506001600160a01b038116600090815260016020908152604080832033845290915290205460ff165b61043f57600080fd5b60405167ffffffffffffffff8416815284907f1d4f9bbfc9cab89d66e1a1562f2233ccbf1308cb4f63de2ead5787adddb8fa689060200160405180910390a25050600091825260208290526040909120600101805467ffffffffffffffff90921674010000000000000000000000000000000000000000027fffffffff0000000000000000ffffffffffffffffffffffffffffffffffffffff909216919091179055565b60008281526020819052604090205482906001600160a01b03163381148061052e57506001600160a01b038116600090815260016020908152604080832033845290915290205460ff165b61053757600080fd5b6040516001600160a01b038416815284907f335721b01866dc23fbee8b6b2c7b1e14d6f05c28cd35a2c934239f94095602a09060200160405180910390a25050600091825260208290526040909120600101805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b03909216919091179055565b60008281526020819052604090205482906001600160a01b03163381148061060057506001600160a01b038116600090815260016020908152604080832033845290915290205460ff165b61060957600080fd5b6000848152602081905260409020805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b0385161790556040516001600160a01b038416815284907fd4735d920b0f87494915f556dd9b54c8f309026070caea5c737245152564d2669060200160405180910390a250505050565b600061068e8686866102cd565b905061069b818484610748565b505050505050565b3360008181526001602090815260408083206001600160a01b0387168085529083529281902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001686151590811790915590519081529192917f17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31910160405180910390a35050565b61073784846105b5565b610742848383610748565b50505050565b6000838152602081905260409020600101546001600160a01b038381169116146107db5760008381526020818152604091829020600101805473ffffffffffffffffffffffffffffffffffffffff19166001600160a01b038616908117909155915191825284917f335721b01866dc23fbee8b6b2c7b1e14d6f05c28cd35a2c934239f94095602a0910160405180910390a25b60008381526020819052604090206001015467ffffffffffffffff8281167401000000000000000000000000000000000000000090920416146108ad576000838152602081815260409182902060010180547fffffffff0000000000000000ffffffffffffffffffffffffffffffffffffffff167401000000000000000000000000000000000000000067ffffffffffffffff861690810291909117909155915191825284917f1d4f9bbfc9cab89d66e1a1562f2233ccbf1308cb4f63de2ead5787adddb8fa68910160405180910390a25b505050565b6000602082840312156108c457600080fd5b5035919050565b80356001600160a01b03811681146108e257600080fd5b919050565b6000806000606084860312156108fc57600080fd5b8335925060208401359150610913604085016108cb565b90509250925092565b803567ffffffffffffffff811681146108e257600080fd5b6000806040838503121561094757600080fd5b823591506109576020840161091c565b90509250929050565b6000806040838503121561097357600080fd5b82359150610957602084016108cb565b600080600080600060a0868803121561099b57600080fd5b85359450602086013593506109b2604087016108cb565b92506109c0606087016108cb565b91506109ce6080870161091c565b90509295509295909350565b600080604083850312156109ed57600080fd5b6109f6836108cb565b915060208301358015158114610a0b57600080fd5b809150509250929050565b60008060008060808587031215610a2c57600080fd5b84359350610a3c602086016108cb565b9250610a4a604086016108cb565b9150610a586060860161091c565b905092959194509250565b60008060408385031215610a7657600080fd5b610a7f836108cb565b9150610957602084016108cb56fea264697066735822122021b17638a4be2d6c2a1553b0dd4be7c60c1cdee62def5def46c5106c04f620ad64736f6c63430008110033",
			"nonce":            "0x0",
			"to":               nil,
			"transactionIndex": nil,
			"value":            "0x0",
			"type":             "0x0",
			"chainId":          "0x539",
			"v":                "0xa96",
			"r":                "0xa7f91ea7510ede164b0d2c5009930194c706b67669baa88b5a12517bfe7fbd5d",
			"s":                "0x6fd3a38aaaf894f1aff486903a747d9d3f0ee485ee6c7d969fa4a67903b79a93",
		}, nil
	} else {
		return map[string]interface{}{
			"blockHash":        nil,
			"blockNumber":      nil,
			"from":             "0x77ddd8c0fd54f68d871355d62b6ee79ea7551e1c",
			"gas":              "0x3946a",
			"gasPrice":         "0x3b9aca00",
			"hash":             "0x3daf1e00bc68459ff63927ea9080bd3b61505e1a61158bfdab9fd4259911201a",
			"input":            "0x608060405234801561001057600080fd5b5060405161047338038061047383398101604081905261002f91610058565b600061003b82826101aa565b5050610269565b634e487b7160e01b600052604160045260246000fd5b6000602080838503121561006b57600080fd5b82516001600160401b038082111561008257600080fd5b818501915085601f83011261009657600080fd5b8151818111156100a8576100a8610042565b604051601f8201601f19908116603f011681019083821181831017156100d0576100d0610042565b8160405282815288868487010111156100e857600080fd5b600093505b8284101561010a57848401860151818501870152928501926100ed565b600086848301015280965050505050505092915050565b600181811c9082168061013557607f821691505b60208210810361015557634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156101a557600081815260208120601f850160051c810160208610156101825750805b601f850160051c820191505b818110156101a15782815560010161018e565b5050505b505050565b81516001600160401b038111156101c3576101c3610042565b6101d7816101d18454610121565b8461015b565b602080601f83116001811461020c57600084156101f45750858301515b600019600386901b1c1916600185901b1785556101a1565b600085815260208120601f198616915b8281101561023b5788860151825594840194600190910190840161021c565b50858210156102595787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b6101fb806102786000396000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c80630e89341c14610030575b600080fd5b61004361003e3660046100ed565b610059565b6040516100509190610106565b60405180910390f35b60606000805461006890610172565b80601f016020809104026020016040519081016040528092919081815260200182805461009490610172565b80156100e15780601f106100b6576101008083540402835291602001916100e1565b820191906000526020600020905b8154815290600101906020018083116100c457829003601f168201915b50505050509050919050565b6000602082840312156100ff57600080fd5b5035919050565b600060208083528351808285015260005b8181101561013357858101830151858201604001528201610117565b5060006040828601015260407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f8301168501019250505092915050565b600181811c9082168061018657607f821691505b6020821081036101bf577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b5091905056fea264697066735822122070e624b94ff2259309e55c3b8032fddc3cf9a21aadc739221bd250387014d1be64736f6c6343000811003300000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000021687474703a2f2f6c6f63616c686f73743a383038302f6e616d652f30787b69647d00000000000000000000000000000000000000000000000000000000000000",
			"nonce":            "0x1",
			"to":               nil,
			"transactionIndex": nil,
			"value":            "0x0",
			"type":             "0x0",
			"chainId":          "0x539",
			"v":                "0xa96",
			"r":                "0xfb930ea0a4f55cda937e889952b673680e0eeeb84097ce68e4a2fb3fcdd625bf",
			"s":                "0x5499e41e5a6faf3e91afc238c82f77157bcde0700dcd75f8d4d91e9b64ce14b9",
		}, nil
	}
}

var nonceadd = false

func mock_getTransactionReceipt(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}
	logs := make([]string, 0)
	if fmt.Sprintf("0x%x", hash.Bytes()) == "0x800dd969fe653f821c976fff5ba73827a3bbea6f7b90218e15dc436e44653f96" {
		nonceadd = true
		return map[string]interface{}{
			"blockHash":         "0xd3cd680bcff36ea3d38e4f9a6999bbdea32f445d27245a7140958cf2eef568fb",
			"blockNumber":       "0x19",
			"contractAddress":   "0x70c060ee15f31fad4f9cd392103117087b524995",
			"cumulativeGasUsed": "0xa34c4",
			"effectiveGasPrice": "0x3b9aca00",
			"from":              "0x77ddd8c0fd54f68d871355d62b6ee79ea7551e1c",
			"gasUsed":           "0xa34c4",
			"logs":              logs,
			"logsBloom":         "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			"status":            "0x1",
			"to":                nil,
			"transactionHash":   "0x800dd969fe653f821c976fff5ba73827a3bbea6f7b90218e15dc436e44653f96",
			"transactionIndex":  "0x0",
			"type":              "0x0",
		}, nil
	} else {
		return map[string]interface{}{
			"blockHash":         "0xca381eefd0af5a77ea11cba8f45c958765a3a6581def1e9a301b3f57e3179f32",
			"blockNumber":       "0x1a",
			"contractAddress":   "0x8d3b356bba38eb130f507dd32df4ce54a4d14239",
			"cumulativeGasUsed": "0x3946a",
			"effectiveGasPrice": "0x3b9aca00",
			"from":              "0x77ddd8c0fd54f68d871355d62b6ee79ea7551e1c",
			"gasUsed":           "0x3946a",
			"logs":              logs,
			"logsBloom":         "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			"status":            "0x1",
			"to":                nil,
			"transactionHash":   "0x3daf1e00bc68459ff63927ea9080bd3b61505e1a61158bfdab9fd4259911201a",
			"transactionIndex":  "0x0",
			"type":              "0x0",
		}, nil
	}
}

func mock_getTransactionCount(ctx context.Context, params []interface{}) (interface{}, error) {
	if nonceadd {
		return "0x1", nil
	} else {
		return "0x0", nil
	}

}

func getVal(v interface{}, k string) string {
	if m, ok := v.(map[string]interface{}); !ok {
		return ""
	} else {
		if data, ok := m[k]; ok {
			if str, ok := data.(string); !ok {
				return ""
			} else {
				return str
			}
		}
	}
	return ""
}
