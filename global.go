package timing

import "context"

func Start(ctx context.Context, name string) Completed {
	t := FromContext(ctx)
	return t.Start(name)
}

func BeginSubRoot(ctx context.Context, name string) context.Context {
	t := FromContext(ctx)
	return t.BeginSubRootContext(ctx, name)
}

func Root(ctx context.Context) map[string]*Location {
	t := FromContext(ctx)
	return t.Root()
}

func ContextWithTiming(ctx context.Context) context.Context {
	t := NewTiming("")
	return context.WithValue(ctx, timingContextKey, t)
}

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
