package dchan

import (
	"sync"
	"testing"

	"github.com/hitsumitomo/dchan"
)

func TestChanModeSendReceive(t *testing.T) {
	var wg sync.WaitGroup
	ch := dchan.New[int](100)

	// Test sending and receiving in single-threaded mode
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	for i := 0; i < 100; i++ {
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}

	// Test sending and receiving in multi-threaded mode
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			ch.Send(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			val, ok := ch.Receive()
			if !ok || val != i {
				t.Errorf("Expected %d, got %d", i, val)
			}
		}
	}()
	wg.Wait()
}

func TestChanModeClose(t *testing.T) {
	ch := dchan.New[int](100)

	// Test Close without parameter
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	ch.Close()
	for i := 0; i < 100; i++ {
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}

	// Test Close with function parameter
	ch = dchan.New[int](100)
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	var received []int
	ch.Close(func(v int) {
		received = append(received, v)
	})
	if len(received) != 100 {
		t.Errorf("Expected received length to be 100, got %d", len(received))
	}
	for i := 0; i < len(received); i++ {
		if received[i] != i {
			t.Errorf("Expected %d, got %d", i, received[i])
		}
	}

	// Test Close with nil parameter
	ch = dchan.New[int](100)
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	// time.Sleep(2 * time.Second) // wait unitl all values are utilized
	ch.Close(nil)
	for i := 0; i < 100; i++ {
		_, ok := ch.Receive()
		if ok {
			t.Errorf("Expected channel to be closed")
		}
	}
}

func TestChanModeHugeData(t *testing.T) {
	var wg sync.WaitGroup
	ch := dchan.New[int](100)

	// Test high volume in multi-threaded mode
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			ch.Send(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			val, ok := ch.Receive()
			if !ok || val != i {
				t.Errorf("Expected %d, got %d", i, val)
			}
		}
	}()
	wg.Wait()
}

func TestChanModeRelaxed(t *testing.T) {
	var wg sync.WaitGroup
	ch := dchan.New[int](100, dchan.Relaxed)

	// Test sending and receiving in single-threaded mode
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	for i := 0; i < 100; i++ {
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}

	// Test sending and receiving in multi-threaded mode
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			ch.Send(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			val, ok := ch.Receive()
			if !ok || val != i {
				t.Errorf("Expected %d, got %d", i, val)
			}
		}
	}()
	wg.Wait()
}

func TestChanModeRelaxedClose(t *testing.T) {
	ch := dchan.New[int](100, dchan.Relaxed)

	for i := 0; i < 100; i++	{
		ch.Send(i)
	}
	ch.Close()
	// sending to closed channel should not panic
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	for i := 0; i < 100; i++ {
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}
	// closing closed channel should not panic
	ch.Close()
}

func TestChanModeLenIsClosed(t *testing.T) {
	ch := dchan.New[int](100, dchan.Relaxed)

	// Test Len and IsClosed without parameter
	for i := 0; i < 100	; i++ {
		ch.Send(i)
	}
	if ch.Len() != 100 {
		t.Errorf("Expected length to be 100, got %d", ch.Len())
	}
	if ch.IsClosed() {
		t.Errorf("Expected channel to be not closed")
	}
	ch.Close()
	if ch.Len() != 100 {
		t.Errorf("Expected length to be 100, got %d", ch.Len())
	}
	if !ch.IsClosed() {
		t.Errorf("Expected channel to be closed")
	}

	// Test Len and IsClosed with Close parameter
	ch = dchan.New[int](100, dchan.Relaxed)
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	if ch.Len() != 100 {
		t.Errorf("Expected length to be 100, got %d", ch.Len())
	}
	if ch.IsClosed() {
		t.Errorf("Expected channel to be not closed")
	}
	ch.Close(nil)
	if ch.Len() != 0 {
		t.Errorf("Expected length to be 0, got %d", ch.Len())
	}
}
func TestChanModeRange(t *testing.T) {
	ch := dchan.New[int](100, dchan.Relaxed)

	for i := 0; i < 100	; i++ {
		ch.Send(i)
	}
	i:=0
	for v,ok := ch.Receive(); ok; v,ok = ch.Receive() {
		if v != i {
			t.Errorf("Expected %d, got %d", i, v)
		}
		i++
		if ch.Len() == 0 {
			ch.Close()
		}
	}
}
// ---------------------------------------------------------------------- SliceMode
func TestSliceModeSendReceive(t *testing.T) {
	var wg sync.WaitGroup
	ch := dchan.New[int](100, dchan.SliceMode)

	// Test sending and receiving in single-threaded mode
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	for i := 0; i < 100; i++ {
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}

	// Test sending and receiving in multi-threaded mode
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			ch.Send(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			val, ok := ch.Receive()
			if !ok || val != i {
				t.Errorf("Expected %d, got %d", i, val)
			}
		}
	}()
	wg.Wait()
}

