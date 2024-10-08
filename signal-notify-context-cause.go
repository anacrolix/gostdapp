package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

// Okay so errors.Is requires equality, which means implementing an Is method, or being a value
// type.
type SignalReceivedError struct {
	Signal os.Signal
}

func (me SignalReceivedError) Error() string {
	return fmt.Sprintf("signal received: %v", me.Signal)
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
				c.cancel(SignalReceivedError{sig})
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
