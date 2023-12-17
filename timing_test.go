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

	assert.Equal(t, uint32(0), tCtx.EntryCount)
	assert.Equal(t, uint32(0), tCtx.ExitCount)

	assert.Equal(t, "", tCtx.String())

	child, complete := Start(tCtx, "child")
	complete()

	child.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "child - 100ms", tCtx.String())
	m := tCtx.ReportMap(" > ", 1000000, false)
	assert.Len(t, m, 1)
	assert.Equal(t, 100.0, m["child"])
}

func Test_NonTrivialRoot(t *testing.T) {
	ctx := context.Background()

	tCtx, complete := Start(ctx, "root")
	time.Sleep(time.Millisecond)
	complete()

	assert.Equal(t, uint32(1), tCtx.EntryCount)
	assert.Equal(t, uint32(1), tCtx.ExitCount)
	assert.Greater(t, tCtx.TotalDuration, time.Duration(0))

	tCtx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "root - 100ms", tCtx.String())
}

func Test_Nesting(t *testing.T) {
	ctx := context.Background()

	rootCtx, rootComplete := Start(ctx, "root")

	assert.Equal(t, time.Duration(0), rootCtx.TotalChildDuration())

	child1Ctx, c1complete := Start(rootCtx, "child 1")
	c1complete()
	child1Ctx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, 100*time.Millisecond, rootCtx.TotalChildDuration())

	child2Ctx, c2complete := Start(rootCtx, "child 2")
	c2complete()
	child2Ctx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, 200*time.Millisecond, rootCtx.TotalChildDuration())

	rootComplete()

	rootCtx.TotalDuration = 210 * time.Millisecond

	assert.Equal(t, "root - 210ms\nroot > child 1 - 100ms\nroot > child 2 - 100ms", rootCtx.String())
	assert.Equal(t, "root - 10ms\nroot.child 1 - 100ms\nroot.child 2 - 100ms", rootCtx.Report(ReportOptions{Separator: ".", ExcludeChildren: true}))
	assert.Equal(t, "root - 210ms\nroot.child 1 - 100ms\nroot.child 2 - 100ms", rootCtx.Report(ReportOptions{Separator: "."}))
	custFmt := func(d time.Duration) string {
		return strconv.Itoa(int(d.Milliseconds()))
	}
	assert.Equal(t, "root - 210\nroot.child 1 - 100\nroot.child 2 - 100", rootCtx.Report(ReportOptions{Separator: ".", DurationFormatter: custFmt}))
	assert.Equal(t, "root - 210ms\n | child 1 - 100ms\n | child 2 - 100ms", rootCtx.Report(ReportOptions{Compact: true}))

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
	expected := `{"name":"root","children":{"child 1":{"name":"child 1","entry-count":1,"exit-count":1,"total-duration":100000000},"child 2":{"name":"child 2","entry-count":1,"exit-count":1,"total-duration":100000000}},"entry-count":1,"exit-count":1,"total-duration":210000000}`
	assert.Equal(t, expected, string(js))
}

func Test_ContextBehavior(t *testing.T) {
	type tt struct {
		v int
	}
	o1 := tt{42}
	o2 := tt{105}

	ctx := context.Background()

	ctxV1 := context.WithValue(ctx, 1, o1)

	rootCtx, _ := Start(ctxV1, "root")

	ctxV2 := context.WithValue(rootCtx, 2, o2)

	child2Ctx, _ := Start(ctxV2, "child 1")

	assert.Equal(t, o1, child2Ctx.Value(1))
	assert.Equal(t, o2, child2Ctx.Value(2))

	expected := `root - 0s entries: 1 exits: 0
root > child 1 - 0s entries: 1 exits: 0`
	assert.Equal(t, expected, rootCtx.String())
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
		StartRoot(nil, "panic")
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
	ctxT, _ := Start(context.Background(), "timer")
	assert.Nil(t, ctxT.Done())
	deadline, ok := ctxT.Deadline()
	assert.True(t, deadline.IsZero())
	assert.False(t, ok)
	assert.NoError(t, ctxT.Err())
}

func Test_MultiStart(t *testing.T) {
	ctx := context.Background()

	rootCtx, rootComplete := Start(ctx, "root")

	child1Ctx, c1complete := Start(rootCtx, "child 1")
	c1complete()

	child1Ctx, c1complete = Start(rootCtx, "child 1")
	c1complete()

	child2Ctx, c2complete := Start(rootCtx, "child 2")
	c2complete()

	rootComplete()
	rootCtx.TotalDuration = 200 * time.Millisecond
	child1Ctx.TotalDuration = 100 * time.Millisecond
	child2Ctx.TotalDuration = 100 * time.Millisecond

	expected := `root - 200ms
root > child 1 - 100ms calls: 2 (50ms/call)
root > child 2 - 100ms`

	assert.Equal(t, expected, rootCtx.String())
	custFmt := func(d time.Duration) string {
		return strconv.Itoa(int(d.Milliseconds()))
	}

	expected = `root - 200
root > child 1 - 100 calls: 2 (50/call)
root > child 2 - 100`

	assert.Equal(t, expected, rootCtx.Report(ReportOptions{DurationFormatter: custFmt}))
}

