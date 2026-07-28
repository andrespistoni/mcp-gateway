package sse

import (
	"fmt"
	"io"
)

func writeEndpoint(writer io.Writer, id SessionID) error {
	_, err := fmt.Fprintf(writer, "event: endpoint\ndata: /message?sessionId=%s\n\n", id.Encode())
	return err
}

func writeMessage(writer io.Writer, payload []byte) error {
	_, err := fmt.Fprintf(writer, "event: message\ndata: %s\n\n", payload)
	return err
}

func writeHeartbeat(writer io.Writer) error {
	_, err := io.WriteString(writer, ": heartbeat\n\n")
	return err
}
