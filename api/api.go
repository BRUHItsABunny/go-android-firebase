package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/BRUHItsABunny/go-android-firebase/constants"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func NotifyInstallationRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	fid := RandomAppFID()
	gmpAppID := appData.GMPAppID
	authVersion := appData.AuthVersion
	sdkVersion := appData.SdkVersion

	installationData, ok := device.FirebaseInstallations[appData.PackageID]
	if ok {
		fid = installationData.FirebaseInstallationID
	}

	data := NotifyInstallationRequestBody{
		FID:         fid,
		AppID:       gmpAppID,
		AuthVersion: authVersion,
		SDKVersion:  sdkVersion,
	}

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", fmt.Sprintf(EndpointInstallations, appData.FirebaseProjectID), bytes.NewBuffer(body))
		if err == nil {
			req.Header = DefaultHeadersFirebase(device, appData, true, true, false)
		}
	}

	return req, err
}

func VerifyPasswordRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, data *VerifyPasswordRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointVerifyPassword, bytes.NewBuffer(body))
		if err == nil {
			req.URL.RawQuery = url.Values{"key": {appData.GoogleAPIKey}}.Encode()
			req.Header = DefaultHeadersFirebase(device, appData, false, false, true)
		}
	}

	return req, err
}

func SignUpNewUser(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, data *SignUpNewUserRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointSignUpNewUser, bytes.NewBuffer(body))
		if err == nil {
			req.URL.RawQuery = url.Values{"key": {appData.GoogleAPIKey}}.Encode()
			req.Header = DefaultHeadersFirebase(device, appData, false, false, true)
		}
	}

	return req, err
}

func SetAccountInto(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, data *SetAccountInfoRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointSetAccountInto, bytes.NewBuffer(body))
		if err == nil {
			req.URL.RawQuery = url.Values{"key": {appData.GoogleAPIKey}}.Encode()
			req.Header = DefaultHeadersFirebase(device, appData, false, false, true)
		}
	}

	return req, err
}

func RefreshSecureTokenRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, data *RefreshSecureTokenRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointRefreshSecureToken, bytes.NewBuffer(body))
		if err == nil {
			req.URL.RawQuery = url.Values{"key": {appData.GoogleAPIKey}}.Encode()
			req.Header = DefaultHeadersFirebase(device, appData, false, false, true)
		}
	}

	return req, err
}

func Auth(ctx context.Context, device *andutils.Device, appData *FirebaseAppData, data url.Values, email, masterToken string) (*http.Request, error) {
	var (
		req *http.Request
		err error
	)

	data["Email"] = []string{email}
	data["Token"] = []string{masterToken}

	data["androidId"] = []string{device.Id.ToHexString()}
	data["lang"] = []string{device.Locale.ToLocale("-", true)}
	data["device_country"] = []string{device.Locale.GetCountry(false)}
	data["sdk_version"] = []string{device.Version.ToAndroidSDK()}

	req, err = http.NewRequestWithContext(ctx, "POST", EndpointAuth, bytes.NewBufferString(data.Encode()))
	if err == nil {
		req.Header = DefaultHeadersAuth(device)
	}

	return req, err
}

// TODO: Add checkin and register3, maybe even later todo also add notification listener impl?
func CheckinAndroidRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, digest, otaCert string, accountCookies ...string) (*http.Request, error) {
	var (
		req *http.Request
	)

	reqObj := NewCheckinRequest(device.Device)
	if len(digest) > 0 {
		reqObj.Digest = &digest
	}
	if len(otaCert) > 0 {
		reqObj.OtaCert = []string{otaCert}
	}
	if len(accountCookies) > 0 {
		reqObj.AccountCookie = accountCookies
	}
	if device.CheckinAndroidID > 0 {
		reqObj.AndroidId = &device.CheckinAndroidID
	}
	if device.CheckinSecurityToken != 0 {
		reqObj.SecurityToken = &device.CheckinSecurityToken
	}
	reqBytes, err := reqObj.MarshalVT()
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointAndroidCheckin, bytes.NewBuffer(reqBytes))
		if err == nil {
			req.Header = DefaultHeadersCheckin(device.Device)
		}
	}

	return req, err
}

func C2DMAndroidRegisterRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData) (*http.Request, error) {

	var (
		req *http.Request
		err error
	)

	installationData, ok := device.FirebaseInstallations[appData.PackageID]
	if !ok {
		return nil, errors.New("no installation available")
	}

	reqBody := url.Values{
		"sender":                             {appData.NotificationSenderID},
		"X-subtype":                          {appData.NotificationSenderID},
		"X-app_ver":                          {appData.AppVersionWithBuild},
		"X-osv":                              {device.Device.Version.ToAndroidSDK()},
		"X-cliv":                             {"fcm-22.0.0"},
		"X-gmsv":                             {"214815028"},
		"X-appid":                            {installationData.FirebaseInstallationID},
		"X-scope":                            {"*"},
		"X-Goog-Firebase-Installations-Auth": {installationData.InstallationAuthentication.AccessToken},
		"X-gmp_app_id":                       {appData.GMPAppID},
		"X-Firebase-Client":                  {device.Device.FormatUserAgent(HeaderValueFireBaseClient)},
		"X-firebase-app-name-hash":           {appData.AppNameHash},
		"X-Firebase-Client-Log-Type":         {"1"},
		"X-app_ver_name":                     {appData.AppVersion},
		"app":                                {appData.PackageID},
		"device":                             {strconv.FormatInt(device.CheckinAndroidID, 10)},
		"app_ver":                            {appData.AppVersionWithBuild},
		"gcm_ver":                            {"214815028"},
		"plat":                               {"0"},
		"cert":                               {strings.ToLower(appData.PackageCertificate)},
		"target_ver":                         {"30"},
	}

	req, err = http.NewRequestWithContext(ctx, "POST", EndpointAndroidRegister, bytes.NewBufferString(reqBody.Encode()))
	if err == nil {
		req.Header = DefaultHeadersAndroidRegister(device)
	}

	return req, err
}
