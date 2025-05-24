// Package timing provides hierarchical timing tracking for Go applications.
// It uses Go's context pattern to maintain parent-child relationships between
// timing operations, making it easy to track where time is spent in complex systems.
//
// Design Philosophy:
//
// The timing package is designed to be used in production code with minimal overhead.
// Functions panic on programming errors (like nil contexts or empty names) rather than
// returning errors. This is intentional - these represent bugs in the calling code that
// should be fixed during development, not runtime errors that need handling.
//
// Callers should NOT use recover() to handle these panics. Instead, ensure:
//   - Always pass non-nil contexts
//   - Always provide non-empty names for timing contexts
//   - Never call Complete functions more than once
//
// These constraints are easy to satisfy and result in cleaner, more performant code
// without error handling boilerplate.
package timing

import (
	"context"
	"time"
)

type Context struct {
	*Location

	prevCtx context.Context
}

type contextTimingType int

const ContextTimingKey contextTimingType = 0

// Start begins a timing context and relates it to a preceding timing context if it exists.
// If a previous context does not exist then this starts a new named root timing context.
//
// Panics if ctx is nil or name is empty. These panics indicate programming errors
// that should be fixed in development, not runtime errors requiring error handling.
func Start(ctx context.Context, name string) (*Context, Complete) {
	c := ForName(ctx, name)
	return c, c.Start()
}

// StartAsync begins a timing context and relates it to a preceding timing context if it exists.
// If a previous context does not exist then this starts a new named root timing context.
// This is similar to Start except that it will mark the context as Async, which means that
// the child contexts will not be excluded from the parent's time. This is useful for timing
// contexts that overlap.
//
// Panics if ctx is nil or name is empty. These panics indicate programming errors
// that should be fixed in development, not runtime errors requiring error handling.
func StartAsync(ctx context.Context, name string) (*Context, Complete) {
	c := ForName(ctx, name)
	c.Async = true
	return c, c.Start()
}

// Root creates a new unnamed timing context. This is similar to Start except there are no timers
// started. This is provided to allow for a simpler report if it's desired.
//
// Panics if ctx is nil. This panic indicates a programming error that should be fixed
// in development, not a runtime error requiring error handling.
func Root(ctx context.Context) *Context {
	if ctx == nil {
		panic("context must be defined")
	}
	c := &Context{
		prevCtx:  ctx,
		Location: &Location{},
	}
	return c
}

// StartRoot creates a new named timing context. Unlike Start, this will create a new unrelated timing
// context regardless if there is a timing context already on the context stack. This is useful
// for any long-running processes that finish after the Goroutine that started them have finished.
//
// Panics if ctx is nil. This panic indicates a programming error that should be fixed
// in development, not a runtime error requiring error handling.
func StartRoot(ctx context.Context, name string) (*Context, Complete) {
	if ctx == nil {
		panic("context must be defined")
	}
	c := &Context{
		prevCtx: ctx,
		Location: &Location{
			Name: name,
		},
	}
	return c, c.Start()
}

// ForName returns an un-started Context. This is generally not used by client code, but
// may be useful for a context that needs to be repeatedly started and completed for some
// reason.
//
// Panics if name is empty or ctx is nil. These panics indicate programming errors
// that should be fixed in development, not runtime errors requiring error handling.
func ForName(ctx context.Context, name string) *Context {
	if name == "" {
		panic("non-root timings must be named")
	}
	if ctx == nil {
		panic("context must be defined")
	}
	p := findParentTiming(ctx)
	if p == nil {
		c := &Context{
			prevCtx: ctx,
			Location: &Location{
				Name: name,
			},
		}
		return c
	} else {
		return p.getChild(ctx, name)
	}
}

// findParentTiming is a global that finds most recent timing context on the context stack.
func findParentTiming(ctx context.Context) *Context {
	value := ctx.Value(ContextTimingKey)
	if value == nil {
		return nil
	}
	if ct, ok := value.(*Context); ok {
		return ct
	}
	panic("invalid context timing type")
}

// context.Context implementation

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.prevCtx.Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.prevCtx.Done()
}

func (c *Context) Err() error {
	return c.prevCtx.Err()
}

func (c *Context) Value(key interface{}) interface{} {
	if key == ContextTimingKey {
		return c
	}
	return c.prevCtx.Value(key)
}
