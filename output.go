package timing

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ReportOptions configures how the report is formatted.
type ReportOptions struct {
	// Prefix gets output prior to every new line in the output.
	Prefix string

	// Separator is output between every level of the output. If Compact is specified then
	// this is printed on the following line and subsequent lines are only indented by
	// len(Separator) spaces. If this is not specified the default is " > ".
	Separator string

	// DurationFormatter, if specified, is called to format durations. Otherwise, the default
	// Golang time.Duration String() is called.
	DurationFormatter DurationFormatter

	// ExcludeChildren controls if the child durations are subtracted from this duration or
	// not. If the Location is marked as Async then the child durations are not subtracted out
	// for that level.
	ExcludeChildren bool

	// Compact controls if the full path is output for each line or if levels are implied
	// with indents. This makes for a far smaller output for deep timing trees.
	Compact bool
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
			} else if l.ExitCount > 1 {
				b.WriteString(fmt.Sprintf(" calls: %d", l.EntryCount))
			}
			if l.ExitCount > 1 {
				perCallDuration := time.Duration(float64(reportDuration) / float64(l.ExitCount))
				var fmtCallDuration string
				if options.DurationFormatter == nil {
					fmtCallDuration = perCallDuration.String()
				} else {
					fmtCallDuration = options.DurationFormatter(perCallDuration)
				}
				b.WriteString(fmt.Sprintf(" (%s/call)", fmtCallDuration))
			}
		}
		b.WriteString(l.formatDetails(options.Prefix))
		if options.Compact {
			childPrefix = path + options.Separator
		} else {
			childPrefix = path + effectiveName + options.Separator
		}
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
