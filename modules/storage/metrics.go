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

package storage

import (
	"time"

	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	CollectTime = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Subsystem: "eshing",
		Name:      "collect_seconds",
		Help:      "The duration of collection step.",
	}, []string{})
	CollectTimeGauge = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Subsystem: "eshing",
		Name:      "collect_seconds_gauge",
		Help:      "The duration of collection step.",
	}, []string{})
	CalcTime = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Subsystem: "eshing",
		Name:      "calc_seconds",
		Help:      "The duration of calculation step.",
	}, []string{})
	CalcTimeGauge = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Subsystem: "eshing",
		Name:      "calc_seconds_gauge",
		Help:      "The duration of calculation step.",
	}, []string{})
)

type Metrics struct {
	collectStart time.Time
	calcStart    time.Time
}

func NewMetrics() actor.Business {
	metrics := &Metrics{
		collectStart: time.Now(),
		calcStart:    time.Now(),
	}
	return metrics
}

func (m *Metrics) Inputs() ([]string, bool) {
	return []string{scommon.MsgInclusive, scommon.MsgExecuted, scommon.MsgAcctHash}, false
}

func (m *Metrics) Outputs() map[string]int {
	return map[string]int{}
}

func (m *Metrics) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInclusive, m.receivedList)
	reg.Register(scommon.MsgExecuted, m.receivedExecuted)
	reg.Register(scommon.MsgAcctHash, m.receivedAcctHash)
}

func (m *Metrics) receivedList(ctx *actor.ActionContext) error {
	m.collectStart = time.Now()
	return nil
}
func (m *Metrics) receivedExecuted(ctx *actor.ActionContext) error {
	m.calcStart = time.Now()
	CollectTime.Observe(m.calcStart.Sub(m.collectStart).Seconds())
	CollectTimeGauge.Set(m.calcStart.Sub(m.collectStart).Seconds())
	return nil
}

func (m *Metrics) receivedAcctHash(ctx *actor.ActionContext) error {
	CalcTime.Observe(time.Since(m.calcStart).Seconds())
	CalcTimeGauge.Set(time.Since(m.calcStart).Seconds())
	return nil
}
