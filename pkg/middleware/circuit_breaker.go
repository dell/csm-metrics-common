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
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when a circuit breaker is in the Open state.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CBState represents the state of a CircuitBreaker.
type CBState int

const (
	// CBStateClosed means the circuit is operating normally.
	CBStateClosed CBState = iota
	// CBStateOpen means the circuit is open and calls are rejected.
	CBStateOpen
	// CBStateHalfOpen means the circuit is testing if the downstream is healthy.
	CBStateHalfOpen
)

// CircuitBreaker implements a three-state circuit breaker pattern.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        CBState
	failures     int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
	arrayID      string
}

// NewCircuitBreaker creates a new CircuitBreaker for the given arrayID.
func NewCircuitBreaker(arrayID string, threshold int, resetTimeout time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 1
	}
	return &CircuitBreaker{
		arrayID:      arrayID,
		threshold:    threshold,
		resetTimeout: resetTimeout,
		state:        CBStateClosed,
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CBState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentState()
}

// currentState resolves OPEN→HALF_OPEN transitions based on time. Must be called with mu held.
func (cb *CircuitBreaker) currentState() CBState {
	if cb.state == CBStateOpen && time.Since(cb.lastFailure) >= cb.resetTimeout {
		cb.state = CBStateHalfOpen
	}
	return cb.state
}

// Call executes fn through the circuit breaker.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	state := cb.currentState()
	cb.mu.Unlock()

	if state == CBStateOpen {
		return fmt.Errorf("circuit breaker open for array %s: last failure at %s: %w",
			cb.arrayID, cb.lastFailure.Format(time.RFC3339), ErrCircuitOpen)
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.threshold {
			cb.state = CBStateOpen
		}
	} else {
		cb.failures = 0
		cb.state = CBStateClosed
	}
	return err
}
