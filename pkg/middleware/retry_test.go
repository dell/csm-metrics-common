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
	"errors"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// U-CMC-16: Retry exhausts maxAttempts on always-failing fn
func TestRetry_ExhaustsAttemptsOnAlwaysFailing(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("backend unavailable")
	}

	ctx := context.Background()
	// Use very short backoff for tests
	start := time.Now()
	err := middleware.Retry(ctx, 3, fn)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Equal(t, 3, attempts, "should attempt exactly 3 times")
	assert.Less(t, elapsed, 30*time.Second, "should not take too long")
}

// U-CMC-17: Retry succeeds on 2nd attempt
func TestRetry_SucceedsOnSecondAttempt(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts == 1 {
			return errors.New("transient error")
		}
		return nil
	}

	ctx := context.Background()
	err := middleware.Retry(ctx, 3, fn)
	assert.NoError(t, err)
	assert.Equal(t, 2, attempts, "should succeed on 2nd attempt")
}

// TestRetry_ContextCancellation stops retry early
func TestRetry_ContextCancellationStopsRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("always fails")
	}

	err := middleware.Retry(ctx, 10, fn)
	assert.Error(t, err)
	assert.Less(t, attempts, 10, "context cancellation should stop retries early")
}
