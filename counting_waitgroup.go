// Package sync contains various utilities surrounding synchronization.
package sync

import (
	"sync"
	"sync/atomic"
)

// A CountingWaitGroup waits for a collection of goroutines to finish.
// The main goroutine calls Add to set the number of
// goroutines to wait for. Then each of the goroutines
// runs and calls Done when finished. At the same time,
// Wait can be used to block until all goroutines have finished.
//
// A CountingWaitGroup must not be copied after first use.
//
// In the terminology of the Go memory model, a call to Done
// “synchronizes before” the return of any Wait call that it unblocks.
// A CountingWaitGroup includes a Len method that returns the number of
// adds/goroutines the WaitGroup would wait on.
type CountingWaitGroup struct {
	noCopy noCopy

	wg    sync.WaitGroup
	count int32
}

// Add adds delta, which may be negative, to the WaitGroup counter.
// If the counter becomes zero, all goroutines blocked on Wait are released.
// If the counter goes negative, Add panics.
//
// Note that calls with a positive delta that occur when the counter is zero
// must happen before a Wait. Calls with a negative delta, or calls with a
// positive delta that start when the counter is greater than zero, may happen
// at any time.
// Typically this means the calls to Add should execute before the statement
// creating the goroutine or other event to be waited for.
// If a CountingWaitGroup is reused to wait for several independent sets of events,
// new Add calls must happen after all previous Wait calls have returned.
// See the CountingWaitGroup example.
func (wg *CountingWaitGroup) Add(delta int) {
	atomic.AddInt32(&wg.count, int32(delta))
	wg.wg.Add(delta)
}

// Done decrements the CountingWaitGroup counter by one.
func (wg *CountingWaitGroup) Done() {
	wg.wg.Done()
	atomic.AddInt32(&wg.count, -1)
}

// Wait blocks until the CountingWaitGroup counter is zero.
func (wg *CountingWaitGroup) Wait() {
	wg.wg.Wait()
}

// Len returns the current number of adds/goroutines the WaitGroup would wait on.
func (wg *CountingWaitGroup) Len() int {
	return int(atomic.LoadInt32(&wg.count))
}
