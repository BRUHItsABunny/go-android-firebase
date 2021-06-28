package go_android_firebase

import (
	"fmt"
	go_android_utils "github.com/BRUHItsABunny/go-android-utils"
)

const (
	Protocol = "https"
	Host     = "firebaseinstallations.googleapis.com"

	EndpointProjects = "/v1/projects/"

	SubEndpointInstallations = "/installations"

	HeaderKeyFireBaseClient  = "x-firebase-client"
	HeaderKeyFireBaseLogType = "x-firebase-log-type"
	HeaderKeyAndroidCert     = "X-Android-Cert"
	HeaderKeyAndroidPackage  = "X-Android-Package"
	HeaderKeyGoogAPIKey      = "x-goog-api-key"
	HeaderKeyUserAgent       = "User-Agent"

	HeaderKeyContentType  = "Content-Type"
	HeaderKeyAccept       = "Accept"
	HeaderKeyCacheControl = "Cache-Control"

	HeaderValueMIMEJSON = "application/json"
)

var (
	HeaderValueFireBaseClient = fmt.Sprintf("kotlin/1.4.10 fire-analytics/19.0.0 android-target-sdk/30 android-min-sdk/24 fire-core/20.0.0 device-name/%s device-model/%s fire-android/%s fire-iid/21.0.1 android-installer/com.android.vending device-brand/%s fire-installations/17.0.0 android-platform/ fire-fcm/20.1.7_1p",
		go_android_utils.DeviceFormatKeyDevice,
		go_android_utils.DeviceFormatKeyDevice,
		go_android_utils.DeviceFormatKeyAndroidSDKLevel,
		go_android_utils.DeviceFormatKeyManufacturer,
	)

	HeaderValueUserAgentPrefix = fmt.Sprintf("Dalvik/2.1.0 ")
)
