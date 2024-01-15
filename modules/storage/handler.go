package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	kafkalib "github.com/arcology-network/streamer/kafka/lib"
)

type data struct {
	Columns []map[string]string `json:"columns"`
	Rows    [][]string          `json:"rows"`
	Type    string              `json:"type"`
}

type Handler struct {
	scanCache *mstypes.ScanCache

	fetchFrequency time.Duration
	max            uint64
	host           string
	step           string
	fetchAhead     int64 //second

	coefficient uint64
}

func NewHandler(scanCache *mstypes.ScanCache, params map[string]interface{}) *Handler {
	handle := Handler{
		scanCache:      scanCache,
		fetchFrequency: time.Duration(params["prometheus_fetch_frequency"].(float64)) * time.Second, //time.Duration(waits) * time.Second
		fetchAhead:     int64(params["prometheus_fetch_ahead"].(float64)),
		host:           params["prometheus_host"].(string),
		step:           params["prometheus_fetch_step"].(string),
		coefficient:    uint64(params["coefficient"].(float64)),
	}

	tim := kafkalib.SyncTimer{}
	tim.StartTimer(handle.fetchFrequency, handle.timerQuery)

	return &handle
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
				raw[1] = fmt.Sprintf("%v", blocks[i].Time) //getTimeStr(blocks[i].Time)
				raw[2] = fmt.Sprintf("%v", blocks[i].TxNumbers)
				raw[3] = blocks[i].Hash
				raw[4] = fmt.Sprintf("%v", blocks[i].GasUsed)
				raws[i] = raw
			}
			jsonStr, _ := json.Marshal([]data{
				{
					Columns: []map[string]string{
						{"text": "Block", "type": "string"},
						{"text": "Age", "type": "string"},
						{"text": "Txn", "type": "string"},
						{"text": "Hash", "type": "string"},
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
				raw := make([]string, 6)
				//{"0xabc...def", "1 min ago", "0x111...111", "0x222...222", "0 Ether"}
				raw[0] = transactions[i].TxHash
				raw[1] = fmt.Sprintf("%v", transactions[i].Time) //getTimeStr(transactions[i].Time)
				raw[2] = transactions[i].Sender
				raw[3] = transactions[i].SendTo
				raw[4] = fmt.Sprintf("%v", transactions[i].Amount)
				raw[5] = transactions[i].Status
				raws[i] = raw
			}
			jsonStr, _ := json.Marshal([]data{
				{
					Columns: []map[string]string{
						{"text": "Txn Hash", "type": "string"},
						{"text": "Age", "type": "string"},
						{"text": "Sender", "type": "string"},
						{"text": "SendTo", "type": "string"},
						{"text": "Amount", "type": "string"},
						{"text": "Status", "type": "string"},
					},
					Rows: raws,
					Type: "table",
				},
			})
			w.Write(jsonStr)
		case "Tabs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			tabs := h.scanCache.GetAllTabs()
			jsonStr, _ := json.Marshal(tabs)
			w.Write(jsonStr)
		}
	}
}

func (h *Handler) timerQuery() {
	current := time.Now().Unix()
	from := strconv.FormatInt(current-h.fetchAhead, 10)
	to := strconv.FormatInt(current, 10)
	for k, _ := range dic {
		res := queryData(k, h.host, from, to, h.step)
		if res == nil || len(res.Data.Result) == 0 {
			continue
		}
		vals := res.Data.Result[0].Values
		if len(vals) == 0 {
			continue
		}
		switch k {
		case "height":
			h.scanCache.SetTabs(mstypes.BlockHeight, parseInt(vals[len(vals)-1][1]))
		case "totals":
			h.scanCache.SetTabs(mstypes.TotalTxs, parseInt(vals[len(vals)-1][1]))
		case "tps":
			latestVal := uint64(0)
			max := uint64(0)
			for i := 0; i < len(vals); i++ {
				latestVal = parseInt(vals[i][1])
				if latestVal > h.max {
					h.max = latestVal
				}
				if latestVal > max {
					max = latestVal
				}
			}
			if max > 0 {
				h.scanCache.SetTabs(mstypes.Tps, max)
			}
			h.scanCache.SetTabs(mstypes.MaxTps, h.max*h.coefficient/100)
		}
	}
}

var (
	dic = map[string]string{
		"height": "tendermint_consensus_height",
		"totals": "consensus_processed_txs_total{job=~\"monaco\"}",
		"tps":    "consensus_real_time_tps{job=~\"monaco\"}",
	}
)

func queryData(target, _host, from, to, step string) *Response {
	params := url.Values{}
	Url, _ := url.Parse(_host + "/api/datasources/proxy/2/api/v1/query_range")
	params.Set("query", dic[target])
	params.Set("start", from) //"1666921554")
	params.Set("end", to)     //"1666921854")   //
	params.Set("step", step)
	Url.RawQuery = params.Encode()
	urlPath := Url.String()
	resp, err := http.Get(urlPath)
	if resp == nil {
		fmt.Printf("query from prometheus err: %v urlPath:%v\n", err, urlPath)
		return nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	// strbody := string(body)
	// fmt.Println("==============" + strbody)

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Printf("query from prometheus Unmarshal err=%v\n", err)
		return nil
	}
	return &response
}

func parseInt(val interface{}) uint64 {
	v := val.(string)
	uv, err := strconv.ParseUint(strings.Split(v, ".")[0], 10, 64)
	if err != nil {
		return 0
	}
	return uv
}

type Response struct {
	Status string `json:"status"`
	Data   INData `json:"data"`
}

type INData struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}
type Result struct {
	Metric Metric          `json:"metric"`
	Values [][]interface{} `json:"values"`
}

type Metric struct {
	Name     string `json:"__name__"`
	Chain_id string `json:"chain_id"`
	Instance string `json:"instance"`
	Job      string `json:"job"`
}
