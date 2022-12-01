package timing

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// DurationFormatter is a function to format a reported duration in whatever way you need.
type DurationFormatter func(d time.Duration) string

// dumpToBuilder is an internal function that recursively outputs the contents of each location
// to the string builder passed in.
func (c *Context) dumpToBuilder(b *strings.Builder, prefix, separator, path string, durFmr DurationFormatter, excludeChildren bool) {
	var childPrefix string
	if c.Name == "" {
		childPrefix = path
	} else {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		reportDuration := c.TotalDuration
		if excludeChildren {
			reportDuration -= c.TotalChildDuration()
		}
		b.WriteString(prefix)
		b.WriteString(path)
		b.WriteString(c.Name)
		b.WriteString(" - ")
		if c.EntryCount > 0 {
			if durFmr == nil {
				b.WriteString(reportDuration.String())
			} else {
				b.WriteString(durFmr(reportDuration))
			}
			if c.EntryCount != c.ExitCount {
				b.WriteString(fmt.Sprintf(" entries: %d exits: %d", c.EntryCount, c.ExitCount))
			} else if c.EntryCount > 1 {
				b.WriteString(fmt.Sprintf(" calls: %d", c.EntryCount))
			}
		}
		childPrefix = path + c.Name + separator
	}
	var keys []string
	for k := range c.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l := c.Children[k]
		l.dumpToBuilder(b, prefix, separator, childPrefix, durFmr, excludeChildren)
	}
}

// dumpToMap is an internal function that recursively outputs the contents of each location
// to the map builder passed in.
func (c *Context) dumpToMap(m map[string]float64, separator, path string, divisor float64, excludeChildren bool) {
	var childPrefix string
	if c.Name == "" {
		childPrefix = path
	} else {
		reportDuration := c.TotalDuration
		if excludeChildren {
			reportDuration -= c.TotalChildDuration()
		}
		key := fmt.Sprintf("%s%s", path, c.Name)
		if c.EntryCount > 0 {
			m[key] = float64(reportDuration.Nanoseconds()) / divisor
		}
		childPrefix = path + c.Name + separator
	}
	for _, c := range c.Children {
		c.dumpToMap(m, separator, childPrefix, divisor, excludeChildren)
	}
}
