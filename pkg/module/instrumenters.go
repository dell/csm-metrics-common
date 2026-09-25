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

// Package module provides instrumenters for CSM modules (Observability, Resiliency,
// Replication, Authorization, Operator).
package module

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/naming"
	"github.com/prometheus/client_golang/prometheus"
)

// ObsInstrumenter instruments a CSM Observability service.
type ObsInstrumenter struct {
	collectionRate    *prometheus.GaugeVec
	exportSuccess     *prometheus.CounterVec
	arrayConnectivity *prometheus.GaugeVec
	processingLatency *prometheus.HistogramVec
}

// NewObsInstrumenter creates and registers ObsInstrumenter metrics with reg.
func NewObsInstrumenter(reg prometheus.Registerer, _, platformIDLabel string) *ObsInstrumenter {
	i := &ObsInstrumenter{
		collectionRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricObsPrefix + "collection_rate",
			Help: "Rate of metrics collected per second.",
		}, []string{"module", platformIDLabel}),
		exportSuccess: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricObsPrefix + "export_success_total",
			Help: "Total metric export attempts by status.",
		}, []string{"module", platformIDLabel, "status"}),
		arrayConnectivity: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricObsPrefix + "array_connectivity",
			Help: "Array connectivity status (1=connected, 0=disconnected).",
		}, []string{"module", platformIDLabel}),
		processingLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    naming.MetricObsPrefix + "processing_latency_seconds",
			Help:    "Latency of processing and export after metric collection, in seconds.",
			Buckets: naming.HistogramBuckets,
		}, []string{"module", platformIDLabel}),
	}
	reg.MustRegister(i.collectionRate, i.exportSuccess, i.arrayConnectivity, i.processingLatency)
	return i
}

// RecordCollectionRate sets the collection rate gauge.
func (i *ObsInstrumenter) RecordCollectionRate(moduleLabel, platformID string, rate float64) {
	i.collectionRate.WithLabelValues(moduleLabel, platformID).Set(rate)
}

// RecordExportSuccess increments the export success counter.
func (i *ObsInstrumenter) RecordExportSuccess(moduleLabel, platformID, status string) {
	i.exportSuccess.WithLabelValues(moduleLabel, platformID, status).Inc()
}

// RecordArrayConnectivity sets the connectivity gauge.
func (i *ObsInstrumenter) RecordArrayConnectivity(moduleLabel, platformID string, connected bool) {
	v := 0.0
	if connected {
		v = 1.0
	}
	i.arrayConnectivity.WithLabelValues(moduleLabel, platformID).Set(v)
}

// RecordProcessingLatency records processing duration.
func (i *ObsInstrumenter) RecordProcessingLatency(moduleLabel, platformID string, d time.Duration) {
	i.processingLatency.WithLabelValues(moduleLabel, platformID).Observe(d.Seconds())
}

// ResiliencyInstrumenter instruments a CSM Resiliency service.
type ResiliencyInstrumenter struct {
	podmonHealthy            *prometheus.GaugeVec
	connectivityCheckTotal   *prometheus.CounterVec
	connectivitySuccessRatio *prometheus.GaugeVec
	failoverTotal            *prometheus.CounterVec
	failoverDuration         *prometheus.HistogramVec
}

// NewResiliencyInstrumenter creates and registers ResiliencyInstrumenter metrics.
func NewResiliencyInstrumenter(reg prometheus.Registerer) *ResiliencyInstrumenter {
	i := &ResiliencyInstrumenter{
		podmonHealthy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricResiliencyPrefix + "podmon_healthy",
			Help: "Podmon health per node (1=healthy, 0=unhealthy).",
		}, []string{"module", "node"}),
		connectivityCheckTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricResiliencyPrefix + "connectivity_check_total",
			Help: "Total connectivity checks performed.",
		}, []string{"module", "protocol"}),
		connectivitySuccessRatio: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricResiliencyPrefix + "connectivity_success_ratio",
			Help: "Connectivity success ratio per protocol.",
		}, []string{"module", "protocol"}),
		failoverTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricResiliencyPrefix + "failover_total",
			Help: "Total failover events.",
		}, []string{"module", "status"}),
		failoverDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    naming.MetricResiliencyPrefix + "failover_duration_seconds",
			Help:    "Failover event duration.",
			Buckets: naming.HistogramBuckets,
		}, []string{"module"}),
	}
	reg.MustRegister(i.podmonHealthy, i.connectivityCheckTotal, i.connectivitySuccessRatio,
		i.failoverTotal, i.failoverDuration)
	return i
}

