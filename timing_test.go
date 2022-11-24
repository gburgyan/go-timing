package timing

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestContextWithTiming(t *testing.T) {
	ctx := ContextWithTiming(context.Background())

	timing := FromContext(ctx)
	testComplete := timing.Start("test")

	assert.Contains(t, timing.root.Children, "test")
	testLocation := timing.root.Children["test"]
	assert.Equal(t, 1, testLocation.EntryCount)
	assert.Equal(t, 0, testLocation.ExitCount)
	testComplete()
	testLocation.TotalDuration = 50 * time.Millisecond
	assert.Equal(t, "test - 50ms\n", timing.String())

	testComplete2 := timing.Start("test")
	assert.Equal(t, 2, testLocation.EntryCount)
	assert.Equal(t, 1, testLocation.ExitCount)
	assert.Equal(t, "test - 50ms entries: 2 exits: 1\n", timing.String())
	testComplete2()

	assert.Equal(t, 2, testLocation.EntryCount)
	assert.Equal(t, 2, testLocation.ExitCount)
	assert.Greater(t, testLocation.TotalDuration, time.Duration(0))

	// Force a time
	testLocation.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "test - 100ms calls: 2\n", timing.String())
}

func TestContextWithSubThreads(t *testing.T) {
	ctx := ContextWithTiming(context.Background())

	outsideComplete := Start(ctx, "test")

	// Prentend this is a new Goroutine
	threadCtx := BeginSubRoot(ctx, "thread")
	insideComplete := Start(threadCtx, "inside")
	insideComplete()
	// End of Goroutine

	outsideComplete()

	timing := FromContext(ctx)
	timing.root.Children["test"].TotalDuration = 250 * time.Millisecond
	timing.root.Children["test"].Children["thread"].Children["inside"].TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "test - 250ms\ntest > (thread) - new timing context\ntest > (thread) > inside - 100ms\n", timing.String())

	js, err := json.Marshal(Root(ctx))
	assert.NoError(t, err)
	// Indented
	// {
	//   "test": {
	//     "entry-count": 1,
	//     "exit-count": 1,
	//     "total-duration-ns": 250000000,
	//     "children": {
	//       "thread": {
	//         "children": {
	//           "inside": {
	//             "entry-count": 1,
	//             "exit-count": 1,
	//             "total-duration-ns": 100000000
	//           }
	//         },
	//         "sub-root": true
	//       }
	//     }
	//   }
	// }
	assert.Equal(t, "{\"test\":{\"entry-count\":1,\"exit-count\":1,\"total-duration-ns\":250000000,\"children\":{\"thread\":{\"children\":{\"inside\":{\"entry-count\":1,\"exit-count\":1,\"total-duration-ns\":100000000}},\"sub-root\":true}}}}", string(js))
}

func TestUninitializedUse(t *testing.T) {
	assert.Panics(t, func() {
		Start(context.Background(), "fail")
	})
}

func TestInvalidName(t *testing.T) {
	ctx := ContextWithTiming(context.Background())
	assert.Panics(t, func() {
		Start(ctx, "")
	})
}

func TestInvalidNameSubRoot(t *testing.T) {
	ctx := ContextWithTiming(context.Background())
	Start(ctx, "task")()
	assert.Panics(t, func() {
		BeginSubRoot(ctx, "task")
	})
}

func TestMessedUpContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, timingContextKey, "Not correct")
	assert.Panics(t, func() {
		FromContext(ctx)
	})
}
