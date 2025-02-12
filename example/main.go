package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/phsym/console-slog"
)

var hdlr *console.Handler

func main() {
	hdlr = console.NewHandler(os.Stderr, &console.HandlerOptions{Level: slog.LevelDebug, AddSource: true})
	logger := slog.New(hdlr)
	slog.SetDefault(logger)
	slog.Info("Hello world!", "foo", "bar")
	slog.Debug("Debug message")
	slog.Warn("Warning message")
	slog.Error("Error message", "err", errors.New("the error"))

	logger = logger.With("foo", "bar").
		WithGroup("the-group").
		With("bar", "baz")

	logger.Info("group info", "attr", "value")

	hdlr.SetIndentation("", "  ")
	slog.Info("factorial", "result", factorial(4))
}

func factorial(number uint) uint {
	hdlr.Increment()
	defer hdlr.Decrement()

	slog.Info("factorial", "number", number)

	if number < 2 {
		return 1
	}
	return number * factorial(number-1)
}
