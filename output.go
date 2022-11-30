package timing

import (
	"fmt"
	"sort"
	"strings"
)

// dumpToBuilder is an internal function that recursively outputs the contents of each location
// to the string builder passed in.
func (c *Context) dumpToBuilder(b *strings.Builder, prefix, separator, path string, excludeChildren bool) {
	var childPrefix string
	if c.Name == "" {
		childPrefix = path
	} else {
		reportDuration := c.TotalDuration
		if excludeChildren {
			reportDuration -= c.TotalChildDuration()
		}
		b.WriteString(fmt.Sprintf("%s%s%s", prefix, path, c.Name))
		if c.EntryCount > 0 {
			b.WriteString(fmt.Sprintf(" - %s", reportDuration.Round(reportDuration)))
			if c.EntryCount != c.ExitCount {
				b.WriteString(fmt.Sprintf(" entries: %d exits: %d", c.EntryCount, c.ExitCount))
			} else if c.EntryCount > 1 {
				b.WriteString(fmt.Sprintf(" calls: %d", c.EntryCount))
			}
			b.WriteString("\n")
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
		l.dumpToBuilder(b, prefix, separator, childPrefix, excludeChildren)
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
