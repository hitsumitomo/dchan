package dchan

import (
	"math"
	"math/bits"
	"sync"
	"sync/atomic"
)

const (
	Relaxed         = 1 << 30
	initialSize     = 1024
	shrinkThreshold = 128 // Number of slice removals before triggering a shrink
)

// Chan is a dynamically growing channel based on a slice of slices ([][]T).
type Chan[T any] struct {
	slices    [][]T // Data storage: each slice grows in size
	headIdx   int   // Read index in the first slice
	tailIdx   int   // Write index in the last slice

	mu        sync.Mutex
	cond      *sync.Cond

	length    int    // Total number of elements in the channel
	closed    uint32 // 1 if closed, 0 otherwise
	relaxed   bool   // If true, disables panics on misuse
	threshold int    // Counter for slice removals to trigger shrinking
}

// New creates a new dynamic channel.
// Usage examples:
//   ch := dchan.New[int]()                   // Default initial size (1024)
//   ch := dchan.New[string](2048)            // Initial size 2048
//   ch := dchan.New[float64](dchan.Relaxed)  // Relaxed mode, default size
//   ch := dchan.New[any](512, dchan.Relaxed) // Relaxed mode, initial size 512
func New[T any](params ...int) *Chan[T] {
	var (
		size    int
		relaxed bool
	)

	for _, param := range params {
		if param & Relaxed > 0 {
			relaxed = true
		}

		s := param & ^Relaxed
		if s > size {
			size = s
		}
	}

	size = nextPowerOfTwo(size)
	if size == 0 {
		size = initialSize
	}

	c := &Chan[T]{
		slices:  [][]T{make([]T, size)},
		relaxed: relaxed,
	}
	c.cond = sync.NewCond(&c.mu)
	return c
}

// Send adds an item to the channel, growing the storage if needed.
func (c *Chan[T]) Send(data T) {
	c.mu.Lock()

	if c.closed == 1 {
		c.mu.Unlock()
		if !c.relaxed {
			panic("send on closed channel")
		}
		return
	}
	last := len(c.slices) - 1

	// Ensure there is space in the last slice for writing
	if last < 0 || c.tailIdx >= len(c.slices[last]) {
		// Grow: double the last slice size or use initialSize if no slices exist
		var newSize int
		if last >= 0 {
			newSize = len(c.slices[last]) * 2
		} else {
			newSize = initialSize
		}
		c.slices = append(c.slices, make([]T, newSize))
		c.tailIdx = 0
		last = len(c.slices) - 1
	}
	c.slices[last][c.tailIdx] = data
	c.tailIdx++
	c.length++

	// trigger condition variable if the channel was empty
	if c.length == 1 {
		c.cond.Signal()
	}
	c.mu.Unlock()
}

// Receive retrieves and removes an item from the channel, blocking if empty and not closed.
// Returns (zero, false) if the channel is closed and empty.
func (c *Chan[T]) Receive() (T, bool) {
	c.mu.Lock()

	for c.length == 0 && c.closed == 0 {
		c.cond.Wait()
	}
	return c.receive() // Unlocks inside
}

// TryReceive retrieves and removes an item from the channel without blocking.
// Returns (zero, false) if the channel is empty.
func (c *Chan[T]) TryReceive() (T, bool) {
	c.mu.Lock()

	if c.length == 0 {
		c.mu.Unlock()
		var zero T
		return zero, false
	}
	return c.receive() // Unlocks inside
}

// receive is the internal implementation for Receive and TryReceive.
// Assumes the lock is already held.
func (c *Chan[T]) receive() (T, bool) {
	if len(c.slices) == 0 || len(c.slices[0]) == 0 {
		c.mu.Unlock()
		var zero T
		return zero, false
	}

	// If channel is closed and empty, return zero value
	if c.closed == 1 && c.length == 0 {
		c.mu.Unlock()
		var zero T
		return zero, false
	}

	val := c.slices[0][c.headIdx]
	c.headIdx++
	c.length--

	// Remove the first slice if it has been fully read
	if c.headIdx >= len(c.slices[0]) {
		if len(c.slices) > 1 {
			c.threshold++
			c.slices[0] = nil
			c.slices = c.slices[1:]

			// Shrink slices if threshold reached
			if c.threshold >= shrinkThreshold {
				c.slices = append([][]T(nil), c.slices...)
				c.threshold = 0
			}

		} else {
			// Only one slice remains, reset indices
			c.tailIdx = 0
		}
		c.headIdx = 0
	}
	isLastRetrieved := c.closed == 1 && c.length == 0
	c.mu.Unlock()

	if isLastRetrieved {
		c.cleanUp()
	}

	return val, true
}

// Ready checks if the first element in the channel satisfies the provided predicate.
// Example: c.Ready(func(item Item) bool { return item.Status == READY })
func (c *Chan[T]) Ready(f func(T) bool) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.length == 0 || len(c.slices) == 0 || c.headIdx >= len(c.slices[0]) {
		return false
	}

	return f(c.slices[0][c.headIdx])
}

// Len returns the current number of items in the channel.
func (c *Chan[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.length
}

// IsClosed returns true if the channel has been closed.
func (c *Chan[T]) IsClosed() bool {
	return atomic.LoadUint32(&c.closed) == 1
}

// Close closes the channel. Optionally drains remaining items using the provided function.
// Panics if already closed, unless in relaxed mode.
func (c *Chan[T]) Close(drainFunc ...func(T)) {
	if atomic.SwapUint32(&c.closed, 1) == 1 {
		if !c.relaxed {
			panic("close of closed channel")
		}
		return
	}
	c.mu.Lock()
	c.cond.Broadcast()
	c.mu.Unlock()
	if len(drainFunc) > 0 && drainFunc[0] != nil {
		for {
			val, ok := c.TryReceive()
			if !ok {
				break
			}
			drainFunc[0](val)
		}
	}
}

// Out returns a receive-only channel that streams items from the dynamic channel.
// The returned channel is buffered with the specified size (default: initialSize).
func (c *Chan[T]) Out(size ...int) <-chan T {
	chanSize := initialSize

	if len(size) > 0 && size[0] > 0 {
		chanSize = size[0]
	}

	outCh := make(chan T, chanSize)
	go func() {
		defer close(outCh)
		for {
			val, ok := c.Receive()
			if !ok {
				return // Channel closed and empty
			}
			outCh <- val
		}
	}()
	return outCh
}

// nextPowerOfTwo returns the next power of two greater than or equal to val.
func nextPowerOfTwo(val int) int {
	if val <= initialSize {
		return initialSize
	}
	if val > math.MaxInt/2 {
		return math.MaxInt
	}
	v := uint(val - 1)
	return 1 << bits.Len(v)
}

// cleanUp releases all resources used by the channel.
// Should be called only after Close() and after all elements have been received.
func (c *Chan[T]) cleanUp() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.slices = nil
	c.length, c.headIdx, c.tailIdx = 0, 0, 0
}
