package sync_test

import (
	"sync/atomic"
	"testing"

	. "github.com/veqryn/go-sync"
)

// NOTICE: count_wait_group_test.go is a direct copy of "sync.waitgroup_test.go", in
// order to test the Count method added to CountingWaitGroup.

func testCountingWaitGroup(t *testing.T, wg1 *CountingWaitGroup, wg2 *CountingWaitGroup) {
	n := 16
	wg1.Add(n)
	wg2.Add(n)

	// Test Count funcion
	if wg1.Len() != n || wg2.Len() != n {
		t.Error("Expected func testCountingWaitGroup(t *testing.T, wg1 *CountingWaitGroup, wg2 *CountingWaitGroup) {\n length: ", n, "; Got: wg1: ", wg1.Len(), "; wg2: ", wg2.Len())
	}

	// Decrement Length and test again
	wg1.Done()
	wg1.Done()
	wg2.Done()
	if wg1.Len() != n-2 || wg2.Len() != n-1 {
		t.Error("Expected func testCountingWaitGroup(t *testing.T, wg1 *CountingWaitGroup, wg2 *CountingWaitGroup) {\n length: wg1: ", n-2, "; wg2: ", n-1, "; Got: wg1: ", wg1.Len(), "; wg2: ", wg2.Len())
	}

	// Re-increment Length and test again
	wg1.Add(1)
	wg1.Add(1)
	wg2.Add(1)
	if wg1.Len() != n || wg2.Len() != n {
		t.Error("Expected func testCountingWaitGroup(t *testing.T, wg1 *CountingWaitGroup, wg2 *CountingWaitGroup) {\n length: ", n, "; Got: wg1: ", wg1.Len(), "; wg2: ", wg2.Len())
	}

	exited := make(chan bool, n)
	for i := 0; i != n; i++ {
		go func() {
			wg1.Done()
			wg2.Wait()
			exited <- true
		}()
	}
	wg1.Wait()
	for i := 0; i != n; i++ {
		select {
		case <-exited:
			t.Fatal("CountingWaitGroup released group too soon")
		default:
		}
		wg2.Done()
	}
	for i := 0; i != n; i++ {
		<-exited // Will block if barrier fails to unlock someone.
	}
}

func TestCountingWaitGroup(t *testing.T) {
	wg1 := &CountingWaitGroup{}
	wg2 := &CountingWaitGroup{}

	// Run the same test a few times to ensure barrier is in a proper state.
	for i := 0; i != 8; i++ {
		testCountingWaitGroup(t, wg1, wg2)
	}
}

func TestCountingWaitGroupMisuse(t *testing.T) {
	defer func() {
		err := recover()
		if err != "sync: negative WaitGroup counter" {
			t.Fatalf("Unexpected panic: %#v", err)
		}
	}()
	wg := &CountingWaitGroup{}
	wg.Add(1)
	wg.Done()
	wg.Done()
	t.Fatal("Should panic")
}

func TestCountingWaitGroupRace(t *testing.T) {
	// Run this test for about 1ms.
	for i := 0; i < 1000; i++ {
		wg := &CountingWaitGroup{}
		n := new(int32)
		// spawn goroutine 1
		wg.Add(1)
		go func() {
			atomic.AddInt32(n, 1)
			wg.Done()
		}()
		// spawn goroutine 2
		wg.Add(1)
		go func() {
			atomic.AddInt32(n, 1)
			wg.Done()
		}()
		// Wait for goroutine 1 and 2
		wg.Wait()
		if atomic.LoadInt32(n) != 2 {
			t.Fatal("Spurious wakeup from Wait")
		}
	}
}

func TestCountingWaitGroupAlign(t *testing.T) {
	type X struct {
		x  byte
		wg CountingWaitGroup
	}
	var x X
	x.wg.Add(1)
	go func(x *X) {
		x.wg.Done()
	}(&x)
	x.wg.Wait()
}

func BenchmarkCountingWaitGroupUncontended(b *testing.B) {
	type PaddedWaitGroup struct {
		CountingWaitGroup
		pad [128]uint8
	}
	b.RunParallel(func(pb *testing.PB) {
		var wg PaddedWaitGroup
		for pb.Next() {
			wg.Add(1)
			wg.Done()
			wg.Wait()
		}
	})
}

func benchmarkCountingWaitGroupAddDone(b *testing.B, localWork int) {
	var wg CountingWaitGroup
	b.RunParallel(func(pb *testing.PB) {
		foo := 0
		for pb.Next() {
			wg.Add(1)
			for i := 0; i < localWork; i++ {
				foo *= 2
				foo /= 2
			}
			wg.Done()
		}
		_ = foo
	})
}

func BenchmarkCountingWaitGroupAddDone(b *testing.B) {
	benchmarkCountingWaitGroupAddDone(b, 0)
}

func BenchmarkCountingWaitGroupAddDoneWork(b *testing.B) {
	benchmarkCountingWaitGroupAddDone(b, 100)
}

func benchmarkCountingWaitGroupWait(b *testing.B, localWork int) {
	var wg CountingWaitGroup
	b.RunParallel(func(pb *testing.PB) {
		foo := 0
		for pb.Next() {
			wg.Wait()
			for i := 0; i < localWork; i++ {
				foo *= 2
				foo /= 2
			}
		}
		_ = foo
	})
}

func BenchmarkCountingWaitGroupWait(b *testing.B) {
	benchmarkCountingWaitGroupWait(b, 0)
}

func BenchmarkCountingWaitGroupWaitWork(b *testing.B) {
	benchmarkCountingWaitGroupWait(b, 100)
}

func BenchmarkCountingWaitGroupActuallyWait(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg CountingWaitGroup
			wg.Add(1)
			go func() {
				wg.Done()
			}()
			wg.Wait()
		}
	})
}
