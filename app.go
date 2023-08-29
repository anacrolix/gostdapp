package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/anacrolix/backtrace"
	"github.com/anacrolix/envpprof"
	"github.com/anacrolix/log"
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
	defer cancel(fmt.Errorf("RunOpts returned"))
	handleMainReturning(mainErr(ctx))
}

func handleMainReturning(mainErr error) {
	envpprof.Stop()
	if mainErr != nil {
		log.Printf("error in main: %v%s", mainErr, backtrace.Sprint(mainErr))
		os.Exit(1)
	}
}

// This is taken from the standard library, and modified to use context cancellation cause.
func signalNotifyContextCause(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(parent)
	c := &signalCtx{
		Context: ctx,
		cancel:  cancel,
		signals: signals,
	}
	c.ch = make(chan os.Signal, 1)
	signal.Notify(c.ch, c.signals...)
	if ctx.Err() == nil {
		go func() {
			select {
			case sig := <-c.ch:
				c.cancel(fmt.Errorf("signal received: %v", sig))
			case <-c.Done():
			}
		}()
	}
	return c, c.stop
}

type signalCtx struct {
	context.Context

	cancel  context.CancelCauseFunc
	signals []os.Signal
	ch      chan os.Signal
}

func (c *signalCtx) stop(cause error) {
	c.cancel(cause)
	signal.Stop(c.ch)
}

func (c *signalCtx) String() string {
	var buf []byte
	// We know that the type of c.Context is context.cancelCtx, and we know that the
	// String method of cancelCtx returns a string that ends with ".WithCancel".
	name := c.Context.(fmt.Stringer).String()
	name = name[:len(name)-len(".WithCancel")]
	buf = append(buf, "signal.NotifyContext("+name...)
	if len(c.signals) != 0 {
		buf = append(buf, ", ["...)
		for i, s := range c.signals {
			buf = append(buf, s.String()...)
			if i != len(c.signals)-1 {
				buf = append(buf, ' ')
			}
		}
		buf = append(buf, ']')
	}
	buf = append(buf, ')')
	return string(buf)
}
