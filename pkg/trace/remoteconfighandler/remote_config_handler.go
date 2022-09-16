// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package remoteconfighandler

import (
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state/products/apmsampling"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
)

// RemoteConfigHandler holds pointers to samplers that need to be updated when APM remote config changes
type RemoteConfigHandler struct {
	remoteClient    config.RemoteClient
	prioritySampler *sampler.PrioritySampler
	rareSampler     *sampler.RareSampler
	errorsSampler   *sampler.ErrorsSampler
}

func New(conf *config.AgentConfig, prioritySampler *sampler.PrioritySampler, rareSampler *sampler.RareSampler, errorsSampler *sampler.ErrorsSampler) *RemoteConfigHandler {
	if conf.RemoteSamplingClient == nil {
		return nil
	}

	return &RemoteConfigHandler{
		remoteClient:    conf.RemoteSamplingClient,
		prioritySampler: prioritySampler,
		rareSampler:     rareSampler,
		errorsSampler:   errorsSampler,
	}
}

func (h *RemoteConfigHandler) Start() {
	if h == nil {
		return
	}

	h.remoteClient.Start()
	h.remoteClient.RegisterAPMUpdate(h.onUpdate)
}

func (h *RemoteConfigHandler) onUpdate(update map[string]state.APMSamplingConfig) {
	h.prioritySampler.UpdateRemoteRates(update)

	for _, conf := range update {
		// We expect the `update` map to contain only one entry for now
		switch conf.Config.RareSamplerState {
		case apmsampling.RareSamplerStateEnabled:
			h.rareSampler.SetEnabled(true)
		case apmsampling.RareSamplerStateDisabled:
			h.rareSampler.SetEnabled(false)
		}
		if conf.Config.ErrorsSamplerTargetTPS != nil {
			h.errorsSampler.ScoreSampler.Sampler.UpdateTargetTPS(conf.Config.ErrorsSamplerTargetTPS.Value)
		}
	}
}
