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

package middleware

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter enforces a maximum request rate per endpoint.
type RateLimiter struct {
	mu      sync.Mutex
	limits  map[string]*rate.Limiter
	rateVal rate.Limit
	burst   int
}

// NewRateLimiter creates a RateLimiter allowing requestsPerMin per endpoint.
func NewRateLimiter(requestsPerMin int) *RateLimiter {
	r := rate.Inf
	b := 1
	if requestsPerMin > 0 {
		r = rate.Limit(float64(requestsPerMin) / 60.0)
		b = requestsPerMin
	}
	return &RateLimiter{
		limits:  make(map[string]*rate.Limiter),
		rateVal: r,
		burst:   b,
	}
}

// Wait blocks until a token is available for endpoint, or ctx is cancelled.
func (r *RateLimiter) Wait(ctx context.Context, endpoint string) error {
	r.mu.Lock()
	l, ok := r.limits[endpoint]
	if !ok {
		l = rate.NewLimiter(r.rateVal, r.burst)
		r.limits[endpoint] = l
	}
	r.mu.Unlock()
	return l.Wait(ctx)
}
