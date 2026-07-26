package main

import (
	"context"
	"io"
	"os"

	"mcp-gateway/internal/app"
	"mcp-gateway/internal/cli"
)

var newApplication = func() (cli.Application, error) {
	return app.NewDefaultRuntime()
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	application, err := newApplication()
	if err != nil {
		_, _ = io.WriteString(stderr, "error: no se pudo inicializar la aplicación\n")
		return 1
	}
	return cli.Run(ctx, args, cli.Streams{Out: stdout, Err: stderr}, application)
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
