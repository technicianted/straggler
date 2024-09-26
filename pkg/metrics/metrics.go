// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package metrics

import (
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	Namespace = "stagger"

	ErrorLabel = "error"
)

// StartMetricsExposer starts prometheus metrics exposer
func StartMetricsExposer(address string, logger logr.Logger) {
	path := "/metrics"
	index := strings.Index(address, "/")
	if index != -1 {
		path = address[index:]
		address = address[0:index]
	}
	http.Handle(path, promhttp.Handler())
	go func() {
		logger.V(1).Info("starting prometheus exposer", "listen", address)
		err := http.ListenAndServe(address, nil)
		logger.V(1).Info("prometheus metrics exposer terminated", "error", err)
	}()
}
