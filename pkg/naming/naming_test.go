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

package naming_test

import (
	"strings"
	"testing"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/naming"
	"github.com/stretchr/testify/assert"
)

// U-CMC-29: All naming constants match spec values
func TestNaming_MetricConstantsMatchSpec(t *testing.T) {
	assert.Equal(t, "dell_csi_operation_total", naming.MetricCSIOperationTotal)
	assert.Equal(t, "dell_csi_operation_duration_seconds", naming.MetricCSIOperationDurationSeconds)
	assert.Equal(t, "dell_csi_operation_failure_total", naming.MetricCSIOperationFailureTotal)
	assert.Equal(t, "dell_csi_driver_uptime_seconds", naming.MetricCSIDriverUptimeSeconds)
	assert.Equal(t, "dell_csi_driver_restart_total", naming.MetricCSIDriverRestartTotal)
	assert.Equal(t, "dell_csi_goroutine_count", naming.MetricCSIGoroutineCount)
	assert.Equal(t, "dell_csi_connection_pool_active", naming.MetricCSIConnectionPoolActive)
	assert.Equal(t, "dell_csi_api_call_total", naming.MetricCSIAPICallTotal)
	assert.Equal(t, "dell_csi_api_call_duration_seconds", naming.MetricCSIAPICallDurationSeconds)
	assert.Equal(t, "dell_csi_volume_total", naming.MetricCSIVolumeTotal)
	assert.Equal(t, "dell_csi_auth_failure_total", naming.MetricCSIAuthFailureTotal)
	assert.Equal(t, "dell_csi_permission_denial_total", naming.MetricCSIPermissionDenialTotal)
	assert.Equal(t, "dell_csi_driver_cpu_usage_percent", naming.MetricCSIDriverCPUUsagePercent)
	assert.Equal(t, "dell_csi_driver_memory_usage_bytes", naming.MetricCSIDriverMemoryUsageBytes)
}

// U-CMC-29: Label key constants
func TestNaming_LabelKeyConstants(t *testing.T) {
	assert.Equal(t, "vendor", naming.LabelVendor)
	assert.Equal(t, "platform", naming.LabelPlatform)
	assert.Equal(t, "operation", naming.LabelOperation)
	assert.Equal(t, "status", naming.LabelStatus)
	assert.Equal(t, "error_code", naming.LabelErrorCode)
	assert.Equal(t, "protocol", naming.LabelProtocol)
	assert.Equal(t, "endpoint", naming.LabelEndpoint)
	assert.Equal(t, "system_id", naming.LabelSystemID)
	assert.Equal(t, "cluster_name", naming.LabelClusterName)
	assert.Equal(t, "array_id", naming.LabelArrayID)
	assert.Equal(t, "global_id", naming.LabelGlobalID)
	assert.Equal(t, "source", naming.LabelSource)
}

// U-CMC-30: HistogramBuckets slice matches spec
func TestNaming_HistogramBuckets(t *testing.T) {
	expected := []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60}
	assert.Equal(t, expected, naming.HistogramBuckets,
		"HistogramBuckets must match spec: [0.1, 0.5, 1, 2, 5, 10, 30, 60]")

	// Test the getter function
	buckets := naming.GetHistogramBuckets()
	assert.Equal(t, expected, buckets)
}

// TestNaming_ModulePrefixes
func TestNaming_ModulePrefixes(t *testing.T) {
	assert.Equal(t, "dell_csm_obs_", naming.MetricObsPrefix)
	assert.Equal(t, "dell_csm_resiliency_", naming.MetricResiliencyPrefix)
	assert.Equal(t, "dell_csm_repl_", naming.MetricReplPrefix)
	assert.Equal(t, "dell_csm_auth_", naming.MetricAuthPrefix)
	assert.Equal(t, "dell_csm_operator_", naming.MetricOperatorPrefix)
}

// TestNaming_LabelValueConstants tests label value constants
func TestNaming_LabelValueConstants(t *testing.T) {
	assert.Equal(t, "success", naming.LabelStatusSuccess)
	assert.Equal(t, "failure", naming.LabelStatusFailure)
}

func TestNaming_AuthMetricConstantsUseAuthPrefix(t *testing.T) {
	metrics := []string{
		naming.MetricAuthPrefix + "request_total",
		naming.MetricAuthPrefix + "request_duration_seconds",
		naming.MetricAuthPrefix + "token_validation_total",
		naming.MetricAuthPrefix + "role_access_total",
		naming.MetricAuthPrefix + "credential_shield_total",
		naming.MetricAuthPrefix + "quota_check_total",
	}

	for _, metricName := range metrics {
		assert.True(t, strings.HasPrefix(metricName, naming.MetricAuthPrefix))
	}
}

func TestNaming_AuthLabelConstants(t *testing.T) {
	assert.Equal(t, "tenant", naming.LabelTenant)
	assert.Equal(t, "role", naming.LabelRole)
	assert.Equal(t, "storage_type", naming.LabelStorageType)
	assert.Equal(t, "decision", naming.LabelDecision)
	assert.Equal(t, "reason", naming.LabelReason)
	assert.Equal(t, "result", naming.LabelResult)
}
