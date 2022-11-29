package timing

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// dumpToBuilder is an internal function that recursively outputs the contents of each location
// to the string builder passed in.
func (c *Context) dumpToBuilder(b *strings.Builder, prefix, separator, path string, onlyLeaf bool) {
	var childPrefix string
	printLine := !onlyLeaf || (c.Children == nil || len(c.Children) == 0)
	if c.Name == "" {
		childPrefix = path
	} else {
		if printLine {
			b.WriteString(fmt.Sprintf("%s%s%s", prefix, path, c.Name))
			if c.EntryCount > 0 {
				b.WriteString(fmt.Sprintf(" - %s", c.TotalDuration.Round(time.Microsecond)))
				if c.EntryCount != c.ExitCount {
					b.WriteString(fmt.Sprintf(" entries: %d exits: %d", c.EntryCount, c.ExitCount))
				} else if c.EntryCount > 1 {
					b.WriteString(fmt.Sprintf(" calls: %d", c.EntryCount))
				}
				b.WriteString("\n")
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
		l.dumpToBuilder(b, prefix, separator, childPrefix, onlyLeaf)
	}
}

// dumpToMap is an internal function that recursively outputs the contents of each location
// to the map builder passed in.
func (c *Context) dumpToMap(m map[string]float64, separator, path string, divisor float64, onlyLeaf bool) {
	var childPrefix string
	reportLine := !onlyLeaf || (c.Children == nil || len(c.Children) == 0)
	if c.Name == "" {
		childPrefix = path
	} else {
		if reportLine {
			key := fmt.Sprintf("%s%s", path, c.Name)
			if c.EntryCount > 0 {
				m[key] = float64(c.TotalDuration.Nanoseconds()) / divisor
			}
		}
		childPrefix = path + c.Name + separator
	}
	for _, c := range c.Children {
		c.dumpToMap(m, separator, childPrefix, divisor, onlyLeaf)
	}
}
