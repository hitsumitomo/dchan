package dchan

import (
	"sync"
)

const (
	// Strict mode will panic on send on a closed channel.
	Strict  = byte(0b00000001)
	// Relaxed mode will ignore send on a closed channel.
	Relaxed = byte(0b00000010)
	// SliceMode will use slices for storage.
	SliceMode = byte(0b00000100)
	// initialSize defines the starting capacity of the internal buffer.
	initialSize = 1024
)

// C is a dynamically growing channel.
type C[T any] struct {
	buffer   [][]T
	channels []chan T
	index   int
	woff    int
	roff    int
	size    int
	closed  bool
	cond    *sync.Cond
	length  int
    relaxed bool
	sliceMode bool
}

// New creates a new dynamic channel.
func New[T any](params ...any) *C[T] {
	s := 1024

	c := &C[T]{
		cond: sync.NewCond(&sync.Mutex{}),
	}
	for _, param := range params {
		switch v := param.(type) {
		case int:
			if v > initialSize {
                s = v / 8 * 8
			}

		case byte:
			if v & Relaxed > 0 {
				c.relaxed = true
			}
			if v & SliceMode > 0 {
				c.sliceMode = true
			}
		}
	}
	if c.sliceMode {
        c.buffer = append(make([][]T, 0), make([]T, s))
    } else {
		c.channels = append(make([]chan T, 0), make(chan T, s))
	}
	c.size = s
	return c
}

// Send adds an item to the dynamic channel.
func (c *C[T]) Send(data T) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	if c.closed {
		if !c.relaxed {
			panic("send on closed channel")
		}
		return
	}

	if c.sliceMode {
		if c.woff == c.size {
			c.buffer = append(c.buffer, make([]T, c.size * 2))
			c.index++
			c.woff = 0
		}
		c.buffer[c.index][c.woff] = data
		c.woff++

	} else {
        select {
        case c.channels[c.index] <- data:
        default:
            c.channels = append(c.channels, make(chan T, c.size))
            c.index++
            c.channels[c.index] <- data
        }
	}
	c.length++
	c.cond.Signal()
}

// Receive retrieves an item from the dynamic channel.
// It returns the item and a boolean indicating whether the retrieval was successful.
func (c *C[T]) Receive() (T, bool) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	for c.length == 0 && !c.closed {
		c.cond.Wait()
	}

	if c.length == 0 && c.closed {
		var zero T
		return zero, false
	}

	var val T
	if c.sliceMode {
		val = c.buffer[0][c.roff]
		c.roff++
		if c.roff == c.size {
			if c.index > 0 {
				c.buffer = append(c.buffer[:0], c.buffer[1:]...)
			}
			c.index--
			c.roff = 0
		}
	} else {
        select {
        case val = <- c.channels[0]:
        default:
            close(c.channels[0])
            c.channels = append(c.channels[:0], c.channels[1:]...)
            c.index--
            val = <- c.channels[0]
        }
	}
	c.length--
	return val, true
}

// Ready checks if item is ready for extraction. Note: works only with sliceMode mode.
func (c *C[T]) Ready(f func(T) bool) bool {
	if f == nil || !c.sliceMode {
		return false
	}
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	if c.length > 0 {
		return f(c.buffer[0][c.roff])
	}
	return false
}

// Len returns the number of items in the dynamic channel.
func (c *C[T]) Len() int {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.length
}

// IsClosed checks if the dynamic channel is closed.
func (c *C[T]) IsClosed() bool {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.closed
}

// Close closes the dynamic channel.
// If function is provided, it will be be executed for each remaining item in the channel.
// If nil is provided, the items will be discarded.
func (c *C[T]) Close(f ...func(T)) {
    c.cond.L.Lock()
    if c.closed {
        c.cond.L.Unlock()
        if !c.relaxed {
            panic("close of closed channel")
        }
        return
    }
    c.closed = true
    c.cond.Broadcast()
    c.cond.L.Unlock()

	if len(f) > 0 {
		for {
			val,ok := c.Receive()
			if !ok {
				return
			}
			if f[0] != nil {
				f[0](val)
			}
		}
	}
}
