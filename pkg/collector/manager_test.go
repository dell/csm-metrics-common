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

package collector_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// fakeCollector counts how many times Collect is called.
type fakeCollector struct {
	name         string
	collectCount atomic.Int64
	returnErr    error
}

func (f *fakeCollector) Collect(_ context.Context) error {
	f.collectCount.Add(1)
	return f.returnErr
}

func (f *fakeCollector) Register(_ prometheus.Registerer) error { return nil }
func (f *fakeCollector) Name() string                           { return f.name }

// U-CMC-01: Manager.Start() calls Collect() each interval
func TestManager_Start_CallsCollectEachInterval(t *testing.T) {
	fc := &fakeCollector{name: "test"}
	mgr := collector.NewManager([]collector.MetricsCollector{fc}, 20*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mgr.Start(ctx)
	<-ctx.Done()
	mgr.Stop()

	count := fc.collectCount.Load()
	assert.GreaterOrEqual(t, count, int64(2), "Collect should have been called at least twice in 100ms with 20ms interval")
}

// U-CMC-02: nil collector in slice is skipped
func TestManager_Start_NilCollectorSkipped(t *testing.T) {
	fc := &fakeCollector{name: "real"}
	mgr := collector.NewManager([]collector.MetricsCollector{nil, fc}, 20*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	// Should not panic
	mgr.Start(ctx)
	<-ctx.Done()
	mgr.Stop()

	assert.GreaterOrEqual(t, fc.collectCount.Load(), int64(1))
}

// U-CMC-03: Empty collector slice — no goroutines, no panic
func TestManager_Start_EmptySlice(t *testing.T) {
	mgr := collector.NewManager([]collector.MetricsCollector{}, 20*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	mgr.Start(ctx)
	<-ctx.Done()
	// Stop should return immediately (no goroutines)
	done := make(chan struct{})
	go func() { mgr.Stop(); close(done) }()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Stop() did not return in time for empty collector slice")
	}
}

// U-CMC-04: Panic in Collect() is recovered; ticker continues
func TestManager_Start_PanicInCollectRecovered(t *testing.T) {
	panicCollector := &panicOnFirstCollect{}
	normalCollector := &fakeCollector{name: "normal"}

	mgr := collector.NewManager(
		[]collector.MetricsCollector{panicCollector, normalCollector},
		20*time.Millisecond,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mgr.Start(ctx)
	<-ctx.Done()
	mgr.Stop()

	// normalCollector should have been invoked (not affected by panic in other goroutine)
	assert.GreaterOrEqual(t, normalCollector.collectCount.Load(), int64(1))
}

// U-CMC-04 helper
type panicOnFirstCollect struct {
	called atomic.Bool
}

func (p *panicOnFirstCollect) Collect(_ context.Context) error {
	if !p.called.Swap(true) {
		panic("simulated panic in Collect")
	}
	return nil
}
func (p *panicOnFirstCollect) Register(_ prometheus.Registerer) error { return nil }
func (p *panicOnFirstCollect) Name() string                           { return "panicky" }

// TestManager_Stop_TerminatesGoroutines
func TestManager_Stop_TerminatesGoroutines(t *testing.T) {
	fc := &fakeCollector{name: "test", returnErr: errors.New("transient error")}
	mgr := collector.NewManager([]collector.MetricsCollector{fc}, 10*time.Millisecond)

	ctx := context.Background()
	mgr.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() { mgr.Stop(); close(done) }()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Stop() did not return within timeout")
	}
}

// TestManager_Register
func TestManager_Register(t *testing.T) {
	manager := collector.NewManager([]collector.MetricsCollector{}, 1*time.Second)

	fake := &fakeCollector{name: "test-collector"}
	err := manager.Register(fake)
	assert.NoError(t, err)
}

// TestManager_Unregister
func TestManager_Unregister(t *testing.T) {
	manager := collector.NewManager([]collector.MetricsCollector{}, 1*time.Second)

	fake := &fakeCollector{name: "test-collector"}
	manager.Register(fake)

	manager.Unregister(fake.Name())

	collectors := manager.GetCollectors()
	assert.Len(t, collectors, 0)
}

// TestManager_CollectAll
func TestManager_CollectAll(t *testing.T) {
	manager := collector.NewManager([]collector.MetricsCollector{}, 1*time.Second)

	fake := &fakeCollector{name: "test-collector"}
	manager.Register(fake)

	err := manager.CollectAll(context.Background())
	assert.NoError(t, err)
}

// TestManager_GetCollectors
func TestManager_GetCollectors(t *testing.T) {
	manager := collector.NewManager([]collector.MetricsCollector{}, 1*time.Second)

	fake := &fakeCollector{name: "test-collector"}
	manager.Register(fake)

	collectors := manager.GetCollectors()
	assert.Len(t, collectors, 1)
	assert.Equal(t, "test-collector", collectors[0])
}
