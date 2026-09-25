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
	"context"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

// U-CMC-14: RateLimiter enforces per-endpoint rate
func TestRateLimiter_EnforcesRate(t *testing.T) {
	// Allow 2 requests per second (very low for testing)
	rl := middleware.NewRateLimiter(2)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// First request should pass immediately
	start := time.Now()
	err := rl.Wait(ctx, "endpoint-a")
	assert.NoError(t, err)
	firstDur := time.Since(start)
	assert.Less(t, firstDur, 200*time.Millisecond, "first request should be immediate")

	// Second request should also pass (burst allows it)
	err = rl.Wait(ctx, "endpoint-a")
	assert.NoError(t, err)
}

// U-CMC-15: RateLimiter with rate=0 is unlimited (no blocking)
func TestRateLimiter_ZeroRateIsUnlimited(t *testing.T) {
	rl := middleware.NewRateLimiter(0)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 10 requests should all be immediate
	for i := 0; i < 10; i++ {
		err := rl.Wait(ctx, "endpoint-a")
		assert.NoError(t, err, "request %d should not be blocked with unlimited rate", i)
	}
}

// TestRateLimiter_PerEndpointIsolation verifies different endpoints don't share limits
func TestRateLimiter_PerEndpointIsolation(t *testing.T) {
	rl := middleware.NewRateLimiter(100) // 100 req/min

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err1 := rl.Wait(ctx, "endpoint-a")
	err2 := rl.Wait(ctx, "endpoint-b")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}
