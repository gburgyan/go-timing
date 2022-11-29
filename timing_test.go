package timing

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_TrivialRoot(t *testing.T) {
	ctx := context.Background()

	tCtx := Root(ctx)

	assert.Equal(t, 0, tCtx.EntryCount)
	assert.Equal(t, 0, tCtx.ExitCount)

	assert.Equal(t, "", tCtx.String())
}

func Test_NonTrivialRoot(t *testing.T) {
	ctx := context.Background()

	tCtx := Start(ctx, "root")
	tCtx.Complete()

	assert.Equal(t, 1, tCtx.EntryCount)
	assert.Equal(t, 1, tCtx.ExitCount)
	assert.Greater(t, tCtx.TotalDuration, time.Duration(0))

	tCtx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "root - 100ms\n", tCtx.String())
}

func Test_Nesting(t *testing.T) {
	ctx := context.Background()

	rootCtx := Start(ctx, "root")

	child1Ctx := Start(rootCtx, "child 1")
	child1Ctx.Complete()

	child2Ctx := Start(rootCtx, "child 2")
	child2Ctx.Complete()

	rootCtx.Complete()

	rootCtx.TotalDuration = 200 * time.Millisecond
	child1Ctx.TotalDuration = 100 * time.Millisecond
	child2Ctx.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "root - 200ms\nroot > child 1 - 100ms\nroot > child 2 - 100ms\n", rootCtx.String())
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

	assert.Equal(t, "root - 0s entries: 1 exits: 0\nroot > child 1 - 0s entries: 1 exits: 0\n", rootCtx.String())
}
