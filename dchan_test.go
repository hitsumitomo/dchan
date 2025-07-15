package dchan

import (
	"runtime"
	"sync"
	"testing"
)

func TestUnlimitedChannel(t *testing.T) {
	c := New[int]()

	// Test Send and Receive
	c.Send(42)
	val, ok := c.Receive()
	if !ok || val != 42 {
		t.Errorf("expected 42, got %v (ok: %v)", val, ok)
	}

	// Test Len
	if c.Len() != 0 {
		t.Errorf("expected length 0, got %d", c.Len())
	}

	// Test Close
	c.Send(100)
	c.Close()
	val, ok = c.Receive()
	if !ok || val != 100 {
		t.Errorf("expected 100 after channel is closed, got %v (ok: %v)", val, ok)
	}
	_, ok = c.Receive()
	if ok {
		t.Errorf("expected no more items after all items are consumed")
	}

	// Test panic on Send after Close in strict mode
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on send to closed channel")
		}
	}()
	c.Send(200)
}

func TestUnlimitedChannel_Out(t *testing.T) {
	c := New[int]()
	outCh := c.Out()

	var received []int
	var wg sync.WaitGroup
	wg.Add(1)

	// Goroutine to receive from the Out() channel
	go func() {
		defer wg.Done()
		for val := range outCh {
			received = append(received, val)
		}
	}()

	// Send some data
	c.Send(1)
	c.Send(2)
	c.Send(3)

	// Close the dchan, which should eventually close outCh
	c.Close()

	// Wait for the receiver goroutine to finish
	wg.Wait()

	// Verify received data
	expected := []int{1, 2, 3}
	if len(received) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(received))
	}
	for i, v := range expected {
		if received[i] != v {
			t.Errorf("expected %d at index %d, got %d", v, i, received[i])
		}
	}
}

func BenchmarkUnlimitedChannel(b *testing.B) {
	b.Run("SendReceive", func(b *testing.B) {
		// Для каждого прогона бенчмарка создаем новый канал
		c := New[int]()
		wg := sync.WaitGroup{}

		b.ResetTimer() // Сбрасываем таймер, чтобы не учитывать инициализацию
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				c.Send(i)
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				c.Receive()
			}
		}()
		wg.Wait()
	})

	b.Run("ConcurrentSend", func(b *testing.B) {
		c := New[int]()
		// Создаем канал для сигнализации завершения получателям
		done := make(chan struct{})

		// Запускаем несколько горутин получателей, которые непрерывно считывают данные
		for i := 0; i < 4; i++ {
			go func() {
				for {
					select {
					case <-done:
						return
					default:
						c.Receive()
					}
				}
			}()
		}

		// Даем получателям время запуститься
		runtime.Gosched()

		// Используем RunParallel для параллельной отправки данных
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.Send(1)
			}
		})

		// Останавливаем получателей
		close(done)
	})

	b.Run("ConcurrentReceive", func(b *testing.B) {
		c := New[int]()
		wg := sync.WaitGroup{}

		// Округляем до кратного 4, как в тесте регулярного канала
		n := b.N / 4 * 4

		// Запускаем получателей заранее
		b.ResetTimer()
		wg.Add(4)
		for i := 0; i < 4; i++ {
			go func() {
				defer wg.Done()
				// Loop until channel is closed and empty
				for {
					_, ok := c.Receive()
					if !ok {
						break // Exit loop when channel is closed and empty
					}
				}
			}()
		}

		// Отправляем данные
		for i := 0; i < n; i++ {
			c.Send(i)
		}
		// Close the channel after sending all items
		c.Close()
		wg.Wait()
	})

	b.Run("OutRange", func(b *testing.B) {
		c := New[int]()
		outCh := c.Out()
		wg := sync.WaitGroup{}

		b.ResetTimer()
		wg.Add(1)
		// Sender goroutine
		go func() {
			for i := 0; i < b.N; i++ {
				c.Send(i)
			}
			c.Close() // Close dchan after sending
		}()

		// Receiver (main benchmark goroutine) using range on Out() channel
		go func() {
			defer wg.Done()
			for range outCh {
				// Consume items
			}
		}()
		wg.Wait() // Wait for the receiver to finish processing all items
	})
}

func BenchmarkRegularChannel(b *testing.B) {
	b.Run("SendReceive", func(b *testing.B) {
		// Создаем новый канал для каждого прогона
		ch := make(chan int, 1024)
		wg := sync.WaitGroup{}

		b.ResetTimer()
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				ch <- i
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				<-ch
			}
		}()
		wg.Wait()
	})

	b.Run("ConcurrentSend", func(b *testing.B) {
		ch := make(chan int, 1024)
		done := make(chan struct{})

		// Запускаем несколько горутин получателей
		for i := 0; i < 4; i++ {
			go func() {
				for {
					select {
					case <-done:
						return
					default:
						<-ch
					}
				}
			}()
		}

		// Даем получателям время запуститься
		runtime.Gosched()

		// Используем RunParallel для параллельной отправки данных
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ch <- 1
			}
		})

		// Останавливаем получателей
		close(done)
	})

	b.Run("ConcurrentReceive", func(b *testing.B) {
		ch := make(chan int, 1024)
		wg := sync.WaitGroup{}

		n := b.N / 4 * 4 // округляем вниз до ближайшего числа, кратного 4

		b.ResetTimer()
		wg.Add(4)
		for i := 0; i < 4; i++ {
			go func() {
				defer wg.Done()
				// Loop until channel is closed and empty
				for range ch {
					// Consume items
				}
			}()
		}
		for i := 0; i < n; i++ {
			ch <- i
		}
		// Close the channel after sending all items
		close(ch)
		wg.Wait()
	})

	b.Run("Range", func(b *testing.B) {
		ch := make(chan int, 1024) // Use a buffer similar to other tests
		wg := sync.WaitGroup{}

		b.ResetTimer()
		wg.Add(1)
		// Sender goroutine
		go func() {
			for i := 0; i < b.N; i++ {
				ch <- i
			}
			close(ch) // Close the channel after sending
		}()

		// Receiver (main benchmark goroutine) using range
		go func() {
			defer wg.Done()
			for range ch {
				// Consume items
			}
		}()
		wg.Wait() // Wait for the receiver to finish processing all items
	})
}
