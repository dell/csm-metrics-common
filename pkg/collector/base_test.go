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

package collector

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestBaseCollector(t *testing.T) {
	collector := NewBaseCollector("test-collector", "system-123", "Test collector for unit tests")

	assert.Equal(t, "test-collector", collector.Name())
	assert.Equal(t, "system-123", collector.SystemID())
	assert.Equal(t, "Test collector for unit tests", collector.description)
}

func TestBaseCollectorLabels(t *testing.T) {
	collector := NewBaseCollector("test", "system-1", "desc")

	collector.SetLabel("protocol", "iscsi")
	collector.SetLabel("array_id", "array-123")

	val, ok := collector.GetLabel("protocol")
	assert.True(t, ok)
	assert.Equal(t, "iscsi", val)

	val, ok = collector.GetLabel("array_id")
	assert.True(t, ok)
	assert.Equal(t, "array-123", val)

	_, ok = collector.GetLabel("nonexistent")
	assert.False(t, ok)
}

func TestBuildLabelValues(t *testing.T) {
	collector := NewBaseCollector("test", "system-1", "desc")

	collector.SetLabel("protocol", "iscsi")
	collector.SetLabel("array_id", "array-123")
	collector.SetLabel("zone", "zone-1")

	keys := []string{"protocol", "array_id", "zone"}
	values := collector.BuildLabelValues(keys)

	assert.Equal(t, []string{"iscsi", "array-123", "zone-1"}, values)

	// Test with missing label
	keys = []string{"protocol", "nonexistent", "array_id"}
	values = collector.BuildLabelValues(keys)
	assert.Equal(t, []string{"iscsi", "unknown", "array-123"}, values)
}

func TestSafeCollect(t *testing.T) {
	collector := NewBaseCollector("test", "system-1", "desc")

	// Test that SafeCollect handles the default error from Collect
	err := collector.SafeCollect(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")

	// Test with a logger function
	err = collector.SafeCollect(context.Background(), func(_ string, _ ...interface{}) {
		// Logger function to capture errors
	})
	assert.Error(t, err)
}

func TestMetricBuilder(t *testing.T) {
	builder := NewMetricBuilder("dell", "powerstore", []string{"system_id", "protocol"})

	assert.Equal(t, "dell_powerstore_test_metric", builder.BuildName("test_metric"))
	assert.Equal(t, []string{"system_id", "protocol"}, builder.GetLabels())

	builder.AddLabels([]string{"zone"})
	assert.Equal(t, []string{"system_id", "protocol", "zone"}, builder.GetLabels())
}

func TestMetricBuilderNoSubsystem(t *testing.T) {
	builder := NewMetricBuilder("dell", "", []string{"system_id"})

	assert.Equal(t, "dell_test_metric", builder.BuildName("test_metric"))
}

// TestCollector implements MetricsCollector for testing
type TestCollector struct {
	*BaseCollector
	called bool
}

func (t *TestCollector) Collect(_ context.Context) error {
	t.called = true
	return nil
}

func (t *TestCollector) Register(_ prometheus.Registerer) error {
	return nil
}

func TestEmbeddedBaseCollector(t *testing.T) {
	testCollector := &TestCollector{
		BaseCollector: NewBaseCollector("embedded-test", "system-1", "Embedded test"),
	}

	assert.Equal(t, "embedded-test", testCollector.Name())
	assert.Equal(t, "system-1", testCollector.SystemID())

	err := testCollector.Collect(context.Background())
	assert.NoError(t, err)
	assert.True(t, testCollector.called)
}

// TestBaseCollector_GetLabels
func TestBaseCollector_GetLabels(t *testing.T) {
	collector := NewBaseCollector("test", "system-1", "desc")

	collector.SetLabel("protocol", "iscsi")
	collector.SetLabel("array_id", "array-123")

	labels := collector.GetLabels()
	assert.Equal(t, "iscsi", labels["protocol"])
	assert.Equal(t, "array-123", labels["array_id"])
	assert.Len(t, labels, 2)
}

// TestTimedCollect
func TestTimedCollect(t *testing.T) {
	collector := NewBaseCollector("test", "system-1", "desc")

	logged := false
	err := collector.TimedCollect(context.Background(), func(_ string, _ ...interface{}) {
		logged = true
	})
	assert.Error(t, err)
	assert.True(t, logged)
}

// TestTimedCollect_NilLogger
func TestTimedCollect_NilLogger(t *testing.T) {
	collector := NewBaseCollector("test", "system-1", "desc")

	err := collector.TimedCollect(context.Background(), nil)
	assert.Error(t, err)
}

// TestSafeCollect_PanicRecovery
func TestSafeCollect_PanicRecovery(t *testing.T) {
	panicCollector := &PanicCollector{
		BaseCollector: NewBaseCollector("panic-test", "system-1", "Panics on collect"),
	}

	logged := false
	err := panicCollector.SafeCollect(context.Background(), func(_ string, _ ...interface{}) {
		logged = true
	})
	assert.Error(t, err)
	assert.True(t, logged, "logger should be called on panic")
}

// PanicCollector is a test collector that panics on Collect
type PanicCollector struct {
	*BaseCollector
}

func (p *PanicCollector) Collect(_ context.Context) error {
	panic("test panic")
}

func (p *PanicCollector) Register(_ prometheus.Registerer) error {
	return nil
}

// TestMetricBuilder_Chaining
func TestMetricBuilder_Chaining(t *testing.T) {
	builder := NewMetricBuilder("dell", "powerstore", []string{"system_id"})

	result := builder.AddLabels([]string{"protocol"}).AddLabels([]string{"zone"})
	assert.Equal(t, []string{"system_id", "protocol", "zone"}, result.GetLabels())
	assert.Equal(t, []string{"system_id", "protocol", "zone"}, builder.GetLabels())
}

// TestBaseCollector_Register
func TestBaseCollector_Register(t *testing.T) {
	collector := NewBaseCollector("test", "system-1", "desc")
	reg := prometheus.NewRegistry()

	err := collector.Register(reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}
