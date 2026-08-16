package firebase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
)

// The tests in this package double as a demonstration of what the library logs: every
// client they build gets a logger that writes into the test output, so running
//
//	go test -run TestRegister3 -v
//
// prints the whole conversation with Google (check-in, installation, registration, MTalk
// frames) instead of only the few values the test dumps at the end.
//
// All of it is configured through the environment, so no test has to be edited to get
// more detail out of it:
//
//	FIREBASE_LOG=off|error|warn|info|debug   how much to log, default debug ("off" is silent)
//	FIREBASE_LOG_BODIES=1                    include HTTP request/response bodies
//
// Nothing is redacted: a test log holds the credentials the run used, so treat one like
// the account it came from.

// testWriter sends the log output through t.Log, which keeps it attached to the test that
// produced it. Writes that arrive after the test finished (a still running MTalk read
// loop, for instance) go to stderr instead, because t.Log would panic there.
type testWriter struct {
	mutex sync.Mutex
	t     *testing.T
	done  bool
}

func (w *testWriter) Write(p []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.done {
		return os.Stderr.Write(p)
	}
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

func (w *testWriter) close() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.done = true
}

// testLogLevel reads FIREBASE_LOG, defaulting to debug so a test run shows everything.
// The bool reports whether logging is wanted at all.
func testLogLevel() (slog.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FIREBASE_LOG"))) {
	case "off", "none", "silent":
		return 0, false
	case "error":
		return slog.LevelError, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "info":
		return slog.LevelInfo, true
	default:
		return slog.LevelDebug, true
	}
}

func testEnvEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// testLogger builds the logger the tests hand to the client. The library only ever talks
// to slog, so swapping this handler is all it takes to send its internals to zerolog (or
// zap, or logrus) instead - that choice belongs to the application, which is why neither
// this library nor its tests depend on a logging library. The zerolog wiring is in the
// README.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()

	level, enabled := testLogLevel()
	if !enabled {
		return slog.New(slog.DiscardHandler)
	}

	writer := &testWriter{t: t}
	t.Cleanup(writer.close)
	return slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level}))
}

// testLogOptions is what every test in this package passes to NewFirebaseClient.
func testLogOptions(t *testing.T) []ClientOption {
	t.Helper()
	return []ClientOption{
		WithLogger(testLogger(t)),
		WithLogBodies(testEnvEnabled("FIREBASE_LOG_BODIES")),
	}
}

// newTestClient builds a client that logs into the test output.
func newTestClient(t *testing.T, hClient *http.Client, device *Device) *Client {
	t.Helper()
	client, err := NewFirebaseClient(hClient, device, testLogOptions(t)...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// TestLoggingShowcase needs no credentials and no network: it points the logging
// transport at a local server that answers the way Google does, so `go test -run
// TestLoggingShowcase -v` shows what the logs look like, and asserts that a logged body
// still reaches the caller intact.
func TestLoggingShowcase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("token=fMEP0Xm3TFa9BvS1cRnZq8LkYd4Wg7Uj\n"))
	}))
	defer server.Close()

	const (
		apiKey        = "AIzaSyC7m9NhFXHiUPryquw7PecqFO0d9YPrVNE"
		masterToken   = "aas_et/AKppINZzGRRUHnPvBFqJ1TnJnNnJfTvXpvKzRyGtqQwEr"
		responseToken = "fMEP0Xm3TFa9BvS1cRnZq8LkYd4Wg7Uj"
	)

	// The library installs this transport itself when a logger is configured, it is
	// exported so it can also be dropped on a client the caller already owns.
	captured := &strings.Builder{}
	logger := slog.New(slog.NewTextHandler(io.MultiWriter(captured, &testWriter{t: t}), &slog.HandlerOptions{Level: slog.LevelDebug}))
	httpClient := &http.Client{Transport: &LoggingTransport{
		Logger:  logger,
		Options: LogOptions{Bodies: true},
	}}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/c2dm/register3?key="+apiKey,
		strings.NewReader("app=org.wikipedia&device=4954293857293847&Token="+masterToken))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "AidLogin 4954293857293847:8571938475918273645")

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// Logging a body does not consume it, the caller still gets everything.
	if !strings.Contains(string(body), responseToken) {
		t.Fatalf("the caller should still receive the full body, got %q", body)
	}

	// Requests, responses, headers and bodies are logged as they are.
	logged := captured.String()
	for _, wanted := range []string{
		"app=org.wikipedia",
		"status=200",
		apiKey,
		masterToken,
		responseToken,
		"AidLogin 4954293857293847:8571938475918273645",
	} {
		if !strings.Contains(logged, wanted) {
			t.Fatalf("expected %q in the logs:\n%s", wanted, logged)
		}
	}
}

