package mcp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

const MaxMessageSize = 1 << 20

type Codec struct {
	reader *bufio.Reader
	writer io.Writer
}

func NewCodec(reader io.Reader, writer io.Writer) *Codec {
	return &Codec{reader: bufio.NewReaderSize(reader, 32*1024), writer: writer}
}

func (c *Codec) Read() (Envelope, error) {
	line, err := c.readLine()
	if err != nil {
		return Envelope{}, err
	}
	return ParseEnvelope(line)
}

func (c *Codec) readLine() ([]byte, error) {
	var data []byte
	for {
		fragment, err := c.reader.ReadSlice('\n')
		data = append(data, fragment...)
		if len(data) > MaxMessageSize+2 {
			if !bytes.HasSuffix(data, []byte{'\n'}) {
				for err == bufio.ErrBufferFull {
					_, err = c.reader.ReadSlice('\n')
				}
			}
			return nil, fmt.Errorf("mensaje stdio supera 1 MiB")
		}
		if err == nil {
			break
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			if len(data) == 0 {
				return nil, fmt.Errorf("EOF stdio inesperado")
			}
			return nil, fmt.Errorf("EOF stdio con mensaje parcial")
		}
		return nil, err
	}
	data = data[:len(data)-1]
	if len(data) > 0 && data[len(data)-1] == '\r' {
		data = data[:len(data)-1]
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("mensaje stdio vacío")
	}
	if len(data) > MaxMessageSize {
		return nil, fmt.Errorf("mensaje stdio supera 1 MiB")
	}
	return data, nil
}

func (c *Codec) Write(envelope Envelope) error {
	if c.writer == nil {
		return fmt.Errorf("writer stdio ausente")
	}
	encoded, err := envelope.MarshalJSON()
	if err != nil {
		return err
	}
	if len(encoded) > MaxMessageSize {
		return fmt.Errorf("mensaje stdio supera 1 MiB")
	}
	encoded = append(encoded, '\n')
	for len(encoded) > 0 {
		written, err := c.writer.Write(encoded)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		encoded = encoded[written:]
	}
	return nil
}
