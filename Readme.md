![Build status](https://github.com/gburgyan/go-timing/actions/workflows/go.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/gburgyan/go-timing)](https://goreportcard.com/report/github.com/gburgyan/go-timing) [![PkgGoDev](https://pkg.go.dev/badge/github.com/gburgyan/go-timing)](https://pkg.go.dev/github.com/gburgyan/go-timing)

# About

Often you want to know where time is being spent for processing a request. This is even more important as you get to more and more complex systems. `go-timing` provides a simple and comprehensive way of recording a hierarchical history of where the time goes.

# Installation

```bash
go get github.com/gburgyan/go-timing
```

# Usage

The simplest use case of this is just start up a timing context and pass that context along to functions you call as you would for any other context:

```go
func ProcessRequest(ctx context.Context) result {
tCtx := timing.Start(ctx, "ProcessRequest")

someFunction(tCtx)
otherFunction(tCtx)
// other processing

tCtx.Complete()

fmt.Print(tCtx)
}

func someFunction(ctx context.Context) {
tCtx := timing.Start(ctx, "someFunction")
defer tCtx.Complete()
// Do work
}

func someFunction(ctx context.Context) {
tCtx := timing.Start(ctx, "otherFunction")
defer tCtx.Complete()
// Do work
}
```

The returned `tCtx` is a context object like any other. This one has the feature that if can track timings. Additionally, if when starting a timing context, there exists a timnig context on the timing stack, the new timing context is added as a child of the parent.

# Reporting

## String()

Since we keep track of the parent-child relationships on the timing, the results can be shown in a tree-like format:

```text
ProcessRequest - 320ms
ProcessRequest > someFunction - 120ms
ProcessRequest > otherFunction - 185ms
```

## Report

If you need more control over the output, you can call `Report` on the timing context. This allows you to apply some formatting to the output:

* A prefix that is written out before each line (default is nothing)
* The separator that is printed between levels (default is " > ")
* A duration formatter (`nil` invokes the default `duration.String()` to format)
* If child times are to be excluded from the parent's time

The last option is important if you want to make a chart or graph of the times. Since time can be reported at multiple levels, it can be counted multiple times. The sum of the above example is 625ms, despite the fact that the run time is only 320ms. This would be misleading on something like a pie chart. By requesting that child times should be excluded, you could get:

```go
tCtx.Report("", " > ", nil, true)
```

Produces:

```text
ProcessRequest - 15ms
ProcessRequest > someFunction - 120ms
ProcessRequest > otherFunction - 185ms
```

This shows that outside of the calls to the children, `ProcessRequest` consumed 15ms on its own.

### Duration formatting

The default Golang `duration` formatting is great for human readability, but it's not as good for machine processing since it involves text parsing of the units. If you need to get something other than the provided functionality, you can pass in a function that takes a duration and returns a string. This allows you to do any transformations, rounding, scaling or anything else.

Originally this was implemented as an explosion of parameters to the function. This wound up being complex and still wouldn't allow for as much flexibility as desired. It was decided that delegating to a function that can do whatever the caller needs is the best solution.

## ReportMap

This is similar to, but simpler than, the text-based `Report` function. This formats the report into an even simpler `map[string]float64` of just the durations for the various timing contexts. This is intended to be easy to consume by a system like Splunk for reporting purposes.

## JSON

The timing context objects are decorated with JSON tags to allow serialization to JSON.

## Custom reporting

All the needed fields are public and easily navigable so if there is a need to output the timing in any other way, this should be easy to do.

# Thread Safety

The `go-timing` module is defined to be completely thread safe while the timings are being logged. There should be no case where a timing is lost or anything behaves incorrectly.

While the normal runtime is designed to be thread safe, the final reporting processes, including the `String()` and the `Report*()` functions, as well as any other interactions like serializing to JSON, are _not_ designed to be thread safe. The intent is that by the time those functions are called, all the processing that was supposed to be timed has already been completed. While not thread safe, the worst case is that incorrect data is printed out.

Logging times for processes that start on the main Goroutine, but end afterward is not supported. If you start a long-running process but log the timing report prior to its completion, you can have no idea how long that took because it's not completed yet. Since this is a logically inconsistent way of running, this is not supported.

If you need timing logs for a long-running process, the correct approach is to start a new `Root` timing context. Since that timing context is unrelated to the original one, everything is fine. When the long-running process has concluded (after the original Goroutine has long since finished), the long-running Goroutine can log its timing.