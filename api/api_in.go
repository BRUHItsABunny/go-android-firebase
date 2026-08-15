package firebase_api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// readBody reads and closes the response body. Unlike gokhttp_responses.ResponseBytes it
// never loses the body on a non-2xx response, the body is exactly where Google puts the
// reason a request was refused.
func readBody(resp *http.Response) ([]byte, error) {
	if resp == nil {
		return nil, ErrEmptyResponse
	}
	if resp.Body == nil {
		return nil, nil
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body) // drain so the connection can be reused
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}
	return body, nil
}

// readBodyExpecting reads the body and turns any unexpected status code into an *HTTPError
// that carries the body along with it.
func readBodyExpecting(resp *http.Response, expectedCodes ...int) ([]byte, error) {
	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}
	if len(expectedCodes) == 0 {
		expectedCodes = []int{http.StatusOK}
	}
	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			return body, nil
		}
	}
	// Prefer the structured Google error over the raw body when there is one.
	if apiErr := parseGoogleAPIError(body); apiErr != nil {
		return body, apiErr
	}
	return body, NewHTTPError(resp, body)
}

// parseGoogleAPIError returns the *GoogleAPIError inside a Google error envelope, or nil.
func parseGoogleAPIError(body []byte) *GoogleAPIError {
	if len(body) == 0 {
		return nil
	}
	envelope := new(googleAPIErrorEnvelope)
	if err := json.Unmarshal(body, envelope); err != nil {
		return nil
	}
	if envelope.Error == nil || (envelope.Error.Code == 0 && envelope.Error.Message == "") {
		return nil
	}
	return envelope.Error
}

// decodeJSON unmarshals an expected-OK response, surfacing Google's error envelope when present.
func decodeJSON(resp *http.Response, result any) error {
	body, err := readBodyExpecting(resp, http.StatusOK)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return ErrEmptyResponse
	}
	if err = json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("json.Unmarshal: %w (body: %s)", err, truncate(string(body)))
	}
	return nil
}

func truncate(s string) string {
	if len(s) > maxErrorBodyLength {
		return s[:maxErrorBodyLength] + "... (truncated)"
	}
	return s
}

// ParseKeyValueBody parses the "key=value" line format that the auth, check-in and
// register3 endpoints answer with. Lines without a "=" are ignored, values may contain
// "=" themselves.
func ParseKeyValueBody(body []byte) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		result[strings.TrimSpace(key)] = value
	}
	return result
}

// ParseExpiresIn parses the duration strings Firebase returns, e.g. "604800s".
// A bare number is accepted too, since not every endpoint appends the unit.
func ParseExpiresIn(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("firebase: empty expiration")
	}
	if seconds, err := strconv.ParseFloat(strings.TrimSuffix(trimmed, "s"), 64); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("firebase: negative expiration %q", value)
		}
		return time.Duration(seconds * float64(time.Second)), nil
	}
	// Fall back to Go duration syntax ("1h30m"), harmless and occasionally handy.
	duration, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("firebase: unparsable expiration %q", value)
	}
	return duration, nil
}

// Validate reports whether the installation response is usable.
func (r *FireBaseInstallationResponse) Validate() error {
	if r == nil {
		return ErrEmptyResponse
	}
	if r.FID == "" {
		return fmt.Errorf("firebase: installation response has no fid")
	}
	if r.AuthToken.Token == "" {
		return fmt.Errorf("firebase: installation response has no auth token")
	}
	if _, err := ParseExpiresIn(r.AuthToken.Expiration); err != nil {
		return fmt.Errorf("firebase: installation response has invalid expiration: %w", err)
	}
	return nil
}

// ExpiresIn returns the lifetime of the installation auth token.
func (r *FireBaseInstallationResponse) ExpiresIn() (time.Duration, error) {
	if r == nil {
		return 0, ErrEmptyResponse
	}
	return ParseExpiresIn(r.AuthToken.Expiration)
}

func NotifyInstallationResult(resp *http.Response) (*FireBaseInstallationResponse, error) {
	result := new(FireBaseInstallationResponse)
	if err := decodeJSON(resp, result); err != nil {
		return nil, err
	}
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return result, nil
}

func VerifyPasswordResult(resp *http.Response) (*GoogleVerifyPasswordResponse, error) {
	result := new(GoogleVerifyPasswordResponse)
	if err := decodeJSON(resp, result); err != nil {
		return nil, err
	}
	return result, nil
}

func SignUpNewUserResult(resp *http.Response) (*GoogleSignUpNewUserResponse, error) {
	result := new(GoogleSignUpNewUserResponse)
	if err := decodeJSON(resp, result); err != nil {
		return nil, err
	}
	return result, nil
}

