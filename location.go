package timing

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Location struct {
	mu sync.Mutex

	// Name is the name of this timing context. If empty this is the non-reporting root of the context.
	Name string `json:"name,omitempty"`

	// Children has all the child timing contexts that have been started under this context.
	Children map[string]*Location `json:"children,omitempty"`

	// EntryCount is the number of times the timing context has been started.
	EntryCount uint32 `json:"entry-count,omitempty"`

	// ExistCount is the number of times the timing context has been completed.
	ExitCount uint32 `json:"exit-count,omitempty"`

	// TotalDuration is the amount of time this context has been started.
	TotalDuration time.Duration `json:"total-duration,omitempty"`

	// Async, if set, causes the children's time to never be excluded. This is used in cases where
	// you have either overlapping timing contexts. This is normally caused when multiple Goroutines
	// are started in parallel in the same timing context.
	Async bool `json:"async,omitempty"`

	// Details allow you to add extra information about the timing location, so you can note the number
	// of items processed or the number of attempts to access a resource.
	Details map[string]anything `json:"details,omitempty"`
}

type anything interface{}

// Complete is a function to call when a concurrent execution is completed.
type Complete func()

// Start begins a timed event for this location. It returns a Complete function that is
// to be called when whatever it is that is being timed is completed.
func (l *Location) Start() Complete {
	ended := false
	atomic.AddUint32(&l.EntryCount, 1)
	startTime := time.Now()
	return func() {
		d := time.Since(startTime)
		if ended {
			panic("timing already completed")
		}
		ended = true
		atomic.AddUint32(&l.ExitCount, 1)
		atomic.AddInt64((*int64)(&l.TotalDuration), int64(d))
	}
}

// String returns a multi-line report of what time was spent and where it was spent.
func (l *Location) String() string {
	b := strings.Builder{}
	l.dumpToBuilder(&b, "", &ReportOptions{Separator: " > "})
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
func (l *Location) Report(options ReportOptions) string {
	if options.Separator == "" {
		if options.Compact {
			options.Separator = " | "
		} else {
			options.Separator = " > "
		}
	}
	b := strings.Builder{}
	l.dumpToBuilder(&b, "", &options)
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
			Name:    name,
			Details: map[string]anything{},
		}
		cc := &Context{
			prevCtx:  ctx,
			Location: cl,
		}
		l.Children[name] = cl
		return cc
	}
}
