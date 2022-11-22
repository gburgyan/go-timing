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

	assert.Contains(t, timing.locations, "test")
	testLocation := timing.locations["test"]
	assert.Equal(t, 1, testLocation.entryCount)
	assert.Equal(t, 0, testLocation.exitCount)
	assert.Len(t, timing.locationStack, 1)
	testComplete()
	assert.Len(t, timing.locationStack, 0)

	testComplete2 := timing.Start("test")
	assert.Equal(t, 2, testLocation.entryCount)
	assert.Equal(t, 1, testLocation.exitCount)
	assert.Len(t, timing.locationStack, 1)
	testComplete2()
	assert.Len(t, timing.locationStack, 0)

	assert.Equal(t, 2, testLocation.entryCount)
	assert.Equal(t, 2, testLocation.exitCount)
	assert.Greater(t, testLocation.totalDuration, time.Duration(0))

	// Force a time
	testLocation.totalDuration = 100 * time.Millisecond

	assert.Equal(t, "test - 100.0000ms calls: 2\n", timing.String())
}
