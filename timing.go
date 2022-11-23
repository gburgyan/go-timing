package timing

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Timing struct {
	mu             sync.Mutex
	asyncProcesses map[string]*Timing
	root           *Location
	current        *Location
}

type Location struct {
	Name          string
	EntryCount    int
	ExitCount     int
	TotalDuration time.Duration
	Owner         *Timing
	Children      map[string]*Location
}

type Completed func()

type key int

const timingContextKey key = 0

func ContextWithTiming(ctx context.Context) context.Context {
	t := NewTiming()
	return context.WithValue(ctx, timingContextKey, t)
}

func GetTiming(ctx context.Context) *Timing {
	tca := ctx.Value(timingContextKey)
	if tca == nil {
		panic("no timing context present")
	}
	if tc, ok := tca.(*Timing); ok {
		return tc
	}
	panic("invalid timing context type")
}

func NewTiming() *Timing {
	t := &Timing{
		asyncProcesses: map[string]*Timing{},
	}

	root := &Location{
		Name:  "", // root
		Owner: t,
	}
	t.current = root
	t.root = root

	return t
}

func (t *Timing) Start(name string) Completed {
	if len(name) == 0 {
		panic("timing name much be a non-zero length string")
	}

	t.mu.Lock()
	parent := t.current
	l := parent.getSubLocation(name)
	t.current = l
	l.EntryCount++
	t.mu.Unlock()

	startTime := time.Now()
	return func() {
		d := time.Since(startTime)

		t.mu.Lock()
		l.ExitCount++
		l.TotalDuration += d
		t.current = parent
		t.mu.Unlock()
	}
}

func (t *Timing) BeginAsyncContext(ctx context.Context, name string) context.Context {
	child := t.BeginAsync(name)
	return context.WithValue(ctx, timingContextKey, child)
}

func (t *Timing) BeginAsync(name string) *Timing {
	child := NewTiming()
	t.mu.Lock()
	t.asyncProcesses[name] = child
	t.mu.Unlock()
	return child
}

func (l *Location) getSubLocation(name string) *Location {
	if l.Children == nil {
		l.Children = map[string]*Location{}
	}

	if l, ok := l.Children[name]; ok {
		return l
	}
	c := &Location{
		Name:  name,
		Owner: l.Owner,
	}
	l.Children[name] = c
	return c
}

func (t *Timing) String() string {
	t.mu.Lock()
	b := strings.Builder{}
	t.root.dumpToBuilder(&b, "")
	t.mu.Unlock()
	return b.String()
}

func (l *Location) dumpToBuilder(b *strings.Builder, prefix string) {
	var childPrefix string
	if len(l.Name) > 0 {
		b.WriteString(prefix)
		b.WriteString(fmt.Sprintf("%s - %s", l.Name, l.TotalDuration.Round(time.Microsecond)))
		if l.EntryCount != l.ExitCount {
			b.WriteString(fmt.Sprintf(" entries: %d exits: %d", l.EntryCount, l.ExitCount))
		} else if l.EntryCount > 1 {
			b.WriteString(fmt.Sprintf(" calls: %d", l.EntryCount))
		}
		b.WriteString("\n")
		childPrefix = prefix + "." + l.Name
	} else {
		childPrefix = prefix
	}

	var keys []string
	for k := range l.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l := l.Children[k]
		l.dumpToBuilder(b, childPrefix)
	}
}
