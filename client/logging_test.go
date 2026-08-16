package firebase_client

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
)

// captureHandler records every log record so tests can assert on them.
type captureHandler struct {
	mutex   sync.Mutex
	records []slog.Record
	level   slog.Level
}

func (h *captureHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= h.level }

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.records = append(h.records, record.Clone())
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *captureHandler) dump() string {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var builder strings.Builder
	for _, record := range h.records {
		builder.WriteString(record.Level.String())
		builder.WriteString(" ")
		builder.WriteString(record.Message)
		record.Attrs(func(attr slog.Attr) bool {
			// Deliberately not calling attr.Value.Resolve(): a handler that forgets to
			// resolve must still not see secrets, which is why the library renders its
			// log values eagerly.
			builder.WriteString(" ")
			builder.WriteString(attr.String())
			return true
		})
		builder.WriteString("\n")
	}
	return builder.String()
}

func (h *captureHandler) messages() []string {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	messages := make([]string, 0, len(h.records))
	for _, record := range h.records {
		messages = append(messages, record.Message)
	}
	return messages
}

func newCaptureLogger(level slog.Level) (*slog.Logger, *captureHandler) {
	handler := &captureHandler{level: level}
	return slog.New(handler), handler
}

func containsMessage(messages []string, want string) bool {
	for _, message := range messages {
		if message == want {
			return true
		}
	}
	return false
}

