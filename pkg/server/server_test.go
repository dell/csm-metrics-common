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

package server_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	l.Close()
	return addr
}

// U-CMC-05: MetricsServer.Start() serves /metrics with HTTP 200
func TestMetricsServer_Start_ServesMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	g := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_gauge", Help: "."})
	reg.MustRegister(g)
	g.Set(42)

	addr := freePort(t)
	srv := server.NewMetricsServer(addr, reg)

	go func() { _ = srv.Start() }()
	time.Sleep(50 * time.Millisecond) // allow server to start

	resp, err := http.Get("http://" + addr + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "test_gauge")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	assert.NoError(t, srv.Shutdown(ctx))
}

// U-CMC-06: MetricsServer serves /healthz and /readyz
func TestMetricsServer_HealthAndReadyEndpoints(t *testing.T) {
	reg := prometheus.NewRegistry()
	addr := freePort(t)
	srv := server.NewMetricsServer(addr, reg)

	go func() { _ = srv.Start() }()
	time.Sleep(50 * time.Millisecond)

	for _, path := range []string{"/healthz", "/readyz"} {
		resp, err := http.Get("http://" + addr + path)
		require.NoError(t, err, "path: %s", path)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "path: %s", path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// U-CMC-07: StartTLS with empty certFile falls back to plain HTTP
func TestMetricsServer_StartTLS_EmptyCertFallsBackToHTTP(t *testing.T) {
	reg := prometheus.NewRegistry()
	addr := freePort(t)
	srv := server.NewMetricsServer(addr, reg)

	go func() { _ = srv.StartTLS("", "") }()
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + addr + "/healthz")
	require.NoError(t, err, "server should fall back to plain HTTP with empty cert/key")
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// U-CMC-08: Shutdown stops the server
func TestMetricsServer_ShutdownStopsServer(t *testing.T) {
	reg := prometheus.NewRegistry()
	addr := freePort(t)
	srv := server.NewMetricsServer(addr, reg)

	go func() { _ = srv.Start() }()
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	_, dialErr := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	assert.Error(t, dialErr, "server should no longer accept connections after Shutdown")
}

// TestMetricsServer_NewMetricsServer_ConfigStruct
func TestMetricsServer_NewMetricsServer_ConfigStruct(t *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := server.Config{
		Port:            ":8080",
		CertFile:        "cert.pem",
		KeyFile:         "key.pem",
		Registry:        reg,
		MinTLSVersion:   0, // should default to TLS12
		StaleMetricName: "test_stale",
		StaleLabels:     []string{"system_id"},
	}

	srv := server.NewMetricsServer(cfg)
	assert.NotNil(t, srv)
}

// TestMetricsServer_NewMetricsServer_NoArgs
func TestMetricsServer_NewMetricsServer_NoArgs(t *testing.T) {
	srv := server.NewMetricsServer()
	assert.NotNil(t, srv)
}

// TestMetricsServer_SetStale
func TestMetricsServer_SetStale(_ *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := server.Config{
		Port:            ":8080",
		Registry:        reg,
		StaleMetricName: "test_stale",
		StaleLabels:     []string{"system_id"},
	}

	srv := server.NewMetricsServer(cfg)
	srv.SetStale([]string{"sys-1"}, true)
	srv.SetStale([]string{"sys-1"}, false)
}

// TestMetricsServer_SetStale_NilGauge
func TestMetricsServer_SetStale_NilGauge(_ *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := server.Config{
		Port:        ":8080",
		Registry:    reg,
		StaleLabels: []string{"system_id"},
	}

	srv := server.NewMetricsServer(cfg)
	srv.SetStale([]string{"sys-1"}, true) // should not panic
}

// TestMetricsServer_SetStale_MismatchedLabelCount
func TestMetricsServer_SetStale_MismatchedLabelCount(_ *testing.T) {
	reg := prometheus.NewRegistry()
	cfg := server.Config{
		Port:            ":8080",
		Registry:        reg,
		StaleMetricName: "test_stale",
		StaleLabels:     []string{"system_id"},
	}

	srv := server.NewMetricsServer(cfg)
	srv.SetStale([]string{"sys-1", "extra"}, true) // should not panic due to mismatched count
}

// TestMetricsServer_Start_WithContext
func TestMetricsServer_Start_WithContext(t *testing.T) {
	reg := prometheus.NewRegistry()
	addr := freePort(t)
	srv := server.NewMetricsServer(addr, reg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = srv.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Server should have stopped
	_, dialErr := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	assert.Error(t, dialErr)
}

// TestMetricsServer_Start_WithCertAndKey
func TestMetricsServer_Start_WithCertAndKey(t *testing.T) {
	reg := prometheus.NewRegistry()
	addr := freePort(t)
	cfg := server.Config{
		Port:     addr,
		Registry: reg,
		CertFile: "nonexistent.pem",
		KeyFile:  "nonexistent.pem",
	}
	srv := server.NewMetricsServer(cfg)

	err := srv.Start()
	assert.Error(t, err) // should fail because cert files don't exist
}

// TestMetricsServer_StartTLS_WithValidCertKey
func TestMetricsServer_StartTLS_WithValidCertKey(t *testing.T) {
	reg := prometheus.NewRegistry()
	addr := freePort(t)
	cfg := server.Config{
		Port:     addr,
		Registry: reg,
		CertFile: "nonexistent.pem",
		KeyFile:  "nonexistent.pem",
	}
	srv := server.NewMetricsServer(cfg)

	err := srv.StartTLS("nonexistent.pem", "nonexistent.pem")
	assert.Error(t, err) // should fail because cert files don't exist
}

// TestMetricsServer_Shutdown_NilServer
func TestMetricsServer_Shutdown_NilServer(t *testing.T) {
	reg := prometheus.NewRegistry()
	srv := server.NewMetricsServer(reg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)
	assert.NoError(t, err) // should not panic
}

// TestTLSConfig
func TestTLSConfig(t *testing.T) {
	// Test with empty cert and key
	cfg, err := server.TLSConfig("", "")
	assert.NoError(t, err)
	assert.Nil(t, cfg)

	// Test with only cert
	_, err = server.TLSConfig("cert.pem", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both cert-file and key-file must be provided")

	// Test with only key
	_, err = server.TLSConfig("", "key.pem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both cert-file and key-file must be provided")

	// Test with non-existent files
	_, err = server.TLSConfig("nonexistent.pem", "nonexistent.pem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading TLS key pair")
}

// TestIsTLSEnabled
func TestIsTLSEnabled(t *testing.T) {
	// Test with empty strings
	assert.False(t, server.IsTLSEnabled("", ""))

	// Test with only cert
	assert.False(t, server.IsTLSEnabled("cert.pem", ""))

	// Test with only key
	assert.False(t, server.IsTLSEnabled("", "key.pem"))

	// Test with both
	assert.True(t, server.IsTLSEnabled("cert.pem", "key.pem"))
}
