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
