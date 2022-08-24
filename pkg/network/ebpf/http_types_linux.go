// Code generated by cmd/cgo -godefs; DO NOT EDIT.
// cgo -godefs -- -fsigned-char http_types.go

package ebpf

type HTTPConnTuple struct {
	Saddr_h  uint64
	Saddr_l  uint64
	Daddr_h  uint64
	Daddr_l  uint64
	Sport    uint16
	Dport    uint16
	Netns    uint32
	Pid      uint32
	Metadata uint32
}
type HTTPBatchState struct {
	Scratch_tx    EbpfHttpTx
	Idx           uint64
	Pos           uint8
	Idx_to_notify uint64
}
type SSLSock struct {
	Tup       HTTPConnTuple
	Fd        uint32
	Pad_cgo_0 [4]byte
}
type SSLReadArgs struct {
	Ctx *byte
	Buf *byte
}
type EbpfHttpTx struct {
	Tup                  HTTPConnTuple
	Request_started      uint64
	Request_method       uint8
	Response_status_code uint16
	Response_last_seen   uint64
	Request_fragment     [160]int8
	Owned_by_src_port    uint16
	Tcp_seq              uint32
	Tags                 uint64
}

type HttpNotification struct {
	Cpu uint32
	Idx uint64
}
type HttpBatch struct {
	Idx uint64
	Pos uint8
	Txs [15]EbpfHttpTx
}
type HttpBatchKey struct {
	Cpu uint32
	Num uint32
}

const (
	HTTPBatchSize  = 0xf
	HTTPBufferSize = 0xa0
	HTTPBatchPages = 0xf
)
