package firebase_api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppDataValidate(t *testing.T) {
	var nilAppData *FirebaseAppData
	if !errors.Is(nilAppData.Validate(), ErrNilAppData) {
		t.Fatal("a nil appData should report ErrNilAppData instead of panicking")
	}

	appData := &FirebaseAppData{}
	err := appData.ValidateForInstallation()
	if err == nil {
		t.Fatal("expected an error")
	}
	missingErr := new(MissingFieldError)
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected a *MissingFieldError, got %T", err)
	}
	if !strings.Contains(err.Error(), "PackageID") {
		t.Fatalf("error should name the missing field: %v", err)
	}

	complete := &FirebaseAppData{
		PackageID:          "com.example.app",
		PackageCertificate: "DEADBEEF",
		GoogleAPIKey:       "AIzaKey",
		FirebaseProjectID:  "project",
		GMPAppID:           "1:1:android:1",
	}
	if err = complete.ValidateForInstallation(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppDataApplyDefaults(t *testing.T) {
	appData := &FirebaseAppData{AppVersion: "1.2.3"}
	appData.ApplyDefaults()
	if appData.AuthVersion != DefaultAuthVersion {
		t.Fatalf("unexpected auth version: %q", appData.AuthVersion)
	}
	if appData.SdkVersion != DefaultSdkVersion {
		t.Fatalf("unexpected sdk version: %q", appData.SdkVersion)
	}
	if appData.AppVersionWithBuild != "1.2.3" {
		t.Fatalf("unexpected app version with build: %q", appData.AppVersionWithBuild)
	}

	explicit := &FirebaseAppData{AuthVersion: "custom", SdkVersion: "custom", AppVersionWithBuild: "9"}
	explicit.ApplyDefaults()
	if explicit.AuthVersion != "custom" || explicit.SdkVersion != "custom" || explicit.AppVersionWithBuild != "9" {
		t.Fatal("ApplyDefaults must not overwrite explicit values")
	}
}

func TestDeviceHelpers(t *testing.T) {
	var nilDevice *FirebaseDevice
	if !errors.Is(nilDevice.Validate(), ErrNilDevice) {
		t.Fatal("a nil device should report ErrNilDevice instead of panicking")
	}
	if nilDevice.HasCheckin() {
		t.Fatal("a nil device has no check-in")
	}
	if _, ok := nilDevice.NotificationToken("com.example.app"); ok {
		t.Fatal("a nil device has no notification token")
	}

	// A device without a hardware profile reports the missing field, it does not panic.
	device := &FirebaseDevice{}
	missingErr := new(MissingFieldError)
	if err := device.ValidateForCheckinScopedCall(); !errors.As(err, &missingErr) {
		t.Fatalf("expected a *MissingFieldError, got %v", err)
	}

	device.FirebaseInstallations = map[string]*FirebaseInstallationData{
		"com.example.app": {NotificationData: &FirebaseNotificationData{NotificationToken: "token"}},
	}
	token, ok := device.NotificationToken("com.example.app")
	if !ok || token != "token" {
		t.Fatalf("unexpected token: %q (%t)", token, ok)
	}
	if _, ok = device.NotificationToken("com.other.app"); ok {
		t.Fatal("unknown package should not have a token")
	}
}

func TestRequestBuildersRejectIncompleteInput(t *testing.T) {
	ctx := t.Context()
	device := &FirebaseDevice{}
	appData := &FirebaseAppData{PackageID: "com.example.app", PackageCertificate: "DEADBEEF"}

	if _, err := NotifyInstallationRequest(ctx, device, appData); err == nil {
		t.Fatal("expected an error for a device without a hardware profile")
	}
	if _, err := C2DMAndroidRegisterRequest(ctx, nil, appData); !errors.Is(err, ErrNilDevice) {
		t.Fatalf("expected ErrNilDevice, got %v", err)
	}
}

func TestResultParsersOverHTTP(t *testing.T) {
	t.Run("register error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Error=AUTHENTICATION_FAILED"))
		}))
		defer server.Close()

		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = AndroidRegisterResult(resp); err == nil {
			t.Fatal("expected an error")
		} else if !IsRetryable(err) {
			t.Fatalf("AUTHENTICATION_FAILED should be retryable: %v", err)
		}
	})

	t.Run("installation http error keeps the body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":403,"message":"API key not valid.","status":"PERMISSION_DENIED"}}`))
		}))
		defer server.Close()

		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		_, err = NotifyInstallationResult(resp)
		if err == nil {
			t.Fatal("expected an error")
		}
		// The old implementation reported only "unexpected http status code: 403".
		if !strings.Contains(err.Error(), "API key not valid.") {
			t.Fatalf("error should carry the response body: %v", err)
		}
		apiErr := new(GoogleAPIError)
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected a *GoogleAPIError, got %T", err)
		}
	})

	t.Run("checkin without credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(nil)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = CheckinResult(resp); !errors.Is(err, ErrEmptyResponse) {
			t.Fatalf("expected ErrEmptyResponse, got %v", err)
		}
	})
}
