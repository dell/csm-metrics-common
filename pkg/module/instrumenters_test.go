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

package module_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/module"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gatherMetric(t *testing.T, reg prometheus.Gatherer, name string) *dto.MetricFamily {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == name {
			return mf
		}
	}
	return nil
}

func gaugeValue(mf *dto.MetricFamily, labels map[string]string) float64 {
	for _, m := range mf.GetMetric() {
		if labelsMatch(m, labels) {
			return m.GetGauge().GetValue()
		}
	}
	return -1
}

func counterValue(mf *dto.MetricFamily, labels map[string]string) float64 {
	for _, m := range mf.GetMetric() {
		if labelsMatch(m, labels) {
			return m.GetCounter().GetValue()
		}
	}
	return -1
}

func labelsMatch(m *dto.Metric, want map[string]string) bool {
	got := make(map[string]string)
	for _, lp := range m.GetLabel() {
		got[lp.GetName()] = lp.GetValue()
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

// U-CMC-23: ObsInstrumenter.RecordCollectionRate updates gauge
func TestObsInstrumenter_RecordCollectionRate(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewObsInstrumenter(reg, "observability", "system_id")

	ins.RecordCollectionRate("observability", "system-1", 3.5)

	mf := gatherMetric(t, reg, "dell_csm_obs_collection_rate")
	require.NotNil(t, mf, "dell_csm_obs_collection_rate should be registered")

	v := gaugeValue(mf, map[string]string{"module": "observability", "system_id": "system-1"})
	assert.InDelta(t, 3.5, v, 0.001)
}

// U-CMC-24: ObsInstrumenter metric HELP text documents the exported contract
func TestObsInstrumenter_HelpText(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewObsInstrumenter(reg, "observability", "system_id")
	ins.RecordCollectionRate("observability", "system-1", 1.0)
	ins.RecordExportSuccess("observability", "system-1", "success")
	ins.RecordProcessingLatency("observability", "system-1", 2*time.Second)

	mf := gatherMetric(t, reg, "dell_csm_obs_collection_rate")
	require.NotNil(t, mf)
	assert.Equal(t, "Rate of metrics collected per second.", mf.GetHelp())

	mf = gatherMetric(t, reg, "dell_csm_obs_export_success_total")
	require.NotNil(t, mf)
	assert.Equal(t, "Total metric export attempts by status.", mf.GetHelp())

	mf = gatherMetric(t, reg, "dell_csm_obs_processing_latency_seconds")
	require.NotNil(t, mf)
	assert.Equal(t, "Latency of processing and export after metric collection, in seconds.", mf.GetHelp())
}

// U-CMC-25: ObsInstrumenter.RecordArrayConnectivity(false) sets gauge to 0
func TestObsInstrumenter_RecordArrayConnectivity_Disconnected(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewObsInstrumenter(reg, "observability", "system_id")

	ins.RecordArrayConnectivity("observability", "system-1", false)

	mf := gatherMetric(t, reg, "dell_csm_obs_array_connectivity")
	require.NotNil(t, mf)
	v := gaugeValue(mf, map[string]string{"module": "observability", "system_id": "system-1"})
	assert.Equal(t, 0.0, v, "disconnected array should report 0")
}

// U-CMC-25: ResiliencyInstrumenter.RecordFailover increments counter and observes duration
func TestResiliencyInstrumenter_RecordFailover(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewResiliencyInstrumenter(reg)

	ins.RecordFailover("resiliency", "success", 2*time.Second)

	mf := gatherMetric(t, reg, "dell_csm_resiliency_failover_total")
	require.NotNil(t, mf)
	v := counterValue(mf, map[string]string{"module": "resiliency", "status": "success"})
	assert.Equal(t, 1.0, v)

	mfDur := gatherMetric(t, reg, "dell_csm_resiliency_failover_duration_seconds")
	require.NotNil(t, mfDur)
}

// U-CMC-26: ReplicationInstrumenter.SetLag updates gauge
func TestReplicationInstrumenter_SetLag(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewReplicationInstrumenter(reg)

	ins.SetLag("replication", "rcg-1", 5.0)

	mf := gatherMetric(t, reg, "dell_csm_repl_lag_seconds")
	require.NotNil(t, mf)
	v := gaugeValue(mf, map[string]string{"module": "replication", "rcg_id": "rcg-1"})
	assert.InDelta(t, 5.0, v, 0.001)
}

// U-CMC-27: AuthInstrumenter.RecordRequest increments request counter and records duration
func TestAuthInstrumenter_RecordRequest(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRequest("powerflex", "tenant-a", "success", 0.5)
	ins.RecordRequest("powerflex", "tenant-a", "success", 0.75)

	mf := gatherMetric(t, reg, "dell_csm_auth_request_total")
	require.NotNil(t, mf, "dell_csm_auth_request_total should be registered")
	v := counterValue(mf, map[string]string{"storage_type": "powerflex", "tenant": "tenant-a", "status": "success"})
	assert.Equal(t, 2.0, v)

	mfDur := gatherMetric(t, reg, "dell_csm_auth_request_duration_seconds")
	require.NotNil(t, mfDur, "dell_csm_auth_request_duration_seconds should be registered")
}

// U-CMC-28: OperatorInstrumenter.RecordReconciliation updates counter and histogram
func TestOperatorInstrumenter_RecordReconciliation(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewOperatorInstrumenter(reg)

	ins.RecordReconciliation("powerflex", "metrics", "success", 500*time.Millisecond)

	mf := gatherMetric(t, reg, "dell_csm_operator_reconciliation_total")
	require.NotNil(t, mf)
	v := counterValue(mf, map[string]string{"driver": "powerflex", "module": "metrics", "status": "success"})
	assert.Equal(t, 1.0, v)

	mfDur := gatherMetric(t, reg, "dell_csm_operator_reconciliation_duration_seconds")
	require.NotNil(t, mfDur)
}

// TestObsInstrumenter_AllMetricsRegistered
func TestObsInstrumenter_AllMetricsRegistered(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewObsInstrumenter(reg, "observability", "system_id")

	// Seed one observation per metric so Gather emits all families.
	ins.RecordCollectionRate("observability", "sys-1", 1)
	ins.RecordExportSuccess("observability", "sys-1", "success")
	ins.RecordArrayConnectivity("observability", "sys-1", true)
	ins.RecordProcessingLatency("observability", "sys-1", time.Millisecond)

	mfs, err := reg.Gather()
	require.NoError(t, err)

	names := make([]string, 0, len(mfs))
	for _, mf := range mfs {
		names = append(names, mf.GetName())
	}

	expected := []string{
		"dell_csm_obs_collection_rate",
		"dell_csm_obs_export_success_total",
		"dell_csm_obs_array_connectivity",
		"dell_csm_obs_processing_latency_seconds",
	}
	for _, exp := range expected {
		found := false
		for _, n := range names {
			if strings.Contains(n, exp) {
				found = true
				break
			}
		}
		assert.True(t, found, "expected metric %q to be registered", exp)
	}
}

// TestObsInstrumenter_RecordExportSuccess
func TestObsInstrumenter_RecordExportSuccess(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewObsInstrumenter(reg, "observability", "system_id")

	ins.RecordExportSuccess("observability", "system-1", "success")
	ins.RecordExportSuccess("observability", "system-1", "success")

	mf := gatherMetric(t, reg, "dell_csm_obs_export_success_total")
	require.NotNil(t, mf)
	v := counterValue(mf, map[string]string{"module": "observability", "system_id": "system-1", "status": "success"})
	assert.Equal(t, 2.0, v)
}

// TestObsInstrumenter_RecordProcessingLatency
func TestObsInstrumenter_RecordProcessingLatency(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewObsInstrumenter(reg, "observability", "system_id")

	ins.RecordProcessingLatency("observability", "system-1", 100*time.Millisecond)

	mf := gatherMetric(t, reg, "dell_csm_obs_processing_latency_seconds")
	require.NotNil(t, mf)
}

// TestResiliencyInstrumenter_SetPodmonHealth
func TestResiliencyInstrumenter_SetPodmonHealth(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewResiliencyInstrumenter(reg)

	ins.SetPodmonHealth("resiliency", "node-1", true)

	mf := gatherMetric(t, reg, "dell_csm_resiliency_podmon_healthy")
	require.NotNil(t, mf)

	ins.SetPodmonHealth("resiliency", "node-1", false)
}

// TestResiliencyInstrumenter_RecordConnectivityCheck
func TestResiliencyInstrumenter_RecordConnectivityCheck(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewResiliencyInstrumenter(reg)

	ins.RecordConnectivityCheck("resiliency", "iscsi")
	ins.RecordConnectivityCheck("resiliency", "iscsi")

	mf := gatherMetric(t, reg, "dell_csm_resiliency_connectivity_check_total")
	require.NotNil(t, mf)
	v := counterValue(mf, map[string]string{"module": "resiliency", "protocol": "iscsi"})
	assert.Equal(t, 2.0, v)
}

// TestResiliencyInstrumenter_SetConnectivitySuccessRatio
func TestResiliencyInstrumenter_SetConnectivitySuccessRatio(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewResiliencyInstrumenter(reg)

	ins.SetConnectivitySuccessRatio("resiliency", "iscsi", 0.95)

	mf := gatherMetric(t, reg, "dell_csm_resiliency_connectivity_success_ratio")
	require.NotNil(t, mf)
	v := gaugeValue(mf, map[string]string{"module": "resiliency", "protocol": "iscsi"})
	assert.InDelta(t, 0.95, v, 0.001)
}

// TestReplicationInstrumenter_SetPairStatus
func TestReplicationInstrumenter_SetPairStatus(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewReplicationInstrumenter(reg)

	ins.SetPairStatus("replication", "rcg-1", "in_sync", 1.0)

	mf := gatherMetric(t, reg, "dell_csm_repl_pair_status")
	require.NotNil(t, mf)
	v := gaugeValue(mf, map[string]string{"module": "replication", "rcg_id": "rcg-1", "state": "in_sync"})
	assert.Equal(t, 1.0, v)
}

// TestReplicationInstrumenter_SetBandwidth
func TestReplicationInstrumenter_SetBandwidth(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewReplicationInstrumenter(reg)

	ins.SetBandwidth("replication", "rcg-1", 1000000.0)

	mf := gatherMetric(t, reg, "dell_csm_repl_bandwidth_bytes")
	require.NotNil(t, mf)
	v := gaugeValue(mf, map[string]string{"module": "replication", "rcg_id": "rcg-1"})
	assert.InDelta(t, 1000000.0, v, 0.001)
}

// TestReplicationInstrumenter_RecordOperation
func TestReplicationInstrumenter_RecordOperation(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewReplicationInstrumenter(reg)

	ins.RecordOperation("replication", "create", "success")

	mf := gatherMetric(t, reg, "dell_csm_repl_operation_total")
	require.NotNil(t, mf)
	v := counterValue(mf, map[string]string{"module": "replication", "operation": "create", "status": "success"})
	assert.Equal(t, 1.0, v)
}

// TestAuthInstrumenter_RecordTokenValidation increments the token validation counter
func TestAuthInstrumenter_RecordTokenValidation(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordTokenValidation("success", "ok")
	ins.RecordTokenValidation("failure", "invalid_token")

	mf := gatherMetric(t, reg, "dell_csm_auth_token_validation_total")
	require.NotNil(t, mf)
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"result": "success", "reason": "ok"}))
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"result": "failure", "reason": "invalid_token"}))
}

