package main

import (
	"bytes"
	"context"
	"testing"

	"mcp-gateway/internal/app"
	"mcp-gateway/internal/cli"
)

type mainFakeApplication struct{}

func (mainFakeApplication) List(context.Context) ([]app.ListItem, error) {
	return []app.ListItem{{Name: "one", Status: app.StatusDisabled, Prefix: "one__", Binary: "one"}}, nil
}

func TestRunDelegatesArgumentsStreamsAndCode(t *testing.T) {
	original := newApplication
	t.Cleanup(func() { newApplication = original })
	newApplication = func() (cli.Application, error) { return mainFakeApplication{}, nil }
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"list"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !bytes.Contains(stdout.Bytes(), []byte("one__")) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