// SetPodmonHealth sets health for a node.
func (i *ResiliencyInstrumenter) SetPodmonHealth(moduleLabel, node string, healthy bool) {
	v := 0.0
	if healthy {
		v = 1.0
	}
	i.podmonHealthy.WithLabelValues(moduleLabel, node).Set(v)
}

// RecordConnectivityCheck increments the connectivity check counter.
func (i *ResiliencyInstrumenter) RecordConnectivityCheck(moduleLabel, protocol string) {
	i.connectivityCheckTotal.WithLabelValues(moduleLabel, protocol).Inc()
}

// SetConnectivitySuccessRatio sets the success ratio gauge.
func (i *ResiliencyInstrumenter) SetConnectivitySuccessRatio(moduleLabel, protocol string, ratio float64) {
	i.connectivitySuccessRatio.WithLabelValues(moduleLabel, protocol).Set(ratio)
}

// RecordFailover records a failover event.
func (i *ResiliencyInstrumenter) RecordFailover(moduleLabel, status string, d time.Duration) {
	i.failoverTotal.WithLabelValues(moduleLabel, status).Inc()
	i.failoverDuration.WithLabelValues(moduleLabel).Observe(d.Seconds())
}

// ReplicationInstrumenter instruments a CSM Replication service.
type ReplicationInstrumenter struct {
	pairStatus     *prometheus.GaugeVec
	lagSeconds     *prometheus.GaugeVec
	bandwidthBytes *prometheus.GaugeVec
	operationTotal *prometheus.CounterVec
}

// NewReplicationInstrumenter creates and registers ReplicationInstrumenter metrics.
func NewReplicationInstrumenter(reg prometheus.Registerer) *ReplicationInstrumenter {
	i := &ReplicationInstrumenter{
		pairStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricReplPrefix + "pair_status",
			Help: "Replication pair status.",
		}, []string{"module", "rcg_id", "state"}),
		lagSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricReplPrefix + "lag_seconds",
			Help: "Replication lag in seconds.",
		}, []string{"module", "rcg_id"}),
		bandwidthBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricReplPrefix + "bandwidth_bytes",
			Help: "Replication bandwidth in bytes/s.",
		}, []string{"module", "rcg_id"}),
		operationTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricReplPrefix + "operation_total",
			Help: "Total replication operations.",
		}, []string{"module", "operation", "status"}),
	}
	reg.MustRegister(i.pairStatus, i.lagSeconds, i.bandwidthBytes, i.operationTotal)
	return i
}

// SetPairStatus updates the pair status gauge.
func (i *ReplicationInstrumenter) SetPairStatus(moduleLabel, rcgID, state string, value float64) {
	i.pairStatus.WithLabelValues(moduleLabel, rcgID, state).Set(value)
}

// SetLag sets the lag gauge.
func (i *ReplicationInstrumenter) SetLag(moduleLabel, rcgID string, lagSec float64) {
	i.lagSeconds.WithLabelValues(moduleLabel, rcgID).Set(lagSec)
}

// SetBandwidth sets the bandwidth gauge.
func (i *ReplicationInstrumenter) SetBandwidth(moduleLabel, rcgID string, bwBytes float64) {
	i.bandwidthBytes.WithLabelValues(moduleLabel, rcgID).Set(bwBytes)
}

// RecordOperation increments the operation counter.
func (i *ReplicationInstrumenter) RecordOperation(moduleLabel, operation, status string) {
	i.operationTotal.WithLabelValues(moduleLabel, operation, status).Inc()
}

