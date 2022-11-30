package timing

import (
	"context"
	"strings"
	"sync"
	"time"
)

type Context struct {
	mu        sync.Mutex
	prevCtx   context.Context
	parent    *Context
	startTime time.Time

	// Name is the name of this timing context. If empty this is the non-reporting root of the context.
	Name string `json:"name,omitempty"`

	// Children has all the child timing contexts that have been started under this context.
	Children map[string]*Context `json:"children,omitempty"`

	// EntryCount is the number of times the timing context has been started.
	EntryCount int `json:"entry-count,omitempty"`

	// ExistCount is the number of times the timing context has been completed.
	ExitCount int `json:"exit-count,omitempty"`

	// TotalDuration is the amount of time this context has been started.
	TotalDuration time.Duration `json:"total-duration,omitempty"`
}

type contextTiming int

const contextTimingKey contextTiming = 0

// ForName returns an un-started Context. This is generally not used by client code, but
// may be useful for a context that needs to be repeatedly started and completed for some
// reason.
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
			Name: name,
		}
		return c
	} else {
		return p.getChild(name)
	}
}

// Start begins a timing context and relates it to a preceding timing context if it exists.
// If a previous context does not exist then this starts a new named root timing context.
func Start(ctx context.Context, name string) *Context {
	c := ForName(ctx, name)
	c.Start(ctx)
	return c
}

// Root creates a new unnamed timing context. This is similar to Start except there are no timers
// started. This is provided to allow for a simpler report if it's desired.
func Root(ctx context.Context) *Context {
	if ctx == nil {
		panic("context must be defined")
	}
	c := &Context{
		prevCtx: ctx,
	}
	return c
}

// Start starts the timer on a given timing context. A timer can only be started if it is not
// already started.
func (c *Context) Start(ctx context.Context) {
	if !c.startTime.IsZero() || c.prevCtx != nil {
		panic("reentrant timing not supported")
	}
	c.prevCtx = ctx
	c.EntryCount++
	c.startTime = time.Now()
}

// Complete marks a timing context as completed and adds the time to the total duration for
// that timing context.
func (c *Context) Complete() {
	d := time.Since(c.startTime)
	if c.startTime.IsZero() || c.prevCtx == nil {
		panic("timing context not started")
	}
	c.startTime = time.Time{} // Zero it out
	c.ExitCount++
	c.prevCtx = nil
	c.TotalDuration += d
}

// String returns a multi-line report of what time was spent and where it was spent.
func (c *Context) String() string {
	b := strings.Builder{}
	c.dumpToBuilder(&b, "", " > ", "", false)
	return b.String()
}

// TotalChildDuration is a helper that computes the total time that the child timing contexts have spent.
func (c *Context) TotalChildDuration() time.Duration {
	d := time.Duration(0)
	for _, child := range c.Children {
		d += child.TotalDuration
	}
	return d
}

// Report generates a report of how much time was spent where.
//
// prefix is prepended to each line if you need something like indented output.
// separator is a string that is used between levels of the timing tree.
// excludeChildren will subtract out of the duration of the children when reporting
// the time.
//
// The reason onlyLeaf exists is if you want to represent the output in a chart, you
// may have double-counting of times. If you have a structure like:
// parent - 100ms
// parent > child1 - 25ms
// parent > child2 - 75ms
// the children's time would be counted twice, once for itself, and once for the parent.
// With onlyLeaf, the parent's line is not directly reported on.
func (c *Context) Report(prefix, separator string, excludeChildren bool) string {
	b := strings.Builder{}
	c.dumpToBuilder(&b, prefix, separator, "", excludeChildren)
	return b.String()
}

// ReportMap takes the timings and formats them into a map keyed on the location names with the
// value of the duration divided by the divisor. With a divisor of 1, the reported time is in the
// native nanoseconds that the Duration keeps track of. This may be annoying to read, so you can
// pass in "1000" to report by microseconds, "1000000" for milliseconds, etc.
//
// separator is a string that is used between levels of the timing tree.
// excludeChildren will subtract out of the duration of the children when reporting
// the time.
func (c *Context) ReportMap(separator string, divisor float64, excludeChildren bool) map[string]float64 {
	result := map[string]float64{}
	c.dumpToMap(result, separator, "", divisor, excludeChildren)
	return result
}

// findParentTiming is a global that finds most recent timing context on the context stack.
func findParentTiming(ctx context.Context) *Context {
	value := ctx.Value(contextTimingKey)
	if value == nil {
		return nil
	}
	if ct, ok := value.(*Context); ok {
		return ct
	}
	panic("invalid context timing type")
}

// getChild gets an existing timing context or creates a child timing context if one
// does not exist.
func (c *Context) getChild(name string) *Context {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Children == nil {
		c.Children = map[string]*Context{}
	}
	if cc, ok := c.Children[name]; ok {
		return cc
	} else {
		cc := &Context{
			parent: c,
			Name:   name,
		}
		c.Children[name] = cc
		return cc
	}
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
	if key == contextTimingKey {
		return c
	}
	return c.prevCtx.Value(key)
}
