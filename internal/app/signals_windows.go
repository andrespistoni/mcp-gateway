//go:build windows

package app

import (
	"context"
	"os"
	"os/signal"
)

func runtimeSignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt)
}
