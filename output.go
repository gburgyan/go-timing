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
func (l *Location) dumpToBuilder(b *strings.Builder, prefix, separator, path string, durFmr DurationFormatter, excludeChildren bool) {
	var childPrefix string
	if l.Name == "" {
		childPrefix = path
	} else {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		reportDuration := l.TotalDuration
		if excludeChildren && !l.Async {
			reportDuration -= l.TotalChildDuration()
		}
		b.WriteString(prefix)
		b.WriteString(path)
		var effectiveName string
		if l.Async {
			effectiveName = "[" + l.Name + "]"
		} else {
			effectiveName = l.Name
		}
		b.WriteString(effectiveName)
		b.WriteString(" - ")
		if l.EntryCount > 0 {
			if durFmr == nil {
				b.WriteString(reportDuration.String())
			} else {
				b.WriteString(durFmr(reportDuration))
			}
			if l.EntryCount != l.ExitCount {
				b.WriteString(fmt.Sprintf(" entries: %d exits: %d", l.EntryCount, l.ExitCount))
			} else if l.EntryCount > 1 {
				b.WriteString(fmt.Sprintf(" calls: %d", l.EntryCount))
			}
		}
		childPrefix = path + effectiveName + separator
	}
	var keys []string
	for k := range l.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l := l.Children[k]
		l.dumpToBuilder(b, prefix, separator, childPrefix, durFmr, excludeChildren)
	}
}

// dumpToMap is an internal function that recursively outputs the contents of each location
// to the map builder passed in.
func (l *Location) dumpToMap(m map[string]float64, separator, path string, divisor float64, excludeChildren bool) {
	var childPrefix string
	if l.Name == "" {
		childPrefix = path
	} else {
		reportDuration := l.TotalDuration
		if excludeChildren {
			reportDuration -= l.TotalChildDuration()
		}
		key := fmt.Sprintf("%s%s", path, l.Name)
		if l.EntryCount > 0 {
			m[key] = float64(reportDuration.Nanoseconds()) / divisor
		}
		childPrefix = path + l.Name + separator
	}
	for _, c := range l.Children {
		c.dumpToMap(m, separator, childPrefix, divisor, excludeChildren)
	}
}
