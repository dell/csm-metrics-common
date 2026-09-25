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

package cache_test

import (
	"sync"
	"testing"
	"time"

	"github.com/Ecosystems/container-storage-modules/src/csm-metrics-common/pkg/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// U-CMC-18: Set then Get within TTL returns stored data
func TestResponseCache_SetAndGetWithinTTL(t *testing.T) {
	c := cache.NewResponseCache(1 * time.Second)
	c.Set("key1", "value1")

	val, ok := c.Get("key1")
	require.True(t, ok, "expected cache hit")
	assert.Equal(t, "value1", val)
}

// U-CMC-19: Get after TTL expiry returns nil, false
func TestResponseCache_GetAfterTTLExpiry(t *testing.T) {
	c := cache.NewResponseCache(10 * time.Millisecond)
	c.Set("key1", "value1")

	time.Sleep(20 * time.Millisecond)

	val, ok := c.Get("key1")
	assert.False(t, ok, "expected cache miss after TTL expiry")
	assert.Nil(t, val)
}

// U-CMC-20: Get with empty key returns nil, false
func TestResponseCache_GetWithEmptyKey(t *testing.T) {
	c := cache.NewResponseCache(1 * time.Second)
	c.Set("", "should-not-store")

	val, ok := c.Get("")
	assert.False(t, ok, "empty key should always miss")
	assert.Nil(t, val)
}

// U-CMC-21: TTL=0 behaves as no-caching
func TestResponseCache_ZeroTTLNoCaching(t *testing.T) {
	c := cache.NewResponseCache(0)
	c.Set("key1", "value1")

	val, ok := c.Get("key1")
	assert.False(t, ok, "TTL=0 should behave as no-caching")
	assert.Nil(t, val)
}

// U-CMC-22: Concurrent Set/Get — no data race
func TestResponseCache_ConcurrentAccess(_ *testing.T) {
	c := cache.NewResponseCache(1 * time.Second)
	var wg sync.WaitGroup
	const goroutines = 20

	for i := 0; i < goroutines; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			c.Set("key", n)
		}(i)
		go func() {
			defer wg.Done()
			c.Get("key") //nolint:errcheck
		}()
	}
	wg.Wait()
	// No race condition panic = pass; validated by -race flag
}

// TestResponseCache_Delete removes cached entry
func TestResponseCache_Delete(t *testing.T) {
	c := cache.NewResponseCache(1 * time.Second)
	c.Set("key1", "value1")
	c.Delete("key1")

	_, ok := c.Get("key1")
	assert.False(t, ok, "deleted key should not be found")
}

// TestResponseCache_SetOverwritesExisting
func TestResponseCache_SetOverwritesExisting(t *testing.T) {
	c := cache.NewResponseCache(1 * time.Second)
	c.Set("key1", "first")
	c.Set("key1", "second")

	val, ok := c.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "second", val)
}
