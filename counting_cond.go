package sync

import (
	"sync"
	"sync/atomic"
)

// CountingCond implements a condition variable, a rendezvous point
// for goroutines waiting for or announcing the occurrence
// of an event.
//
// Each Cond has an associated Locker L (often a *Mutex or *RWMutex),
// which must be held when changing the condition and
// when calling the Wait method.
//
// A Cond must not be copied after first use.
//
// In the terminology of the Go memory model, Cond arranges that
// a call to Broadcast or Signal “synchronizes before” any Wait call
// that it unblocks.
//
// For many simple use cases, users will be better off using channels than a
// Cond (Broadcast corresponds to closing a channel, and Signal corresponds to
// sending on a channel).
//
// For more on replacements for sync.Cond, see [Roberto Clapis's series on
// advanced concurrency patterns], as well as [Bryan Mills's talk on concurrency
// patterns].
//
// [Roberto Clapis's series on advanced concurrency patterns]: https://blogtitle.github.io/categories/concurrency/
// [Bryan Mills's talk on concurrency patterns]: https://drive.google.com/file/d/1nPdvhB0PutEJzdCq5ms6UI58dp50fcAN/view
// A CountingCond includes a Len method that returns the number of goroutines that
// have either acquired the lock or are attempting to acquire the lock,
// that the Condition is waiting on, if a broadcast or signal were to occur.
type CountingCond struct {
	noCopy noCopy

	cond  sync.Cond
	count int32
}

// NewCountingCond returns a new CountingCond with Locker l.
func NewCountingCond(l sync.Locker) *CountingCond {
	return &CountingCond{cond: sync.Cond{L: l}}
}

// Wait atomically unlocks c.cond.L and suspends execution
// of the calling goroutine. After later resuming execution,
// Wait locks c.cond.L before returning. Unlike in other systems,
// Wait cannot return unless awoken by Broadcast or Signal.
//
// Because c.cond.L is not locked when Wait first resumes, the caller
// typically cannot assume that the condition is true when
// Wait returns. Instead, the caller should Wait in a loop:
//
//	c.Lock()
//	for !condition() {
//	    c.Wait()
//	}
//	... make use of condition ...
//	c.Unlock()
func (c *CountingCond) Wait() {
	atomic.AddInt32(&c.count, -1)
	c.cond.Wait()
	atomic.AddInt32(&c.count, 1)
}

// WaitReturnLen atomically unlocks c.cond.L and suspends execution
// of the calling goroutine. After later resuming execution,
// WaitReturnLen locks c.cond.L before returning the current count/len. Unlike in other systems,
// WaitReturnLen cannot return unless awoken by Broadcast or Signal.
//
// Because c.cond.L is not locked when Wait first resumes, the caller
// typically cannot assume that the condition is true when
// Wait returns. Instead, the caller should Wait in a loop:
//
//	c.Lock()
//	for !condition() {
//	    c.Wait()
//	}
//	... make use of condition ...
//	c.Unlock()
func (c *CountingCond) WaitReturnLen() int {
	atomic.AddInt32(&c.count, -1)
	c.cond.Wait()
	return int(atomic.AddInt32(&c.count, 1))
}

// Signal wakes one goroutine waiting on c, if there is any.
//
// It is allowed but not required for the caller to hold c.cond.L
// during the call.
//
// Signal() does not affect goroutine scheduling priority; if other goroutines
// are attempting to lock c.cond.L, they may be awoken before a "waiting" goroutine.
func (c *CountingCond) Signal() {
	c.cond.Signal()
}

// Broadcast wakes all goroutines waiting on c.
//
// It is allowed but not required for the caller to hold c.cond.L
// during the call.
func (c *CountingCond) Broadcast() {
	c.cond.Broadcast()
}

// Lock locks the condition's Locker.
// Normally, if the lock is already in use, the calling goroutine blocks until lock is available.
func (c *CountingCond) Lock() {
	atomic.AddInt32(&c.count, 1)
	c.cond.L.Lock()
}

// LockReturnLen locks the condition's Locker and returns the current count/len.
// Normally, if the lock is already in use, the calling goroutine blocks until lock is available.
func (c *CountingCond) LockReturnLen() int {
	i := atomic.AddInt32(&c.count, 1)
	c.cond.L.Lock()
	return int(i)
}

// Unlock unlocks the condition's Locker.
// It is usually a run-time error if the Locker is not locked on entry to Unlock.
// Normally, a locked Locker is not associated with a particular goroutine.
// It is normally allowed for one goroutine to lock a Locker and then arrange for another goroutine to unlock it.
func (c *CountingCond) Unlock() {
	c.cond.L.Unlock()
	atomic.AddInt32(&c.count, -1)
}

// UnlockReturnLen unlocks the condition's Locker and returns the current count/len.
// It is usually a run-time error if the Locker is not locked on entry to Unlock.
// Normally, a locked Locker is not associated with a particular goroutine.
// It is normally allowed for one goroutine to lock a Locker and then arrange for another goroutine to unlock it.
func (c *CountingCond) UnlockReturnLen() int {
	c.cond.L.Unlock()
	return int(atomic.AddInt32(&c.count, -1))
}

// Len returns the current number of goroutines that have either acquired the lock or are attempting to acquire the lock,
// that the Condition is waiting on, if a broadcast or signal were to occur.
func (c *CountingCond) Len() int {
	return int(atomic.LoadInt32(&c.count))
}
