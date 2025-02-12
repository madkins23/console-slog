package console

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var bufferPool = &sync.Pool{
	New: func() any { return new(buffer) },
}

var cwd, _ = os.Getwd()

// HandlerOptions are options for a ConsoleHandler.
// A zero HandlerOptions consists entirely of default values.
type HandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level slog.Leveler

	// Disable colorized output
	NoColor bool

	// TimeFormat is the format used for time.DateTime
	TimeFormat string

	// Theme defines the colorized output using ANSI escape sequences
	Theme Theme
}

type Indenter interface {
	SetIndentation(prefix, indent string)
	Increment()
	Decrement()
}

// indentation defines support data for enhanced message and arg indentation by depth.
type indentation struct {
	// Prefix for indentation string.
	prefix string

	// Indent string for each depth level.
	indent string

	// Depth is the current indentation depth
	depth uint
}

func (hd *indentation) indentMsg(msg string) string {
	if hd.prefix == "" && (hd.indent == "" || hd.depth < 1) {
		// No indentation.
		return msg
	}

	// Build indentation string.
	builder := strings.Builder{}
	if hd.prefix != "" {
		builder.WriteString(hd.prefix)
	}
	if hd.indent != "" && hd.depth > 0 {
		var i uint
		for i = 0; i < hd.depth; i++ {
			builder.WriteString(hd.indent)
		}
	}
	builder.WriteString(msg)
	return builder.String()
}

type Handler struct {
	opts    HandlerOptions
	out     io.Writer
	group   string
	context buffer
	enc     *encoder
	indentation
}

var _ slog.Handler = (*Handler)(nil)

// NewHandler creates a Handler that writes to w,
// using the given options.
// If opts is nil, the default options are used.
func NewHandler(out io.Writer, opts *HandlerOptions) *Handler {
	if opts == nil {
		opts = new(HandlerOptions)
	}
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.DateTime
	}
	if opts.Theme == nil {
		opts.Theme = NewDefaultTheme()
	}
	return &Handler{
		opts:    *opts, // Copy struct
		out:     out,
		group:   "",
		context: nil,
		enc:     &encoder{opts: *opts},
	}
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.opts.Level.Level()
}

// Handle implements slog.Handler.
func (h *Handler) Handle(_ context.Context, rec slog.Record) error {
	buf := bufferPool.Get().(*buffer)

	h.enc.writeTimestamp(buf, rec.Time)
	h.enc.writeLevel(buf, rec.Level)
	if h.opts.AddSource && rec.PC > 0 {
		h.enc.writeSource(buf, rec.PC, cwd)
	}
	// TODO: Add indentation here.
	h.enc.writeMessage(buf, rec.Level, h.indentMsg(rec.Message))
	buf.copy(&h.context)
	rec.Attrs(func(a slog.Attr) bool {
		h.enc.writeAttr(buf, a, h.group)
		return true
	})
	h.enc.NewLine(buf)
	if _, err := buf.WriteTo(h.out); err != nil {
		buf.Reset()
		bufferPool.Put(buf)
		return err
	}
	bufferPool.Put(buf)
	return nil
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newCtx := h.context
	for _, a := range attrs {
		h.enc.writeAttr(&newCtx, a, h.group)
	}
	newCtx.Clip()
	return &Handler{
		opts:    h.opts,
		out:     h.out,
		group:   h.group,
		context: newCtx,
		enc:     h.enc,
	}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	name = strings.TrimSpace(name)
	if h.group != "" {
		name = h.group + "." + name
	}
	return &Handler{
		opts:    h.opts,
		out:     h.out,
		group:   name,
		context: h.context,
		enc:     h.enc,
	}
}

var _ Indenter = (*Handler)(nil)

func (h *Handler) SetIndentation(prefix, indent string) {
	h.prefix = prefix
	h.indent = indent
	h.depth = 0
}

func (h *Handler) Increment() {
	h.depth++
}

func (h *Handler) Decrement() {
	if h.depth > 0 {
		h.depth--
	}
}
