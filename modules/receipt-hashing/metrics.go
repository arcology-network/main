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

package receipthashing

import (
	"fmt"
	"os"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
)

type Metrics struct {
	actor.WorkerThread
	firstItem *mtypes.TPSGasBurned
	maxTps    int
	maxGas    int

	isOutput   bool
	maxRecords int
	logfile    string

	linesCounter int
}

// return a Subscriber struct
func NewMetrics(concurrency int, groupid string) actor.IWorkerEx {
	cr := Metrics{}
	cr.Set(concurrency, groupid)
	return &cr
}

func (cr *Metrics) Inputs() ([]string, bool) {
	return []string{actor.MsgReceiptInfo}, true
}

func (cr *Metrics) Outputs() map[string]int {
	return map[string]int{}
}

func (cr *Metrics) OnStart() {
}

func (cr *Metrics) Stop() {

}

func (cr *Metrics) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgReceiptInfo:
			if cr.isOutput {
				go func() {
					show := cr.Calculate(v.Data.(*mtypes.ReceiptInfo).TpsGas, v.Height)
					cr.WriteFile(show)
				}()
			}
		}
	}
	return nil
}

func (cr *Metrics) Config(params map[string]interface{}) {
	if v, ok := params["logfile"]; !ok {
		panic("parameter not found: logfile")
	} else {
		cr.logfile = v.(string)
	}

	if len(cr.logfile) > 0 {
		cr.isOutput = true
	} else {
		cr.isOutput = false
	}

	if v, ok := params["maxRecords"]; !ok {
		panic("parameter not found: maxRecords")
	} else {
		cr.maxRecords = int(v.(float64))
	}
}

func (cr *Metrics) Calculate(item *mtypes.TPSGasBurned, height uint64) string {
	if cr.firstItem == nil {
		cr.firstItem = item
		return ""
	}
	mseconds := item.Timestamp - cr.firstItem.Timestamp
	realTps := int(item.TotalTxs) * 1000 / int(mseconds)
	realGasUsed := int(item.GasUsed) * 1000 / int(mseconds)
	if realTps > cr.maxTps {
		cr.maxTps = realTps
	}
	if realGasUsed > cr.maxGas {
		cr.maxGas = realGasUsed
	}

	item.Height = height
	cr.firstItem = item

	showMemo := fmt.Sprintf("height = %v,", height)
	if item.TotalTxs > 0 {
		showMemo = fmt.Sprintf("%s total = %v, success = %v, fail = %v,", showMemo, item.TotalTxs, item.SuccessfulTxs, item.TotalTxs-item.SuccessfulTxs)
	} else {
		showMemo = fmt.Sprintf("%s empty block,", showMemo)
	}
	showMemo = fmt.Sprintf("%s timestamp = %v, maxTps = %v, realtimeTps = %v", showMemo, item.Timestamp, cr.maxTps, realTps)
	showMemo = fmt.Sprintf("%s maxGasBurned = %v, realtimeGasBurned = %v \n", showMemo, cr.maxGas, realGasUsed)
	return showMemo
}

func (cr *Metrics) WriteFile(txt string) {
	if cr.linesCounter >= cr.maxRecords {
		cr.linesCounter = 0
		overwritrefile(cr.logfile, txt)
	}
	cr.linesCounter++
	appendfile(cr.logfile, txt)
}

func overwritrefile(filename, txt string) {
	data := []byte(txt)
	err := os.WriteFile(filename, data, 0644)
	if err != nil {
		fmt.Println("Error overwrite writing to file:", err)
		return
	}
}

func appendfile(filename, txt string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(txt)
	if err != nil {
		fmt.Println("Error apppend writing to file:", err)
		return
	}
}
