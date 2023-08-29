package app

import (
	"context"
	"errors"
	"github.com/anacrolix/backtrace"
	"github.com/anacrolix/envpprof"
	"github.com/anacrolix/log"
	"os"
)

// Deprecated: Use RunContext. Doesn't return on error.
func Run(mainErr func() error) {
	// Can't wrap RunContext, because that hooks SIGINT and this mainErr won't see the cancellation.
	handleMainReturning(mainErr())
}

// Doesn't return on error.
func RunContext(
	mainErr func(ctx context.Context) error,
) {
	ctx, cancel := signalNotifyContextCause(context.Background(), os.Interrupt)
	defer cancel(errors.New("main returned"))
	handleMainReturning(mainErr(ctx))
}

func handleMainReturning(mainErr error) {
	envpprof.Stop()
	if mainErr != nil {
		log.Printf("error in main: %v%s", mainErr, backtrace.Sprint(mainErr))
		os.Exit(1)
	}
}
