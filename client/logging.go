package firebase_client

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
)

// LogOptions controls how much detail the client logs.
//
// Everything is logged as it is: tokens, keys, credentials and bodies reach the handler
// verbatim, so point the logger somewhere you are willing to have those end up.
type LogOptions struct {
	// Bodies includes HTTP request and response bodies in the debug logs.
	Bodies bool
	// MaxBodyLength caps how much of a body is logged, 0 means DefaultMaxLoggedBody.
	MaxBodyLength int
}

// DefaultMaxLoggedBody caps logged HTTP bodies so a stray large response cannot flood
// the logs.
const DefaultMaxLoggedBody = 8 << 10 // 8 KiB

func (o LogOptions) maxBodyLength() int {
	if o.MaxBodyLength > 0 {
		return o.MaxBodyLength
	}
	return DefaultMaxLoggedBody
}

// WithLogger attaches a *slog.Logger to the client. Nothing is logged without one.
//
// Any slog handler works, which is how zerolog (or zap, or logrus) gets plugged in
// without this library depending on it:
//
//	zl := zerolog.New(os.Stderr) // the bridge writes the record's own timestamp
//	handler := slogzerolog.Option{Level: slog.LevelDebug, Logger: &zl}.NewZerologHandler()
//	client, err := firebase.NewFirebaseClient(nil, nil, firebase.WithLogger(slog.New(handler)))
//
// Levels used by this library:
//   - Debug: every request/response, MTalk frames, retry decisions
//   - Info:  state changes worth knowing about (check-in, installation, token obtained)
//   - Warn:  a failure that is being retried
//   - Error: a failure that ended an operation or the MTalk read loop
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *FireBaseClient) {
		c.logger = logger
	}
}

// WithLogBodies makes the debug logs include HTTP request/response bodies.
// Bodies are logged as they are, credentials and all.
func WithLogBodies(enabled bool) ClientOption {
	return func(c *FireBaseClient) {
		c.logOptions.Bodies = enabled
	}
}

// WithLogOptions sets every logging detail at once.
func WithLogOptions(options LogOptions) ClientOption {
	return func(c *FireBaseClient) {
		c.logOptions = options
	}
}

// WithLogContext sets the context used for records that belong to no particular call: the
// startup record, and the MTalk read loop, which runs in the background long after the
// call that started it returned.
//
// It exists for handlers that take attributes off the context (a request id, an account, a
// trace) - those handlers see this context when there is no call context to use. Only the
// values are kept, cancelling it does not stop anything.
func WithLogContext(ctx context.Context) ClientOption {
	return func(c *FireBaseClient) {
		if ctx != nil {
			c.logCtx = context.WithoutCancel(ctx)
		}
	}
}

// logContext returns the fallback context for records without one, never nil.
func (c *FireBaseClient) logContext() context.Context {
	if c == nil || c.logCtx == nil {
		return context.Background()
	}
	return c.logCtx
}

// Logger returns the client's logger, never nil. Calls on the returned logger are cheap
// no-ops when no logger was configured.
func (c *FireBaseClient) Logger() *slog.Logger {
	if c == nil || c.logger == nil {
		return discardLogger
	}
	return c.logger
}

// discardLogger is what every logging call falls back to when logging is not configured.
var discardLogger = slog.New(slog.DiscardHandler)

// logOperation emits one debug (success) or error (failure) record for an API call.
func (c *FireBaseClient) logOperation(ctx context.Context, operation string, start time.Time, err error, attrs ...any) {
	if ctx == nil {
		// Callers do pass nil contexts, and a handler is free to read from the one it gets.
		ctx = c.logContext()
	}
	logger := c.Logger()
	level := slog.LevelDebug
	if err != nil {
		level = slog.LevelError
	}
	if !logger.Enabled(ctx, level) {
		return
	}

	args := make([]any, 0, len(attrs)+4)
	args = append(args, slog.String("operation", operation), slog.Duration("took", time.Since(start)))
	args = append(args, attrs...)
	if err != nil {
		args = append(args,
			slog.String("error", err.Error()),
			slog.Bool("retryable", firebase_api.IsRetryable(err)),
		)
	}
	logger.Log(ctx, level, "firebase call", args...)
}

// logValuer is anything that renders itself as a group of named log attributes. Every type
// this library logs implements it.
type logValuer interface{ LogValue() slog.Value }

// logAttr renders a value eagerly instead of handing the raw struct to the handler.
// slog handlers are supposed to call Value.Resolve() (which would reach LogValue anyway),
// but not every third-party handler does, and a handler that forgets would log the raw
// struct instead of the named attributes.
func logAttr(key string, value logValuer) slog.Attr {
	return slog.Attr{Key: key, Value: value.LogValue()}
}

