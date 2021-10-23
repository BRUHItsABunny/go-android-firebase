package api

import (
	. "github.com/BRUHItsABunny/go-android-firebase/constants"
	"net/http"
)

func DefaultHeadersFirebase(device *FirebaseDevice, includeAPIKey, includeFireBaseClient, includeClientVersion bool) http.Header {
	headers := http.Header{
		HeaderKeyContentType:  {HeaderValueMIMEJSON},
		HeaderKeyAccept:       {HeaderValueMIMEJSON},
		HeaderKeyCacheControl: {"no-cache"},
		HeaderKeyAndroidCert: {device.AndroidCert},
		HeaderKeyAndroidPackage: {device.AndroidPackage},
		HeaderKeyUserAgent: {"Dalvik/2.1.0 " + device.Device.GetUserAgent()},
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