// AuthInstrumenter instruments a CSM Authorization proxy service.
// It owns all auth-specific Prometheus metrics including the liveness gauge
// (dell_csm_auth_up) and the rolling success rate gauge (dell_csm_auth_success_rate)
// so that consumers need only call NewAuthInstrumenter and the Record* methods.
type AuthInstrumenter struct {
	// lifecycle / health
	up          prometheus.Gauge
	successRate prometheus.Gauge

	// atomic request counters used to compute successRate without a mutex
	totalCount   int64
	successCount int64

	// per-request metrics
	requestTotal          *prometheus.CounterVec
	requestDuration       *prometheus.HistogramVec
	tokenValidationTotal  *prometheus.CounterVec
	roleAccessTotal       *prometheus.CounterVec
	credentialShieldTotal *prometheus.CounterVec
	quotaCheckTotal       *prometheus.CounterVec
}

// NewAuthInstrumenter creates and registers all AuthInstrumenter metrics with reg.
// dell_csm_auth_up is set to 1 immediately so that the /metrics endpoint returns
// non-empty content before any proxy traffic has been handled.
// dell_csm_auth_success_rate starts at 0 and is updated after every RecordRequest call.
func NewAuthInstrumenter(reg prometheus.Registerer) *AuthInstrumenter {
	i := &AuthInstrumenter{
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: naming.MetricAuthPrefix + "up",
			Help: "1 if the CSM Authorization proxy is running.",
		}),
		successRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: naming.MetricAuthPrefix + "success_rate",
			Help: "Rolling ratio of successful authorization requests to total requests (0.0 - 1.0).",
		}),
		requestTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricAuthPrefix + "request_total",
			Help: "Total authorization requests.",
		}, []string{naming.LabelStorageType, naming.LabelTenant, naming.LabelStatus}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    naming.MetricAuthPrefix + "request_duration_seconds",
			Help:    "Authorization request duration in seconds.",
			Buckets: naming.HistogramBuckets,
		}, []string{naming.LabelStorageType, naming.LabelTenant}),
		tokenValidationTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricAuthPrefix + "token_validation_total",
			Help: "Total token validation outcomes.",
		}, []string{naming.LabelResult, naming.LabelReason}),
		roleAccessTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricAuthPrefix + "role_access_total",
			Help: "Total role access decisions.",
		}, []string{naming.LabelRole, naming.LabelStorageType, naming.LabelDecision}),
		credentialShieldTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricAuthPrefix + "credential_shield_total",
			Help: "Total credential shielding results.",
		}, []string{naming.LabelStorageType, naming.LabelStatus}),
		quotaCheckTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricAuthPrefix + "quota_check_total",
			Help: "Total quota check results.",
		}, []string{naming.LabelStorageType, naming.LabelTenant, naming.LabelResult}),
	}
	reg.MustRegister(
		i.up,
		i.successRate,
		i.requestTotal,
		i.requestDuration,
		i.tokenValidationTotal,
		i.roleAccessTotal,
		i.credentialShieldTotal,
		i.quotaCheckTotal,
	)
	i.up.Set(1)
	return i
}

// RecordRequest increments the request counter, optionally observes duration, and
// updates the dell_csm_auth_success_rate gauge atomically.
func (i *AuthInstrumenter) RecordRequest(storageType, tenant, status string, durationSeconds float64) {
	storageType = sanitizeAuthLabel(storageType)
	tenant = sanitizeAuthLabel(tenant)
	status = sanitizeAuthLabel(status)
	i.requestTotal.WithLabelValues(storageType, tenant, status).Inc()
	if durationSeconds > 0 {
		i.requestDuration.WithLabelValues(storageType, tenant).Observe(durationSeconds)
	}
	total := atomic.AddInt64(&i.totalCount, 1)
	var success int64
	if status == "success" {
		success = atomic.AddInt64(&i.successCount, 1)
	} else {
		success = atomic.LoadInt64(&i.successCount)
	}
	i.successRate.Set(float64(success) / float64(total))
}

