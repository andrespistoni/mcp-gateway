package cli

import (
	"bytes"
	"context"
	"testing"
)

func TestAddCLIMapsArgumentsAndWarnsOnSkipValidation(t *testing.T) {
	application := &workflowApplication{}
	var stdout, stderr bytes.Buffer
	args := []string{"add", "custom", "--prefix", "custom__", "--binary", "missing", "--arg", "a b", "--env", "TOKEN=${TOKEN}", "--skip-validation"}
	code := Run(context.Background(), args, Streams{Out: &stdout, Err: &stderr}, application)
	if code != 0 || application.add.Name != "custom" || application.add.Disabled || !application.add.SkipValidation ||
		application.add.Args[0] != "a b" || application.add.Environment["TOKEN"] != "${TOKEN}" ||
		!bytes.Contains(stderr.Bytes(), []byte("sin verificar")) {
		t.Fatalf("code=%d request=%#v stdout=%q stderr=%q", code, application.add, stdout.String(), stderr.String())
	}
}
