package firebase_api

import (
	"log/slog"
)

// The types in this file implement slog.LogValuer so that logging one of them produces a
// group of named attributes instead of a struct dump: an app is logged as its package id,
// its app id and its API key, a device as its check-in id and its versions.
//
// Values are logged as they are. Anything this library logs - tokens, keys, credentials,
// request and response bodies - reaches the handler verbatim, so point the logger at
// somewhere you are willing to have those end up.

// LogValue implements slog.LogValuer for an app's Firebase constants.
func (a *FirebaseAppData) LogValue() slog.Value {
	if a == nil {
		return slog.StringValue("<nil>")
	}
	return slog.GroupValue(
		slog.String("package_id", a.PackageID),
		slog.String("gmp_app_id", a.GMPAppID),
		slog.String("firebase_project_id", a.FirebaseProjectID),
		slog.String("notification_sender_id", a.NotificationSenderID),
		slog.String("app_version", a.AppVersion),
		slog.String("google_api_key", a.GoogleAPIKey),
	)
}

// LogValue implements slog.LogValuer for a device.
func (d *FirebaseDevice) LogValue() slog.Value {
	if d == nil {
		return slog.StringValue("<nil>")
	}
	attrs := []slog.Attr{
		slog.Int64("checkin_android_id", d.CheckinAndroidID),
		slog.Bool("has_checkin", d.HasCheckin()),
		slog.String("gms_version", d.GmsVersion),
		slog.String("firebase_client_version", d.FirebaseClientVersion),
		slog.Int("installations", len(d.FirebaseInstallations)),
	}
	if d.Device != nil {
		attrs = append(attrs,
			slog.String("device_model", d.Device.Model),
			slog.String("android_sdk", d.Device.Version.ToAndroidSDK()),
		)
	}
	return slog.GroupValue(attrs...)
}

// LogValue implements slog.LogValuer for a stored installation.
func (i *FirebaseInstallationData) LogValue() slog.Value {
	if i == nil {
		return slog.StringValue("<nil>")
	}
	attrs := []slog.Attr{slog.String("fid", i.FirebaseInstallationID)}
	if i.InstallationAuthentication != nil {
		attrs = append(attrs, slog.Attr{Key: "auth", Value: i.InstallationAuthentication.LogValue()})
	}
	if i.NotificationData != nil {
		attrs = append(attrs, slog.String("notification_token", i.NotificationData.NotificationToken))
	}
	return slog.GroupValue(attrs...)
}

// LogValue implements slog.LogValuer for stored authentication.
func (a *FirebaseAuthentication) LogValue() slog.Value {
	if a == nil {
		return slog.StringValue("<nil>")
	}
	attrs := []slog.Attr{
		slog.String("access_token", a.AccessToken),
		slog.String("refresh_token", a.RefreshToken),
	}
	if a.Expires != nil {
		attrs = append(attrs, slog.Time("expires", a.Expires.AsTime()))
	}
	return slog.GroupValue(attrs...)
}

// LogValue implements slog.LogValuer for an installation response.
func (r *FireBaseInstallationResponse) LogValue() slog.Value {
	if r == nil {
		return slog.StringValue("<nil>")
	}
	return slog.GroupValue(
		slog.String("fid", r.FID),
		slog.String("name", r.Name),
		slog.String("auth_token", r.AuthToken.Token),
		slog.String("expires_in", r.AuthToken.Expiration),
		slog.String("refresh_token", r.RefreshToken),
	)
}

// LogValue implements slog.LogValuer for an auth response.
func (r *AuthResponse) LogValue() slog.Value {
	if r == nil {
		return slog.StringValue("<nil>")
	}
	return slog.GroupValue(
		slog.String("token", r.Token),
		slog.Time("expires", r.Expires),
		slog.Any("scopes", r.Scopes),
	)
}

// LogValue implements slog.LogValuer for a registration failure, so the code and the
// extra fields the endpoint returned end up as separate attributes.
func (e *RegisterError) LogValue() slog.Value {
	if e == nil {
		return slog.StringValue("<nil>")
	}
	return slog.GroupValue(
		slog.String("code", e.Code),
		slog.Bool("retryable", e.IsRetryable()),
		slog.Int("fields", len(e.Fields)),
	)
}
