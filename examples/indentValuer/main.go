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
