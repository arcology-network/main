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
	"crypto/sha256"
	"math/big"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	eucommon "github.com/arcology-network/eu/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

type ExecutingSequence struct {
	Msgs       []*eucommon.StandardMessage
	Parallel   bool
	SequenceId ethCommon.Hash
}

func NewExecutingSequence(msgs []*eucommon.StandardMessage, parallel bool) *ExecutingSequence {
	buffers := make([][]byte, len(msgs))
	for i, msg := range msgs {
		buffers[i] = msg.TxHash[:]
	}

	hash := sha256.Sum256(codec.Byteset(buffers).Encode())
	return &ExecutingSequence{
		Msgs:       msgs,
		Parallel:   parallel,
		SequenceId: ethCommon.BytesToHash(hash[:]),
	}
}

type ExecutingSequences []*ExecutingSequence

func (this ExecutingSequences) Encode() ([]byte, error) {
	if this == nil {
		return []byte{}, nil
	}

	data := make([][]byte, len(this))
	worker := func(start, end, idx int, args ...interface{}) {
		executingSequences := args[0].([]interface{})[0].(ExecutingSequences)
		data := args[0].([]interface{})[1].([][]byte)
		for i := start; i < end; i++ {
			standardMessages := eucommon.StandardMessages(executingSequences[i].Msgs)
			standardMessagesData, err := standardMessages.Encode()
			if err != nil {
				standardMessagesData = []byte{}
			}

			tmpData := [][]byte{
				standardMessagesData,
				codec.Bools([]bool{executingSequences[i].Parallel}).Encode(),
				executingSequences[i].SequenceId[:],
			}
			data[i] = codec.Byteset(tmpData).Encode()
		}
	}
	common.ParallelWorker(len(this), types.Concurrency, worker, this, data)
	return codec.Byteset(data).Encode(), nil
}

func (this *ExecutingSequences) Decode(data []byte) ([]*ExecutingSequence, error) {
	fields := codec.Byteset{}.Decode(data).(codec.Byteset)
	v := ExecutingSequences(make([]*ExecutingSequence, len(fields)))
	this = &v

	worker := func(start, end, idx int, args ...interface{}) {
		data := args[0].([]interface{})[0].(codec.Byteset)
		executingSequences := args[0].([]interface{})[1].(ExecutingSequences)

		for i := start; i < end; i++ {
			executingSequence := new(ExecutingSequence)

			datafields := codec.Byteset{}.Decode(data[i]).(codec.Byteset)
			msgResults, err := new(eucommon.StandardMessages).Decode(datafields[0])
			if err != nil {
				msgResults = []*eucommon.StandardMessage{}
			}
			executingSequence.Msgs = msgResults
			parallels := new(codec.Bools).Decode(datafields[1]).([]bool)
			if len(parallels) > 0 {
				executingSequence.Parallel = parallels[0]
			}
			executingSequence.SequenceId = ethCommon.BytesToHash(datafields[2])
			executingSequences[i] = executingSequence

		}
	}
	common.ParallelWorker(len(fields), types.Concurrency, worker, fields, *this)
	return ([]*ExecutingSequence)(*this), nil
}

type ExecutorRequest struct {
	Sequences     []*ExecutingSequence
	Height        uint64
	GenerationIdx uint32

	Timestamp   *big.Int
	Parallelism uint64
	Debug       bool
}

func (this *ExecutorRequest) GobEncode() ([]byte, error) {
	executingSequences := ExecutingSequences(this.Sequences)
	executingSequencesData, err := executingSequences.Encode()
	if err != nil {
		return []byte{}, err
	}

	timeStampData := []byte{}
	if this.Timestamp != nil {
		timeStampData = this.Timestamp.Bytes()
	}

	data := [][]byte{
		executingSequencesData,
		common.Uint64ToBytes(this.Height),
		common.Uint32ToBytes(this.GenerationIdx),
		timeStampData,
		common.Uint64ToBytes(this.Parallelism),
		codec.Bool(this.Debug).Encode(),
	}
	return codec.Byteset(data).Encode(), nil
}

func (this *ExecutorRequest) GobDecode(data []byte) error {
	fields := codec.Byteset{}.Decode(data).(codec.Byteset)
	msgResults, err := new(ExecutingSequences).Decode(fields[0])
	if err != nil {
		return err
	}
	this.Sequences = msgResults
	this.Height = common.BytesToUint64(fields[1])
	this.GenerationIdx = common.BytesToUint32(fields[2])
	this.Timestamp = new(big.Int).SetBytes(fields[3])
	this.Parallelism = common.BytesToUint64(fields[4])
	this.Debug = bool(codec.Bool(this.Debug).Decode(fields[5]).(codec.Bool))
	return nil
}
