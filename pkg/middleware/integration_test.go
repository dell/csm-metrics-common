/*
 Copyright © 2025-2026 Dell Inc. or its subsidiaries. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
      http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package middleware_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// I-CMC-02: CircuitBreaker trips after threshold failures; once OPEN all calls fail immediately.
func TestIntegration_CircuitBreaker_TripsAfterThreshold(t *testing.T) {
	// threshold=3: circuit opens after 3 consecutive failures
	cb := middleware.NewCircuitBreaker("array-test", 3, 30*time.Second)

	callCount := 0
	alwaysFail := func() error {
		callCount++
		return errors.New("backend down")
	}

	// Drive 3 failures to open the circuit
	for i := 0; i < 3; i++ {
		_ = cb.Call(alwaysFail)
	}

	assert.Equal(t, 3, callCount, "all 3 calls before threshold should reach the backend")
	assert.Equal(t, middleware.CBStateOpen, cb.State(), "circuit must be OPEN after threshold failures")

	// Now the circuit is OPEN — next call must fail immediately without invoking fn
	callsBeforeOpen := callCount
	err := cb.Call(func() error {
		callCount++
		return nil
	})

	require.Error(t, err, "OPEN circuit must return an error")
	assert.ErrorIs(t, err, middleware.ErrCircuitOpen, "error must wrap ErrCircuitOpen")
	assert.Equal(t, callsBeforeOpen, callCount, "OPEN circuit must not invoke the inner function")
}

// I-CMC-02b: CircuitBreaker resets on success; remains CLOSED after successful calls.
func TestIntegration_CircuitBreaker_RemainsClosedAfterSuccess(t *testing.T) {
	cb := middleware.NewCircuitBreaker("array-ok", 5, 30*time.Second)

	// 4 successful calls — should not trip the circuit
	for i := 0; i < 4; i++ {
		err := cb.Call(func() error { return nil })
		require.NoError(t, err)
	}

	assert.Equal(t, middleware.CBStateClosed, cb.State(), "circuit must remain CLOSED after successful calls")
}

// I-CMC-02c: RateLimiter allows first burst immediately; blocks when burst exceeded.
func TestIntegration_RateLimiter_AllowsInitialBurst(t *testing.T) {
	// 120 req/min = 2 req/s, burst=120
	rl := middleware.NewRateLimiter(120)

	// The first 120 tokens are immediately available (burst)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx, "endpoint-A")
	assert.NoError(t, err, "first request within burst should not block")
}

// I-CMC-02d: Retry function retries on failure and succeeds on second attempt.
func TestIntegration_Retry_SucceedsOnSecondAttempt(t *testing.T) {
	attempts := 0
	ctx := context.Background()

	err := middleware.Retry(ctx, 3, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("transient error")
		}
		return nil
	})

	require.NoError(t, err, "retry should succeed when fn succeeds on second attempt")
	assert.Equal(t, 2, attempts, "fn should be called exactly twice")
}
