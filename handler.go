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

	// Indent defines a way for the message and attributes to be indented.
	Indent Indentation
}

const defaultIndentKey = "depth"

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
	Tab string

	// Key is the attribute key that will hold the depth number.
	// When not provided the key is set to "depth".
	Key string
}

func (indent *Indentation) indentString(depth int64) string {
	if indent.isZero() || depth <= 0 {
		return ""
	}
	// Build indentation string.
	builder := strings.Builder{}
	if indent.Prefix != "" {
		builder.WriteString(indent.Prefix)
	}
	if indent.Tab != "" {
		// TODO: Is there a more efficient way to do this?
		var i int64
		for i = 0; i < depth; i++ {
			builder.WriteString(indent.Tab)
		}
	}
	return builder.String()
}

// DefaultIndentation returns an Indentation struct with the specified tag string,
// no Prefix string, and Key set to the default indentation key.
// Other configurations (e.g. featuring a Prefix or non-default Key)
// must be set manually.
func DefaultIndentation(tab string) Indentation {
	return Indentation{
		Tab: "  ",
		Key: defaultIndentKey,
	}
}

func (indent *Indentation) isZero() bool {
	return indent.Prefix == "" && indent.Tab == ""
}

type Handler struct {
	opts    HandlerOptions
	out     io.Writer
	group   string
	context buffer
	enc     *encoder
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
	if opts.Indent.Key == "" {
		opts.Indent.Key = defaultIndentKey
	}
	return &Handler{
		opts:    *opts, // Copy struct
		out:     out,
		group:   "",
		context: nil,
		enc:     &encoder{opts: *opts},
	}
}

// / Enabled implements slog.Handler.
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
	if h.opts.Indent.isZero() {
		h.enc.writeMessage(buf, rec.Level, rec.Message)
		buf.copy(&h.context)
		rec.Attrs(func(a slog.Attr) bool {
			h.enc.writeAttr(buf, a, h.group)
			return true
		})
	} else {
		// NewHandler() should always set h.opts.IndentKey to a non-empty value.
		key := h.opts.Indent.Key
		// Indent the message and attributes.
		// Can't just ask for the depth key, must iterate through attributes.
		var attributes []slog.Attr
		var depth int64
		rec.Attrs(func(a slog.Attr) bool {
			if a.Key == key {
				value := a.Value
				if value.Kind() == slog.KindLogValuer {
					value = a.Value.LogValuer().LogValue()
				}
				if value.Kind() == slog.KindInt64 {
					depth = value.Int64()
				}
			} else {
				attributes = append(attributes, a)
			}
			return true
		})
		h.enc.writeMessage(buf, rec.Level, h.opts.Indent.indentString(depth)+rec.Message)
		buf.copy(&h.context)
		for _, a := range attributes {
			h.enc.writeAttr(buf, a, h.group)
		}
	}
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

// DepthValuer represents an indentation depth object (an int64).
// These objects can be incremented on entry to a function and decremented in a defer statement.
// Use a pointer to a DepthValuer object in logging statements, do not pass the object itself.
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