// TestAuthInstrumenter_RecordRoleAccess increments the role access counter
func TestAuthInstrumenter_RecordRoleAccess(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRoleAccess("admin", "powerflex", "allow")
	ins.RecordRoleAccess("admin", "powerflex", "allow")

	mf := gatherMetric(t, reg, "dell_csm_auth_role_access_total")
	require.NotNil(t, mf)
	v := counterValue(mf, map[string]string{"role": "admin", "storage_type": "powerflex", "decision": "allow"})
	assert.Equal(t, 2.0, v)
}

// TestAuthInstrumenter_RecordCredentialShield increments the credential shield counter
func TestAuthInstrumenter_RecordCredentialShield(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordCredentialShield("powerstore", "success")
	ins.RecordCredentialShield("powerstore", "failure")

	mf := gatherMetric(t, reg, "dell_csm_auth_credential_shield_total")
	require.NotNil(t, mf)
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"storage_type": "powerstore", "status": "success"}))
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"storage_type": "powerstore", "status": "failure"}))
}

// TestAuthInstrumenter_RecordQuotaCheck increments the quota check counter
func TestAuthInstrumenter_RecordQuotaCheck(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordQuotaCheck("powerflex", "tenant-a", "allowed")
	ins.RecordQuotaCheck("powerflex", "tenant-a", "denied")

	mf := gatherMetric(t, reg, "dell_csm_auth_quota_check_total")
	require.NotNil(t, mf)
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"storage_type": "powerflex", "tenant": "tenant-a", "result": "allowed"}))
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"storage_type": "powerflex", "tenant": "tenant-a", "result": "denied"}))
}

