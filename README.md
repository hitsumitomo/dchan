# dchan

`dchan` is a Go library that provides a dynamic channel with unlimited capacity and additional features for more efficient data operations.

## Functions

- `New[T any](params ...any) *C[T]`: Initializes a new channel.
- `Send(data T)`: Sends a value to the channel.
- `Receive() (T, bool)`: Retrieves a value from the channel.
- `Close(f ...func(T)) `: Closes the channel; optionally accepts a function to handle any remaining values or nil to discard them.
- `Ready(f func(T) bool) bool`: Checks if the first element in the channel is ready for retrieval based on a custom boolean function, without actually removing it from the channel (available only in SliceMode).
- `Len() int `: Returns the current length of the channel.
- `IsClosed() bool`: Checks whether the channel is closed.
## Usage

### Creating a new channel ([T any])

```go
ch := dchan.New[int64]()
// ch := dchan.New[*MyStruct]()
```
By default, the initial capacity is set to 1024 values, with storage implemented as a slice of Go channels
that automatically expands as it fills. The behavior closely mirrors that of standard Go channels.

Additional configuration options:

```go
ch := dchan.New[int](10240) // type *dchan.C
// ch := dchan.New[int](dchan.Relaxed)
// ch := dchan.New[int](10240, dchan.Relaxed)
```
- `10240` - Initial capacity for storing up to 10,240 values (automatically expanding as it fills).
- `flag dchan.SliceMode` - will use slices for storage.
- `flag dchan.Relaxed` - Permits sending to a closed channel and closing the channel multiple times without causing a panic.
Note that while correct uses of `dchan.Relaxed` do exist, they are rare, and use of `dchan.Relaxed` is often a sign of a deeper problem in the program's logic.
### Sending and receiving values

```go
ch.Send(1)
val, ok := ch.Receive()

```
### Iterating over channel values

```go
// Equivalent to standard channel reading: for val := range ch { ... }
for val,ok := ch.Receive(); ok; val,ok = ch.Receive() {
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

- `ch.Close(func(T))` - function to process (or utilize) elements remaining in the channel.
- `ch.Close(nil)` - discards any remaining elements in the channel.

### Determines if the first element in the channel is ready (SliceMode only)

```go
ready := ch.Ready(func(v T) bool { return v.Status == READY })

// func readyFunc(v int) bool {
// 	return v == 10
// }
// ...
// if ch.Ready(readyFunc) {
//     ...
// }
```

### Getting the length of the channel

```go
length := ch.Len()
```

### Checking whether the channel is closed.

```go
closed := ch.IsClosed()
```

### A basic example:
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

    // Method1 (Equivalent to standard channel reading: for val := range ch { ... })
    go func(){
        i := 0
        for val,ok := ch.Receive(); ok; val,ok = ch.Receive() {
            fmt.Println("Method1:", val)
            if i++; i == 4 {
                break
            }
        }
    }()

    // Method2
    go func(){
        i := 0
        for {
            val,ok := ch.Receive()
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
    ch = dchan.New[int](dchan.SliceMode | dchan.Relaxed)
    // also allowed:
    // ch = dchan.New[int64](2048)
    // ch = dchan.New[int64](2048, dchan.Relaxed)

    go func(){
        for val,ok := ch.Receive(); ok; val,ok = ch.Receive() {
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
BenchmarkDchanSend-4                    1000000000        0.06726 ns/op    0 B/op     0 allocs/op
BenchmarkDchanReceive-4                 1000000000        0.06754 ns/op    0 B/op     0 allocs/op
BenchmarkDchanConcurrent-4                 6965311      172.7 ns/op        9 B/op     0 allocs/op
BenchmarkDchanSliceModeSend-4           1000000000        0.03978 ns/op    0 B/op     0 allocs/op
BenchmarkDchanSliceModeReceive-4        1000000000        0.02774 ns/op    0 B/op     0 allocs/op
BenchmarkDchanSliceModeConcurrent-4       11675958      125.3 ns/op       16 B/op     0 allocs/op
BenchmarkChannelSend-4                  1000000000        0.03446 ns/op    0 B/op     0 allocs/op
BenchmarkChannelReceive-4               1000000000        0.03369 ns/op    0 B/op     0 allocs/op
BenchmarkChannelConcurrent-4               3614706      328.8 ns/op        0 B/op     0 allocs/op
</pre>


