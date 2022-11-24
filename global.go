package timing

import "context"

// ContextWithTiming takes a context and returns a new context that has a timing in it that can be
// used to track timings of things in a structured, hierarchical way.
func ContextWithTiming(ctx context.Context) context.Context {
	t := NewTiming("")
	return context.WithValue(ctx, timingContextKey, t)
}

// NewTiming return a new timing object that can be used by itself.
func NewTiming(rootName string) *Timing {
	root := &Location{
		Name: rootName, // root
	}
	t := &Timing{
		current: root,
		root:    root,
	}

	return t
}

// FromContext returns the active Timing object from the context. If not found this function panics.
func FromContext(ctx context.Context) *Timing {
	tca := ctx.Value(timingContextKey)
	if tca == nil {
		panic("no timing context present")
	}
	if tc, ok := tca.(*Timing); ok {
		return tc
	}
	panic("invalid timing context type") // There should be bo way of getting here.
}

// Start pulls the Timing object out of the context and starts timing of a process. It returns
// a Completed function that should be called when whatever process is completed.
func Start(ctx context.Context, name string) Completed {
	t := FromContext(ctx)
	return t.Start(name)
}

// BeginSubRootContext starts a new related Timing context with a give name. In general the
// Timing object is not thread safe, by starting a new Timing context we solve that issue. As long
// as only one Goroutine is working with a given Timing object, everything is safe.
func BeginSubRootContext(ctx context.Context, name string) context.Context {
	t := FromContext(ctx)
	return t.BeginSubRootContext(ctx, name)
}

// Root returns to root location of the current Timing context. This can be used to report on
// all of the timing activity that has been recorded.
func Root(ctx context.Context) map[string]*Location {
	t := FromContext(ctx)
	return t.Root()
}
