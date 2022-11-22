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
	locations      map[string]*location
	locationStack  []string
}

type location struct {
	level         int
	name          string
	fullName      string
	totalDuration time.Duration
	entryCount    int
	exitCount     int
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
	return &Timing{
		asyncProcesses: map[string]*Timing{},
		locations:      map[string]*location{},
	}
}

func (t *Timing) Start(name string) Completed {
	l := t.getSubLocation(name)
	l.entryCount++
	startTime := time.Now()
	return func() {
		d := time.Since(startTime)
		l.exitCount++
		l.totalDuration += d
		t.locationStack = t.locationStack[0 : l.level-1]
	}
}

func (t *Timing) BeginAsyncContext(ctx context.Context, name string) context.Context {
	child := t.BeginAsync(name)
	return context.WithValue(ctx, timingContextKey, child)
}

func (t *Timing) BeginAsync(name string) *Timing {
	child := NewTiming()
	t.asyncProcesses[name] = child
	return child
}

func (t *Timing) getSubLocation(name string) *location {
	t.locationStack = append(t.locationStack, name)
	fullName := strings.Join(t.locationStack, ".")
	t.mu.Lock()
	defer t.mu.Unlock()
	if l, ok := t.locations[fullName]; ok {
		return l
	}
	l := &location{
		name:     name,
		fullName: fullName,
		level:    len(t.locationStack),
	}
	t.locations[fullName] = l
	return l
}

func (t *Timing) String() string {
	var keys []string
	b := strings.Builder{}
	for k := range t.locations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l := t.locations[k]
		b.WriteString(fmt.Sprintf("%s - %.4fms", k, float64(l.totalDuration.Microseconds()/1000.0)))
		if l.entryCount != l.exitCount {
			b.WriteString(fmt.Sprintf(" entries: %d exits: %d", l.entryCount, l.exitCount))
		} else if l.entryCount > 1 {
			b.WriteString(fmt.Sprintf(" calls: %d", l.entryCount))
		}
		b.WriteString("\n")
	}
	return b.String()
}