// RecordTokenValidation increments the token validation counter.
func (i *AuthInstrumenter) RecordTokenValidation(result, reason string) {
	i.tokenValidationTotal.WithLabelValues(sanitizeAuthLabel(result), sanitizeAuthLabel(reason)).Inc()
}

// RecordRoleAccess increments the role access counter.
func (i *AuthInstrumenter) RecordRoleAccess(role, storageType, decision string) {
	i.roleAccessTotal.WithLabelValues(sanitizeAuthLabel(role), sanitizeAuthLabel(storageType), sanitizeAuthLabel(decision)).Inc()
}

// RecordCredentialShield increments the credential shield counter.
func (i *AuthInstrumenter) RecordCredentialShield(storageType, status string) {
	i.credentialShieldTotal.WithLabelValues(sanitizeAuthLabel(storageType), sanitizeAuthLabel(status)).Inc()
}

// RecordQuotaCheck increments the quota check counter.
func (i *AuthInstrumenter) RecordQuotaCheck(storageType, tenant, result string) {
	i.quotaCheckTotal.WithLabelValues(sanitizeAuthLabel(storageType), sanitizeAuthLabel(tenant), sanitizeAuthLabel(result)).Inc()
}

func sanitizeAuthLabel(v string) string {
	if strings.TrimSpace(v) == "" {
		return "unknown"
	}
	return v
}

// OperatorInstrumenter instruments the csm-operator.
type OperatorInstrumenter struct {
	clustersManagedTotal   prometheus.Gauge
	clusterConnectivity    *prometheus.GaugeVec
	reconciliationTotal    *prometheus.CounterVec
	reconciliationDuration *prometheus.HistogramVec
	crCount                *prometheus.GaugeVec
}

// NewOperatorInstrumenter creates and registers OperatorInstrumenter metrics.
func NewOperatorInstrumenter(reg prometheus.Registerer) *OperatorInstrumenter {
	i := &OperatorInstrumenter{
		clustersManagedTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: naming.MetricOperatorPrefix + "clusters_managed_total",
			Help: "Total ContainerStorageModule CRs managed.",
		}),
		clusterConnectivity: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricOperatorPrefix + "cluster_connectivity",
			Help: "Cluster connectivity status.",
		}, []string{"cluster_name", "platform_id"}),
		reconciliationTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: naming.MetricOperatorPrefix + "reconciliation_total",
			Help: "Total reconciliation attempts.",
		}, []string{"driver", "module", "status"}),
		reconciliationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    naming.MetricOperatorPrefix + "reconciliation_duration_seconds",
			Help:    "Reconciliation duration.",
			Buckets: naming.HistogramBuckets,
		}, []string{"driver", "module"}),
		crCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: naming.MetricOperatorPrefix + "cr_count",
			Help: "CR count by driver and state.",
		}, []string{"driver", "state"}),
	}
	reg.MustRegister(i.clustersManagedTotal, i.clusterConnectivity, i.reconciliationTotal,
		i.reconciliationDuration, i.crCount)
	return i
}

// SetClustersManaged sets the clusters managed gauge.
func (i *OperatorInstrumenter) SetClustersManaged(n float64) {
	i.clustersManagedTotal.Set(n)
}

// SetClusterConnectivity sets connectivity for a cluster.
func (i *OperatorInstrumenter) SetClusterConnectivity(clusterName, platformID string, connected bool) {
	v := 0.0
	if connected {
		v = 1.0
	}
	i.clusterConnectivity.WithLabelValues(clusterName, platformID).Set(v)
}

// RecordReconciliation records a reconciliation event.
func (i *OperatorInstrumenter) RecordReconciliation(driver, module, status string, d time.Duration) {
	i.reconciliationTotal.WithLabelValues(driver, module, status).Inc()
	i.reconciliationDuration.WithLabelValues(driver, module).Observe(d.Seconds())
}

// SetCRCount sets the CR count gauge.
func (i *OperatorInstrumenter) SetCRCount(driver, state string, n float64) {
	i.crCount.WithLabelValues(driver, state).Set(n)
}
