package main

import (
	"os"

	"mcp-gateway/internal/testsupport/fakemcp"
)

func main() {
	os.Exit(fakemcp.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
