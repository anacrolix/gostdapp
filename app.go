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
	code := OnMainReturned(mainErr)
	if code != 0 {
		os.Exit(code)
	}
}

// Does the cleanup expected at the end of main, without exiting. Abstracting this out lets us
// handle additions to cleanup behaviour in the future.
func OnMainReturned(mainErr error) (exitCode int) {
	envpprof.Stop()
	if mainErr != nil {
		log.Printf("error in main: %v%s", mainErr, backtrace.Sprint(mainErr))
		return 1
	}
	return 0
}
