package app

import (
	"context"
	"errors"
	"os"
	"syscall"

	"github.com/anacrolix/backtrace"
	"github.com/anacrolix/envpprof"
	"github.com/anacrolix/log"
)

// Deprecated: Use RunContext. Doesn't return on error.
func Run(mainErr func() error) {
	// Can't wrap RunContext, because that hooks SIGINT and this mainErr won't see the cancellation.
	handleMainReturning(context.TODO(), mainErr())
}

// Doesn't return on error.
func RunContext(
	mainErr func(ctx context.Context) error,
) {
	ctx, cancel := signalNotifyContextCause(context.Background(), os.Interrupt)
	defer cancel(errors.New("main returned"))
	handleMainReturning(ctx, mainErr(ctx))
}

func handleMainReturning(mainCtx context.Context, mainErr error) {
	code := OnMainReturned(mainCtx, mainErr)
	if code != 0 {
		os.Exit(code)
	}
}

// Does the cleanup expected at the end of main, without exiting. Abstracting this out lets us
// handle additions to cleanup behaviour in the future.
func OnMainReturned(mainCtx context.Context, mainErr error) (exitCode int) {
	defer envpprof.Stop()

	// Beware if the errors are not value equal or implement the error Is method. Note that
	// apparently you should treat a clean SIGINT shutdown as a success, but I don't see many
	// programs actually doing that.
	if errors.Is(mainErr, context.Cause(mainCtx)) {
		return 0
	}
	log.Levelf(log.Critical, "error in main: %v%s", mainErr, backtrace.Sprint(mainErr))
	// Here we could extract an exit code from errors that have an ExitCoder interface.
	var sigErr SignalReceivedError
	if errors.As(mainErr, &sigErr) {
		return 128 + int(sigErr.Signal.(syscall.Signal))
	}
	return 1
}
