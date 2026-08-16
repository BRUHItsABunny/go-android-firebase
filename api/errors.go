package firebase_api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Sentinel errors, these are meant to be used with errors.Is
var (
	// ErrNilAppData is returned when a call is made without the *FirebaseAppData it needs.
	ErrNilAppData = errors.New("firebase: appData is nil")
	// ErrNilDevice is returned when a call is made without the *FirebaseDevice it needs.
	ErrNilDevice = errors.New("firebase: device is nil")
	// ErrNoInstallation is returned when an app has not been installed (see FireBaseClient.NotifyInstallation) yet.
	ErrNoInstallation = errors.New("firebase: no installation available for package, call NotifyInstallation first")
	// ErrNoInstallationAuth is returned when an installation exists but holds no (valid) authentication.
	ErrNoInstallationAuth = errors.New("firebase: installation has no authentication, call NotifyInstallation first")
	// ErrNoCheckin is returned when a call requires check-in credentials that the device does not have yet.
	ErrNoCheckin = errors.New("firebase: no check-in credentials, call Checkin first")
	// ErrEmptyResponse is returned when the server responded successfully but without a usable body.
	ErrEmptyResponse = errors.New("firebase: empty response body")
	// ErrNotConnected is returned when an MTalk operation requires a connection that isn't established.
	ErrNotConnected = errors.New("firebase: MTalk is not connected")
	// ErrAlreadyConnected is returned when MTalk.Connect is called on a live connection.
	ErrAlreadyConnected = errors.New("firebase: MTalk is already connected")
)

// HTTPError is returned when an endpoint answers with an unexpected status code.
// The (truncated) body is kept around because Google's endpoints put the actual
// reason in there far more often than in the status code itself.
type HTTPError struct {
	Endpoint   string
	StatusCode int
	Status     string
	Body       string
}

const maxErrorBodyLength = 2048

// NewHTTPError builds an HTTPError from a response, the body is expected to be read already.
func NewHTTPError(resp *http.Response, body []byte) *HTTPError {
	endpoint := ""
	if resp.Request != nil && resp.Request.URL != nil {
		endpoint = resp.Request.URL.String()
	}
	bodyStr := strings.TrimSpace(string(body))
	if len(bodyStr) > maxErrorBodyLength {
		bodyStr = bodyStr[:maxErrorBodyLength] + "... (truncated)"
	}
	return &HTTPError{
		Endpoint:   endpoint,
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       bodyStr,
	}
}

func (e *HTTPError) Error() string {
	msg := fmt.Sprintf("firebase: unexpected HTTP status %d", e.StatusCode)
	if e.Endpoint != "" {
		msg += " for " + e.Endpoint
	}
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}

// IsRetryable reports whether retrying the same request later could plausibly succeed.
func (e *HTTPError) IsRetryable() bool {
	switch e.StatusCode {
	case http.StatusTooManyRequests, http.StatusRequestTimeout,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// Registration error codes as returned by the c2dm register3 endpoint.
const (
	RegisterErrorPhoneRegistration   = "PHONE_REGISTRATION_ERROR"
	RegisterErrorAuthentication      = "AUTHENTICATION_FAILED"
	RegisterErrorTooManyRegistrants  = "TOO_MANY_REGISTRATIONS"
	RegisterErrorInvalidSender       = "INVALID_SENDER"
	RegisterErrorInvalidParameters   = "INVALID_PARAMETERS"
	RegisterErrorMissingRegistration = "MISSING_REGISTRATION"
	RegisterErrorServiceNotAvailable = "SERVICE_NOT_AVAILABLE"
	RegisterErrorInternalServerError = "INTERNAL_SERVER_ERROR"
)

// RegisterError is returned when the c2dm register3 endpoint answers with "Error=SOMETHING".
// register3 answers with HTTP 200 even when it refuses to register, so the body is the only
// place the failure shows up.
type RegisterError struct {
	// Code is the raw value of the Error field, e.g. "PHONE_REGISTRATION_ERROR".
	Code string
	// Fields holds every key=value pair of the response, so callers can inspect extras
	// such as "Retry-After" without re-parsing the body.
	Fields map[string]string
}

func (e *RegisterError) Error() string {
	return fmt.Sprintf("firebase: registration refused: %s%s", e.Code, e.hint())
}

func (e *RegisterError) hint() string {
	switch e.Code {
	case RegisterErrorAuthentication:
		return " (check-in credentials rejected, run Checkin again)"
	case RegisterErrorPhoneRegistration:
		return " (check-in credentials are not known to GCM/FCM yet, retry in a few seconds)"
	case RegisterErrorTooManyRegistrants:
		return " (this device registered too many apps, use a fresh device)"
	case RegisterErrorInvalidSender:
		return " (NotificationSenderID/GMPAppID does not match the app)"
	}
	return ""
}

// IsRetryable reports whether retrying the registration later could plausibly succeed.
// PHONE_REGISTRATION_ERROR in particular is usually a propagation delay right after check-in.
func (e *RegisterError) IsRetryable() bool {
	switch e.Code {
	case RegisterErrorPhoneRegistration, RegisterErrorServiceNotAvailable,
		RegisterErrorInternalServerError, RegisterErrorAuthentication:
		return true
	}
	return false
}

// AuthError is returned when the android auth endpoint answers with "Error=SOMETHING".
type AuthError struct {
	Code   string
	Fields map[string]string
}

func (e *AuthError) Error() string {
	msg := fmt.Sprintf("firebase: auth failed: %s", e.Code)
	if info, ok := e.Fields["Info"]; ok && info != "" {
		msg += " (" + info + ")"
	}
	if url, ok := e.Fields["Url"]; ok && url != "" {
		msg += " see: " + url
	}
	return msg
}

// IsRetryable reports whether retrying the same auth request could plausibly succeed.
func (e *AuthError) IsRetryable() bool {
	return e.Code == "ServiceUnavailable" || e.Code == "Timeout"
}

// GoogleAPIError is the error envelope the identitytoolkit/securetoken endpoints return.
type GoogleAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
	Errors  []struct {
		Message string `json:"message"`
		Domain  string `json:"domain"`
		Reason  string `json:"reason"`
	} `json:"errors"`
}

func (e *GoogleAPIError) Error() string {
	msg := fmt.Sprintf("firebase: API error %d", e.Code)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if e.Status != "" {
		msg += " (" + e.Status + ")"
	}
	return msg
}

// IsRetryable reports whether retrying the same request later could plausibly succeed.
func (e *GoogleAPIError) IsRetryable() bool {
	return e.Code == http.StatusTooManyRequests || e.Code >= http.StatusInternalServerError
}

// googleAPIErrorEnvelope is the wire shape of a Google API error response.
type googleAPIErrorEnvelope struct {
	Error *GoogleAPIError `json:"error"`
}

// IsRetryable reports whether err (or anything it wraps) is worth retrying.
// It recognises every error type this package returns and defaults to false.
func IsRetryable(err error) bool {
	var retryable interface{ IsRetryable() bool }
	if errors.As(err, &retryable) {
		return retryable.IsRetryable()
	}
	return false
}
