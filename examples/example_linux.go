//go:build linux
// +build linux

package main

import (
	"fmt"
	"log"
	"time"

	serial "github.com/luhtfiimanal/go-linux-serial"
)

func main() {
	cfg := serial.Config{
		Device:    "/dev/ttyUSB0", // Change to your serial device
		BaudRate:  115200,
		Delimiter: "\r\n",
	}
	r, err := serial.Open(cfg)
	if err != nil {
		log.Fatalf("Failed to open serial: %v", err)
	}
	defer r.Close()

	go r.ReadLinesLoop(
		func(line string) {
			fmt.Printf("Received: %s\n", line)
		},
		func(err error) {
			log.Printf("Read error: %v\n", err)
		},
	)

	// Example write
	err = r.WriteLine("C,INFO", cfg.Delimiter)
	if err != nil {
		log.Printf("Write error: %v\n", err)
	}

	time.Sleep(3 * time.Second) // let it read for a while
}
