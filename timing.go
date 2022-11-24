package timing

import (
	"context"
	"strings"
	"time"
)

type Timing struct {
	root    *Location
	current *Location
}

type Location struct {
	Name          string               `json:"-"`
	EntryCount    int                  `json:"entry-count,omitempty"`
	ExitCount     int                  `json:"exit-count,omitempty"`
	TotalDuration time.Duration        `json:"total-duration-ns,omitempty"`
	Children      map[string]*Location `json:"children,omitempty"`
	SubRoot       bool                 `json:"sub-root,omitempty"`
}

type Completed func()

type key int

const timingContextKey key = 0

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

func (t *Timing) BeginSubRootContext(ctx context.Context, name string) context.Context {
	child := t.BeginSubRoot(name)
	return context.WithValue(ctx, timingContextKey, child)
}

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

func (t *Timing) Root() map[string]*Location {
	return t.root.Children
}

func (t *Timing) String() string {
	b := strings.Builder{}
	t.root.dumpToBuilder(&b, false, "", " > ", "")
	return b.String()
}
