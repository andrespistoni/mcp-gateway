package sse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

const (
	maxRequestTarget = 8 * 1024
	maxHeaderBytes   = 16 * 1024
	// HTTP does not prescribe a method length limit. The gate nevertheless
	// bounds its own request-line storage so an invalid method cannot consume
	// unbounded memory before the target limit can be evaluated.
	maxRequestLineBytes = maxRequestTarget + 1024
)

type gateError struct {
	status int
	err    error
}

func (e *gateError) Error() string { return e.err.Error() }

type requestHead struct {
	raw           []byte
	bodyPrefix    []byte
	contentLength int64
}

// parseRequestHead accepts the deliberately small HTTP/1.1 subset understood
// by the gateway. It returns the exact bytes for replay to net/http.
func parseRequestHead(reader *bufio.Reader) (requestHead, error) {
	raw := make([]byte, 0, maxRequestLineBytes+maxHeaderBytes+2)
	line, err := readRequestLine(reader, &raw)
	if err != nil {
		return requestHead{}, err
	}
	parts := strings.Split(string(line), " ")
	if len(parts) != 3 || parts[0] == "" || !validToken([]byte(parts[0])) || parts[1] == "" || parts[2] != "HTTP/1.1" || !strings.HasPrefix(parts[1], "/") {
		return requestHead{}, badHead("request-line inválida")
	}
	if len(parts[1]) > maxRequestTarget {
		return requestHead{}, &gateError{status: 414, err: fmt.Errorf("request target supera 8 KiB")}
	}
	if _, err := url.ParseRequestURI(parts[1]); err != nil {
		return requestHead{}, badHead("request target inválido")
	}
	for _, b := range line {
		if b == 0 || b < 0x20 || b == 0x7f {
			return requestHead{}, badHead("request-line contiene controles")
		}
	}

	headers := make(map[string]string)
	headerBytes := 0
	for {
		line, err = readCRLFLine(reader, &raw, maxHeaderBytes-headerBytes)
		if err != nil {
			return requestHead{}, err
		}
		if len(line) == 0 {
			break
		}
		headerBytes += len(line) + 2
		if line[0] == ' ' || line[0] == '\t' || bytes.IndexByte(line, 0) >= 0 {
			return requestHead{}, badHead("header inválido")
		}
		colon := bytes.IndexByte(line, ':')
		if colon <= 0 || line[colon-1] == ' ' || line[colon-1] == '\t' || !validToken(line[:colon]) {
			return requestHead{}, badHead("nombre de header inválido")
		}
		for _, b := range line[colon+1:] {
			if (b < 0x20 && b != '\t') || b == 0x7f {
				return requestHead{}, badHead("valor de header inválido")
			}
		}
		name := strings.ToLower(string(line[:colon]))
		value := strings.TrimSpace(string(line[colon+1:]))
		if _, exists := headers[name]; exists {
			return requestHead{}, badHead("header duplicado")
		}
		headers[name] = value
	}
	if host, ok := headers["host"]; !ok || host == "" {
		return requestHead{}, badHead("Host ausente")
	}
	if _, ok := headers["transfer-encoding"]; ok {
		return requestHead{}, badHead("Transfer-Encoding no admitido")
	}
	length := int64(0)
	if value, ok := headers["content-length"]; ok {
		if value == "" || strings.ContainsAny(value, "+- ") {
			return requestHead{}, badHead("Content-Length inválido")
		}
		for _, b := range value {
			if b < '0' || b > '9' {
				return requestHead{}, badHead("Content-Length inválido")
			}
		}
		length, err = strconv.ParseInt(value, 10, 64)
		if err != nil || length < 0 {
			return requestHead{}, badHead("Content-Length inválido")
		}
	}
	buffered := reader.Buffered()
	prefix := make([]byte, buffered)
	if buffered > 0 {
		_, _ = io.ReadFull(reader, prefix)
	}
	if int64(len(prefix)) > length {
		prefix = prefix[:length]
	}
	return requestHead{raw: raw, bodyPrefix: prefix, contentLength: length}, nil
}

func readRequestLine(reader *bufio.Reader, raw *[]byte) ([]byte, error) {
	line := make([]byte, 0, maxRequestLineBytes)
	spaces, targetBytes := 0, 0
	for {
		byteValue, err := reader.ReadByte()
		if err != nil {
			return nil, incompleteHeadError(err)
		}
		if len(line) == maxRequestLineBytes {
			return nil, badHead("request-line supera el límite interno")
		}
		if byteValue == ' ' {
			spaces++
		} else if spaces == 1 && byteValue != '\r' && byteValue != '\n' {
			targetBytes++
			if targetBytes > maxRequestTarget {
				return nil, &gateError{status: 414, err: fmt.Errorf("request target supera 8 KiB")}
			}
		}
		line = append(line, byteValue)
		if byteValue != '\n' {
			continue
		}
		if len(line) < 2 || line[len(line)-2] != '\r' {
			return nil, badHead("HTTP requiere CRLF")
		}
		*raw = append(*raw, line...)
		return line[:len(line)-2], nil
	}
}

// readCRLFLine bounds each field-line before it is appended to raw. maxBytes
// is the remaining header budget, excluding the final empty CRLF delimiter.
func readCRLFLine(reader *bufio.Reader, raw *[]byte, maxBytes int) ([]byte, error) {
	line := make([]byte, 0, min(maxBytes, 1024))
	for {
		byteValue, err := reader.ReadByte()
		if err != nil {
			return nil, incompleteHeadError(err)
		}
		nextLength := len(line) + 1
		isEmptyDelimiter := (len(line) == 0 && byteValue == '\r') ||
			(len(line) == 1 && line[0] == '\r' && byteValue == '\n')
		if nextLength > maxBytes && !isEmptyDelimiter {
			return nil, &gateError{status: 431, err: fmt.Errorf("headers superan 16 KiB")}
		}
		line = append(line, byteValue)
		if byteValue != '\n' {
			continue
		}
		if len(line) < 2 || line[len(line)-2] != '\r' {
			return nil, badHead("HTTP requiere CRLF")
		}
		*raw = append(*raw, line...)
		return line[:len(line)-2], nil
	}
}

func incompleteHeadError(err error) error {
	if err == io.EOF {
		return badHead("head HTTP incompleto")
	}
	return err
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func validToken(value []byte) bool {
	for _, b := range value {
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b))) {
			return false
		}
	}
	return true
}

func badHead(message string) error { return &gateError{status: 400, err: fmt.Errorf("%s", message)} }