func SetAccountInfoResult(resp *http.Response) (*GoogleSetAccountInfoResponse, error) {
	result := new(GoogleSetAccountInfoResponse)
	if err := decodeJSON(resp, result); err != nil {
		return nil, err
	}
	return result, nil
}

func RefreshSecureTokenResult(resp *http.Response) (*SecureTokenRefreshResponse, error) {
	result := new(SecureTokenRefreshResponse)
	if err := decodeJSON(resp, result); err != nil {
		return nil, err
	}
	return result, nil
}

func AuthResult(resp *http.Response) (*AuthResponse, error) {
	body, err := readBody(resp)
	if err != nil {
		return nil, fmt.Errorf("readBody: %w", err)
	}
	return ParseAuthResponse(body, resp.StatusCode)
}

// ParseAuthResponse parses an android auth endpoint body. The endpoint answers with
// "Error=BadAuthentication" style bodies (sometimes with HTTP 200, sometimes not), those
// are returned as an *AuthError rather than an empty AuthResponse.
func ParseAuthResponse(body []byte, statusCode int) (*AuthResponse, error) {
	fields := ParseKeyValueBody(body)
	if len(fields) == 0 {
		return nil, ErrEmptyResponse
	}

	if errCode, ok := fields["Error"]; ok {
		return nil, &AuthError{Code: strings.TrimSpace(errCode), Fields: fields}
	}

	result := new(AuthResponse)
	if token, ok := fields["Auth"]; ok {
		result.Token = token
	}
	if token, ok := fields["it"]; ok { // "it" wins, it is the more specific token
		result.Token = token
	}
	if metadata, ok := fields["itMetadata"]; ok {
		result.Metadata = metadata
	}
	if scopes, ok := fields["grantedScopes"]; ok {
		result.Scopes = strings.Fields(scopes)
	}
	if expiry, ok := fields["Expiry"]; ok {
		timeStamp, err := strconv.ParseInt(strings.TrimSpace(expiry), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("firebase: unparsable auth Expiry %q: %w", expiry, err)
		}
		result.Expires = time.Unix(timeStamp, 0)
	}

	if result.Token == "" {
		if statusCode != http.StatusOK {
			return nil, fmt.Errorf("firebase: auth failed with HTTP %d: %s", statusCode, truncate(string(body)))
		}
		return nil, fmt.Errorf("firebase: auth response contains no token: %s", truncate(string(body)))
	}
	return result, nil
}

func CheckinResult(resp *http.Response) (*CheckinResponse, error) {
	responseBody, err := readBodyExpecting(resp, http.StatusOK)
	if err != nil {
		return nil, err
	}
	if len(responseBody) == 0 {
		return nil, ErrEmptyResponse
	}

	result := new(CheckinResponse)
	if err = result.UnmarshalVT(responseBody); err != nil {
		return nil, fmt.Errorf("result.UnmarshalVT: %w", err)
	}
	if result.AndroidId == nil || *result.AndroidId == 0 {
		return nil, fmt.Errorf("firebase: check-in response contains no android id")
	}
	if result.SecurityToken == nil || *result.SecurityToken == 0 {
		return nil, fmt.Errorf("firebase: check-in response contains no security token")
	}
	return result, nil
}

// AndroidRegisterResult parses a c2dm register3 response.
//
// The endpoint answers with a "key=value" body and uses HTTP 200 even for failures, the
// failure shows up as "Error=PHONE_REGISTRATION_ERROR" instead. Those are returned as a
// *RegisterError, which reports whether a retry is worthwhile via IsRetryable.
func AndroidRegisterResult(resp *http.Response) (string, error) {
	body, err := readBody(resp)
	if err != nil {
		return "", fmt.Errorf("readBody: %w", err)
	}
	return ParseAndroidRegisterResponse(body, resp.StatusCode)
}

// ParseAndroidRegisterResponse parses a c2dm register3 body into a notification token.
func ParseAndroidRegisterResponse(body []byte, statusCode int) (string, error) {
	fields := ParseKeyValueBody(body)
	if len(fields) == 0 {
		if statusCode != http.StatusOK {
			return "", fmt.Errorf("firebase: registration failed with HTTP %d: %s", statusCode, truncate(string(body)))
		}
		return "", ErrEmptyResponse
	}

	if errCode, ok := fields["Error"]; ok {
		return "", &RegisterError{Code: strings.TrimSpace(errCode), Fields: fields}
	}

	token, ok := fields["token"]
	if !ok || token == "" {
		if statusCode != http.StatusOK {
			return "", fmt.Errorf("firebase: registration failed with HTTP %d: %s", statusCode, truncate(string(body)))
		}
		return "", fmt.Errorf("firebase: registration response contains no token: %s", truncate(string(body)))
	}
	return token, nil
}
