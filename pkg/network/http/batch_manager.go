// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf
// +build linux_bpf

package http

import (
	"unsafe"

	"fmt"

	netebpf "github.com/DataDog/datadog-agent/pkg/network/ebpf"
	"github.com/DataDog/datadog-agent/pkg/network/http/transaction"
	"github.com/cilium/ebpf"
)

func toHTTPNotification(data []byte) netebpf.HttpNotification {
	return *(*netebpf.HttpNotification)(unsafe.Pointer(&data[0]))
}

type httpBatchKey netebpf.HttpBatchKey
type httpBatch netebpf.HttpBatch

// Prepare the httpBatchKey for a map lookup
func (k *httpBatchKey) Prepare(n netebpf.HttpNotification) {
	k.Cpu = n.Cpu
	k.Num = uint32(int(n.Idx) % netebpf.HTTPBatchPages)
}

const maxLookupsPerCPU = 2

type usrBatchState struct {
	idx, pos int
}

type batchManager struct {
	batchMap   *ebpf.Map
	stateByCPU []usrBatchState
	numCPUs    int
}

func newBatchManager(batchMap, batchStateMap *ebpf.Map, numCPUs int) *batchManager {
	batch := new(netebpf.HttpBatch)
	state := new(netebpf.HTTPBatchState)
	stateByCPU := make([]usrBatchState, numCPUs)

	for i := 0; i < numCPUs; i++ {
		// Initialize eBPF maps
		batchStateMap.Put(unsafe.Pointer(&i), unsafe.Pointer(state))
		for j := 0; j < netebpf.HTTPBatchPages; j++ {
			key := &httpBatchKey{Cpu: uint32(i), Num: uint32(j)}
			batchMap.Put(unsafe.Pointer(key), unsafe.Pointer(batch))
		}
	}

	return &batchManager{
		batchMap:   batchMap,
		stateByCPU: stateByCPU,
		numCPUs:    numCPUs,
	}
}

func (m *batchManager) GetTransactionsFrom(notification netebpf.HttpNotification) ([]transaction.HttpTX, error) {
	var (
		state    = &m.stateByCPU[notification.Cpu]
		batch    = new(httpBatch)
		batchKey = new(httpBatchKey)
	)

	batchKey.Prepare(notification)
	err := m.batchMap.Lookup(unsafe.Pointer(batchKey), unsafe.Pointer(batch))
	if err != nil {
		return nil, fmt.Errorf("error retrieving http batch for cpu=%d", notification.Cpu)
	}

	if int(batch.Idx) < state.idx {
		// This means this batch was processed via GetPendingTransactions
		return nil, nil
	}

	if batch.IsDirty(notification) {
		// This means the batch was overridden before we a got chance to read it
		return nil, transaction.ErrLostBatch
	}

	offset := state.pos
	state.idx = int(notification.Idx) + 1
	state.pos = 0

	txns := make([]transaction.HttpTX, len(batch.Transactions()[offset:]))
	tocopy := batch.Transactions()[offset:]
	for idx := range tocopy {
		txns[idx] = &tocopy[idx]
	}
	return txns, nil
}

func (m *batchManager) GetPendingTransactions() []transaction.HttpTX {
	transactions := make([]transaction.HttpTX, 0, netebpf.HTTPBatchSize*netebpf.HTTPBatchPages/2)
	for i := 0; i < m.numCPUs; i++ {
		for lookup := 0; lookup < maxLookupsPerCPU; lookup++ {
			var (
				usrState = &m.stateByCPU[i]
				pageNum  = usrState.idx % netebpf.HTTPBatchPages
				batchKey = &httpBatchKey{Cpu: uint32(i), Num: uint32(pageNum)}
				batch    = new(httpBatch)
			)

			err := m.batchMap.Lookup(unsafe.Pointer(batchKey), unsafe.Pointer(batch))
			if err != nil {
				break
			}

			krnStateIDX := int(batch.Idx)
			krnStatePos := int(batch.Pos)
			if krnStateIDX != usrState.idx || krnStatePos <= usrState.pos {
				break
			}

			all := batch.Transactions()
			pending := all[usrState.pos:krnStatePos]
			for _, tx := range pending {
				var newtx = tx
				transactions = append(transactions, &newtx)
			}

			if krnStatePos == netebpf.HTTPBatchSize {
				// We detected a full batch before the http_notification_t was processed.
				// In this case we update the userspace state accordingly and try to
				// preemptively read the next batch in order to return as many
				// completed HTTP transactions as possible
				usrState.idx++
				usrState.pos = 0
				continue
			}

			usrState.pos = krnStatePos
			// Move on to the next CPU core
			break
		}
	}

	return transactions
}

// IsDirty detects whether the batch page we're supposed to read from is still
// valid.  A "dirty" page here means that between the time the
// http_notification_t message was sent to userspace and the time we performed
// the batch lookup the page was overridden.
func (batch *httpBatch) IsDirty(notification netebpf.HttpNotification) bool {
	return batch.Idx != notification.Idx
}

// Transactions returns the slice of HTTP transactions embedded in the batch
func (batch *httpBatch) Transactions() []transaction.EbpfHttpTx {
	return (*(*[netebpf.HTTPBatchSize]transaction.EbpfHttpTx)(unsafe.Pointer(&batch.Txs)))[:]
}
