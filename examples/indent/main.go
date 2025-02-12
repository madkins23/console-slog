package main

import (
	"log/slog"
	"os"

	"github.com/phsym/console-slog"
)

var hdlr *console.Handler

func main() {
	hdlr = console.NewHandler(os.Stderr, &console.HandlerOptions{Level: slog.LevelDebug})
	hdlr.SetIndentation("", "  ")
	slog.SetDefault(slog.New(hdlr))
	slog.Info("factorial", "result", factorial(7))
}

func factorial(number uint) uint {
	hdlr.Increment()
	defer hdlr.Decrement()

	slog.Debug("factorial", "number", number)
	var result uint
	if number < 2 {
		result = 1
	} else {
		result = number * factorial(number-1)
	}
	slog.Debug("factorial", "result", result)
	return result
}