func TestSliceModeClose(t *testing.T) {
	ch := dchan.New[int](100, dchan.SliceMode)

	// Test Close without parameter
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	ch.Close()
	for i := 0; i < 100; i++ {
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}

	// Test Close with function parameter
	ch = dchan.New[int](100, dchan.SliceMode)
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	var received []int
	ch.Close(func(v int) {
		received = append(received, v)
	})
	if len(received) != 100 {
		t.Errorf("Expected received length to be 100, got %d", len(received))
	}
	for i := 0; i < len(received); i++ {
		if received[i] != i {
			t.Errorf("Expected %d, got %d", i, received[i])
		}
	}

	// Test Close with nil parameter
	ch = dchan.New[int](100, dchan.SliceMode)
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	// time.Sleep(2 * time.Second) // wait unitl all values are utilized
	ch.Close(nil)
	for i := 0; i < 100; i++ {
		_, ok := ch.Receive()
		if ok {
			t.Errorf("Expected channel to be closed")
		}
	}
}

func TestSliceModeHugeData(t *testing.T) {
	var wg sync.WaitGroup
	ch := dchan.New[int](100, dchan.SliceMode)

	// Test high volume in multi-threaded mode
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			ch.Send(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			val, ok := ch.Receive()
			if !ok || val != i {
				t.Errorf("Expected %d, got %d", i, val)
			}
		}
	}()
	wg.Wait()
}

func TestSliceModeRelaxed(t *testing.T) {
	var wg sync.WaitGroup
	ch := dchan.New[int](100, dchan.SliceMode | dchan.Relaxed)

	// Test sending and receiving in single-threaded mode
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	for i := 0; i < 100; i++ {
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}

	// Test sending and receiving in multi-threaded mode
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			ch.Send(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			val, ok := ch.Receive()
			if !ok || val != i {
				t.Errorf("Expected %d, got %d", i, val)
			}
		}
	}()
	wg.Wait()
}

func TestSliceModeRelaxedClose(t *testing.T) {
	ch := dchan.New[int](100, dchan.SliceMode | dchan.Relaxed)

	for i := 0; i < 100; i++	{
		ch.Send(i)
	}
	ch.Close()
	// sending to closed channel should not panic
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	for i := 0; i < 100; i++ {
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}
	// closing closed channel should not panic
	ch.Close()
}

