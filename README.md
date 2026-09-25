# CSM Metrics Common

A shared Go library for Dell Container Storage Modules (CSM) that provides standardized Prometheus metrics instrumentation across CSM CSI drivers and modules.

## Overview

`csm-metrics-common` is a foundational library that implements common metrics infrastructure for Dell CSM drivers (PowerStore, PowerScale, PowerMax, PowerFlex, Unity, and COSI). It provides:

- **Standardized metric names** and label conventions
- **Production-grade resilience** features (circuit breaker, rate limiting, caching, retry)
- **HTTP metrics server** with TLS support
- **Collector framework** for managing multiple metric collectors
- **CSI operation interceptors** for gRPC-based operation tracking
- **Shared naming constants** for consistency across all CSM drivers

## Features

### 📊 Standardized Metrics
- Common CSI driver metric names (operations, duration, failures)
- Standard label keys (operation, status, protocol, error_code, etc.)
- Histogram buckets for CSI operation latency
- Module-specific metric prefixes (obs, resiliency, repl, auth, operator)

### 🛡️ Resilience Features
- **Circuit Breaker**: Three-state pattern (Closed, Open, Half-Open) to prevent cascading failures
- **Rate Limiter**: Per-endpoint throttling to prevent API overload
- **Response Caching**: Configurable TTL with stale data fallback
- **Retry Mechanism**: Exponential backoff for transient failures
- **Timeout Management**: Configurable timeouts to prevent hanging calls

### 🌐 HTTP Server
- Prometheus metrics endpoint (`/metrics`)
- Health endpoint (`/healthz`)
- Readiness endpoint (`/readyz`)
- TLS support with configurable minimum version
- Stale metric indicator for circuit breaker states

### 🔧 Collector Framework
- Dynamic collector registration and deregistration
- Lifecycle management with start/stop controls
- Panic recovery and error logging
- Timed collection with duration tracking
- Multi-collector orchestration

### 🔍 CSI Interceptor
- gRPC interceptor for CSI operations
- Operation counting and duration tracking
- Failure and error classification
- Authentication and permission denial tracking

## Installation

```bash
go get github.com/Ecosystems/container-storage-modules/src/csm-metrics-common
```

## Quick Start

### Using the Metrics Server

```go
package main

import (
    "context"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/server"
)

func main() {
    reg := prometheus.NewRegistry()
    
    cfg := server.Config{
        Port:            ":8443",
        Registry:        reg,
        CertFile:        "/path/to/cert.pem",
        KeyFile:         "/path/to/key.pem",
        StaleMetricName: "dell_powerstore_metrics_stale",
        StaleLabels:     []string{"global_id"},
    }
    
    srv := server.NewMetricsServer(cfg)
    srv.Start(context.Background())
}
```

### Using the Circuit Breaker

```go
package main

import (
    "errors"
    "time"
    "github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/middleware"
)

func main() {
    cb := middleware.NewCircuitBreaker("array-1", 3, 30*time.Second)
    
    err := cb.Call(func() error {
        // Your API call here
        return callAPI()
    })
    
    if errors.Is(err, middleware.ErrCircuitOpen) {
        // Circuit is open, use cached data or fallback
    }
}
```

### Using the Collector Manager

```go
package main

import (
    "context"
    "time"
    "github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/collector"
)

func main() {
    collectors := []collector.MetricsCollector{
        NewMyCollector("collector-1"),
        NewMyCollector("collector-2"),
    }
    
    manager := collector.NewCollectorManager(collectors, 30*time.Second)
    manager.Start(context.Background())
    
    // Later, stop the manager
    manager.Stop()
}
```

### Using Standard Metric Names

```go
package main

import "github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/naming"

func main() {
    // Use standardized metric names
    operationMetric := naming.MetricCSIOperationTotal
    durationMetric := naming.MetricCSIOperationDurationSeconds
    failureMetric := naming.MetricCSIOperationFailureTotal
    
    // Use standardized label keys
    operationLabel := naming.LabelOperation
    statusLabel := naming.LabelStatus
    protocolLabel := naming.LabelProtocol
    
    // Use standardized histogram buckets
    buckets := naming.HistogramBuckets // [0.1, 0.5, 1, 2, 5, 10, 30, 60]
}
```

## Architecture

```
csm-metrics-common/
├── pkg/
│   ├── naming/           # Standardized metric names and labels
│   ├── collector/       # Collector framework and interfaces
│   ├── server/          # HTTP metrics server
│   ├── middleware/      # Resilience features (circuit breaker, rate limiter, retry)
│   ├── cache/           # Response caching layer
│   ├── interceptor/     # CSI gRPC interceptor
│   └── module/          # Module instrumenters
```

