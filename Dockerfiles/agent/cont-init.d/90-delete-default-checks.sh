#!/bin/bash

# Delete default checks if the user disables them
if [[ -z "${DD_DISABLE_DEFAULT_CHECKS}" ]]; then
    exit 0
fi

find /etc/datadog-agent/conf.d/ -iname *.yaml.default -delete;
