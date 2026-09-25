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

package collector_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// I-CMN-01: CollectorManager drives multiple collectors concurrently.
func TestCollectorManager_MultipleConcurrentCollectors(t *testing.T) {
	colA := &fakeCollector{name: "A"}
	colB := &fakeCollector{name: "B"}

	mgr := collector.NewManager([]collector.MetricsCollector{colA, colB}, 50*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 220*time.Millisecond)
	defer cancel()

	mgr.Start(ctx)
	<-ctx.Done()
	mgr.Stop()

	assert.GreaterOrEqual(t, colA.collectCount.Load(), int64(2), "collector A should run multiple times")
	assert.GreaterOrEqual(t, colB.collectCount.Load(), int64(2), "collector B should run multiple times")
}

// panicCollector panics on every Collect call.
type panicCollector struct{ name string }

func (p *panicCollector) Collect(_ context.Context) error        { panic("simulated collect panic") }
func (p *panicCollector) Register(_ prometheus.Registerer) error { return nil }
func (p *panicCollector) Name() string                           { return p.name }

// I-CMN-02: Panicking collector does not crash manager or stop other collectors.
func TestCollectorManager_PanicInOneCollectorDoesNotStopOthers(t *testing.T) {
	safeCol := &fakeCollector{name: "safe-col"}
	panicCol := &panicCollector{name: "panic-col"}

	mgr := collector.NewManager([]collector.MetricsCollector{panicCol, safeCol}, 40*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Millisecond)
	defer cancel()

	mgr.Start(ctx)
	<-ctx.Done()
	mgr.Stop()

	assert.GreaterOrEqual(t, safeCol.collectCount.Load(), int64(2),
		"safe collector must continue running even if another panics")
}

// I-CMN-03: Stop() terminates collection loop cleanly without leaking goroutines.
func TestCollectorManager_StopTerminatesCleanly(t *testing.T) {
	col := &fakeCollector{name: "counting-col"}

	mgr := collector.NewManager([]collector.MetricsCollector{col}, 20*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	mgr.Start(ctx)
	time.Sleep(80 * time.Millisecond)
	cancel()
	mgr.Stop()

	snapshot := col.collectCount.Load()
	time.Sleep(60 * time.Millisecond)

	assert.Equal(t, snapshot, col.collectCount.Load(),
		"no further collections should occur after Stop()")
}
