// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build ignore
// +build ignore

package ebpf

/*
#include "./c/tracer.h"
#include "./c/http-types.h"
*/
import "C"

type HTTPConnTuple C.conn_tuple_t
type HTTPBatchState C.http_batch_state_t
type SSLSock C.ssl_sock_t
type SSLReadArgs C.ssl_read_args_t
type EbpfHttpTx C.http_transaction_t

type HttpNotification C.http_batch_notification_t
type HttpBatch C.http_batch_t
type HttpBatchKey C.http_batch_key_t

type LibPath = C.lib_path_t

const (
	HTTPBatchSize  = C.HTTP_BATCH_SIZE
	HTTPBufferSize = C.HTTP_BUFFER_SIZE
	HTTPBatchPages = C.HTTP_BATCH_PAGES

	PathMaxSize = C.LIB_PATH_MAX_SIZE
)
