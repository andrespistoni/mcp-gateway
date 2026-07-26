package proxy

import (
	"io"
	"sync"
)

const StderrLimit = 64 * 1024

type StderrSnapshot struct {
	Data      []byte
	Truncated bool
}

type stderrCapture struct {
	mu        sync.Mutex
	data      []byte
	truncated bool
}

func (c *stderrCapture) drain(reader io.Reader) {
	buffer := make([]byte, 32*1024)
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			c.append(buffer[:count])
		}
		if err != nil {
			return
		}
	}
}

func (c *stderrCapture) append(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	remaining := StderrLimit - len(c.data)
	if remaining > 0 {
		if len(data) < remaining {
			remaining = len(data)
		}
		c.data = append(c.data, data[:remaining]...)
	}
	if remaining < len(data) {
		c.truncated = true
	}
}

func (c *stderrCapture) snapshot() StderrSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return StderrSnapshot{Data: append([]byte(nil), c.data...), Truncated: c.truncated}
}
