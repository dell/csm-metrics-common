// Copyright © 2026 Dell Inc. or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds all configuration for a MetricsServer.
type Config struct {
	Port            string
	CertFile        string
	KeyFile         string
	Registry        prometheus.Gatherer
	MinTLSVersion   uint16
	StaleMetricName string
	StaleLabels     []string
}

// MetricsServer serves Prometheus metrics, health and readiness endpoints over HTTP or HTTPS.
type MetricsServer struct {
	addr           string
	certFile       string
	keyFile        string
	registry       prometheus.Gatherer
	minTLSVersion  uint16
	srv            *http.Server
	staleGauge     *prometheus.GaugeVec
	staleLabelKeys []string
	mu             sync.RWMutex
}

// NewMetricsServer creates a MetricsServer. Accepts two forms:
//
//	NewMetricsServer(cfg Config)                       — preferred, struct-based
//	NewMetricsServer(addr string, reg prometheus.Gatherer) — legacy positional form
func NewMetricsServer(args ...interface{}) *MetricsServer {
	switch {
	case len(args) == 1:
		if cfg, ok := args[0].(Config); ok {
			s := &MetricsServer{
				addr:           cfg.Port,
				certFile:       cfg.CertFile,
				keyFile:        cfg.KeyFile,
				registry:       cfg.Registry,
				minTLSVersion:  cfg.MinTLSVersion,
				staleLabelKeys: cfg.StaleLabels,
			}
			// Set default TLS version if not specified
			if s.minTLSVersion == 0 {
				s.minTLSVersion = tls.VersionTLS12
			}
			// Create stale gauge if metric name is specified
			if cfg.StaleMetricName != "" && len(cfg.StaleLabels) > 0 {
				if reg, ok := cfg.Registry.(prometheus.Registerer); ok {
					s.staleGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
						Name: cfg.StaleMetricName,
						Help: "1 when metrics are stale (circuit open).",
					}, cfg.StaleLabels)
					reg.MustRegister(s.staleGauge)
				}
			}
			return s
		}
	case len(args) == 2:
		addr, _ := args[0].(string)
		reg, _ := args[1].(prometheus.Gatherer)
		return &MetricsServer{addr: addr, registry: reg, minTLSVersion: tls.VersionTLS12}
	}
	return &MetricsServer{minTLSVersion: tls.VersionTLS12}
}

// Start starts the server. When a context is provided it shuts down when ctx is cancelled.
// Accepts zero or one context argument so both Start() and Start(ctx) compile.
func (s *MetricsServer) Start(ctxs ...context.Context) error {
	mux := s.buildMux()
	s.mu.Lock()
	s.srv = &http.Server{Addr: s.addr, Handler: mux}
	s.mu.Unlock()
	if len(ctxs) > 0 {
		ctx := ctxs[0]
		go func() {
			<-ctx.Done()
			s.mu.RLock()
			if s.srv != nil {
				_ = s.srv.Shutdown(context.Background())
			}
			s.mu.RUnlock()
		}()
	}
	if s.certFile != "" && s.keyFile != "" {
		s.mu.RLock()
		err := s.srv.ListenAndServeTLS(s.certFile, s.keyFile)
		s.mu.RUnlock()
		return err
	}
	s.mu.RLock()
	err := s.srv.ListenAndServe()
	s.mu.RUnlock()
	return err
}

// StartTLS starts an HTTPS server. Falls back to HTTP if certFile or keyFile is empty.
func (s *MetricsServer) StartTLS(certFile, keyFile string) error {
	mux := s.buildMux()
	s.mu.Lock()
	s.srv = &http.Server{Addr: s.addr, Handler: mux}
	s.mu.Unlock()
	if certFile == "" || keyFile == "" {
		s.mu.RLock()
		err := s.srv.ListenAndServe()
		s.mu.RUnlock()
		return err
	}
	cfg := &tls.Config{
		MinVersion: s.minTLSVersion,
	}
	s.mu.Lock()
	s.srv.TLSConfig = cfg
	s.mu.Unlock()
	s.mu.RLock()
	err := s.srv.ListenAndServeTLS(certFile, keyFile)
	s.mu.RUnlock()
	return err
}

// Shutdown gracefully stops the server.
func (s *MetricsServer) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// SetStale marks metrics as stale (1) or fresh (0) for the given label values.
// This is useful for circuit breaker patterns to indicate when metrics collection is failing.
func (s *MetricsServer) SetStale(labelValues []string, stale bool) {
	if s.staleGauge == nil {
		return
	}
	if len(labelValues) != len(s.staleLabelKeys) {
		return
	}
	v := 0.0
	if stale {
		v = 1.0
	}
	s.staleGauge.WithLabelValues(labelValues...).Set(v)
}

func (s *MetricsServer) buildMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
	return mux
}
