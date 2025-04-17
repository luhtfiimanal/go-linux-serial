# go-linux-serial

Ultra-low-latency, killable, production-grade serial communication for Go (Linux-only)

## Features
- Interruptible read loop (self-pipe mechanism for instant killability)
- Safe, concurrent read/write on a single connection
- Robust error handling and resource cleanup
- Comprehensive unit tests using PTY simulation
- Simple, modern API for production use

## Installation

```
go get github.com/luhtfiimanal/go-linux-serial
```

## Usage Example

```go
package main

import (
    "fmt"
    "github.com/luhtfiimanal/go-linux-serial"
)

func main() {
    cfg := serial.Config{
        Device: "/dev/ttyUSB0",
        BaudRate: 115200,
        Delimiter: "\r\n",
    }
    sr, err := serial.Open(cfg)
    if err != nil {
        panic(err)
    }
    defer sr.Close()

    go sr.ReadLinesLoop(
        func(line string) { fmt.Println("Line:", line) },
        func(err error) { fmt.Println("Error:", err) },
    )

    sr.WriteLine("C,INFO", cfg.Delimiter)
    // ...
}
```

## License

MIT

---

**Contributions welcome!**

- Please file issues or PRs for bugs, improvements, or new features.
- See `serialreader_test.go` for PTY-based testing examples.
