package firebase_api

import (
	"fmt"
	"strings"
)

// Defaults that are applied when the caller leaves the corresponding field empty.
const (
	// DefaultAuthVersion is the only auth version Firebase installations accepts today.
	DefaultAuthVersion = "FIS_v2"
	// DefaultSdkVersion mirrors a recent firebase-installations release.
	DefaultSdkVersion = "a:17.1.3"
	// DefaultFirebaseClientVersion mirrors a recent firebase-messaging release.
	DefaultFirebaseClientVersion = "fcm-23.1.2"
	// DefaultGmsVersion mirrors a recent Google Play services release.
	DefaultGmsVersion = "241718022"
)

// MissingFieldError names the fields a call needs but did not get.
type MissingFieldError struct {
	// Type is the struct the fields belong to, e.g. "FirebaseAppData".
	Type string
	// Fields are the names of the missing fields.
	Fields []string
	// Operation is the call that needed them, empty when not applicable.
	Operation string
}

func (e *MissingFieldError) Error() string {
	msg := fmt.Sprintf("firebase: %s is missing required field(s): %s", e.Type, strings.Join(e.Fields, ", "))
	if e.Operation != "" {
		msg += " (needed by " + e.Operation + ")"
	}
	return msg
}

func missing(typeName, operation string, fields []string) error {
	if len(fields) == 0 {
		return nil
	}
	return &MissingFieldError{Type: typeName, Fields: fields, Operation: operation}
}

// ApplyDefaults fills in the fields that have a sane default so callers only have to
// provide the app specific values. It is called automatically by the client.
func (a *FirebaseAppData) ApplyDefaults() {
	if a == nil {
		return
	}
	if a.AuthVersion == "" {
		a.AuthVersion = DefaultAuthVersion
	}
	if a.SdkVersion == "" {
		a.SdkVersion = DefaultSdkVersion
	}
	if a.AppVersionWithBuild == "" {
		a.AppVersionWithBuild = a.AppVersion
	}
}

// Validate checks the fields every app scoped call needs.
func (a *FirebaseAppData) Validate() error {
	if a == nil {
		return ErrNilAppData
	}
	var fields []string
	if a.PackageID == "" {
		fields = append(fields, "PackageID")
	}
	if a.PackageCertificate == "" {
		fields = append(fields, "PackageCertificate")
	}
	return missing("FirebaseAppData", "", fields)
}

// ValidateForInstallation checks the fields NotifyInstallation needs.
func (a *FirebaseAppData) ValidateForInstallation() error {
	if err := a.Validate(); err != nil {
		return err
	}
	var fields []string
	if a.GoogleAPIKey == "" {
		fields = append(fields, "GoogleAPIKey")
	}
	if a.FirebaseProjectID == "" {
		fields = append(fields, "FirebaseProjectID")
	}
	if a.GMPAppID == "" {
		fields = append(fields, "GMPAppID")
	}
	return missing("FirebaseAppData", "NotifyInstallation", fields)
}

// ValidateForRegistration checks the fields C2DMRegisterAndroid needs.
func (a *FirebaseAppData) ValidateForRegistration() error {
	if err := a.Validate(); err != nil {
		return err
	}
	var fields []string
	if a.NotificationSenderID == "" {
		fields = append(fields, "NotificationSenderID")
	}
	if a.GMPAppID == "" {
		fields = append(fields, "GMPAppID")
	}
	if a.AppVersion == "" && a.AppVersionWithBuild == "" {
		fields = append(fields, "AppVersion")
	}
	return missing("FirebaseAppData", "C2DMRegisterAndroid", fields)
}

// ValidateForAPIKeyCall checks the fields the identitytoolkit calls need.
func (a *FirebaseAppData) ValidateForAPIKeyCall(operation string) error {
	if err := a.Validate(); err != nil {
		return err
	}
	var fields []string
	if a.GoogleAPIKey == "" {
		fields = append(fields, "GoogleAPIKey")
	}
	return missing("FirebaseAppData", operation, fields)
}

// ApplyDefaults fills in the device fields that have a sane default.
func (d *FirebaseDevice) ApplyDefaults() {
	if d == nil {
		return
	}
	if d.GmsVersion == "" {
		d.GmsVersion = DefaultGmsVersion
	}
	if d.FirebaseClientVersion == "" {
		d.FirebaseClientVersion = DefaultFirebaseClientVersion
	}
}

// Validate checks that the device carries the hardware profile every call needs.
func (d *FirebaseDevice) Validate() error {
	if d == nil {
		return ErrNilDevice
	}
	if d.Device == nil {
		return &MissingFieldError{Type: "FirebaseDevice", Fields: []string{"Device"}}
	}
	return nil
}

// HasCheckin reports whether the device holds usable check-in credentials.
func (d *FirebaseDevice) HasCheckin() bool {
	return d != nil && d.CheckinAndroidID != 0 && d.CheckinSecurityToken != 0
}

// ValidateForCheckinScopedCall checks that the device has been checked in.
func (d *FirebaseDevice) ValidateForCheckinScopedCall() error {
	if err := d.Validate(); err != nil {
		return err
	}
	if !d.HasCheckin() {
		return ErrNoCheckin
	}
	return nil
}

// Installation returns the stored installation for a package, if any.
func (d *FirebaseDevice) Installation(packageID string) (*FirebaseInstallationData, bool) {
	if d == nil || d.FirebaseInstallations == nil {
		return nil, false
	}
	installation, ok := d.FirebaseInstallations[packageID]
	return installation, ok && installation != nil
}

// NotificationToken returns the stored FCM notification token for a package, if any.
func (d *FirebaseDevice) NotificationToken(packageID string) (string, bool) {
	installation, ok := d.Installation(packageID)
	if !ok || installation.NotificationData == nil || installation.NotificationData.NotificationToken == "" {
		return "", false
	}
	return installation.NotificationData.NotificationToken, true
}