// TestStructuredLoggingShowcase proves the records survive as structured data: one object
// per record, every attribute its own field. Run it with -v to see the JSON.
//
// The handler here is the standard library's, so this test carries no dependency. Swap it
// for a bridge and the same records reach zerolog unchanged, which is all the wiring an
// application needs (see the README):
//
//	zl := zerolog.New(os.Stderr) // the bridge writes the record's own timestamp
//	handler := slogzerolog.Option{Level: slog.LevelDebug, Logger: &zl}.NewZerologHandler()
//	client, err := firebase.NewFirebaseClient(nil, nil, firebase.WithLogger(slog.New(handler)))
func TestStructuredLoggingShowcase(t *testing.T) {
	const (
		apiKey        = "AIzaSyC7m9NhFXHiUPryquw7PecqFO0d9YPrVNE"
		responseToken = "fMEP0Xm3TFa9BvS1cRnZq8LkYd4Wg7Uj"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"projects/1/installations/abc","fid":"abc","authToken":{"token":"` + responseToken + `","expiresIn":"604800s"}}`))
	}))
	defer server.Close()

	captured := &strings.Builder{}
	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(captured, &testWriter{t: t}),
		&slog.HandlerOptions{Level: slog.LevelDebug}))

	httpClient := &http.Client{Transport: &LoggingTransport{
		Logger:  logger,
		Options: LogOptions{Bodies: true},
	}}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/projects/demo/installations?key="+apiKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err = io.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}

	// Apps and devices reach a handler as a group of named attributes: they implement
	// slog.LogValuer, and this library resolves them eagerly, so a handler that skips
	// resolving still gets the fields rather than a struct dump.
	logger.Info("firebase state",
		slog.Any("app", &firebase_api.FirebaseAppData{PackageID: "org.wikipedia", GoogleAPIKey: apiKey}),
	)

	logged := captured.String()
	for _, line := range strings.Split(strings.TrimSpace(logged), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("expected one JSON object per record, got %q: %v", line, err)
		}
		for _, field := range []string{"level", "time", "msg"} {
			if _, ok := record[field]; !ok {
				t.Fatalf("expected a %q field in the record: %s", field, line)
			}
		}
	}
	// Attributes arrive as fields, values as they are.
	for _, wanted := range []string{`"status":200`, `"package_id":"org.wikipedia"`, apiKey, responseToken} {
		if !strings.Contains(logged, wanted) {
			t.Fatalf("expected %q in the structured output:\n%s", wanted, logged)
		}
	}
}

// accountKey is the context key the showcase below hangs its per-account field on.
type accountKey struct{}

// contextAttrHandler adds attributes taken off the context of each record. It is what
// slog-zerolog's AttrFromContext option does, written out here in a dozen lines so the
// test needs no bridge to prove the library feeds one properly.
type contextAttrHandler struct {
	slog.Handler
	attrs func(ctx context.Context) []slog.Attr
}

func (h contextAttrHandler) Handle(ctx context.Context, record slog.Record) error {
	record.AddAttrs(h.attrs(ctx)...)
	return h.Handler.Handle(ctx, record)
}

// WithAttrs and WithGroup have to re-wrap: the promoted methods of the embedded handler
// would return the inner handler and quietly drop the context attributes from every
// logger built with slog.Logger.With.
func (h contextAttrHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextAttrHandler{Handler: h.Handler.WithAttrs(attrs), attrs: h.attrs}
}

func (h contextAttrHandler) WithGroup(name string) slog.Handler {
	return contextAttrHandler{Handler: h.Handler.WithGroup(name), attrs: h.attrs}
}

// TestCallerFieldsShowcase shows the three ways a caller adds their own fields to what
// this library logs, and that all three reach the records - including the ones the MTalk
// read loop produces in the background, which have no call context of their own.
//
// With zerolog the first two are `zerolog.New(w).With().Str(...)` and the bridge's
// AttrFromContext, see the README. The library cannot tell the difference: it only ever
// hands records and contexts to a slog.Handler.
func TestCallerFieldsShowcase(t *testing.T) {
	captured := &strings.Builder{}

	// 1. A field on the handler, on every record. (zerolog: a field on the zerolog logger)
	base := slog.NewJSONHandler(io.MultiWriter(captured, &testWriter{t: t}),
		&slog.HandlerOptions{Level: slog.LevelDebug}).
		WithAttrs([]slog.Attr{slog.String("component", "fcm-worker")})

	// 2. A field taken off the context of each record, for per-account or per-request
	//    labels that no argument has to be threaded through the library for.
	handler := contextAttrHandler{
		Handler: base,
		attrs: func(ctx context.Context) []slog.Attr {
			if account, ok := ctx.Value(accountKey{}).(string); ok {
				return []slog.Attr{slog.String("account", account)}
			}
			return nil
		},
	}

	// 3. A field on the slog logger itself.
	logger := slog.New(handler).With(slog.String("worker_id", "w-7"))

	ctx := context.WithValue(context.Background(), accountKey{}, "acct-42")
	client, err := NewFirebaseClient(nil, nil,
		WithLogger(logger),
		// Without this the startup and MTalk records have no context to read "account"
		// from, every call below passes its own.
		WithLogContext(ctx),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	// A failing call (nil app data, so this stays offline) and a background MTalk record.
	_, _ = client.NotifyInstallation(ctx, nil)
	client.MTalk.OnError(errors.New("connection lost"), true)

	logged := captured.String()
	lines := strings.Split(strings.TrimSpace(logged), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected a startup, a call and an mtalk record, got:\n%s", logged)
	}
	for _, line := range lines {
		for _, field := range []string{`"component":"fcm-worker"`, `"worker_id":"w-7"`, `"account":"acct-42"`} {
			if !strings.Contains(line, field) {
				t.Fatalf("expected %s on every record, missing from:\n%s", field, line)
			}
		}
	}
	if !strings.Contains(logged, `"msg":"mtalk error"`) {
		t.Fatalf("expected the background mtalk record:\n%s", logged)
	}
}

// TestLoggingShowcaseLevels documents which level carries what, so a caller can pick a
// level instead of turning everything on.
func TestLoggingShowcaseLevels(t *testing.T) {
	logger := testLogger(t)

	device := &firebase_api.FirebaseDevice{CheckinAndroidID: 4954293857293847}
	device.ApplyDefaults()
	appData := &firebase_api.FirebaseAppData{
		PackageID:            "org.wikipedia",
		GMPAppID:             "1:296120793014:android:34d2ba8d355ca9259a7317",
		NotificationSenderID: "296120793014",
		AppVersion:           "2.7.50394",
		GoogleAPIKey:         "AIzaSyC7m9NhFXHiUPryquw7PecqFO0d9YPrVNE",
	}

	// Both of these implement slog.LogValuer, so handing a device or an app to any slog
	// handler is safe: the API key comes out as a fingerprint.
	logger.Debug("debug: every request, response, MTalk frame and retry decision")
	logger.Info("info: state changes", slog.Any("device", device), slog.Any("app", appData))
	logger.Warn("warn: a failure that is being retried", slog.String("error", "PHONE_REGISTRATION_ERROR"))
	logger.Error("error: a failure that ended the operation", slog.Bool("retryable", IsRetryable(firebase_api.ErrNoCheckin)))
}
