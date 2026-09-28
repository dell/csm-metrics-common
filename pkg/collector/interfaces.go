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

	"github.com/prometheus/client_golang/prometheus"
)

// PowerScaleClusterCollector collects metrics from a single PowerScale cluster.
type PowerScaleClusterCollector interface {
	MetricsCollector
	// ClusterName returns the name of the cluster this collector targets.
	ClusterName() string
	// SetRegistry replaces the Prometheus registry used for metrics.
	SetRegistry(reg prometheus.Registerer)
}

// PowerScaleCollectorManager manages the lifecycle of multiple
// PowerScaleClusterCollectors across all configured clusters.
type PowerScaleCollectorManager interface {
	// AddCluster registers a new cluster collector.
	AddCluster(c PowerScaleClusterCollector)
	// RemoveCluster deregisters and stops a cluster collector by name.
	RemoveCluster(clusterName string)
	// StartAll starts collection for all registered clusters.
	StartAll(ctx context.Context)
	// StopAll stops all running cluster collectors.
	StopAll()
}
