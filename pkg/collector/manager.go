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
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector is implemented by every platform-specific collector.
type MetricsCollector interface {
	// Collect fetches metrics from the storage array and updates the registry.
	Collect(ctx context.Context) error
	// Register registers all Prometheus descriptors with reg.
	Register(reg prometheus.Registerer) error
	// Name returns a human-readable identifier.
	Name() string
}

// Manager manages the lifecycle of a set of MetricsCollectors.
type Manager struct {
	collectors      []MetricsCollector
	collectInterval time.Duration
	stopCh          chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
}

// NewManager creates a Manager.
func NewManager(collectors []MetricsCollector, interval time.Duration) *Manager {
	return &Manager{
		collectors:      collectors,
		collectInterval: interval,
		stopCh:          make(chan struct{}),
	}
}

// Start begins background collection for all registered collectors.
func (m *Manager) Start(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, c := range m.collectors {
		if c == nil {
			continue
		}
		m.wg.Add(1)
		go m.runCollector(ctx, c)
	}
}

// Stop signals all collector goroutines to stop and waits for them to finish.
func (m *Manager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// Register dynamically adds a collector to the manager.
func (m *Manager) Register(collector MetricsCollector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.collectors = append(m.collectors, collector)
	return nil
}

// Unregister removes a collector from the manager by name.
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.collectors {
		if c != nil && c.Name() == name {
			m.collectors = append(m.collectors[:i], m.collectors[i+1:]...)
			return
		}
	}
}

// CollectAll triggers an immediate collection from all registered collectors.
func (m *Manager) CollectAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errs []error
	for _, c := range m.collectors {
		if c == nil {
			continue
		}
		if err := c.Collect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple collectors failed: %v", errs)
	}

	return nil
}

// GetCollectors returns a list of registered collector names.
func (m *Manager) GetCollectors() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.collectors))
	for _, c := range m.collectors {
		if c != nil {
			names = append(names, c.Name())
		}
	}
	return names
}

func (m *Manager) runCollector(ctx context.Context, c MetricsCollector) {
	defer m.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("collector %s panicked: %v", c.Name(), r)
		}
	}()

	// Collect immediately on start
	collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	if err := c.Collect(collectCtx); err != nil {
		log.Printf("Initial collection failed for %s: %v", c.Name(), err)
	}
	cancel()

	ticker := time.NewTicker(m.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			_ = c.Collect(collectCtx)
			cancel()
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}
