package timing

import (
	"context"
	"strings"
	"sync"
	"time"
)

type Context struct {
	mu      sync.Mutex
	prevCtx context.Context

	Name          string
	Parent        *Context
	Children      map[string]*Context
	EntryCount    int
	ExitCount     int
	TotalDuration time.Duration
	startTime     time.Time
}

type contextTiming int

const contextTimingKey contextTiming = 0

func Start(ctx context.Context, name string) *Context {
	if name == "" {
		panic("non-root timings must be named")
	}
	p := findParentTiming(ctx)
	if p == nil {
		c := &Context{
			Name: name,
		}
		c.Start(ctx)
		return c
	} else {
		return p.startChild(ctx, name)
	}
}

func Root(ctx context.Context) *Context {
	c := &Context{
		prevCtx: ctx,
		Name:    "",
	}
	return c
}

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

func (c *Context) startChild(ctx context.Context, name string) *Context {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Children == nil {
		c.Children = map[string]*Context{}
	}
	if cc, ok := c.Children[name]; ok {
		cc.Start(ctx)
		return cc
	} else {
		cc := &Context{
			Parent: c,
			Name:   name,
		}
		c.Children[name] = cc
		cc.Start(ctx)
		return cc
	}
}

func (c *Context) Start(ctx context.Context) {
	if !c.startTime.IsZero() || c.prevCtx != nil {
		panic("reentrant timing not supported")
	}
	c.prevCtx = ctx
	c.EntryCount++
	c.startTime = time.Now()
}

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

// Report generates a report of how much time was spent where.
//
// prefix is prepended to each line if you need something like indented output.
// separator is a string that is used between levels of the timing tree.
// onlyLeaf determines if only leaf nodes are reported on.
//
// The reason onlyLeaf exists is if you want to represent the output in a chart, you
// may have double-counting of times. If you have a structure like:
// parent - 100ms
// parent > child1 - 25ms
// parent > child2 - 75ms
// the children's time would be counted twice, once for itself, and once for the parent.
// With onlyLeaf, the parent's line is not directly reported on.
func (c *Context) Report(prefix, separator string, onlyLeaf bool) string {
	b := strings.Builder{}
	c.dumpToBuilder(&b, prefix, separator, "", onlyLeaf)
	return b.String()
}

// ReportMap takes the timings and formats them into a map keyed on the location names with the
// value of the duration divided by the divisor. With a divisor of 1, the reported time is in the
// native nanoseconds that the Duration keeps track of. This may be annoying to read, so you can
// pass in "1000" to report by microseconds, "1000000" for milliseconds, etc.
//
// separator is a string that is used between levels of the timing tree.
// onlyLeaf determines if only leaf nodes are reported on.
func (c *Context) ReportMap(separator string, divisor float64, onlyLeaf bool) map[string]float64 {
	result := map[string]float64{}
	c.dumpToMap(result, separator, "", divisor, onlyLeaf)
	return result
}
