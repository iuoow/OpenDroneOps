package signals

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func Context(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	return ctx, cancel
}
