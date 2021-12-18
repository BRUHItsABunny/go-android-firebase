package constants

import (
	"fmt"
	go_android_utils "github.com/BRUHItsABunny/go-android-utils"
)

const (
	protocol = "https://"
	host     = protocol + "firebaseinstallations.googleapis.com"

	endpointProjects = host + "/v1/projects/%s"

	EndpointInstallations      = endpointProjects + "/installations"
	EndpointVerifyPassword     = "https://www.googleapis.com/identitytoolkit/v3/relyingparty/verifyPassword"
	EndpointRefreshSecureToken = "https://securetoken.googleapis.com/v1/token"
	EndpointAuth               = "https://android.googleapis.com/auth"

	HeaderKeyFireBaseClient  = "x-firebase-client"
	HeaderKeyClientVersion   = "x-client-version"
	HeaderKeyFireBaseLogType = "x-firebase-log-type"
	HeaderKeyAndroidCert     = "X-Android-Cert"
	HeaderKeyAndroidPackage  = "X-Android-Package"
	HeaderKeyGoogAPIKey      = "x-goog-api-key"
	HeaderKeyUserAgent       = "User-Agent"

	HeaderKeyContentType  = "Content-Type"
	HeaderKeyAccept       = "Accept"
	HeaderKeyCacheControl = "Cache-Control"

	HeaderValueMIMEJSON      = "application/json"
	HeaderValueClientVersion = "Android/Fallback/X20000001/FirebaseCore-Android"
)

var (
	HeaderValueFireBaseClient = fmt.Sprintf("kotlin/1.4.10 fire-analytics/19.0.0 android-target-sdk/30 android-min-sdk/24 fire-core/20.0.0 device-name/%s device-model/%s fire-android/%s fire-iid/21.0.1 android-installer/com.android.vending device-brand/%s fire-installations/17.0.0 android-platform/ fire-fcm/20.1.7_1p",
		go_android_utils.DeviceFormatKeyDevice,
		go_android_utils.DeviceFormatKeyDevice,
		go_android_utils.DeviceFormatKeyAndroidSDKLevel,
		go_android_utils.DeviceFormatKeyManufacturer,
	)
)
