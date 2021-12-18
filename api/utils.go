package api

import (
	"fmt"
	. "github.com/BRUHItsABunny/go-android-firebase/constants"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"net/http"
)

func DefaultHeadersFirebase(device *FirebaseDevice, includeAPIKey, includeFireBaseClient, includeClientVersion bool) http.Header {
	headers := http.Header{
		HeaderKeyContentType:    {HeaderValueMIMEJSON},
		HeaderKeyAccept:         {HeaderValueMIMEJSON},
		HeaderKeyCacheControl:   {"no-cache"},
		HeaderKeyAndroidCert:    {device.AndroidCert},
		HeaderKeyAndroidPackage: {device.AndroidPackage},
		HeaderKeyUserAgent:      {"Dalvik/2.1.0 " + device.Device.GetUserAgent()},
	}

	if includeAPIKey {
		headers[HeaderKeyGoogAPIKey] = []string{device.GoogleAPIKey}
	}
	if includeFireBaseClient {
		headers[HeaderKeyFireBaseClient] = []string{device.Device.FormatUserAgent(HeaderValueFireBaseClient)}
	}
	if includeClientVersion {
		// TODO: Constant or variable?
		headers[HeaderKeyClientVersion] = []string{HeaderValueClientVersion}
	}
	return headers
}

func DefaultHeadersAuth(device *andutils.Device) http.Header {
	headers := http.Header{
		HeaderKeyContentType: {"application/x-www-form-urlencoded"},
		"app":                {"com.google.android.gm"},
		"device":             {device.Id.ToHexString()},
		HeaderKeyUserAgent:   {fmt.Sprintf("GoogleAuth/1.4 (%s %s); gzip", andutils.DeviceFormatKeyModel, andutils.DeviceFormatKeyBuild)},
	}

	return headers
}
