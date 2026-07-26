package cli

import (
	"bytes"
	"context"
	"testing"
)

func TestManageCLIInvokesMutationsAndReportsIdempotence(t *testing.T) {
	application := &workflowApplication{}
	for _, args := range [][]string{{"remove", "one"}, {"enable", "one"}, {"disable", "one"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(context.Background(), args, Streams{Out: &stdout, Err: &stderr}, application); code != 0 || stderr.Len() != 0 || !bytes.Contains(stdout.Bytes(), []byte("sin cambios")) {
			t.Fatalf("args=%v code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
	if application.removed != "one" || application.enabled == nil || *application.enabled {
		t.Fatalf("mutaciones no invocadas: %#v", application)
	}
}