// TestAuthInstrumenter_SanitizesEmptyLabels verifies empty strings are replaced with "unknown"
func TestAuthInstrumenter_SanitizesEmptyLabels(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRequest("", "", "", 0.5)
	ins.RecordTokenValidation("", "")
	ins.RecordRoleAccess("", "", "")
	ins.RecordCredentialShield("", "")
	ins.RecordQuotaCheck("", "", "")

	mf := gatherMetric(t, reg, "dell_csm_auth_request_total")
	require.NotNil(t, mf)
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"storage_type": "unknown", "tenant": "unknown", "status": "unknown"}))

	mfTV := gatherMetric(t, reg, "dell_csm_auth_token_validation_total")
	require.NotNil(t, mfTV)
	assert.Equal(t, 1.0, counterValue(mfTV, map[string]string{"result": "unknown", "reason": "unknown"}))
}

// TestAuthInstrumenter_RequestDurationSkippedWhenZero verifies histogram is not observed for zero duration
func TestAuthInstrumenter_RequestDurationSkippedWhenZero(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRequest("powerflex", "tenant-a", "failure", 0)

	mf := gatherMetric(t, reg, "dell_csm_auth_request_total")
	require.NotNil(t, mf)
	v := counterValue(mf, map[string]string{"storage_type": "powerflex", "tenant": "tenant-a", "status": "failure"})
	assert.Equal(t, 1.0, v)

	mfDur := gatherMetric(t, reg, "dell_csm_auth_request_duration_seconds")
	if mfDur != nil {
		for _, metric := range mfDur.GetMetric() {
			assert.Equal(t, uint64(0), metric.GetHistogram().GetSampleCount())
		}
	}
}

