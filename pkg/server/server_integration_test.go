/*
 Copyright © 2025-2026 Dell Inc. or its subsidiaries. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
      http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// I-SRV-01: MetricsServer HTTP endpoint serves /metrics with registered gauge.
func TestMetricsServer_HTTP_ServesRegisteredMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_integration_gauge",
		Help: "Integration test gauge.",
	}, []string{"label"})
	reg.MustRegister(g)
	g.WithLabelValues("v1").Set(42)

	port := unusedPort(t)
	cfg := server.Config{
		Port:     fmt.Sprintf(":%d", port),
		CertFile: "",
		KeyFile:  "",
		Registry: reg,
	}

	srv := server.NewMetricsServer(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = srv.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "test_integration_gauge")
}

// I-SRV-02: /healthz returns 200 OK.
func TestMetricsServer_HTTP_HealthzOK(t *testing.T) {
	reg := prometheus.NewRegistry()
	port := unusedPort(t)
	cfg := server.Config{
		Port:     fmt.Sprintf(":%d", port),
		CertFile: "",
		KeyFile:  "",
		Registry: reg,
	}

	srv := server.NewMetricsServer(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = srv.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/healthz", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// I-SRV-03: /readyz returns 200 OK.
func TestMetricsServer_HTTP_ReadyzOK(t *testing.T) {
	reg := prometheus.NewRegistry()
	port := unusedPort(t)
	cfg := server.Config{
		Port:     fmt.Sprintf(":%d", port),
		CertFile: "",
		KeyFile:  "",
		Registry: reg,
	}

	srv := server.NewMetricsServer(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = srv.Start(ctx) }()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/readyz", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// I-SRV-04: Context cancellation triggers graceful shutdown.
func TestMetricsServer_HTTP_GracefulShutdown(t *testing.T) {
	reg := prometheus.NewRegistry()
	port := unusedPort(t)
	cfg := server.Config{
		Port:     fmt.Sprintf(":%d", port),
		CertFile: "",
		KeyFile:  "",
		Registry: reg,
	}

	srv := server.NewMetricsServer(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		_ = srv.Start(ctx)
		close(done)
	}()
	time.Sleep(80 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down within 2s after context cancellation")
	}
}
