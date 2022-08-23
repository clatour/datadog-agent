// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build (windows && npm) || linux_bpf
// +build windows,npm linux_bpf

package transaction

// Method is the type used to represent HTTP request methods
type Method int

const (
	// MethodUnknown represents an unknown request method
	MethodUnknown Method = iota
	// MethodGet represents the GET request method
	MethodGet
	// MethodPost represents the POST request method
	MethodPost
	// MethodPut represents the PUT request method
	MethodPut
	// MethodDelete represents the DELETE request method
	MethodDelete
	// MethodHead represents the HEAD request method
	MethodHead
	// MethodOptions represents the OPTIONS request method
	MethodOptions
	// MethodPatch represents the PATCH request method
	MethodPatch
)

// Method returns a string representing the HTTP method of the request
func (m Method) String() string {
	switch m {
	case MethodGet:
		return "GET"
	case MethodPost:
		return "POST"
	case MethodPut:
		return "PUT"
	case MethodHead:
		return "HEAD"
	case MethodDelete:
		return "DELETE"
	case MethodOptions:
		return "OPTIONS"
	case MethodPatch:
		return "PATCH"
	default:
		return "UNKNOWN"
	}
}

type HttpTX interface {
	ReqFragment() []byte
	StatusClass() int
	RequestLatency() float64
	isIPV4() bool
	SrcIPLow() uint64
	SrcIPHigh() uint64
	SrcPort() uint16
	DstIPLow() uint64
	DstIPHigh() uint64
	DstPort() uint16
	Method() Method
	StatusCode() uint16
	SetStatusCode(uint16)
	StaticTags() uint64
	DynamicTags() []string
	String() string
	Incomplete() bool
	Path(buffer []byte) ([]byte, bool)
	ResponseLastSeen() uint64
	SetResponseLastSeen(ls uint64)
	RequestStarted() uint64
	SetRequestMethod(uint32)
	RequestMethod() uint32
}
