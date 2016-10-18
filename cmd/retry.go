/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"math/rand"
	"sync"
	"time"
)

// lockedRandSource provides protected rand source, implements rand.Source interface.
type lockedRandSource struct {
	lk  sync.Mutex
	src rand.Source
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an
// int64.
func (r *lockedRandSource) Int63() (n int64) {
	r.lk.Lock()
	n = r.src.Int63()
	r.lk.Unlock()
	return
}

// Seed uses the provided seed value to initialize the generator to a
// deterministic state.
func (r *lockedRandSource) Seed(seed int64) {
	r.lk.Lock()
	r.src.Seed(seed)
	r.lk.Unlock()
}

// MaxJitter will randomize over the full exponential backoff time
const MaxJitter = 1.0

// NoJitter disables the use of jitter for randomizing the exponential backoff time
const NoJitter = 0.0

// Global random source for fetching random values.
var globalRandomSource = rand.New(&lockedRandSource{
	src: rand.NewSource(time.Now().UTC().UnixNano()),
})

// newRetryTimer creates a timer with exponentially increasing delays
// until the maximum retry attempts are reached.
func newRetryTimer(unit time.Duration, cap time.Duration, jitter float64, doneCh chan struct{}) <-chan struct{} {
	attemptCh := make(chan struct{})

	// computes the exponential backoff duration according to
	// https://www.awsarchitectureblog.com/2015/03/backoff.html
	exponentialBackoffWait := func(attempt int) time.Duration {
		// normalize jitter to the range [0, 1.0]
		if jitter < NoJitter {
			jitter = NoJitter
		}
		if jitter > MaxJitter {
			jitter = MaxJitter
		}

		//sleep = random_between(0, min(cap, base * 2 ** attempt))
		sleep := unit * time.Duration(1<<uint(attempt))
		if sleep > cap {
			sleep = cap
		}
		if jitter != NoJitter {
			sleep -= time.Duration(globalRandomSource.Float64() * float64(sleep) * jitter)
		}
		return sleep
	}

	go func() {
		defer close(attemptCh)
		var nextBackoff int
		for {
			select {
			// Attempts starts.
			case attemptCh <- struct{}{}:
				nextBackoff++
			case <-globalWakeupCh:
				// Reset nextBackoff to reduce the subsequent wait and re-read
				// format.json from all disks again.
				nextBackoff = 0
			case <-doneCh:
				// Stop the routine.
				return
			}
			time.Sleep(exponentialBackoffWait(nextBackoff))
		}
	}()
	return attemptCh
}