// TestAuthInstrumenter_AllMetricsRegistered verifies all 8 auth metric families are emitted
func TestAuthInstrumenter_AllMetricsRegistered(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRequest("powerflex", "tenant-a", "success", 0.1)
	ins.RecordTokenValidation("success", "ok")
	ins.RecordRoleAccess("admin", "powerflex", "allow")
	ins.RecordCredentialShield("powerflex", "success")
	ins.RecordQuotaCheck("powerflex", "tenant-a", "allowed")

	mfs, err := reg.Gather()
	require.NoError(t, err)

	names := make(map[string]bool, len(mfs))
	for _, mf := range mfs {
		names[mf.GetName()] = true
	}

	expected := []string{
		"dell_csm_auth_up",
		"dell_csm_auth_success_rate",
		"dell_csm_auth_request_total",
		"dell_csm_auth_request_duration_seconds",
		"dell_csm_auth_token_validation_total",
		"dell_csm_auth_role_access_total",
		"dell_csm_auth_credential_shield_total",
		"dell_csm_auth_quota_check_total",
	}
	for _, exp := range expected {
		assert.True(t, names[exp], "expected metric %q to be present", exp)
	}
}

// TestAuthInstrumenter_UpGaugePresentAndIsOne verifies dell_csm_auth_up is 1 at startup
func TestAuthInstrumenter_UpGaugePresentAndIsOne(t *testing.T) {
	reg := prometheus.NewRegistry()
	_ = module.NewAuthInstrumenter(reg)

	mf := gatherMetric(t, reg, "dell_csm_auth_up")
	require.NotNil(t, mf, "dell_csm_auth_up must be registered at startup")
	v := gaugeValue(mf, map[string]string{})
	assert.Equal(t, 1.0, v, "dell_csm_auth_up must be 1 immediately after creation")
}

// TestAuthInstrumenter_UpGaugeRemainsOneAfterRequests verifies up stays 1 after traffic
func TestAuthInstrumenter_UpGaugeRemainsOneAfterRequests(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRequest("powerflex", "tenant-a", "success", 0.1)
	ins.RecordRequest("powerflex", "tenant-a", "failure", 0)

	mf := gatherMetric(t, reg, "dell_csm_auth_up")
	require.NotNil(t, mf)
	assert.Equal(t, 1.0, gaugeValue(mf, map[string]string{}))
}

// TestAuthInstrumenter_SuccessRateAtStartupIsZero verifies success_rate is 0 before any requests
func TestAuthInstrumenter_SuccessRateAtStartupIsZero(t *testing.T) {
	reg := prometheus.NewRegistry()
	_ = module.NewAuthInstrumenter(reg)

	mf := gatherMetric(t, reg, "dell_csm_auth_success_rate")
	require.NotNil(t, mf, "dell_csm_auth_success_rate must be registered at startup")
	v := gaugeValue(mf, map[string]string{})
	assert.Equal(t, 0.0, v, "success_rate must be 0 before any requests (no division-by-zero)")
}

// TestAuthInstrumenter_SuccessRateAllSuccess verifies 1.0 when all requests succeed
func TestAuthInstrumenter_SuccessRateAllSuccess(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRequest("powerflex", "tenant-a", "success", 0.1)
	ins.RecordRequest("powerflex", "tenant-a", "success", 0.2)
	ins.RecordRequest("powerflex", "tenant-a", "success", 0.3)

	mf := gatherMetric(t, reg, "dell_csm_auth_success_rate")
	require.NotNil(t, mf)
	assert.InDelta(t, 1.0, gaugeValue(mf, map[string]string{}), 0.001)
}

