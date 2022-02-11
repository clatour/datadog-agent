// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"encoding/json"
	"net/http"

	"github.com/DataDog/datadog-agent/pkg/process/util"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

func statusHandler(w http.ResponseWriter, _ *http.Request) {
	log.Info("Got a request for the status. Making status.")

	agentStatus, err := util.GetStatus()
	if err != nil {
		if err != nil {
			_ = log.Warn("failed to get status from agent:", agentStatus)
			body, _ := json.Marshal(map[string]string{"error": err.Error()})
			http.Error(w, string(body), http.StatusInternalServerError)
		}
	}

	b, err := json.Marshal(agentStatus)
	if err != nil {
		_ = log.Warn("failed to serialize status response from agent:", err)
		body, _ := json.Marshal(map[string]string{"error": err.Error()})
		http.Error(w, string(body), http.StatusInternalServerError)
	}

	_, err = w.Write(b)
	if err != nil {
		_ = log.Warn("received response from agent but failed write it to client:", err)
	}
}