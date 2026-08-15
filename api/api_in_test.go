package firebase_api

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestParseAndroidRegisterResponse(t *testing.T) {
	t.Run("token", func(t *testing.T) {
		token, err := ParseAndroidRegisterResponse([]byte("token=cxWt1abcDEF:APA91bH_token-value"), http.StatusOK)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "cxWt1abcDEF:APA91bH_token-value" {
			t.Fatalf("unexpected token: %q", token)
		}
	})

	t.Run("token with trailing newline", func(t *testing.T) {
		token, err := ParseAndroidRegisterResponse([]byte("token=abc123\r\n"), http.StatusOK)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "abc123" {
			t.Fatalf("unexpected token: %q", token)
		}
	})

	t.Run("error is typed and retryable", func(t *testing.T) {
		// The old implementation returned "REGISTRATION_ERROR" here by blindly cutting 6 bytes.
		_, err := ParseAndroidRegisterResponse([]byte("Error=PHONE_REGISTRATION_ERROR"), http.StatusOK)
		if err == nil {
			t.Fatal("expected an error")
		}
		registerErr := new(RegisterError)
		if !errors.As(err, &registerErr) {
			t.Fatalf("expected a *RegisterError, got %T", err)
		}
		if registerErr.Code != RegisterErrorPhoneRegistration {
			t.Fatalf("unexpected code: %q", registerErr.Code)
		}
		if !IsRetryable(err) {
			t.Fatal("PHONE_REGISTRATION_ERROR should be retryable")
		}
	})

	t.Run("non retryable error", func(t *testing.T) {
		_, err := ParseAndroidRegisterResponse([]byte("Error=INVALID_SENDER"), http.StatusOK)
		if err == nil {
			t.Fatal("expected an error")
		}
		if IsRetryable(err) {
			t.Fatal("INVALID_SENDER should not be retryable")
		}
	})

	t.Run("short body does not panic", func(t *testing.T) {
		// The old implementation panicked on any body shorter than 6 bytes.
		for _, body := range []string{"", "ab", "err"} {
			if _, err := ParseAndroidRegisterResponse([]byte(body), http.StatusOK); err == nil {
				t.Fatalf("expected an error for body %q", body)
			}
		}
	})

	t.Run("html error page", func(t *testing.T) {
		_, err := ParseAndroidRegisterResponse([]byte("<html><title>502 Bad Gateway</title></html>"), http.StatusBadGateway)
		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestParseAuthResponse(t *testing.T) {
	t.Run("token", func(t *testing.T) {
		body := "SID=xyz\nAuth=auth-token\nExpiry=1700000000\ngrantedScopes=scope-a scope-b\n"
		result, err := ParseAuthResponse([]byte(body), http.StatusOK)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Token != "auth-token" {
			t.Fatalf("unexpected token: %q", result.Token)
		}
		if !result.Expires.Equal(time.Unix(1700000000, 0)) {
			t.Fatalf("unexpected expiry: %v", result.Expires)
		}
		if len(result.Scopes) != 2 {
			t.Fatalf("unexpected scopes: %v", result.Scopes)
		}
	})

	t.Run("value containing an equals sign", func(t *testing.T) {
		result, err := ParseAuthResponse([]byte("Auth=aa==bb\n"), http.StatusOK)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Token != "aa==bb" {
			t.Fatalf("unexpected token: %q", result.Token)
		}
	})

	t.Run("error is typed", func(t *testing.T) {
		_, err := ParseAuthResponse([]byte("Error=BadAuthentication\nInfo=InvalidToken\n"), http.StatusOK)
		authErr := new(AuthError)
		if !errors.As(err, &authErr) {
			t.Fatalf("expected an *AuthError, got %T (%v)", err, err)
		}
		if authErr.Code != "BadAuthentication" {
			t.Fatalf("unexpected code: %q", authErr.Code)
		}
	})

	t.Run("line without separator does not panic", func(t *testing.T) {
		// The old implementation indexed entryParts[1] unconditionally.
		if _, err := ParseAuthResponse([]byte("garbage\nmore garbage"), http.StatusOK); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestParseExpiresIn(t *testing.T) {
	cases := map[string]time.Duration{
		"604800s": 604800 * time.Second,
		"604800":  604800 * time.Second,
		" 3600s ": time.Hour,
		"1h30m":   90 * time.Minute,
	}
	for input, want := range cases {
		got, err := ParseExpiresIn(input)
		if err != nil {
			t.Fatalf("ParseExpiresIn(%q): unexpected error: %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseExpiresIn(%q) = %v, want %v", input, got, want)
		}
	}

	for _, input := range []string{"", "s", "abc", "-5s"} {
		// The old implementation cut the last byte and silently used 0 seconds for these.
		if _, err := ParseExpiresIn(input); err == nil {
			t.Fatalf("ParseExpiresIn(%q): expected an error", input)
		}
	}
}

func TestParseKeyValueBody(t *testing.T) {
	fields := ParseKeyValueBody([]byte("a=1\r\nb=2=3\n\nnot-a-pair\nc=\n"))
	if fields["a"] != "1" || fields["b"] != "2=3" || fields["c"] != "" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
	if _, ok := fields["not-a-pair"]; ok {
		t.Fatal("lines without a separator should be ignored")
	}
}

func TestParseGoogleAPIError(t *testing.T) {
	body := []byte(`{"error":{"code":400,"message":"API key not valid.","status":"INVALID_ARGUMENT"}}`)
	apiErr := parseGoogleAPIError(body)
	if apiErr == nil {
		t.Fatal("expected an error to be parsed")
	}
	if apiErr.Code != 400 || apiErr.Message != "API key not valid." {
		t.Fatalf("unexpected error: %#v", apiErr)
	}
	if IsRetryable(apiErr) {
		t.Fatal("400 should not be retryable")
	}
	if !IsRetryable(&GoogleAPIError{Code: 503}) {
		t.Fatal("503 should be retryable")
	}

	if parseGoogleAPIError([]byte(`{"fid":"abc"}`)) != nil {
		t.Fatal("a success body should not parse as an error")
	}
	if parseGoogleAPIError([]byte("not json")) != nil {
		t.Fatal("a non-JSON body should not parse as an error")
	}
}

func TestInstallationResponseValidate(t *testing.T) {
	valid := &FireBaseInstallationResponse{
		FID:       "abc",
		AuthToken: FireBaseAuthToken{Token: "token", Expiration: "604800s"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expiresIn, err := valid.ExpiresIn()
	if err != nil || expiresIn != 604800*time.Second {
		t.Fatalf("unexpected expiry: %v (%v)", expiresIn, err)
	}

	invalid := []*FireBaseInstallationResponse{
		{},
		{FID: "abc"},
		{FID: "abc", AuthToken: FireBaseAuthToken{Token: "token"}},
		{FID: "abc", AuthToken: FireBaseAuthToken{Token: "token", Expiration: "soon"}},
	}
	for i, result := range invalid {
		if err := result.Validate(); err == nil {
			t.Fatalf("case %d: expected an error", i)
		}
	}
}
