package timing

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestContextWithTiming(t *testing.T) {
	ctx := ContextWithTiming(context.Background())

	timing := GetTiming(ctx)
	testComplete := timing.Start("test")

	assert.Contains(t, timing.root.Children, "test")
	testLocation := timing.root.Children["test"]
	assert.Equal(t, 1, testLocation.EntryCount)
	assert.Equal(t, 0, testLocation.ExitCount)
	testComplete()

	testComplete2 := timing.Start("test")
	assert.Equal(t, 2, testLocation.EntryCount)
	assert.Equal(t, 1, testLocation.ExitCount)
	testComplete2()

	assert.Equal(t, 2, testLocation.EntryCount)
	assert.Equal(t, 2, testLocation.ExitCount)
	assert.Greater(t, testLocation.TotalDuration, time.Duration(0))

	// Force a time
	testLocation.TotalDuration = 100 * time.Millisecond

	assert.Equal(t, "test - 100ms calls: 2\n", timing.String())
}
