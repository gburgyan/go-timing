package timing

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// dumpToBuilder is an internal function that recursively outputs the contents of each location
// to the string builder passed in.
func (l *Location) dumpToBuilder(b *strings.Builder, prefix, separator, path string, onlyLeaf bool) {
	var childPrefix string
	printLine := !onlyLeaf || (l.Children == nil || len(l.Children) == 0)
	root := len(l.Name) == 0
	if l.SubRoot {
		name := "(" + l.Name + ")"
		if printLine {
			b.WriteString(fmt.Sprintf("%s%s%s - new timing context\n", prefix, path, name))
		}
		childPrefix = path + name + separator
	} else {
		if !root && printLine {
			b.WriteString(fmt.Sprintf("%s%s%s", prefix, path, l.Name))
			if l.EntryCount > 0 {
				b.WriteString(fmt.Sprintf(" - %s", l.TotalDuration.Round(time.Microsecond)))
				if l.EntryCount != l.ExitCount {
					b.WriteString(fmt.Sprintf(" entries: %d exits: %d", l.EntryCount, l.ExitCount))
				} else if l.EntryCount > 1 {
					b.WriteString(fmt.Sprintf(" calls: %d", l.EntryCount))
				}
				b.WriteString("\n")
			}
			childPrefix = path + l.Name + separator
		} else {
			childPrefix = path
		}
	}
	var keys []string
	for k := range l.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l := l.Children[k]
		l.dumpToBuilder(b, prefix, separator, childPrefix, onlyLeaf)
	}
}

// dumpToMap is an internal function that recursively outputs the contents of each location
// to the map builder passed in.
func (l *Location) dumpToMap(m map[string]float64, separator, path string, divisor float64, onlyLeaf bool) {
	var childPrefix string
	reportLine := !onlyLeaf || (l.Children == nil || len(l.Children) == 0)
	root := len(l.Name) == 0
	if l.SubRoot {
		name := "(" + l.Name + ")"
		childPrefix = path + name + separator
	} else {
		if !root && reportLine {
			key := fmt.Sprintf("%s%s", path, l.Name)
			if l.EntryCount > 0 {
				m[key] = float64(l.TotalDuration.Nanoseconds()) / divisor
			}
			childPrefix = path + l.Name + separator
		} else {
			childPrefix = path
		}
	}
	for _, c := range l.Children {
		c.dumpToMap(m, separator, childPrefix, divisor, onlyLeaf)
	}
}
