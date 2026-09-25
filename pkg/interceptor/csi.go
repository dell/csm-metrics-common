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

// Package interceptor provides gRPC interceptors for CSI operation metrics.
package interceptor

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// CSIOperationConfig holds configuration for CSI operation metrics.
type CSIOperationConfig struct {
	Registry          prometheus.Registerer
	SystemIDLabel     string
	AdditionalLabels  []string
	HistogramBuckets  []float64
	ClassifyErrorFunc func(error) string
}

// CSIOperationInterceptor tracks CSI operation metrics using standardized naming.
type CSIOperationInterceptor struct {
	opTotal    *prometheus.CounterVec
	opDuration *prometheus.HistogramVec
	opFailure  *prometheus.CounterVec
	labels     []string
}

// NewCSIOperationInterceptor creates a new CSI operation metrics interceptor.
func NewCSIOperationInterceptor(cfg CSIOperationConfig) *CSIOperationInterceptor {
	labels := append([]string{"system_id", "operation", "status"}, cfg.AdditionalLabels...)

	interceptor := &CSIOperationInterceptor{
		labels: labels,
		opTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dell_csi_operation_total",
			Help: "Total CSI operations by operation type and status.",
		}, labels),
		opDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "dell_csi_operation_duration_seconds",
			Help:    "CSI operation latency histogram.",
			Buckets: cfg.HistogramBuckets,
		}, append([]string{"system_id", "operation"}, cfg.AdditionalLabels...)),
		opFailure: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dell_csi_operation_failure_total",
			Help: "Total CSI operation failures by error code.",
		}, append([]string{"system_id", "operation", "error_code"}, cfg.AdditionalLabels...)),
	}

	cfg.Registry.MustRegister(interceptor.opTotal, interceptor.opDuration, interceptor.opFailure)
	return interceptor
}

// ExtractOperation extracts the CSI operation name from the gRPC method path.
// Example: /csi.v1.Controller/CreateVolume -> CreateVolume
func ExtractOperation(fullMethod string) string {
	parts := strings.Split(fullMethod, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// ClassifyGRPCError converts a gRPC error to a standardized error code string.
func ClassifyGRPCError(err error) string {
	if err == nil {
		return "none"
	}
	st, ok := status.FromError(err)
	if ok {
		return st.Code().String()
	}
	return "unknown"
}

// UnaryInterceptor returns a gRPC UnaryServerInterceptor that records CSI operation metrics.
func (i *CSIOperationInterceptor) UnaryInterceptor(systemID string, extractAdditionalLabels func(context.Context, interface{}) []string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		operation := ExtractOperation(info.FullMethod)

		// Extract additional labels if provided
		additionalLabels := []string{}
		if extractAdditionalLabels != nil {
			additionalLabels = extractAdditionalLabels(ctx, req)
		}

		resp, err := handler(ctx, req)
		duration := time.Since(start).Seconds()

		// Build label values
		totalLabels := append([]string{systemID, operation}, additionalLabels...)
		durationLabels := append([]string{systemID, operation}, additionalLabels...)

		i.opDuration.WithLabelValues(durationLabels...).Observe(duration)

		if err != nil {
			errorCode := ClassifyGRPCError(err)
			failureLabels := append([]string{systemID, operation, errorCode}, additionalLabels...)
			i.opTotal.WithLabelValues(append(totalLabels, "failure")...).Inc()
			i.opFailure.WithLabelValues(failureLabels...).Inc()
		} else {
			i.opTotal.WithLabelValues(append(totalLabels, "success")...).Inc()
		}

		return resp, err
	}
}
