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

package middleware_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// U-CMC-09: Circuit CLOSED → OPEN after 3 failures
func TestCircuitBreaker_ClosedToOpenAfterThresholdFailures(t *testing.T) {
	cb := middleware.NewCircuitBreaker("array-1", 3, 30*time.Second)
	assert.Equal(t, middleware.CBStateClosed, cb.State())

	failFn := func() error { return errors.New("backend error") }

	for i := 0; i < 3; i++ {
		_ = cb.Call(failFn)
	}

	assert.Equal(t, middleware.CBStateOpen, cb.State(), "circuit should be OPEN after 3 failures")

	// Next call should be rejected immediately
	err := cb.Call(func() error { return nil })
	require.Error(t, err)
	assert.ErrorIs(t, err, middleware.ErrCircuitOpen)
}

// U-CMC-10: Circuit OPEN → HALF_OPEN after resetTimeout
func TestCircuitBreaker_OpenToHalfOpenAfterReset(t *testing.T) {
	cb := middleware.NewCircuitBreaker("array-1", 1, 10*time.Millisecond)

	_ = cb.Call(func() error { return errors.New("error") })
	assert.Equal(t, middleware.CBStateOpen, cb.State())

	time.Sleep(20 * time.Millisecond)
	// State transitions to HALF_OPEN on next State() call
	assert.Equal(t, middleware.CBStateHalfOpen, cb.State())
}

// U-CMC-11: Circuit HALF_OPEN → CLOSED on success
func TestCircuitBreaker_HalfOpenToClosedOnSuccess(t *testing.T) {
	cb := middleware.NewCircuitBreaker("array-1", 1, 10*time.Millisecond)

	_ = cb.Call(func() error { return errors.New("error") })
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, middleware.CBStateHalfOpen, cb.State())

	err := cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, middleware.CBStateClosed, cb.State())
}

// U-CMC-12: Circuit HALF_OPEN → OPEN on failure
func TestCircuitBreaker_HalfOpenToOpenOnFailure(t *testing.T) {
	cb := middleware.NewCircuitBreaker("array-1", 1, 10*time.Millisecond)

	_ = cb.Call(func() error { return errors.New("error") })
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, middleware.CBStateHalfOpen, cb.State())

	_ = cb.Call(func() error { return errors.New("still failing") })
	assert.Equal(t, middleware.CBStateOpen, cb.State())
}

// U-CMC-13: Circuit breaker threshold=0 treated as 1
func TestCircuitBreaker_ZeroThresholdTreatedAsOne(t *testing.T) {
	cb := middleware.NewCircuitBreaker("array-1", 0, 30*time.Second)

	// Should open after exactly 1 failure (not zero)
	_ = cb.Call(func() error { return errors.New("error") })
	assert.Equal(t, middleware.CBStateOpen, cb.State(),
		"circuit with threshold=0 should open after 1 failure (threshold treated as 1)")
}

// TestCircuitBreaker_SuccessResetsFailureCount verifies a success clears the failure count
func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := middleware.NewCircuitBreaker("array-1", 3, 30*time.Second)

	_ = cb.Call(func() error { return errors.New("error") })
	_ = cb.Call(func() error { return errors.New("error") })
	// success resets
	_ = cb.Call(func() error { return nil })
	// Two more failures should not open (count reset to 0)
	_ = cb.Call(func() error { return errors.New("error") })
	_ = cb.Call(func() error { return errors.New("error") })
	assert.Equal(t, middleware.CBStateClosed, cb.State(),
		"failure count should have been reset by the success")
}
