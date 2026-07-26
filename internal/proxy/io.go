package proxy

import (
	"io"

	"mcp-gateway/internal/mcp"
)

type readResult struct {
	envelope mcp.Envelope
	err      error
}

type writeRequest struct {
	envelope mcp.Envelope
	done     chan error
}

func readStdout(stdout io.ReadCloser, inbound chan<- readResult, actorGone <-chan struct{}) {
	defer stdout.Close()
	codec := mcp.NewCodec(stdout, nil)
	for {
		envelope, err := codec.Read()
		result := readResult{envelope: envelope, err: err}
		select {
		case inbound <- result:
		case <-actorGone:
			if err == nil {
				continue
			}
		}
		if err != nil {
			// A protocol error is terminal, but bytes already in the parent pipe still
			// belong to this reader and must be drained before it closes the handle.
			_, _ = io.Copy(io.Discard, stdout)
			return
		}
	}
}

func writeStdin(stdin io.WriteCloser, requests <-chan writeRequest) {
	defer stdin.Close()
	codec := mcp.NewCodec(nil, stdin)
	for request := range requests {
		request.done <- codec.Write(request.envelope)
	}
}
