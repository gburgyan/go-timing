package timing

import (
	"context"
	"strings"
	"time"
)

// Timing is the base do the time-tracking module. It keeps track of what it is that is
// currently being done, and is used to start timing and creation of sub-timing trackers
// that can be used on other Goroutines.
type Timing struct {
	root    *Location
	current *Location
}

// Location represents the time spent doing a given task. It's called "location" because
// certain things may be called multiple times--in that case this tracks the total time
// as well as the number of calls to it.
type Location struct {
	Name          string               `json:"-"`
	EntryCount    int                  `json:"entry-count,omitempty"`
	ExitCount     int                  `json:"exit-count,omitempty"`
	TotalDuration time.Duration        `json:"total-duration-ns,omitempty"`
	Children      map[string]*Location `json:"children,omitempty"`
	SubRoot       bool                 `json:"sub-root,omitempty"`
}

// Completed is a function that is returned when you start time tracking of a task. It should
// be called when the task is completed.
//
// A common way to do this is to invoke it with a defer:
//
//	func task(ctx context.Context) {
//	  complete := timing.Start(ctx, "task")
//	  defer complete()
//	  // do work
//	}
type Completed func()

type key int

const timingContextKey key = 0

// Start starts timing of a task and returns a function that should be called when the task is completed.
func (t *Timing) Start(name string) Completed {
	if len(name) == 0 {
		panic("timing name much be a non-zero length string")
	}

	parent := t.current
	l := parent.getSubLocation(name)
	t.current = l
	l.EntryCount++

	startTime := time.Now()
	return func() {
		d := time.Since(startTime)

		l.ExitCount++
		l.TotalDuration += d
		t.current = parent
	}
}

func (l *Location) getSubLocation(name string) *Location {
	if l.Children == nil {
		l.Children = map[string]*Location{}
	}

	if c, ok := l.Children[name]; ok {
		return c
	}
	c := &Location{
		Name: name,
	}
	l.Children[name] = c
	return c
}

// BeginSubRootContext creates a new Timing object, attaches it to the context and
// returns the new decorated context. This is used because a Timing object is not
// natively thread safe--creating a new Timing that is only used by the other Goroutine
// solves this issue since each Goroutine has its own Timing object.
func (t *Timing) BeginSubRootContext(ctx context.Context, name string) context.Context {
	child := t.BeginSubRoot(name)
	return context.WithValue(ctx, timingContextKey, child)
}

// BeginSubRoot creates a new Timing object. This is used because a Timing object is not
// natively thread safe--creating a new Timing that is only used by the other Goroutine
// solves this issue since each Goroutine has its own Timing object.
func (t *Timing) BeginSubRoot(name string) *Timing {
	if t.current.Children == nil {
		t.current.Children = map[string]*Location{}
	}
	if _, ok := t.current.Children[name]; ok {
		panic("sub-threads require a new timing location")
	}
	childLoc := &Location{
		Name:    name,
		SubRoot: true,
	}
	t.current.Children[name] = childLoc

	child := &Timing{
		current: childLoc,
		root:    childLoc,
	}

	return child
}

// Root returns the top-level map of the Locations that are contained within this Timing object.
func (t *Timing) Root() map[string]*Location {
	return t.root.Children
}

// String returns a multi-line report of what time was spent and where it was spent.
func (t *Timing) String() string {
	b := strings.Builder{}
	t.root.dumpToBuilder(&b, "", " > ", "", false)
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
func (t *Timing) Report(prefix, separator string, onlyLeaf bool) string {
	b := strings.Builder{}
	t.root.dumpToBuilder(&b, prefix, separator, "", onlyLeaf)
	return b.String()
}

// ReportMap takes the timings and formats them into a map keyed on the location names with the
// value of the duration divided by the divisor. With a divisor of 1, the reported time is in the
// native nanoseconds that the Duration keeps track of. This may be annoying to read, so you can
// pass in "1000" to report by microseconds, "1000000" for milliseconds, etc.
//
// separator is a string that is used between levels of the timing tree.
// onlyLeaf determines if only leaf nodes are reported on.
func (t *Timing) ReportMap(separator string, divisor float64, onlyLeaf bool) map[string]float64 {
	result := map[string]float64{}
	t.root.dumpToMap(result, separator, "", divisor, onlyLeaf)
	return result
}
