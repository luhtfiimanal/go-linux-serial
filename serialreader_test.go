package serial

import (
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/require"
)

func TestSerialReader_BasicRead(t *testing.T) {
	// 1. Create a PTY pair (master/slave)
	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { master.Close(); slave.Close() })

	// 2. Configure SerialReader to use the slave path
	cfg := Config{
		Device:    slave.Name(),
		BaudRate:  115200,
		Delimiter: "\n",
	}
	reader, err := Open(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { reader.Close() })

	// 3. Start ReadLinesLoop in a goroutine
	lines := make(chan string, 1)
	errors := make(chan error, 1)
	go reader.ReadLinesLoop(
		func(line string) { lines <- line },
		func(err error) { errors <- err },
	)

	// 4. Write data to master
	_, err = master.Write([]byte("hello\n"))
	require.NoError(t, err)

	// 5. Expect to receive the line promptly
	select {
	case l := <-lines:
		require.Equal(t, "hello", l)
	case err := <-errors:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for line")
	}
}

func TestSerialReader_Killability(t *testing.T) {
	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { master.Close(); slave.Close() })

	cfg := Config{
		Device:    slave.Name(),
		BaudRate:  115200,
		Delimiter: "\n",
	}
	reader, err := Open(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { reader.Close() })

	done := make(chan struct{})
	go func() {
		reader.ReadLinesLoop(
			func(line string) {},
			func(err error) {},
		)
		close(done)
	}()

	// Give the goroutine a chance to block
	time.Sleep(10 * time.Millisecond)
	// Now close the reader, which should unblock the loop
	err = reader.Close()
	require.NoError(t, err)

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for ReadLinesLoop to exit after Close")
	}
}

func TestSerialReader_ErrorPropagation(t *testing.T) {
	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { master.Close(); slave.Close() })

	cfg := Config{
		Device:    slave.Name(),
		BaudRate:  115200,
		Delimiter: "\n",
	}
	reader, err := Open(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { reader.Close() })

	errors := make(chan error, 1)
	go reader.ReadLinesLoop(
		func(line string) {},
		func(err error) { errors <- err },
	)

	// Simulate device disconnect by closing master
	require.NoError(t, master.Close())

	select {
	case err := <-errors:
		require.Error(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for error after device disconnect")
	}
}