// TestAuthInstrumenter_SuccessRateAllFailure verifies 0.0 when all requests fail
func TestAuthInstrumenter_SuccessRateAllFailure(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRequest("powerflex", "tenant-a", "failure", 0)
	ins.RecordRequest("powerflex", "tenant-a", "failure", 0)

	mf := gatherMetric(t, reg, "dell_csm_auth_success_rate")
	require.NotNil(t, mf)
	assert.InDelta(t, 0.0, gaugeValue(mf, map[string]string{}), 0.001)
}

// TestAuthInstrumenter_SuccessRateMixed verifies correct ratio for mixed traffic
func TestAuthInstrumenter_SuccessRateMixed(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordRequest("powerflex", "tenant-a", "success", 0.1)
	ins.RecordRequest("powerflex", "tenant-a", "success", 0.1)
	ins.RecordRequest("powerflex", "tenant-a", "success", 0.1)
	ins.RecordRequest("powerflex", "tenant-a", "failure", 0)

	mf := gatherMetric(t, reg, "dell_csm_auth_success_rate")
	require.NotNil(t, mf)
	assert.InDelta(t, 0.75, gaugeValue(mf, map[string]string{}), 0.001)
}

// TestAuthInstrumenter_SuccessRateConcurrent verifies thread-safety of success rate updates
func TestAuthInstrumenter_SuccessRateConcurrent(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	const workers = 20
	const reqsEach = 50
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for j := range reqsEach {
				status := "success"
				if j%4 == 0 {
					status = "failure"
				}
				ins.RecordRequest("powerflex", "tenant-a", status, 0.1)
			}
		}()
	}
	wg.Wait()

	mf := gatherMetric(t, reg, "dell_csm_auth_success_rate")
	require.NotNil(t, mf)
	v := gaugeValue(mf, map[string]string{})
	assert.Greater(t, v, 0.0, "success_rate must be >0 after mixed traffic")
	assert.LessOrEqual(t, v, 1.0, "success_rate must be <=1")
}

// TestAuthInstrumenter_QuotaCheckErrorResult verifies the 'error' result label is recorded
func TestAuthInstrumenter_QuotaCheckErrorResult(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewAuthInstrumenter(reg)

	ins.RecordQuotaCheck("powerflex", "tenant-a", "allowed")
	ins.RecordQuotaCheck("powerflex", "tenant-a", "denied")
	ins.RecordQuotaCheck("powerflex", "tenant-a", "error")

	mf := gatherMetric(t, reg, "dell_csm_auth_quota_check_total")
	require.NotNil(t, mf)
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"storage_type": "powerflex", "tenant": "tenant-a", "result": "allowed"}))
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"storage_type": "powerflex", "tenant": "tenant-a", "result": "denied"}))
	assert.Equal(t, 1.0, counterValue(mf, map[string]string{"storage_type": "powerflex", "tenant": "tenant-a", "result": "error"}))
}

// TestOperatorInstrumenter_SetClustersManaged
func TestOperatorInstrumenter_SetClustersManaged(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewOperatorInstrumenter(reg)

	ins.SetClustersManaged(5.0)

	mf := gatherMetric(t, reg, "dell_csm_operator_clusters_managed_total")
	require.NotNil(t, mf)
	v := gaugeValue(mf, map[string]string{})
	assert.InDelta(t, 5.0, v, 0.001)
}

// TestOperatorInstrumenter_SetClusterConnectivity
func TestOperatorInstrumenter_SetClusterConnectivity(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewOperatorInstrumenter(reg)

	ins.SetClusterConnectivity("cluster-1", "platform-1", true)

	mf := gatherMetric(t, reg, "dell_csm_operator_cluster_connectivity")
	require.NotNil(t, mf)

	ins.SetClusterConnectivity("cluster-1", "platform-1", false)
}

// TestOperatorInstrumenter_SetCRCount
func TestOperatorInstrumenter_SetCRCount(t *testing.T) {
	reg := prometheus.NewRegistry()
	ins := module.NewOperatorInstrumenter(reg)

	ins.SetCRCount("powerflex", "present", 3.0)

	mf := gatherMetric(t, reg, "dell_csm_operator_cr_count")
	require.NotNil(t, mf)
	v := gaugeValue(mf, map[string]string{"driver": "powerflex", "state": "present"})
	assert.InDelta(t, 3.0, v, 0.001)
}