### Component Overview

#### `pkg/naming`
Standardized constants for metric names, label keys, and histogram buckets. Ensures consistency across all CSM drivers.

#### `pkg/collector`
- `BaseCollector`: Common functionality for metrics collectors
- `CollectorManager`: Lifecycle management for multiple collectors
- `Interfaces`: PowerScale-specific collector interfaces
- Dynamic registration and deregistration

#### `pkg/server`
- `MetricsServer`: HTTP server with TLS support
- Endpoints: `/metrics`, `/healthz`, `/readyz`
- Stale metric indicator for circuit breaker integration

#### `pkg/middleware`
- `CircuitBreaker`: Three-state circuit breaker pattern
- `RateLimiter`: Per-endpoint request throttling
- `Retry`: Exponential backoff retry mechanism

#### `pkg/cache`
- Response caching with configurable TTL
- Stale data fallback support
- Cache invalidation mechanisms

#### `pkg/interceptor`
- gRPC interceptor for CSI operations
- Operation counting and duration tracking
- Failure and error classification

#### `pkg/module`
- Instrumentation hooks for CSM modules
- OpenTelemetry integration support

## Configuration

### Circuit Breaker

```go
// Create a circuit breaker with custom threshold and timeout
cb := middleware.NewCircuitBreaker(
    "array-id",           // Array identifier
    3,                    // Failure threshold
    30*time.Second,       // Reset timeout
)
```

### Rate Limiter

```go
// Create a rate limiter with custom limit
rl := middleware.NewRateLimiter(
    "endpoint",           // Endpoint identifier
    100,                  // Requests per minute
)
```

### Metrics Server

```go
cfg := server.Config{
    Port:            ":8443",
    CertFile:        "/path/to/cert.pem",
    KeyFile:         "/path/to/key.pem",
    Registry:        prometheus.NewRegistry(),
    MinTLSVersion:   tls.VersionTLS12,
    StaleMetricName: "dell_driver_metrics_stale",
    StaleLabels:     []string{"cluster_name"},
}
```

## Standard Metric Names

### CSI Driver Metrics

| Metric Name | Description |
|-------------|-------------|
| `dell_csi_operation_total` | Total CSI operations by operation type and status |
| `dell_csi_operation_duration_seconds` | CSI operation latency histogram |
| `dell_csi_operation_failure_total` | Total CSI operation failures by error code |
| `dell_csi_driver_uptime_seconds` | Driver uptime in seconds |
| `dell_csi_driver_restart_total` | Total driver restarts |
| `dell_csi_goroutine_count` | Number of active goroutines |
| `dell_csi_connection_pool_active` | Active connections to storage array |
| `dell_csi_api_call_total` | Total storage API calls by endpoint and status |
| `dell_csi_api_call_duration_seconds` | API call duration histogram |
| `dell_csi_volume_total` | Total CSI volumes by protocol |
| `dell_csi_auth_failure_total` | Total authentication failures |
| `dell_csi_permission_denial_total` | Total permission denials |
| `dell_csi_driver_cpu_usage_percent` | Driver CPU usage percentage |
| `dell_csi_driver_memory_usage_bytes` | Driver memory usage in bytes |

### Standard Label Keys

| Label Key | Description |
|-----------|-------------|
| `vendor` | Vendor identifier |
| `platform` | Platform identifier |
| `operation` | CSI operation name |
| `status` | Success/failure status |
| `error_code` | Error classification |
| `protocol` | Storage protocol (iSCSI, NFS, NVMeTCP, FC) |
| `endpoint` | API endpoint |
| `system_id` | System identifier |
| `cluster_name` | Cluster name |
| `array_id` | Array identifier |
| `global_id` | Global identifier |
| `source` | Source identifier |

## Testing

Run tests with coverage:

```bash
go test ./... -cover
```

Run specific package tests:

```bash
go test ./pkg/collector -v
go test ./pkg/middleware -v
go test ./pkg/server -v
```

## Usage in CSM Drivers

This library is used by:
- **csi-powerstore**: PowerStore CSI driver metrics
- **csi-powerscale**: PowerScale CSI driver metrics
- **csi-powermax**: PowerMax CSI driver metrics
- **csi-powerflex**: PowerFlex CSI driver metrics
- **csi-unity**: Unity CSI driver metrics
- **csi-cosi**: COSI CSI driver metrics

## Contributing

1. Follow the existing code style and conventions
2. Add tests for new features
3. Update this README for significant changes
4. Ensure all tests pass before submitting

## License

Copyright © 2026 Dell Inc. or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
