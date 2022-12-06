package timing

import (
	"context"
	"strings"
	"sync"
	"time"
)

type Location struct {
	mu        sync.Mutex
	startTime time.Time

	// Name is the name of this timing context. If empty this is the non-reporting root of the context.
	Name string `json:"name,omitempty"`

	// Children has all the child timing contexts that have been started under this context.
	Children map[string]*Location `json:"children,omitempty"`

	// EntryCount is the number of times the timing context has been started.
	EntryCount int `json:"entry-count,omitempty"`

	// ExistCount is the number of times the timing context has been completed.
	ExitCount int `json:"exit-count,omitempty"`

	// TotalDuration is the amount of time this context has been started.
	TotalDuration time.Duration `json:"total-duration,omitempty"`

	// Async, if set, causes the children's time to never be excluded. This is used in cases where
	// you have either overlapping timing contexts. This is normally caused when multiple Goroutines
	// are started in parallel in the same timing context.
	Async bool `json:"async,omitempty"`
}

// Start starts the timer on a given timing context. A timer can only be started if it is not
// already started.
func (l *Location) Start() {
	if !l.startTime.IsZero() {
		panic("reentrant timing not supported")
	}
	l.EntryCount++
	l.startTime = time.Now()
}

// Complete marks a timing context as completed and adds the time to the total duration for
// that timing context.
func (l *Location) Complete() {
	d := time.Since(l.startTime)
	if l.startTime.IsZero() {
		panic("timing context not started")
	}
	l.startTime = time.Time{} // Zero it out
	l.ExitCount++
	l.TotalDuration += d
}

// String returns a multi-line report of what time was spent and where it was spent.
func (l *Location) String() string {
	b := strings.Builder{}
	l.dumpToBuilder(&b, "", " > ", "", nil, false)
	return b.String()
}

// TotalChildDuration is a helper that computes the total time that the child timing contexts have spent.
func (l *Location) TotalChildDuration() time.Duration {
	d := time.Duration(0)
	for _, child := range l.Children {
		d += child.TotalDuration
	}
	return d
}

// Report generates a report of how much time was spent where.
//
//   - prefix is prepended to each line if you need something like indented output.
//   - separator is a string that is used between levels of the timing tree.
//   - durFmt is a function to format (round, display, etc.) the duration to report in whatever
//     way is suitable for your needs.
//   - excludeChildren will subtract out of the duration of the children when reporting
//     the time.
//
// The reason excludeChildren exists is if you want to represent the output in a chart, you
// may have double-counting of times. If you have a structure like:
//
//	parent - 100ms
//	parent > child1 - 25ms
//	parent > child2 - 75ms
//
// the children's time would be counted twice, once for itself, and once for the parent.
// With onlyLeaf, the parent's line is not directly reported on.
func (l *Location) Report(prefix, separator string, durFmt DurationFormatter, excludeChildren bool) string {
	b := strings.Builder{}
	l.dumpToBuilder(&b, prefix, separator, "", durFmt, excludeChildren)
	return b.String()
}

// ReportMap takes the timings and formats them into a map keyed on the location names with the
// value of the duration divided by the divisor. With a divisor of 1, the reported time is in the
// native nanoseconds that the Duration keeps track of. This may be annoying to read, so you can
// pass in "1000" to report by microseconds, "1000000" for milliseconds, etc.
//
//   - separator is a string that is used between levels of the timing tree.
//   - excludeChildren will subtract out of the duration of the children when reporting
//     the time.
func (l *Location) ReportMap(separator string, divisor float64, excludeChildren bool) map[string]float64 {
	result := map[string]float64{}
	l.dumpToMap(result, separator, "", divisor, excludeChildren)
	return result
}

// getChild gets an existing timing context or creates a child timing context if one
// does not exist.
func (l *Location) getChild(ctx context.Context, name string) *Context {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.Children == nil {
		l.Children = map[string]*Location{}
	}
	if cl, ok := l.Children[name]; ok {
		return &Context{
			prevCtx:  ctx,
			Location: cl,
		}
	} else {
		cl := &Location{
			Name: name,
		}
		cc := &Context{
			prevCtx:  ctx,
			Location: cl,
		}
		l.Children[name] = cl
		return cc
	}
}
