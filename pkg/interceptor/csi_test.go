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

package interceptor

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestExtractOperation(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		want       string
	}{
		{
			name:       "standard CSI method",
			fullMethod: "/csi.v1.Controller/CreateVolume",
			want:       "CreateVolume",
		},
		{
			name:       "node method",
			fullMethod: "/csi.v1.Node/NodePublishVolume",
			want:       "NodePublishVolume",
		},
		{
			name:       "identity method",
			fullMethod: "/csi.v1.Identity/Probe",
			want:       "Probe",
		},
		{
			name:       "malformed method",
			fullMethod: "invalid",
			want:       "invalid",
		},
		{
			name:       "empty method",
			fullMethod: "",
			want:       "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractOperation(tt.fullMethod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClassifyGRPCError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
			err:  nil,
			want: "none",
		},
		{
			name: "gRPC status error",
			err:  status.Error(codes.NotFound, "not found"),
			want: "NotFound",
		},
		{
			name: "gRPC permission error",
			err:  status.Error(codes.PermissionDenied, "permission denied"),
			want: "PermissionDenied",
		},
		{
			name: "generic error",
			err:  assert.AnError,
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyGRPCError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewCSIOperationInterceptor(t *testing.T) {
	registry := prometheus.NewRegistry()
	cfg := CSIOperationConfig{
		Registry:          registry,
		SystemIDLabel:     "system_id",
		AdditionalLabels:  []string{"protocol"},
		HistogramBuckets:  []float64{0.1, 0.5, 1, 2, 5},
		ClassifyErrorFunc: ClassifyGRPCError,
	}

	interceptor := NewCSIOperationInterceptor(cfg)
	assert.NotNil(t, interceptor)
	assert.NotNil(t, interceptor.opTotal)
	assert.NotNil(t, interceptor.opDuration)
	assert.NotNil(t, interceptor.opFailure)
}

func TestCSIOperationInterceptor_UnaryInterceptor(t *testing.T) {
	registry := prometheus.NewRegistry()
	cfg := CSIOperationConfig{
		Registry:          registry,
		SystemIDLabel:     "system_id",
		AdditionalLabels:  []string{"protocol"},
		HistogramBuckets:  []float64{0.1, 0.5, 1, 2, 5},
		ClassifyErrorFunc: ClassifyGRPCError,
	}

	interceptor := NewCSIOperationInterceptor(cfg)

	// Test with successful handler
	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	unaryInterceptor := interceptor.UnaryInterceptor("test-system", func(_ context.Context, _ interface{}) []string {
		return []string{"iscsi"}
	})

	info := &grpc.UnaryServerInfo{FullMethod: "/csi.v1.Controller/CreateVolume"}
	resp, err := unaryInterceptor(context.Background(), "request", info, handler)

	assert.True(t, handlerCalled)
	assert.Equal(t, "response", resp)
	assert.NoError(t, err)

	// Test with error handler
	handlerCalled = false
	errorHandler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return nil, status.Error(codes.NotFound, "not found")
	}

	resp, err = unaryInterceptor(context.Background(), "request", info, errorHandler)

	assert.True(t, handlerCalled)
	assert.Nil(t, resp)
	assert.Error(t, err)
}
