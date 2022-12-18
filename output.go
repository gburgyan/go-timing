package timing

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReportOptions struct {
	Prefix            string
	Separator         string
	DurationFormatter DurationFormatter
	ExcludeChildren   bool
	Compact           bool
}

// DurationFormatter is a function to format a reported duration in whatever way you need.
type DurationFormatter func(d time.Duration) string

// dumpToBuilder is an internal function that recursively outputs the contents of each location
// to the string builder passed in.
func (l *Location) dumpToBuilder(b *strings.Builder, path string, options *ReportOptions) {
	var childPrefix string
	if l.Name == "" {
		childPrefix = path
	} else {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		reportDuration := l.TotalDuration
		if options.ExcludeChildren && !l.Async {
			reportDuration -= l.TotalChildDuration()
		}
		b.WriteString(options.Prefix)
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
			if options.DurationFormatter == nil {
				b.WriteString(reportDuration.String())
			} else {
				b.WriteString(options.DurationFormatter(reportDuration))
			}
			if l.EntryCount != l.ExitCount {
				b.WriteString(fmt.Sprintf(" entries: %d exits: %d", l.EntryCount, l.ExitCount))
			} else if l.EntryCount > 1 {
				b.WriteString(fmt.Sprintf(" calls: %d", l.EntryCount))
			}
		}
		b.WriteString(l.formatDetails(options.Prefix))
		childPrefix = path + effectiveName + options.Separator
	}
	var keys []string
	for k := range l.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l := l.Children[k]
		l.dumpToBuilder(b, childPrefix, options)
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

func (l *Location) formatDetails(prefix string) string {
	if len(l.Details) == 0 {
		return ""
	}
	var keys []string
	for k := range l.Details {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	anyNewlines := false
	formattedDetails := map[string]string{}
	for _, k := range keys {
		o := l.Details[k]
		s := fmt.Sprintf("%+v", o)
		if strings.Contains(s, "\n") {
			anyNewlines = true
		}
		formattedDetails[k] = s
	}
	builder := strings.Builder{}
	if !anyNewlines {
		builder.WriteString(" (")
		for i, k := range keys {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(k)
			builder.WriteString(":")
			builder.WriteString(formattedDetails[k])
		}
		builder.WriteString(")")
	} else {
		const baseIndent = 4
		for _, k := range keys {
			lines := strings.Split(strings.TrimRight(formattedDetails[k], "\n"), "\n")
			keyIndent := len(k) + 1 + baseIndent
			for i, line := range lines {
				builder.WriteString("\n")
				builder.WriteString(prefix)
				if i == 0 {
					builder.WriteString(strings.Repeat(" ", baseIndent))
					builder.WriteString(k)
					builder.WriteString(":")
					builder.WriteString(line)
				} else {
					builder.WriteString(strings.Repeat(" ", keyIndent))
					builder.WriteString(line)
				}
			}
		}
	}
	return builder.String()
}
