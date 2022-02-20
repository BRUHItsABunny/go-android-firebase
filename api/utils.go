package api

import (
	"encoding/base64"
	"fmt"
	. "github.com/BRUHItsABunny/go-android-firebase/constants"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"math/rand"
	"net/http"
	"strings"
)

func DefaultHeadersFirebase(device *FirebaseDevice, appData *FirebaseAppData, includeAPIKey, includeFireBaseClient, includeClientVersion bool) http.Header {
	headers := http.Header{
		HeaderKeyContentType:    {HeaderValueMIMEJSON},
		HeaderKeyAccept:         {HeaderValueMIMEJSON},
		HeaderKeyCacheControl:   {"no-cache"},
		HeaderKeyAndroidCert:    {strings.ToUpper(appData.PackageCertificate)},
		HeaderKeyAndroidPackage: {appData.PackageID},
		HeaderKeyUserAgent:      {"Dalvik/2.1.0 " + device.Device.GetUserAgent()},
	}

	if includeAPIKey {
		headers[HeaderKeyGoogAPIKey] = []string{appData.GoogleAPIKey}
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
		HeaderKeyUserAgent:   {device.FormatUserAgent(fmt.Sprintf("GoogleAuth/1.4 (%s %s); gzip", andutils.DeviceFormatKeyModel, andutils.DeviceFormatKeyBuild))},
	}

	return headers
}

func DefaultHeadersCheckin(device *andutils.Device) http.Header {
	headers := http.Header{
		HeaderKeyContentType: {"application/x-protobuffer"},
		HeaderKeyUserAgent:   {device.FormatUserAgent(fmt.Sprintf("Android-Checkin/2.0 (%s %s); gzip", andutils.DeviceFormatKeyDevice, andutils.DeviceFormatKeyBuild))},
	}

	return headers
}

func DefaultHeadersAndroidRegister(device *FirebaseDevice) http.Header {
	headers := http.Header{
		HeaderKeyContentType: {"application/x-www-form-urlencoded"},
		HeaderKeyUserAgent:   {device.Device.FormatUserAgent(fmt.Sprintf("Android-GCM/1.5 (%s %s)", andutils.DeviceFormatKeyDevice, andutils.DeviceFormatKeyBuild))},
		"Authorization":      {fmt.Sprintf("AidLogin %d:%d", device.CheckinAndroidID, device.CheckinSecurityToken)},
	}
	return headers
}

func RandomAppFID() string {
	// Mimic: https://firebase.google.com/docs/reference/android/com/google/firebase/installations/FirebaseInstallations#public-taskstring-getid
	// url-safe ase84 of 128bit integer as bytes, our approach 16 random bytes
	fakeInt := make([]byte, 16)
	rand.Read(fakeInt)
	return base64.RawURLEncoding.EncodeToString(fakeInt)
}
