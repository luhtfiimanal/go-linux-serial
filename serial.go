package serial

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// SerialReader provides low-latency, killable, line-oriented access to a Linux serial port.
// It is safe for concurrent use by multiple goroutines.
type SerialReader struct {
	fd        int
	file      *os.File
	done      chan struct{}
	closeOnce sync.Once
	config    Config
	pipeR     int // self-pipe read fd
	pipeW     int // self-pipe write fd
}

// Config holds configuration parameters for opening a serial port.
type Config struct {
	Device      string
	BaudRate    int
	Delimiter   string // default "\r\n"
	ReadTimeout time.Duration
}

// Open opens a serial port using the provided Config and returns a SerialReader.
// The port is configured for raw, low-latency, non-buffered operation.
func Open(cfg Config) (*SerialReader, error) {
	fd, err := syscall.Open(cfg.Device, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, fmt.Errorf("open failed: %w", err)
	}

	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, fmt.Errorf("get termios: %w", err)
	}

	// Raw mode
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8

	// Baud rate
	baud := baudToUnix(cfg.BaudRate)
	termios.Cflag &^= unix.CBAUD
	termios.Cflag |= baud

	// Set VMIN=1, VTIME=0 for immediate, non-blocking reads
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		return nil, fmt.Errorf("set termios: %w", err)
	}

	// Turn back into blocking mode now that config is done
	syscall.SetNonblock(fd, false)

	// Create self-pipe for killability
	pipeFds := make([]int, 2)
	if err := unix.Pipe(pipeFds); err != nil {
		return nil, fmt.Errorf("pipe: %w", err)
	}

	file := os.NewFile(uintptr(fd), cfg.Device)
	return &SerialReader{
		fd:        fd,
		file:      file,
		done:      make(chan struct{}),
		closeOnce: sync.Once{},
		config:    cfg,
		pipeR:     pipeFds[0],
		pipeW:     pipeFds[1],
	}, nil
}

// WriteLine writes a line (with specified newline) to the serial port.
func (s *SerialReader) WriteLine(line string, newline string) error {
	_, err := s.file.WriteString(line + newline)
	return err
}

// ReadLine reads a line using a custom buffer, avoiding bufio for lowest latency.
// ReadLine reads a single line from the serial port, blocking until a full line is received or an error occurs.
// The delimiter is specified in Config. This avoids bufio for lowest latency.
func (s *SerialReader) ReadLine() (string, error) {
	buf := make([]byte, 4096)
	line := ""
	for {
		// Use poll to wait for data or kill signal
		pfd := []unix.PollFd{
			{Fd: int32(s.fd), Events: unix.POLLIN},
			{Fd: int32(s.pipeR), Events: unix.POLLIN},
		}
		_, err := unix.Poll(pfd, -1)
		if err != nil {
			return "", err
		}
		// Check killability
		select {
		case <-s.done:
			return "", fmt.Errorf("serialreader closed")
		default:
		}
		if pfd[1].Revents&unix.POLLIN != 0 {
			// Drain pipe
			var b [1]byte
			unix.Read(s.pipeR, b[:])
			return "", fmt.Errorf("serialreader closed")
		}
		if pfd[0].Revents&unix.POLLIN != 0 {
			n, err := s.file.Read(buf)
			if err != nil {
				return "", err
			}
			line += string(buf[:n])
			if idx := strings.Index(line, s.config.Delimiter); idx >= 0 {
				result := line[:idx]
				return result, nil
			}
		}
	}
}

// ReadLinesLoop reads lines with lowest latency, using poll and custom buffer, and reports errors immediately.
// ReadLinesLoop continuously reads lines from the serial port and invokes onLine for each complete line.
// If an error occurs, onError is called and the loop exits.
func (s *SerialReader) ReadLinesLoop(onLine func(string), onError func(error)) {
	buf := make([]byte, 4096)
	line := ""
	for {
		// Use poll to wait for data or kill signal
		pfd := []unix.PollFd{
			{Fd: int32(s.fd), Events: unix.POLLIN},
			{Fd: int32(s.pipeR), Events: unix.POLLIN},
		}
		_, err := unix.Poll(pfd, -1)
		if err != nil {
			onError(err)
			return
		}
		// Check killability
		select {
		case <-s.done:
			return
		default:
		}
		if pfd[1].Revents&unix.POLLIN != 0 {
			// Drain pipe
			var b [1]byte
			unix.Read(s.pipeR, b[:])
			return
		}
		if pfd[0].Revents&unix.POLLIN != 0 {
			n, err := s.file.Read(buf)
			if err != nil {
				onError(err)
				return
			}
			line += string(buf[:n])
			for {
				idx := strings.Index(line, s.config.Delimiter)
				if idx < 0 {
					break
				}
				onLine(line[:idx])
				line = line[idx+len(s.config.Delimiter):]
			}
		}
	}
}

// Close closes the serial port and unblocks any ReadLine/ReadLinesLoop calls.
// Safe to call multiple times; subsequent calls are no-ops.
func (s *SerialReader) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.done)
		// Wake up poll using self-pipe
		if s.pipeW > 0 {
			unix.Write(s.pipeW, []byte{1})
		}
		if s.file != nil {
			err = s.file.Close()
		}
		syscall.Close(s.fd)
		if s.pipeR > 0 {
			unix.Close(s.pipeR)
		}
		if s.pipeW > 0 {
			unix.Close(s.pipeW)
		}
	})
	return err
}

func baudToUnix(baud int) uint32 {
	switch baud {
	case 9600:
		return unix.B9600
	case 19200:
		return unix.B19200
	case 38400:
		return unix.B38400
	case 57600:
		return unix.B57600
	case 115200:
		return unix.B115200
	case 230400:
		return unix.B230400
	default:
		return unix.B115200 // fallback
	}
}
