# dchan

`dchan` is a Go library that provides a dynamic channel with unlimited capacity and additional features for efficient data operations.

## Functions

- `New[T any](params ...int) *dchan.Chan[T]`: Initializes a new dynamic channel.
- `Send(data T)`: Sends a value to the channel.
- `Receive() (T, bool)`: Retrieves a value from the channel, blocking if empty and not closed.
- `TryReceive() (T, bool)`: Retrieves a value from the channel without blocking; returns false if empty.
- `Close(f ...func(T))`: Closes the channel; optionally accepts a function to handle any remaining values.
- `Ready(f func(T) bool) bool`: Checks if the first element in the channel satisfies a predicate, without removing it.
- `Len() int`: Returns the current number of items in the channel.
- `IsClosed() bool`: Checks whether the channel is closed.
- `Out(size ...int) <-chan T`: Returns a receive-only channel for iterating over values.

## Usage

### Creating a new channel

```go
ch := dchan.New[int64]()
// ch := dchan.New[*MyStruct]()
```
By default, the initial capacity is set to 1024 values, with storage implemented as a slice of slices
that automatically expands as it fills. The behavior closely mirrors that of standard Go channels.

Additional configuration options:

```go
ch := dchan.New[int](10240) // initial capacity 10,240
// ch := dchan.New[int](dchan.Relaxed)
// ch := dchan.New[int](10240, dchan.Relaxed)
```
- `10240` - Initial capacity for storing up to 10,240 values (automatically expanding as it fills).
- `dchan.Relaxed` - Permits sending to a closed channel and closing the channel multiple times without causing a panic.
  Use of `dchan.Relaxed` is rare and should be considered carefully.

### Sending and receiving values

```go
ch.Send(1)
val, ok := ch.Receive()
```

### Iterating over channel values

```go
for val, ok := ch.Receive(); ok; val, ok = ch.Receive() {
    // process val
}
```

Or using the `Out` method:

```go
for val := range ch.Out() {
    // process val
}
```

### Closing the channel

```go
ch.Close()
// ch.Close(f)
// ch.Close(nil)
```
Extra parameters are: `func(T)` or `nil`.

- `ch.Close(func(T))` - function to process elements remaining in the channel.
- `ch.Close(nil)` - discards any remaining elements in the channel.

### Checking if the first element is ready

```go
ready := ch.Ready(func(v T) bool { return v == 10 })
// or for structs:
// ready := ch.Ready(func(v MyStruct) bool { return v.Status == READY })
```

### Getting the length of the channel

```go
length := ch.Len()
```

### Checking whether the channel is closed

```go
closed := ch.IsClosed()
```

### A basic example

```go
package main

import (
    "fmt"
    "time"
    "github.com/hitsumitomo/dchan"
)

func utilizeFunc(i int) {
    fmt.Printf("Utilize: %v\n", i)
}

func main() {
    ch := dchan.New[int]()

    // Method 1: Loop with Receive
    go func() {
        i := 0
        for val, ok := ch.Receive(); ok; val, ok = ch.Receive() {
            fmt.Println("Method1:", val)
            if i++; i == 4 {
                break
            }
        }
    }()

    // Method 2: Loop with Receive and break
    go func() {
        i := 0
        for {
            val, ok := ch.Receive()
            if !ok {
                break
            }
            fmt.Println("Method2:", val)
            if i++; i == 4 {
                break
            }
        }
    }()

    for i := 0; i < 10; i++ {
        ch.Send(i)
    }

    time.Sleep(time.Second)
    ch.Close(utilizeFunc)
    time.Sleep(time.Second)

    fmt.Println()
    // ------------------------------
    ch = dchan.New[int](dchan.Relaxed)
    // also allowed:
    // ch = dchan.New[int64](2048)
    // ch = dchan.New[int64](2048, dchan.Relaxed)

    go func() {
        for val, ok := ch.Receive(); ok; val, ok = ch.Receive() {
            fmt.Println("Method3:", val)
        }
        fmt.Println("Channel is closed")
    }()

    for i := 0; i < 5; i++ {
        ch.Send(i)
    }

    time.Sleep(time.Second)
    ch.Close()
    ch.Send(100)
    ch.Close()
    time.Sleep(time.Second)
}
// Method1: 0
// Method1: 2
// Method1: 3
// Method1: 4
// Method2: 1
// Method2: 5
// Method2: 6
// Method2: 7
// Utilize: 8
// Utilize: 9

// Method3: 0
// Method3: 1
// Method3: 2
// Method3: 3
// Method3: 4
// Channel is closed
```

### Benchmarks
<pre>
BenchmarkUnlimitedChannel/SendReceive-2       45708661    27.42 ns/op    11 B/op    0 allocs/op
BenchmarkUnlimitedChannel/ConcurrentSend-2    44617639    38.84 ns/op    12 B/op    0 allocs/op
BenchmarkUnlimitedChannel/ConcurrentReceive-2 45536180    34.15 ns/op    11 B/op    0 allocs/op
BenchmarkUnlimitedChannel/OutRange-2          15693538    77.31 ns/op     8 B/op    0 allocs/op

BenchmarkRegularChannel/SendReceive-2         27289593    43.89 ns/op     0 B/op    0 allocs/op
BenchmarkRegularChannel/ConcurrentSend-2      17264353    64.79 ns/op     0 B/op    0 allocs/op
BenchmarkRegularChannel/ConcurrentReceive-2   24897272    48.82 ns/op     0 B/op    0 allocs/op
BenchmarkRegularChannel/Range-2               26300221    58.73 ns/op     0 B/op    0 allocs/op
</pre>