func Test_MultiRoot(t *testing.T) {
	ctx := context.Background()

	rootCtx, rootComplete := Start(ctx, "root")

	child1Ctx, c1complete := Start(rootCtx, "child 1")

	root2Ctx, grComplete := StartRoot(child1Ctx, "goroutine")
	grComplete()
	root2Ctx.TotalDuration = 100 * time.Millisecond

	c1complete()

	rootComplete()
	rootCtx.TotalDuration = 200 * time.Millisecond
	child1Ctx.TotalDuration = 100 * time.Millisecond

	expected := `root - 200ms
root > child 1 - 100ms`

	assert.Equal(t, expected, rootCtx.String())
	assert.Equal(t, "goroutine - 100ms", root2Ctx.String())
}

func Test_Async(t *testing.T) {
	ctx := context.Background()

	rootCtx, rootComplete := Start(ctx, "root")
	rootCtx.Async = true

	child1Ctx, c1Complete := Start(rootCtx, "child 1")
	c1Complete()

	child1Ctx, c1Complete = Start(rootCtx, "child 1")
	c1Complete()

	child2Ctx, c2Complete := Start(rootCtx, "child 2")
	c2Complete()

	rootComplete()
	rootCtx.TotalDuration = 110 * time.Millisecond
	child1Ctx.TotalDuration = 100 * time.Millisecond
	child2Ctx.TotalDuration = 100 * time.Millisecond

	expected := `[root] - 110ms
[root] > child 1 - 100ms calls: 2 (50ms/call)
[root] > child 2 - 100ms`
	assert.Equal(t, expected, rootCtx.Report(ReportOptions{ExcludeChildren: true}))
}

func Test_Async2(t *testing.T) {
	ctx := context.Background()

	rootCtx, rootComplete := StartAsync(ctx, "root")

	child1Ctx, c1Complete := Start(rootCtx, "child 1")
	c1Complete()

	child1Ctx, c1Complete = Start(rootCtx, "child 1")
	c1Complete()

	child2Ctx, c2Complete := Start(rootCtx, "child 2")
	c2Complete()

	rootComplete()
	rootCtx.TotalDuration = 110 * time.Millisecond
	child1Ctx.TotalDuration = 100 * time.Millisecond
	child2Ctx.TotalDuration = 100 * time.Millisecond

	expected := `[root] - 110ms
[root] > child 1 - 100ms calls: 2 (50ms/call)
[root] > child 2 - 100ms`
	assert.Equal(t, expected, rootCtx.Report(ReportOptions{ExcludeChildren: true}))
}

func Test_ReentrantPanics(t *testing.T) {
	ctx := context.Background()

	rootCtx, _ := Start(ctx, "root")
	_, childComplete := Start(rootCtx, "child")

	Start(rootCtx, "child") // Ignores returns

	childComplete()
	assert.Panics(t, func() {
		childComplete()
	})

	fmt.Print()
}

func Test_DetailsPlain(t *testing.T) {
	ctx := context.Background()

	rootCtx, rootComplete := Start(ctx, "root")
	rootComplete()

	rootCtx.TotalDuration = time.Microsecond
	rootCtx.AddDetails("string", "foo")
	rootCtx.AddDetails("int", 42)

	assert.Equal(t, "root - 1µs (int:42, string:foo)", rootCtx.String())
}

func Test_DetailsNewlines(t *testing.T) {
	ctx := context.Background()

	rootCtx, rootComplete := Start(ctx, "root")
	childCtx, childComplete := Start(rootCtx, "child")
	childComplete()
	rootComplete()

	rootCtx.TotalDuration = 100 * time.Microsecond
	rootCtx.AddDetails("short", "alice\nbob\ncarol\n")
	rootCtx.AddDetails("longer", "alice\neve\nbob")

	childCtx.TotalDuration = 50 * time.Microsecond
	childCtx.AddDetails("lines", "multiple\nlines")

	result := rootCtx.String()
	//fmt.Println(result)
	expected := `root - 100µs
    longer:alice
           eve
           bob
    short:alice
          bob
          carol
root > child - 50µs
    lines:multiple
          lines`
	assert.Equal(t, expected, result)

	result = rootCtx.Report(ReportOptions{Prefix: "* "})
	//fmt.Println(result)
	expected = `* root - 100µs
*     longer:alice
*            eve
*            bob
*     short:alice
*           bob
*           carol
* root > child - 50µs
*     lines:multiple
*           lines`
	assert.Equal(t, expected, result)

	result = rootCtx.Report(ReportOptions{Prefix: "* ", Separator: " | ", Compact: true})
	//fmt.Println(result)
	expected = `* root - 100µs
*  |     longer:alice
*  |            eve
*  |            bob
*  |     short:alice
*  |           bob
*  |           carol
*  | child - 50µs
*  |  |     lines:multiple
*  |  |           lines`
	assert.Equal(t, "* root - 100µs\n*  |     longer:alice\n*  |            eve\n*  |            bob\n*  |     short:alice\n*  |           bob\n*  |           carol\n*  | child - 50µs\n*  |  |     lines:multiple\n*  |  |           lines", result)
}
