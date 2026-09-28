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

// Package naming provides standardized metric name constants for Dell CSM metrics.
package naming

// Common CSI driver metric names
const (
	MetricCSIOperationTotal           = "dell_csi_operation_total"
	MetricCSIOperationDurationSeconds = "dell_csi_operation_duration_seconds"
	MetricCSIOperationFailureTotal    = "dell_csi_operation_failure_total"
	MetricCSIDriverUptimeSeconds      = "dell_csi_driver_uptime_seconds"
	MetricCSIDriverRestartTotal       = "dell_csi_driver_restart_total"
	MetricCSIGoroutineCount           = "dell_csi_goroutine_count"
	MetricCSIConnectionPoolActive     = "dell_csi_connection_pool_active"
	MetricCSIAPICallTotal             = "dell_csi_api_call_total"
	MetricCSIAPICallDurationSeconds   = "dell_csi_api_call_duration_seconds"
	MetricCSIVolumeTotal              = "dell_csi_volume_total"
	MetricCSIAuthFailureTotal         = "dell_csi_auth_failure_total"
	MetricCSIPermissionDenialTotal    = "dell_csi_permission_denial_total"
	MetricCSIDriverCPUUsagePercent    = "dell_csi_driver_cpu_usage_percent"
	MetricCSIDriverMemoryUsageBytes   = "dell_csi_driver_memory_usage_bytes"
)

// Common label key constants
const (
	LabelVendor      = "vendor"
	LabelPlatform    = "platform"
	LabelOperation   = "operation"
	LabelStatus      = "status"
	LabelErrorCode   = "error_code"
	LabelProtocol    = "protocol"
	LabelEndpoint    = "endpoint"
	LabelSystemID    = "system_id"
	LabelClusterName = "cluster_name"
	LabelArrayID     = "array_id"
	LabelGlobalID    = "global_id"
	LabelSource      = "source"
)

// Common label value constants
const (
	LabelStatusSuccess = "success"
	LabelStatusFailure = "failure"
)

// CSM module metric name prefixes
const (
	MetricObsPrefix        = "dell_csm_obs_"
	MetricResiliencyPrefix = "dell_csm_resiliency_"
	MetricReplPrefix       = "dell_csm_repl_"
	MetricAuthPrefix       = "dell_csm_auth_"
	MetricOperatorPrefix   = "dell_csm_operator_"
)

// HistogramBuckets are the standard latency buckets for CSI operations (in seconds).
var HistogramBuckets = []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60}

// CSM Authorization label key constants
const (
	LabelTenant      = "tenant"
	LabelRole        = "role"
	LabelStorageType = "storage_type"
	LabelDecision    = "decision"
	LabelReason      = "reason"
	LabelResult      = "result"
)

// GetHistogramBuckets returns the standard histogram buckets.
func GetHistogramBuckets() []float64 {
	return HistogramBuckets
}
