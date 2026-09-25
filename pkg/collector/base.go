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
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// BaseCollector provides common functionality for metrics collectors.
// It implements the MetricsCollector interface and can be embedded in specific collectors.
type BaseCollector struct {
	name        string
	systemID    string
	labels      map[string]string
	description string
}

// NewBaseCollector creates a new BaseCollector with common configuration.
func NewBaseCollector(name, systemID, description string) *BaseCollector {
	return &BaseCollector{
		name:        name,
		systemID:    systemID,
		labels:      make(map[string]string),
		description: description,
	}
}

// Name returns the collector name.
func (b *BaseCollector) Name() string {
	return b.name
}

// SystemID returns the system identifier.
func (b *BaseCollector) SystemID() string {
	return b.systemID
}

// SetLabel sets a label key-value pair.
func (b *BaseCollector) SetLabel(key, value string) {
	b.labels[key] = value
}

// GetLabel returns the value for a label key.
func (b *BaseCollector) GetLabel(key string) (string, bool) {
	val, ok := b.labels[key]
	return val, ok
}

// GetLabels returns all labels as a map.
func (b *BaseCollector) GetLabels() map[string]string {
	return b.labels
}

// BuildLabelValues builds label values in the order of provided keys.
func (b *BaseCollector) BuildLabelValues(keys []string) []string {
	values := make([]string, len(keys))
	for i, key := range keys {
		if val, ok := b.labels[key]; ok {
			values[i] = val
		} else {
			values[i] = "unknown"
		}
	}
	return values
}

// Collect is a no-op implementation that should be overridden by embedded collectors.
func (b *BaseCollector) Collect(_ context.Context) error {
	return fmt.Errorf("Collect method not implemented for %s", b.name)
}

// Register is a no-op implementation that should be overridden by embedded collectors.
func (b *BaseCollector) Register(_ prometheus.Registerer) error {
	return fmt.Errorf("Register method not implemented for %s", b.name)
}

// SafeCollect wraps the Collect method with error recovery and logging.
func (b *BaseCollector) SafeCollect(ctx context.Context, logger func(format string, args ...interface{})) error {
	defer func() {
		if r := recover(); r != nil {
			if logger != nil {
				logger("collector %s panicked: %v", b.name, r)
			}
		}
	}()

	err := b.Collect(ctx)
	if err != nil && logger != nil {
		logger("collector %s collection failed: %v", b.name, err)
	}
	return err
}

// TimedCollect measures and logs the duration of a collection operation.
func (b *BaseCollector) TimedCollect(ctx context.Context, logger func(format string, args ...interface{})) error {
	start := time.Now()
	err := b.SafeCollect(ctx, logger)
	duration := time.Since(start)

	if logger != nil {
		logger("collector %s completed in %v", b.name, duration)
	}
	return err
}

// MetricBuilder helps build Prometheus metrics with consistent labeling.
type MetricBuilder struct {
	namespace string
	subsystem string
	labels    []string
}

// NewMetricBuilder creates a new MetricBuilder.
func NewMetricBuilder(namespace, subsystem string, labels []string) *MetricBuilder {
	return &MetricBuilder{
		namespace: namespace,
		subsystem: subsystem,
		labels:    labels,
	}
}

// BuildName creates a fully qualified metric name.
func (m *MetricBuilder) BuildName(name string) string {
	if m.subsystem != "" {
		return fmt.Sprintf("%s_%s_%s", m.namespace, m.subsystem, name)
	}
	return fmt.Sprintf("%s_%s", m.namespace, name)
}

// GetLabels returns the configured label keys.
func (m *MetricBuilder) GetLabels() []string {
	return m.labels
}

// AddLabels adds additional label keys.
func (m *MetricBuilder) AddLabels(additionalLabels []string) *MetricBuilder {
	m.labels = append(m.labels, additionalLabels...)
	return m
}
