package timing

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func Test_TrivialRoot(t *testing.T) {
	ctx := context.Background()

	tCtx := Root(ctx)

	assert.Equal(t, 0, tCtx.EntryCount)
	assert.Equal(t, 0, tCtx.ExitCount)

	assert.Equal(t, "", tCtx.String())

	child := Start(tCtx, "child")
	child.Complete()

	child.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "child - 100ms", tCtx.String())
	m := tCtx.ReportMap(" > ", 1000000, false)
	assert.Len(t, m, 1)
	assert.Equal(t, 100.0, m["child"])
}

func Test_NonTrivialRoot(t *testing.T) {
	ctx := context.Background()

	tCtx := Start(ctx, "root")
	time.Sleep(time.Millisecond)
	tCtx.Complete()

	assert.Equal(t, 1, tCtx.EntryCount)
	assert.Equal(t, 1, tCtx.ExitCount)
	assert.Greater(t, tCtx.TotalDuration, time.Duration(0))

	tCtx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "root - 100ms", tCtx.String())
}

func Test_Nesting(t *testing.T) {
	ctx := context.Background()

	rootCtx := Start(ctx, "root")

	assert.Equal(t, time.Duration(0), rootCtx.TotalChildDuration())

	child1Ctx := Start(rootCtx, "child 1")
	child1Ctx.Complete()
	child1Ctx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, 100*time.Millisecond, rootCtx.TotalChildDuration())

	child2Ctx := Start(rootCtx, "child 2")
	child2Ctx.Complete()
	child2Ctx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, 200*time.Millisecond, rootCtx.TotalChildDuration())

	rootCtx.Complete()

	rootCtx.TotalDuration = 210 * time.Millisecond

	assert.Equal(t, "root - 210ms\nroot > child 1 - 100ms\nroot > child 2 - 100ms", rootCtx.String())
	assert.Equal(t, "root - 10ms\nroot.child 1 - 100ms\nroot.child 2 - 100ms", rootCtx.Report("", ".", nil, true))
	assert.Equal(t, "root - 210ms\nroot.child 1 - 100ms\nroot.child 2 - 100ms", rootCtx.Report("", ".", nil, false))
	custFmt := func(d time.Duration) string {
		return strconv.Itoa(int(d.Milliseconds()))
	}
	assert.Equal(t, "root - 210\nroot.child 1 - 100\nroot.child 2 - 100", rootCtx.Report("", ".", custFmt, false))

	fmt.Println(rootCtx)

	m := rootCtx.ReportMap(" > ", 1000000, true)
	assert.Len(t, m, 3)
	assert.Equal(t, 10.0, m["root"])
	assert.Equal(t, 100.0, m["root > child 1"])
	assert.Equal(t, 100.0, m["root > child 2"])

	m = rootCtx.ReportMap(".", 1000000, false)
	assert.Len(t, m, 3)
	assert.Equal(t, 210.0, m["root"])
	assert.Equal(t, 100.0, m["root.child 1"])
	assert.Equal(t, 100.0, m["root.child 2"])

	js, err := json.Marshal(rootCtx)
	assert.NoError(t, err)
	assert.Equal(t, "{\"name\":\"root\",\"children\":{\"child 1\":{\"name\":\"child 1\",\"entry-count\":1,\"exit-count\":1,\"total-duration\":100000000},\"child 2\":{\"name\":\"child 2\",\"entry-count\":1,\"exit-count\":1,\"total-duration\":100000000}},\"entry-count\":1,\"exit-count\":1,\"total-duration\":210000000}", string(js))
}

func Test_ContextBehavior(t *testing.T) {
	type tt struct {
		v int
	}
	o1 := tt{42}
	o2 := tt{105}

	ctx := context.Background()

	ctxV1 := context.WithValue(ctx, 1, o1)

	rootCtx := Start(ctxV1, "root")

	ctxV2 := context.WithValue(rootCtx, 2, o2)

	child2Ctx := Start(ctxV2, "child 1")

	assert.Equal(t, o1, child2Ctx.Value(1))
	assert.Equal(t, o2, child2Ctx.Value(2))

	assert.Equal(t, "root - 0s entries: 1 exits: 0\nroot > child 1 - 0s entries: 1 exits: 0", rootCtx.String())
}

func Test_StartPanics(t *testing.T) {
	assert.Panics(t, func() {
		Start(nil, "root")
	})
	assert.Panics(t, func() {
		Root(nil)
	})
	assert.Panics(t, func() {
		Start(context.Background(), "")
	})
	assert.Panics(t, func() {
		StartNew(nil, "panic")
	})
}

func Test_ParentTimingPanic(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextTimingKey, 1)
	assert.Panics(t, func() {
		findParentTiming(ctx)
	})
}

func Test_Context(t *testing.T) {
	ctxT := Start(context.Background(), "timer")
	assert.Nil(t, ctxT.Done())
	deadline, ok := ctxT.Deadline()
	assert.True(t, deadline.IsZero())
	assert.False(t, ok)
	assert.NoError(t, ctxT.Err())
}

func Test_MultiStart(t *testing.T) {
	ctx := context.Background()

	rootCtx := Start(ctx, "root")

	child1Ctx := Start(rootCtx, "child 1")
	child1Ctx.Complete()

	child1Ctx = Start(rootCtx, "child 1")
	child1Ctx.Complete()

	child2Ctx := Start(rootCtx, "child 2")
	child2Ctx.Complete()

	rootCtx.Complete()
	rootCtx.TotalDuration = 200 * time.Millisecond
	child1Ctx.TotalDuration = 100 * time.Millisecond
	child2Ctx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "root - 200ms\nroot > child 1 - 100ms calls: 2\nroot > child 2 - 100ms", rootCtx.String())
}

func Test_MultiRoot(t *testing.T) {
	ctx := context.Background()

	rootCtx := Start(ctx, "root")

	child1Ctx := Start(rootCtx, "child 1")

	root2Ctx := StartNew(child1Ctx, "goroutine")
	root2Ctx.Complete()
	root2Ctx.TotalDuration = 100 * time.Millisecond

	child1Ctx.Complete()

	rootCtx.Complete()
	rootCtx.TotalDuration = 200 * time.Millisecond
	child1Ctx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "root - 200ms\nroot > child 1 - 100ms", rootCtx.String())
	assert.Equal(t, "goroutine - 100ms", root2Ctx.String())
}

func Test_ReentrantPanics(t *testing.T) {
	ctx := context.Background()

	rootCtx := Start(ctx, "root")
	childCtx := Start(rootCtx, "child")

	assert.Panics(t, func() {
		Start(rootCtx, "child")
	})

	childCtx.Complete()
	assert.Panics(t, func() {
		childCtx.Complete()
	})

	fmt.Print()
}
