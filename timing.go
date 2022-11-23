package timing

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Timing struct {
	root      *Location
	current   *Location
	subThread bool
}

type Location struct {
	Name          string
	EntryCount    int
	ExitCount     int
	TotalDuration time.Duration
	Children      map[string]*Location
}

type Completed func()

type key int

const timingContextKey key = 0

func ContextWithTiming(ctx context.Context) context.Context {
	t := NewTiming("")
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

func (t *Timing) BeginSubThreadContext(ctx context.Context, name string) context.Context {
	child := t.BeginSubThread(name)
	return context.WithValue(ctx, timingContextKey, child)
}

func (t *Timing) BeginSubThread(name string) *Timing {
	childLoc := t.current.getSubLocation(name)
	if childLoc.EntryCount > 0 {
		panic("sub-threads require a new timing location")
	}

	child := &Timing{
		current:   childLoc,
		root:      childLoc,
		subThread: true,
	}

	return child
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

func (t *Timing) String() string {
	b := strings.Builder{}
	t.root.dumpToBuilder(&b, "", "")
	return b.String()
}

func (l *Location) dumpToBuilder(b *strings.Builder, prefix, path string) {
	var childPrefix string
	if len(l.Name) > 0 {
		b.WriteString(fmt.Sprintf("%s%s%s", prefix, path, l.Name))
		if l.EntryCount > 0 {
			b.WriteString(fmt.Sprintf(" - %s", l.TotalDuration.Round(time.Microsecond)))
			if l.EntryCount != l.ExitCount {
				b.WriteString(fmt.Sprintf(" entries: %d exits: %d", l.EntryCount, l.ExitCount))
			} else if l.EntryCount > 1 {
				b.WriteString(fmt.Sprintf(" calls: %d", l.EntryCount))
			}
		}
		b.WriteString("\n")
		childPrefix = path + l.Name + "."
	} else {
		childPrefix = path
	}

	var keys []string
	for k := range l.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l := l.Children[k]
		l.dumpToBuilder(b, prefix, childPrefix)
	}
}
