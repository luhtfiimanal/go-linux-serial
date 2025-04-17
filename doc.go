// Package serial provides a minimal, Linux-only serial port reader
// designed for high-frequency unbuffered communication with embedded devices.
//
// This package is optimized for real-time use cases such as scientific
// instrumentation (e.g., seismometers), where data arrives with high frequency
// (e.g., 200Hz) and must be read as soon as newline-delimited lines are available.
//
// Features:
//   - Raw syscall-based serial I/O on Linux, no buffering delays
//   - Line-based reading with custom newline (default: \r\n)
//   - Safe for concurrent usage with killability
//   - Self-pipe mechanism for killability
//   - PTY-based tests for reliability
//
// This package does **not** support Windows.
//
// Example usage:
//
//	cfg := serial.Config{
//	    Device:   "/dev/ttyUSB0",
//	    BaudRate:  115200,
//	    Delimiter: "\r\n",
//	}
//	reader, err := serial.Open(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer reader.Close()
//
//	// Start reading lines in a goroutine
//	go func() {
//	    reader.ReadLinesLoop(
//	        func(line string) {
//	            fmt.Println("Received:", line)
//	        },
//	        func(err error) {
//	            log.Println("Read error:", err)
//	        },
//	    )
//	}()
//
//	// Write a command
//	err = reader.WriteLine("C,START", "\r\n")
//	if err != nil {
//	    log.Println("Write failed:", err)
//	}
//
//	// ... to stop reading, call reader.Close() from another goroutine
package serial
