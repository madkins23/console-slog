# console-slog

[![Go Reference](https://pkg.go.dev/badge/github.com/phsym/console-slog.svg)](https://pkg.go.dev/github.com/phsym/console-slog) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/phsym/console-slog/master/LICENSE) [![Build](https://github.com/phsym/console-slog/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/phsym/slog-console/actions/workflows/go.yml) [![codecov](https://codecov.io/gh/phsym/console-slog/graph/badge.svg?token=ZIJT9L79QP)](https://codecov.io/gh/phsym/console-slog) [![Go Report Card](https://goreportcard.com/badge/github.com/phsym/console-slog)](https://goreportcard.com/report/github.com/phsym/console-slog)

A handler for slog that prints colorized logs, similar to zerolog's console writer output without sacrificing performances.

## Installation
```bash
go get github.com/phsym/console-slog@latest
```

## Example
```go
package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/phsym/console-slog"
)

func main() {
	logger := slog.New(
		console.NewHandler(os.Stderr, &console.HandlerOptions{Level: slog.LevelDebug}),
	)
	slog.SetDefault(logger)
	slog.Info("Hello world!", "foo", "bar")
	slog.Debug("Debug message")
	slog.Warn("Warning message")
	slog.Error("Error message", "err", errors.New("the error"))

	logger = logger.With("foo", "bar").
		WithGroup("the-group").
		With("bar", "baz")

	logger.Info("group info", "attr", "value")
}
```

![output](./doc/img/output.png)

When setting `console.HandlerOptions.AddSource` to `true`:
```go
console.NewHandler(os.Stderr, &console.HandlerOptions{Level: slog.LevelDebug, AddSource: true})
```
![output-with-source](./doc/img/output-with-source.png)

## Performances
See [benchmark file](./bench_test.go) for details.

The handler itself performs quite well compared to std-lib's handlers. It does no allocation:
```
goos: linux
goarch: amd64
pkg: github.com/phsym/console-slog
cpu: Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz
BenchmarkHandlers/dummy-4               128931026            8.732 ns/op               0 B/op          0 allocs/op
BenchmarkHandlers/console-4               849837              1294 ns/op               0 B/op          0 allocs/op
BenchmarkHandlers/std-text-4              542583              2097 ns/op               4 B/op          2 allocs/op
BenchmarkHandlers/std-json-4              583784              1911 ns/op             120 B/op          3 allocs/op
```

However, the go 1.21.0 `slog.Logger` adds some overhead:
```
goos: linux
goarch: amd64
pkg: github.com/phsym/console-slog
cpu: Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz
BenchmarkLoggers/dummy-4                 1239873             893.2 ns/op             128 B/op          1 allocs/op
BenchmarkLoggers/console-4                483354              2338 ns/op             128 B/op          1 allocs/op
BenchmarkLoggers/std-text-4               368828              3141 ns/op             132 B/op          3 allocs/op
BenchmarkLoggers/std-json-4               393322              2909 ns/op             248 B/op          4 allocs/op
```

## Indentation
Indentation of functions can occasionally be useful for delineating
calls by indenting callees more than callers.
This can sometimes help interpret logs during development.

The specific case of runaway recursion shows up clearly as the messages
march repeatedly across the screen.

Indentation is configured using `console.HandlerOptins.Indent` which uses the following type:
```go
// Indentation configures indentation for message and attribute data.
// If both Prefix and Tab strings are empty ("") indentation will not occur. 
// When indenting, the message is prepended by the Prefix string (if not empty)
// followed by the depth level iterations of the Tab string.
// The depth level is provided by an attribute named by the Key string.
// This should be either an integer or a pointer to a DepthLevel object.
type Indentation struct {
// Prefix string used before indentation (optional).
Prefix string

// Tab represents the additional indentation per depth level.
// It is probably better to not use an actual tab character here,
// but instead use some number of spaces.
Tab    string

// Key is the attribute key that will hold the depth number.
// When not provided the key is set to "depth".
Key    string
}}
```
A convenience function can be used instead of
filling out the `console.HandlerOptins.Indent` object manually for simple cases:
```go
// DefaultIndentation returns an Indentation struct with the specified tag string,
// no Prefix string, and Key set to the default indentation key.
// Other configurations (e.g. featuring a Prefix or non-default Key)
// must be set manually.
func DefaultIndentation(tab string) Indentation
```

### Example

```go
package main

import (
	"log/slog"
	"os"

	"github.com/phsym/console-slog"
)

func main() {
	slog.SetDefault(slog.New(console.NewHandler(os.Stderr, &console.HandlerOptions{
		Level:  slog.LevelDebug,
		Indent: console.DefaultIndentation("  "),
	})))
	slog.Info("factorial", "result", factorial(7, 0))
}

func factorial(number, depth int64) int64 {
	slog.Debug("factorial", "number", number, "depth", depth)
	var result int64
	if number < 2 {
		result = 1
	} else {
		result = number * factorial(number-1, depth+1)
	}
	slog.Debug("factorial", "result", result, "depth", depth)
	return result
}
```

![factorial-indent](./doc/img/factorial-indent.png)

Where passing the depth number as an argument is contraindicated or just plain irritating
it is possible to use the following object for the depth value in logging statements:
```go
// DepthValuer represents an indentation depth object (an int64).
// These objects can be incremented on entry to a function and decremented in a defer statement.
// This object is not thread safe. Using it as a global variable (the mostly likely usage)
// in a multithreaded application will result in unpredictable values in different threads.
type DepthValuer struct {
	depth int
}

// LogValue implements the slog.LogValuer interface.
func (dv *DepthValuer) LogValue() slog.Value {
	return slog.IntValue(int(dv.depth))
}

// Increment the depth value.
// Use this at the head of a function or important code block.
func (dv *DepthValuer) Increment() {
	dv.depth += 1
}

// Decrement the depth value.
// Use this after an important code block or as a defer statement in a function.
func (dv *DepthValuer) Decrement() {
	dv.depth -= 1
}
```

The use case for `DepthValuer` is to invoke its `Increment()` function at the start of a function,
followed by a `defer` of its `Decrement()` object.
All logging calls in the function need to append the `Indentation.Key` and a pointer to the `DepthValuer`.

_It is not possible_ to add this attribute via a `WithAttrs()` call as the value is
only evaluated once and cached by the `console.Handler` for performance reasons.
The `Indentation.Key` and `DepthValuer` pointer must be specified in each logging call.

This mechanism should be usable in code blocks but `defer` doesn't work in this case.
The programmer is responsible for making certain that the `Decrement()` call is made in all situations.

## Example:
```go
package main

import (
	"log/slog"
	"os"

	"github.com/phsym/console-slog"
)

var depthValuer = &console.DepthValuer{}

func main() {
	slog.SetDefault(slog.New(console.NewHandler(os.Stderr, &console.HandlerOptions{
		Level:  slog.LevelDebug,
		Indent: console.DefaultIndentation("  "),
	})))
	slog.Info("factorial", "result", factorial(7))
}

func factorial(number int64) int64 {
	depthValuer.Increment()
	defer depthValuer.Decrement()
	slog.Debug("factorial", "number", number, "depth", depthValuer)
	var result int64
	if number < 2 {
		result = 1
	} else {
		result = number * factorial(number-1)
	}
	slog.Debug("factorial", "result", result, "depth", depthValuer)
	return result
}
```