func readyFunc(v int) bool {
	return v == 10
}
func TestSliceModeReady(t *testing.T) {
	ch := dchan.New[int](100, dchan.SliceMode)

	// Test Ready without parameter
	for i := 0; i < 100	; i++ {
		ch.Send(i)
	}
	for i := 0; i < 100; i++ {
		if i == 10 && !ch.Ready(readyFunc) {
			t.Errorf("Expected channel to be ready")
		}
		val, ok := ch.Receive()
		if !ok || val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}
	if ch.Ready(readyFunc) {
		t.Errorf("Expected Ready(readyFunc) to be false")
	}
}
func TestSliceModeLenIsClosed(t *testing.T) {
	ch := dchan.New[int](100, dchan.SliceMode | dchan.Relaxed)

	// Test Len and IsClosed without parameter
	for i := 0; i < 100	; i++ {
		ch.Send(i)
	}
	if ch.Len() != 100 {
		t.Errorf("Expected length to be 100, got %d", ch.Len())
	}
	if ch.IsClosed() {
		t.Errorf("Expected channel to be not closed")
	}
	ch.Close()
	if ch.Len() != 100 {
		t.Errorf("Expected length to be 100, got %d", ch.Len())
	}
	if !ch.IsClosed() {
		t.Errorf("Expected channel to be closed")
	}

	// Test Len and IsClosed with Close parameter
	ch = dchan.New[int](100, dchan.SliceMode | dchan.Relaxed)
	for i := 0; i < 100; i++ {
		ch.Send(i)
	}
	if ch.Len() != 100 {
		t.Errorf("Expected length to be 100, got %d", ch.Len())
	}
	if ch.IsClosed() {
		t.Errorf("Expected channel to be not closed")
	}
	ch.Close(nil)
	if ch.Len() != 0 {
		t.Errorf("Expected length to be 0, got %d", ch.Len())
	}
}
func TestSliceModeRange(t *testing.T) {
	ch := dchan.New[int](100, dchan.SliceMode | dchan.Relaxed)

	for i := 0; i < 100	; i++ {
		ch.Send(i)
	}
	i:=0
	for v,ok := ch.Receive(); ok; v,ok = ch.Receive() {
		if v != i {
			t.Errorf("Expected %d, got %d", i, v)
		}
		i++
		if ch.Len() == 0 {
			ch.Close()
		}
	}
	if ch.Ready(readyFunc) {
		t.Errorf("Expected channel to be not ready")
	}
}
// ---------------------------------------------------------------------- /SliceMode
func BenchmarkDchanSend(b *testing.B) {
    dc := dchan.New[int]()
    b.ResetTimer()

    for i := 0; i < 1024000; i++ {
        dc.Send(i)
    }
}

func BenchmarkDchanReceive(b *testing.B) {
    dc := dchan.New[int]()
    for i := 0; i < 1024000; i++ {
        dc.Send(i) // Pre-fill the channel
    }
    b.ResetTimer()

    for i := 0; i < 1024000; i++ {
        dc.Receive()
    }
}

func BenchmarkDchanConcurrent(b *testing.B) {
    dc := dchan.New[int]()
    done := make(chan struct{})

    go func() {
        for {
            select {
            case <-done:
                return
            default:
                dc.Receive()
            }
        }
    }()

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            dc.Send(1)
        }
    })

    close(done)
    dc.Close()
}

func BenchmarkDchanSliceModeSend(b *testing.B) {
    dc := dchan.New[int](dchan.SliceMode)
    b.ResetTimer()

    for i := 0; i < 1024000; i++ {
        dc.Send(i)
    }
}

func BenchmarkDchanSliceModeReceive(b *testing.B) {
    dc := dchan.New[int](dchan.SliceMode)
    for i := 0; i < 1024000; i++ {
        dc.Send(i)
    }
    b.ResetTimer()

    for i := 0; i < 1024000; i++ {
        dc.Receive()
    }
}

func BenchmarkDchanSliceModeConcurrent(b *testing.B) {
    dc := dchan.New[int](dchan.SliceMode)
    done := make(chan struct{})

    go func() {
        for {
            select {
            case <-done:
                return
            default:
                dc.Receive()
            }
        }
    }()

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            dc.Send(1)
        }
    })

    close(done)
    dc.Close()
}

func BenchmarkChannelSend(b *testing.B) {
    ch := make(chan int, 1024000)
    b.ResetTimer()

    for i := 0; i < 1024000; i++ {
        ch <- i
    }
    close(ch)
}

func BenchmarkChannelReceive(b *testing.B) {
    ch := make(chan int, 1024000)
    for i := 0; i < 1024000; i++ {
        ch <- i // Pre-fill the channel
    }
    close(ch)
    b.ResetTimer()

    for i := 0; i < 1024000; i++ {
        <- ch
    }
}

func BenchmarkChannelConcurrent(b *testing.B) {
    ch := make(chan int, 1000)
    done := make(chan struct{})

    go func() {
        for {
            select {
            case <-ch:
            case <-done:
                return
            }
        }
    }()

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            ch <- 1
        }
    })

    close(done)
    close(ch)
}

