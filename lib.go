// Package firebase is the entry point of go-android-firebase, a library that speaks the
// Firebase/GCM protocols the way an Android device does: it checks a (virtual) device in
// with Google, installs an app with Firebase, obtains an FCM notification token and can
// hold the MTalk connection that receives push notifications.
//
// The quickest path to a working push notification receiver:
//
//	client, err := firebase.NewFirebaseClient(nil, nil)   // random device, default HTTP client
//	token, err := client.RegisterForNotifications(ctx, appData, nil)
//	client.MTalk.OnNotification = func(n *firebase_api.DataMessageStanza) { ... }
//	err = client.MTalk.Connect()
//	defer client.Close()
package firebase

import (
	"net/http"

	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	firebase_client "github.com/BRUHItsABunny/go-android-firebase/client"
)

// Re-exported types so that callers only need to import this package for the common case.
type (
	// Client talks to Firebase on behalf of one device.
	Client = firebase_client.FireBaseClient
	// ClientOption customises a Client at construction time.
	ClientOption = firebase_client.ClientOption
	// Device holds the persistent state of a virtual Android device.
	Device = firebase_api.FirebaseDevice
	// AppData holds the Firebase constants of the app being impersonated.
	AppData = firebase_api.FirebaseAppData
	// RegisterOptions tunes Client.RegisterForNotifications.
	RegisterOptions = firebase_client.RegisterOptions
	// LogOptions controls how much detail the client logs.
	LogOptions = firebase_client.LogOptions
	// LoggingTransport logs HTTP round trips, reusable on a custom *http.Client.
	LoggingTransport = firebase_client.LoggingTransport
	// Notification is a push notification as it arrives over MTalk.
	Notification = firebase_api.DataMessageStanza
)

// Re-exported errors so callers can use errors.Is without importing the api package.
var (
	ErrNilAppData         = firebase_api.ErrNilAppData
	ErrNilDevice          = firebase_api.ErrNilDevice
	ErrNoInstallation     = firebase_api.ErrNoInstallation
	ErrNoInstallationAuth = firebase_api.ErrNoInstallationAuth
	ErrNoCheckin          = firebase_api.ErrNoCheckin
	ErrEmptyResponse      = firebase_api.ErrEmptyResponse
	ErrNotConnected       = firebase_api.ErrNotConnected
	ErrAlreadyConnected   = firebase_api.ErrAlreadyConnected
)

// NewFirebaseClient builds a client. A nil hClient gets a default client with a timeout,
// a nil device gets a freshly generated random one.
func NewFirebaseClient(hClient *http.Client, device *firebase_api.FirebaseDevice, opts ...firebase_client.ClientOption) (*firebase_client.FireBaseClient, error) {
	return firebase_client.NewFirebaseClient(hClient, device, opts...)
}

// Logging options, re-exported so callers only need this package.
//
// The library logs through log/slog, so any slog handler receives its internals: no
// dependency on a specific logging library, and nothing is logged until a logger is set.
var (
	// WithLogger attaches a *slog.Logger, see the client package for the levels used.
	WithLogger = firebase_client.WithLogger
	// WithLogBodies includes HTTP request/response bodies in the debug records.
	WithLogBodies = firebase_client.WithLogBodies
	// WithLogOptions sets every logging detail at once.
	WithLogOptions = firebase_client.WithLogOptions
	// WithLogContext sets the context for records with no call of their own (startup, the
	// MTalk read loop), for handlers that take attributes off the context.
	WithLogContext = firebase_client.WithLogContext
)

// RandomAppFID generates a Firebase Installation ID, returning an empty string in the
// (practically impossible) case that no randomness is available. Use RandomAppFIDErr when
// that case matters to you.
func RandomAppFID() string {
	result, _ := firebase_api.RandomAppFID()
	return result
}

// RandomAppFIDErr generates a Firebase Installation ID and reports why it could not.
func RandomAppFIDErr() (string, error) {
	return firebase_api.RandomAppFID()
}

// IsRetryable reports whether an error returned by this library is worth retrying, e.g. a
// PHONE_REGISTRATION_ERROR right after a check-in or a 5xx from Google.
func IsRetryable(err error) bool {
	return firebase_api.IsRetryable(err)
}
