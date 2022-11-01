package sync_test

import (
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	. "github.com/veqryn/go-sync"
)

// NOTICE: count_wait_group_test.go is a direct copy of "sync.cond_test.go", in
// order to test the Count method added to CountingCond.

func TestCondSignal(t *testing.T) {
	var m sync.Mutex
	c := NewCountingCond(&m)

	if c.Len() != 0 {
		t.Fatal("expected len zero")
	}
	if c.LockReturnLen() != 1 {
		t.Fatal("expected len one")
	}
	c.UnlockReturnLen()
	if c.Len() != 0 {
		t.Fatal("expected len zero")
	}

	n := 2
	running := make(chan bool, n)
	awake := make(chan bool, n)
	for i := 0; i < n; i++ {
		go func() {
			m.Lock()
			running <- true
			c.WaitReturnLen()
			awake <- true
			m.Unlock()
		}()
	}
	for i := 0; i < n; i++ {
		<-running // Wait for everyone to run.
	}
	for n > 0 {
		select {
		case <-awake:
			t.Fatal("goroutine not asleep")
		default:
		}
		m.Lock()
		c.Signal()
		m.Unlock()
		<-awake // Will deadlock if no goroutine wakes up
		select {
		case <-awake:
			t.Fatal("too many goroutines awake")
		default:
		}
		n--
	}
	c.Signal()
}

func TestCondSignalGenerations(t *testing.T) {
	var m sync.Mutex
	c := NewCountingCond(&m)
	n := 100
	running := make(chan bool, n)
	awake := make(chan int, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			m.Lock()
			running <- true
			c.Wait()
			awake <- i
			m.Unlock()
		}(i)
		if i > 0 {
			a := <-awake
			if a != i-1 {
				t.Fatalf("wrong goroutine woke up: want %d, got %d", i-1, a)
			}
		}
		<-running
		m.Lock()
		c.Signal()
		m.Unlock()
	}
}

func TestCondBroadcast(t *testing.T) {
	var m sync.Mutex
	c := NewCountingCond(&m)
	n := 200
	running := make(chan int, n)
	awake := make(chan int, n)
	exit := false
	for i := 0; i < n; i++ {
		go func(g int) {
			m.Lock()
			for !exit {
				running <- g
				c.Wait()
				awake <- g
			}
			m.Unlock()
		}(i)
	}
	for i := 0; i < n; i++ {
		for i := 0; i < n; i++ {
			<-running // Will deadlock unless n are running.
		}
		if i == n-1 {
			m.Lock()
			exit = true
			m.Unlock()
		}
		select {
		case <-awake:
			t.Fatal("goroutine not asleep")
		default:
		}
		m.Lock()
		c.Broadcast()
		m.Unlock()
		seen := make([]bool, n)
		for i := 0; i < n; i++ {
			g := <-awake
			if seen[g] {
				t.Fatal("goroutine woke up twice")
			}
			seen[g] = true
		}
	}
	select {
	case <-running:
		t.Fatal("goroutine did not exit")
	default:
	}
	c.Broadcast()
}

func TestRace(t *testing.T) {
	x := 0
	c := NewCountingCond(&sync.Mutex{})
	done := make(chan bool)
	go func() {
		c.Lock()
		x = 1
		c.Wait()
		if x != 2 {
			t.Error("want 2")
		}
		x = 3
		c.Signal()
		c.Unlock()
		done <- true
	}()
	go func() {
		c.Lock()
		for {
			if x == 1 {
				x = 2
				c.Signal()
				break
			}
			c.Unlock()
			runtime.Gosched()
			c.Lock()
		}
		c.Unlock()
		done <- true
	}()
	go func() {
		c.Lock()
		for {
			if x == 2 {
				c.Wait()
				if x != 3 {
					t.Error("want 3")
				}
				break
			}
			if x == 3 {
				break
			}
			c.Unlock()
			runtime.Gosched()
			c.Lock()
		}
		c.Unlock()
		done <- true
	}()
	<-done
	<-done
	<-done
}

func TestCondSignalStealing(t *testing.T) {
	for iters := 0; iters < 1000; iters++ {
		var m sync.Mutex
		cond := NewCountingCond(&m)

		// Start a waiter.
		ch := make(chan struct{})
		go func() {
			m.Lock()
			ch <- struct{}{}
			cond.Wait()
			m.Unlock()

			ch <- struct{}{}
		}()

		<-ch
		m.Lock()
		m.Unlock()

		// We know that the waiter is in the cond.Wait() call because we
		// synchronized with it, then acquired/released the mutex it was
		// holding when we synchronized.
		//
		// Start two goroutines that will race: one will broadcast on
		// the cond var, the other will wait on it.
		//
		// The new waiter may or may not get notified, but the first one
		// has to be notified.
		done := false
		go func() {
			cond.Broadcast()
		}()

		go func() {
			m.Lock()
			for !done {
				cond.Wait()
			}
			m.Unlock()
		}()

		// Check that the first waiter does get signaled.
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("First waiter didn't get broadcast.")
		}

		// Release the second waiter in case it didn't get the
		// broadcast.
		m.Lock()
		done = true
		m.Unlock()
		cond.Broadcast()
	}
}

func TestCondCopy(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil || err.(string) != "sync.Cond is copied" {
			t.Fatalf("got %v, expect sync.Cond is copied", err)
		}
	}()
	// c := CountingCond{L: &sync.Mutex{}}
	c := *NewCountingCond(&sync.Mutex{})
	c.Signal()
	var c2 CountingCond
	reflect.ValueOf(&c2).Elem().Set(reflect.ValueOf(&c).Elem()) // c2 := c, hidden from vet
	c2.Signal()
}

func BenchmarkCond1(b *testing.B) {
	benchmarkCond(b, 1)
}

func BenchmarkCond2(b *testing.B) {
	benchmarkCond(b, 2)
}

func BenchmarkCond4(b *testing.B) {
	benchmarkCond(b, 4)
}

func BenchmarkCond8(b *testing.B) {
	benchmarkCond(b, 8)
}

func BenchmarkCond16(b *testing.B) {
	benchmarkCond(b, 16)
}

func BenchmarkCond32(b *testing.B) {
	benchmarkCond(b, 32)
}

func benchmarkCond(b *testing.B, waiters int) {
	c := NewCountingCond(&sync.Mutex{})
	done := make(chan bool)
	id := 0

	for routine := 0; routine < waiters+1; routine++ {
		go func() {
			for i := 0; i < b.N; i++ {
				c.Lock()
				if id == -1 {
					c.Unlock()
					break
				}
				id++
				if id == waiters+1 {
					id = 0
					c.Broadcast()
				} else {
					c.Wait()
				}
				c.Unlock()
			}
			c.Lock()
			id = -1
			c.Broadcast()
			c.Unlock()
			done <- true
		}()
	}
	for routine := 0; routine < waiters+1; routine++ {
		<-done
	}
}
