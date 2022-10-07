package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"time"

	mstypes "github.com/HPISTechnologies/main/modules/storage/types"
)

type data struct {
	Columns []map[string]string `json:"columns"`
	Rows    [][]string          `json:"rows"`
	Type    string              `json:"type"`
}

type Handler struct {
	scanCache *mstypes.ScanCache
}

func NewHandler(scanCache *mstypes.ScanCache) *Handler {
	return &Handler{
		scanCache: scanCache,
	}
}

func getTimeStr(timestamp *big.Int) string {
	start := time.Unix(timestamp.Int64(), 0)
	span := time.Now().Sub(start)
	if span < 0 {
		return "0 seconds ago"
	}
	hs := span.Hours()
	nhs := math.Floor(hs)
	if nhs > 0 {
		ds := nhs / 24
		if ds > 0 {
			return fmt.Sprintf("%v days ago", ds)
		} else {
			return fmt.Sprintf("%v hours ago", nhs)
		}

	}
	ms := span.Minutes()
	nms := math.Floor(ms)
	if nms > 0 {
		return fmt.Sprintf("%v minutes ago", nms)
	}
	ss := span.Seconds()
	nss := math.Floor(ss)
	if nss > 0 {
		return fmt.Sprintf("%v seconds ago", nss)
	}
	return "0 seconds ago"
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	// fmt.Printf("method: %v, url: %v, body: %v\n", r.Method, r.URL, string(body))

	switch r.URL.Path {
	case "/search":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		jsonStr, _ := json.Marshal([]string{"Blocks", "Transactions"})
		w.Write(jsonStr)
	case "/query":
		type params struct {
			Targets []map[string]string
		}
		var p params
		json.Unmarshal(body, &p)
		// fmt.Printf("query: %v\n", p)
		switch p.Targets[0]["target"] {
		case "Blocks":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			blocks := h.scanCache.GetLatestBlocks()
			raws := make([][]string, len(blocks))
			for i := range blocks {
				raw := make([]string, 5)
				//{"1", "1 min ago", "50000", "0xabc...def", "30,000,000"}
				raw[0] = fmt.Sprintf("%v", blocks[i].Height)
				raw[1] = getTimeStr(blocks[i].Time)
				raw[2] = fmt.Sprintf("%v", blocks[i].TxNumbers)
				raw[3] = blocks[i].Proposer
				raw[4] = fmt.Sprintf("%v", blocks[i].GasUsed)
				raws[i] = raw
			}
			jsonStr, _ := json.Marshal([]data{
				{
					Columns: []map[string]string{
						{"text": "Block", "type": "string"},
						{"text": "Age", "type": "string"},
						{"text": "Txn", "type": "string"},
						{"text": "Producer", "type": "string"},
						{"text": "Gas Used", "type": "string"},
					},
					Rows: raws,
					Type: "table",
				},
			})
			w.Write(jsonStr)
		case "Transactions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			transactions := h.scanCache.GetLatestTxs()
			raws := make([][]string, len(transactions))
			for i := range transactions {
				raw := make([]string, 5)
				//{"0xabc...def", "1 min ago", "0x111...111", "0x222...222", "0 Ether"}
				raw[0] = transactions[i].TxHash
				raw[1] = getTimeStr(transactions[i].Time)
				raw[2] = transactions[i].From
				raw[3] = transactions[i].To
				raw[4] = fmt.Sprintf("%v", transactions[i].Value)
				raws[i] = raw
			}
			jsonStr, _ := json.Marshal([]data{
				{
					Columns: []map[string]string{
						{"text": "Txn Hash", "type": "string"},
						{"text": "Age", "type": "string"},
						{"text": "From", "type": "string"},
						{"text": "To", "type": "string"},
						{"text": "Value", "type": "string"},
					},
					Rows: raws,
					Type: "table",
				},
			})
			w.Write(jsonStr)
		}
	}
}
