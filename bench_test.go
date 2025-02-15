package console

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type DummyHandler struct{}

func (*DummyHandler) Enabled(context.Context, slog.Level) bool   { return true }
func (*DummyHandler) Handle(context.Context, slog.Record) error  { return nil }
func (h *DummyHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *DummyHandler) WithGroup(name string) slog.Handler       { return h }

var handlers = []struct {
	name string
	hdl  slog.Handler
}{
	{"dummy", &DummyHandler{}},
	{"console", NewHandler(io.Discard, &HandlerOptions{Level: slog.LevelDebug, AddSource: false})},
	{"console-indent", NewHandler(io.Discard, &HandlerOptions{Level: slog.LevelDebug, Indent: DefaultIndentation("  ")})},
	{"std-text", slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: false})},
	{"std-json", slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: false})},
	//{"console-ind-valuer", NewHandler(io.Discard, &HandlerOptions{Level: slog.LevelDebug, Indent: DefaultIndentation("  ")})},
}

var attrs = []slog.Attr{
	slog.String("foo", "bar"),
	slog.Int("int", 12),
	slog.Duration("dur", 3*time.Second),
	slog.Bool("bool", true),
	slog.Float64("float", 23.7),
	slog.Time("thetime", time.Now()),
	slog.Any("err", errors.New("yo")),
	slog.Group("empty"),
	slog.Group("group", slog.String("bar", "baz")),
}

var attrsAny = func() (a []any) {
	for _, attr := range attrs {
		a = append(a, attr)
	}
	return
}()

func BenchmarkHandlers(b *testing.B) {
	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	rec.AddAttrs(attrs...)

	for _, tc := range handlers {
		b.Run(tc.name, func(b *testing.B) {
			l := tc.hdl.WithAttrs(attrs).WithGroup("test").WithAttrs(attrs)
			// Warm-up
			_ = l.Handle(ctx, rec)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = l.Handle(ctx, rec)
			}
		})
	}
	b.ReportAllocs()
}

func BenchmarkHandlersIndent(b *testing.B) {
	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	rec.AddAttrs(attrs...)
	rec.AddAttrs(slog.Attr{
		Key:   "depth",
		Value: slog.IntValue(5),
	})

	for _, tc := range handlers {
		b.Run(tc.name, func(b *testing.B) {
			l := tc.hdl.WithAttrs(attrs).WithGroup("test").WithAttrs(attrs)
			// Warm-up
			_ = l.Handle(ctx, rec)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = l.Handle(ctx, rec)
			}
		})
	}
	b.ReportAllocs()
}

func BenchmarkLoggersIndent(b *testing.B) {
	for _, tc := range handlers {
		ctx := context.Background()
		b.Run(tc.name, func(b *testing.B) {
			attrsIndent := attrs
			attrsIndent = append(attrsIndent, slog.Int64("depth", 5))
			l := slog.New(tc.hdl).With(attrsAny...).WithGroup("test").With(attrsAny...)
			// Warm-up
			l.LogAttrs(ctx, slog.LevelInfo, "hello", attrs...)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				l.LogAttrs(ctx, slog.LevelInfo, "hello", attrs...)
			}
		})
	}
	b.ReportAllocs()
}
