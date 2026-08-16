package firebase_client

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testClient(t *testing.T) *FireBaseClient {
	t.Helper()
	client, err := NewFirebaseClient(nil, nil)
	if err != nil {
		t.Fatalf("NewFirebaseClient: %v", err)
	}
	return client
}

func TestNewFirebaseClientDefaults(t *testing.T) {
	client := testClient(t)
	if client.Device == nil || client.Device.Device == nil {
		t.Fatal("a nil device should be replaced by a random one")
	}
	if client.Client == nil || client.Client.Timeout == 0 {
		t.Fatal("the default HTTP client should have a timeout")
	}
	if client.Device.GmsVersion == "" || client.Device.FirebaseClientVersion == "" {
		t.Fatal("device defaults should be applied")
	}
	if client.MTalk == nil || client.MTalk.Connected() {
		t.Fatal("MTalk should exist but not be connected")
	}
	if client.Device.MTalkPrivateKey == "" || client.Device.MTalkPublicKey == "" {
		// The private key used to be dropped on the floor, which made a persisted device
		// unable to decrypt web push notifications after a restart.
		t.Fatal("the generated key pair should be stored on the device")
	}
}

func TestNewFirebaseClientKeepsCallerClient(t *testing.T) {
	provided := &http.Client{}
	client, err := NewFirebaseClient(provided, nil, WithTimeout(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if client.Client != provided {
		t.Fatal("the caller's HTTP client should be used as-is")
	}
	if provided.Timeout != 0 {
		t.Fatal("WithTimeout must not mutate a client we do not own")
	}
}

func TestMTalkKeysSurviveARoundTrip(t *testing.T) {
	first := testClient(t)
	device := first.Device

	// Rebuild a client from the persisted device, as a caller restoring saved state would.
	second, err := NewFirebaseClient(nil, device)
	if err != nil {
		t.Fatalf("NewFirebaseClient: %v", err)
	}
	if second.Device.MTalkPrivateKey != first.Device.MTalkPrivateKey {
		t.Fatal("the private key should be reused, not regenerated")
	}
	if second.Device.MTalkPublicKey != first.Device.MTalkPublicKey {
		t.Fatal("the public key should match the persisted private key")
	}
}

func TestCallsFailFastOnIncompleteInput(t *testing.T) {
	client := testClient(t)
	ctx := t.Context()

	if _, err := client.NotifyInstallation(ctx, nil); !errors.Is(err, firebase_api.ErrNilAppData) {
		t.Fatalf("expected ErrNilAppData, got %v", err)
	}

	appData := &firebase_api.FirebaseAppData{
		PackageID:            "com.example.app",
		PackageCertificate:   "DEADBEEF",
		NotificationSenderID: "1234567890",
		GMPAppID:             "1:1234567890:android:abc",
		AppVersion:           "1.0.0",
	}
	// No installation and no check-in yet, this must not reach the network.
	if _, err := client.C2DMRegisterAndroid(ctx, appData); !errors.Is(err, firebase_api.ErrNoCheckin) {
		t.Fatalf("expected ErrNoCheckin, got %v", err)
	}

	client.Device.CheckinAndroidID = 1
	client.Device.CheckinSecurityToken = 1
	if _, err := client.C2DMRegisterAndroid(ctx, appData); !errors.Is(err, firebase_api.ErrNoInstallation) {
		t.Fatalf("expected ErrNoInstallation, got %v", err)
	}

	// An installation without authentication is just as unusable.
	client.Device.FirebaseInstallations = map[string]*firebase_api.FirebaseInstallationData{
		appData.PackageID: {FirebaseInstallationID: "fid"},
	}
	if _, err := client.C2DMRegisterAndroid(ctx, appData); !errors.Is(err, firebase_api.ErrNoInstallationAuth) {
		t.Fatalf("expected ErrNoInstallationAuth, got %v", err)
	}
}

func TestInstallationExpired(t *testing.T) {
	client := testClient(t)
	const packageID = "com.example.app"

	if !client.InstallationExpired(packageID) {
		t.Fatal("an unknown package counts as expired")
	}

	client.Device.FirebaseInstallations = map[string]*firebase_api.FirebaseInstallationData{
		packageID: {InstallationAuthentication: &firebase_api.FirebaseAuthentication{
			AccessToken: "token",
			Expires:     timestamppb.New(time.Now().Add(time.Hour)),
		}},
	}
	if client.InstallationExpired(packageID) {
		t.Fatal("a token valid for another hour is not expired")
	}

	client.Device.FirebaseInstallations[packageID].InstallationAuthentication.Expires =
		timestamppb.New(time.Now().Add(-time.Minute))
	if !client.InstallationExpired(packageID) {
		t.Fatal("an expired token should be reported as expired")
	}
}

func TestNotificationTokenAccessor(t *testing.T) {
	client := testClient(t)
	if _, ok := client.NotificationToken("com.example.app"); ok {
		t.Fatal("no token should be stored yet")
	}

	client.deviceMutex.Lock()
	client.installationLocked("com.example.app").NotificationData.NotificationToken = "token"
	client.deviceMutex.Unlock()

	token, ok := client.NotificationToken("com.example.app")
	if !ok || token != "token" {
		t.Fatalf("unexpected token: %q (%t)", token, ok)
	}
}

func TestRegisterForNotificationsValidatesEarly(t *testing.T) {
	client := testClient(t)
	if _, err := client.RegisterForNotifications(t.Context(), nil, nil); !errors.Is(err, firebase_api.ErrNilAppData) {
		t.Fatalf("expected ErrNilAppData, got %v", err)
	}
}

func TestRegisterForNotificationsHonoursContext(t *testing.T) {
	client := testClient(t)
	client.Device.Device = andutils.GetRandomDevice()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	appData := &firebase_api.FirebaseAppData{PackageID: "com.example.app", PackageCertificate: "DEADBEEF"}
	_, err := client.RegisterForNotifications(ctx, appData, &RegisterOptions{Retries: 1, RetryDelay: time.Millisecond})
	if err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
}

func TestMTalkCloseWithoutConnect(t *testing.T) {
	client := testClient(t)
	if err := client.MTalk.Close(); !errors.Is(err, firebase_api.ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
	// Client.Close treats a never-connected MTalk as a no-op.
	if err := client.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMTalkConnectRequiresCheckin(t *testing.T) {
	client := testClient(t)
	if err := client.MTalk.Connect(); !errors.Is(err, firebase_api.ErrNoCheckin) {
		t.Fatalf("expected ErrNoCheckin, got %v", err)
	}
}

func TestParseAppDataValue(t *testing.T) {
	result := parseAppDataValue("salt=abc;dh=def;;broken")
	if result["salt"] != "abc" || result["dh"] != "def" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if _, ok := result["broken"]; ok {
		t.Fatal("entries without a separator should be ignored")
	}
}