func TestNoLoggerIsSilentAndTransportUntouched(t *testing.T) {
	client, err := NewFirebaseClient(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.Logger() != discardLogger {
		t.Fatal("without WithLogger the client should use the discard logger")
	}
	if client.Client.Transport != nil {
		t.Fatal("without a logger the transport should be left alone")
	}
	// Must not panic.
	client.Logger().Debug("ignored")
}

func TestLoggerIsWiredIntoTransportAndMTalk(t *testing.T) {
	logger, handler := newCaptureLogger(slog.LevelDebug)
	client, err := NewFirebaseClient(nil, nil, WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := client.Client.Transport.(*LoggingTransport); !ok {
		t.Fatalf("expected a *LoggingTransport, got %T", client.Client.Transport)
	}
	if client.MTalk.Logger() != logger {
		t.Fatal("the MTalk connection should share the client's logger")
	}
	if !containsMessage(handler.messages(), "firebase client ready") {
		t.Fatalf("expected a startup record, got: %s", handler.dump())
	}
}

func TestLoggerDoesNotMutateCallersHTTPClient(t *testing.T) {
	logger, _ := newCaptureLogger(slog.LevelDebug)
	provided := &http.Client{}
	client, err := NewFirebaseClient(provided, nil, WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}
	if provided.Transport != nil {
		t.Fatal("the caller's client must not be modified")
	}
	if client.Client == provided {
		t.Fatal("the client should work on a copy once logging is enabled")
	}
}

func TestFailedCallIsLoggedWithRetryability(t *testing.T) {
	logger, handler := newCaptureLogger(slog.LevelDebug)
	client, err := NewFirebaseClient(nil, nil, WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	if _, err = client.NotifyInstallation(t.Context(), nil); err == nil {
		t.Fatal("expected an error")
	}

	dump := handler.dump()
	if !strings.Contains(dump, "ERROR firebase call") {
		t.Fatalf("expected an error record for the failed call, got: %s", dump)
	}
	if !strings.Contains(dump, "operation=NotifyInstallation") {
		t.Fatalf("the record should name the operation, got: %s", dump)
	}
	if !strings.Contains(dump, "retryable=false") {
		t.Fatalf("the record should say whether a retry helps, got: %s", dump)
	}
}

func TestLoggingTransportLogsAndPreservesBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := readAll(r)
		if body != `{"password":"hunter2","fid":"abc"}` {
			t.Errorf("the server should receive the original body, got %q", body)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("token=TOKENVALUE\n"))
	}))
	defer server.Close()

	logger, handler := newCaptureLogger(slog.LevelDebug)
	transport := &LoggingTransport{Logger: logger, Options: LogOptions{Bodies: true}}
	httpClient := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodPost, server.URL+"?key=APIKEYVALUE", strings.NewReader(`{"password":"hunter2","fid":"abc"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "AidLogin 1234567890:9876543210")

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	responseBody, err := readAllResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	// Logging a body must not consume it.
	if responseBody != "token=TOKENVALUE\n" {
		t.Fatalf("the caller should still receive the full body, got %q", responseBody)
	}

	dump := handler.dump()
	// Everything is logged as it is, bodies, headers and query parameters alike.
	for _, wanted := range []string{
		`{"password":"hunter2","fid":"abc"}`,
		"token=TOKENVALUE",
		"key=APIKEYVALUE",
		"AidLogin 1234567890:9876543210",
		"status=200",
	} {
		if !strings.Contains(dump, wanted) {
			t.Fatalf("expected %q in the logs: %s", wanted, dump)
		}
	}
}

func TestLoggingTransportSkipsWorkWhenDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	logger, handler := newCaptureLogger(slog.LevelInfo) // debug records are dropped
	httpClient := &http.Client{Transport: &LoggingTransport{Logger: logger, Options: LogOptions{Bodies: true}}}

	resp, err := httpClient.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	body, err := readAllResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	if body != "ok" {
		t.Fatalf("unexpected body: %q", body)
	}
	if len(handler.messages()) != 0 {
		t.Fatalf("nothing should be logged below the handler's level: %s", handler.dump())
	}
}

func TestBodyFormatting(t *testing.T) {
	transport := &LoggingTransport{}

	form := transport.formatBody("application/x-www-form-urlencoded", []byte("app=com.example&Token=TOKENVALUE"))
	if form != "app=com.example&Token=TOKENVALUE" {
		t.Fatalf("a body should be logged as it is: %s", form)
	}

	empty := transport.formatBody("text/plain", nil)
	if empty != "" {
		t.Fatalf("an empty body has nothing to log: %q", empty)
	}

	binary := transport.formatBody("application/x-protobuffer", []byte{0x08, 0x00, 0x12, 0x03, 0x01})
	if !strings.Contains(binary, "binary bytes") {
		t.Fatalf("binary bodies should not be dumped verbatim: %s", binary)
	}

	long := transport.formatBody("text/plain", []byte(strings.Repeat("a", DefaultMaxLoggedBody+100)))
	if !strings.HasSuffix(long, "(truncated)") {
		t.Fatal("long bodies should be truncated")
	}
}

func TestDeviceAndAppDataLogValues(t *testing.T) {
	appData := &firebase_api.FirebaseAppData{
		PackageID:    "com.example.app",
		GoogleAPIKey: "APIKEYVALUE",
	}
	device := &firebase_api.FirebaseDevice{CheckinAndroidID: 42}

	t.Run("structs are logged as named attributes", func(t *testing.T) {
		logger, handler := newCaptureLogger(slog.LevelDebug)
		logger.Info("state", logAttr("app", appData), logAttr("device", device))

		dump := handler.dump()
		// Eagerly resolved, so a handler that never calls Value.Resolve still gets the
		// attributes rather than a struct dump.
		for _, wanted := range []string{"package_id=com.example.app", "google_api_key=APIKEYVALUE", "checkin_android_id=42"} {
			if !strings.Contains(dump, wanted) {
				t.Fatalf("expected %q in the record: %s", wanted, dump)
			}
		}
	})

	t.Run("resolving handlers get the same thing", func(t *testing.T) {
		// This is what a well behaved handler (including slog's own) does.
		value := slog.AnyValue(appData).Resolve()
		if !strings.Contains(value.String(), "package_id=com.example.app") {
			t.Fatalf("unexpected resolved value: %s", value)
		}
	})

	t.Run("nil values are safe", func(t *testing.T) {
		var (
			nilApp    *firebase_api.FirebaseAppData
			nilDevice *firebase_api.FirebaseDevice
		)
		logger, _ := newCaptureLogger(slog.LevelDebug)
		logger.Info("state", logAttr("app", nilApp), logAttr("device", nilDevice))
	})
}

// contextKey is what the handler below reads its extra attribute from, the way a bridge
// configured with AttrFromContext (slog-zerolog) or a middleware would.
type contextKey struct{}

// contextHandler adds an attribute taken off the record's context, so a test can tell
// which context a record was emitted with.
type contextHandler struct {
	*captureHandler
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if account, ok := ctx.Value(contextKey{}).(string); ok {
		record.AddAttrs(slog.String("account", account))
	}
	return h.captureHandler.Handle(ctx, record)
}

func TestLogContextReachesRecordsWithoutACall(t *testing.T) {
	capture := &captureHandler{level: slog.LevelDebug}
	logger := slog.New(contextHandler{captureHandler: capture})

	ctx := context.WithValue(context.Background(), contextKey{}, "acct-42")
	client, err := NewFirebaseClient(nil, nil, WithLogger(logger), WithLogContext(ctx))
	if err != nil {
		t.Fatal(err)
	}

	// A record with no call behind it: emitted during construction.
	if !strings.Contains(capture.dump(), "account=acct-42") {
		t.Fatalf("the startup record should carry the context attributes: %s", capture.dump())
	}

	// A record from the MTalk read loop, which has no call context at all. The client
	// hands its log context to the connection, so these are decorated too.
	client.MTalk.OnError(errors.New("connection lost"), true)

	dump := capture.dump()
	if !strings.Contains(dump, "mtalk error") {
		t.Fatalf("expected an mtalk error record: %s", dump)
	}
	if strings.Count(dump, "account=acct-42") != 2 {
		t.Fatalf("both records should carry the context attributes: %s", dump)
	}
}

func TestConnectContextValuesOutliveTheCall(t *testing.T) {
	capture := &captureHandler{level: slog.LevelDebug}
	connection, err := NewMTalkCon(&firebase_api.FirebaseDevice{CheckinAndroidID: 1})
	if err != nil {
		t.Fatal(err)
	}
	connection.SetLogger(slog.New(contextHandler{captureHandler: capture}))

	// ConnectContext hands the read loop the values of its context, without its
	// cancellation: the loop keeps logging after the dial deadline has passed.
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "acct-42"))
	connection.SetLogContext(ctx)
	cancel()

	if connection.logContext().Err() != nil {
		t.Fatal("the stored log context should not be cancellable")
	}
	connection.OnError(errors.New("connection lost"), false)
	if !strings.Contains(capture.dump(), "account=acct-42") {
		t.Fatalf("the read loop's records should carry the context attributes: %s", capture.dump())
	}
}

func readAll(r *http.Request) (string, error) {
	defer func() { _ = r.Body.Close() }()
	body, err := io.ReadAll(r.Body)
	return string(body), err
}

func readAllResponse(resp *http.Response) (string, error) {
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	return string(body), err
}