// LoggingTransport logs every HTTP round trip through it at debug level. It is installed
// automatically when a logger is configured, and is exported so it can be reused on a
// custom *http.Client.
type LoggingTransport struct {
	// Base is the transport doing the actual work, nil means http.DefaultTransport.
	Base http.RoundTripper
	// Logger receives the records, nil disables logging entirely.
	Logger *slog.Logger
	// Options control body logging.
	Options LogOptions
}

// RoundTrip implements http.RoundTripper.
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	logger := t.Logger
	if logger == nil || !logger.Enabled(req.Context(), slog.LevelDebug) {
		return base.RoundTrip(req)
	}

	requestAttrs := []any{
		slog.String("method", req.Method),
		slog.String("url", urlString(req.URL)),
		slog.Any("headers", headerValue(req.Header)),
	}
	if t.Options.Bodies {
		body, err := t.captureRequestBody(req)
		if err != nil {
			return nil, err
		}
		if body != "" {
			requestAttrs = append(requestAttrs, slog.String("body", body))
		}
	}
	logger.DebugContext(req.Context(), "http request", requestAttrs...)

	start := time.Now()
	resp, err := base.RoundTrip(req)
	took := time.Since(start)
	if err != nil {
		logger.DebugContext(req.Context(), "http response",
			slog.String("method", req.Method),
			slog.String("url", urlString(req.URL)),
			slog.Duration("took", took),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	responseAttrs := []any{
		slog.String("method", req.Method),
		slog.String("url", urlString(req.URL)),
		slog.Int("status", resp.StatusCode),
		slog.Duration("took", took),
		slog.Any("headers", headerValue(resp.Header)),
	}
	if t.Options.Bodies {
		body, replayErr := t.captureResponseBody(resp)
		if replayErr != nil {
			responseAttrs = append(responseAttrs, slog.String("body_error", replayErr.Error()))
		} else if body != "" {
			responseAttrs = append(responseAttrs, slog.String("body", body))
		}
	}
	logger.DebugContext(req.Context(), "http response", responseAttrs...)
	return resp, nil
}

// captureRequestBody reads the request body for logging and puts it back so the request
// can still be sent.
func (t *LoggingTransport) captureRequestBody(req *http.Request) (string, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return "", nil
	}
	raw, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return "", err
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(raw)), nil
	}
	return t.formatBody(req.Header.Get("Content-Type"), raw), nil
}

// captureResponseBody reads the response body for logging and replaces it so the caller
// still receives it.
func (t *LoggingTransport) captureResponseBody(resp *http.Response) (string, error) {
	if resp.Body == nil {
		return "", nil
	}
	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		// The body is gone, hand the caller an empty one rather than a closed reader.
		resp.Body = io.NopCloser(bytes.NewReader(nil))
		return "", err
	}
	resp.Body = io.NopCloser(bytes.NewReader(raw))
	return t.formatBody(resp.Header.Get("Content-Type"), raw), nil
}

// formatBody renders a body for logging: text as it is, binary as a size, both truncated
// to maxBodyLength so a stray large response cannot flood the logs.
func (t *LoggingTransport) formatBody(_ string, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	text := string(body)
	if isProbablyBinary(body) {
		// Protobuf and friends are unreadable in a log and would just be noise.
		return "<" + itoa(len(body)) + " binary bytes>"
	}

	if max := t.Options.maxBodyLength(); len(text) > max {
		text = text[:max] + "... (truncated)"
	}
	return text
}

// headerValue renders headers as a log group.
func headerValue(header http.Header) slog.Value {
	attrs := make([]slog.Attr, 0, len(header))
	for key, values := range header {
		attrs = append(attrs, slog.String(key, strings.Join(values, ", ")))
	}
	return slog.GroupValue(attrs...)
}

// urlString renders a request URL, tolerating a nil one.
func urlString(target *url.URL) string {
	if target == nil {
		return ""
	}
	return target.String()
}

// isProbablyBinary reports whether a body is protobuf or similar, logging those verbatim
// would just be noise.
func isProbablyBinary(body []byte) bool {
	limit := min(len(body), 512)
	for _, b := range body[:limit] {
		if b == 0 || (b < 0x09 && b != 0x00) || (b > 0x0D && b < 0x20) {
			return true
		}
	}
	return false
}

// itoa is a tiny helper so this file does not need strconv for one call.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := [20]byte{}
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}
