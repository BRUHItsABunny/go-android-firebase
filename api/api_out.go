package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BRUHItsABunny/gOkHttp/requests"
	. "github.com/BRUHItsABunny/go-android-firebase/constants"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func NotifyInstallationRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData) (*http.Request, error) {
	fid, _ := RandomAppFID()
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

	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	req, err := requests.MakePOSTRequest(ctx, fmt.Sprintf(EndpointInstallations, appData.FirebaseProjectID),
		requests.NewPOSTRawOption(bytes.NewBuffer(body), HeaderValueMIMEJSON, int64(len(body))),
		requests.NewHeaderOption(DefaultHeadersFirebase(device, appData, true, true, false)),
	)
	if err != nil {
		return nil, fmt.Errorf("requests.MakePOSTRequest: %w", err)
	}
	return req, err
}

func VerifyPasswordRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, data *VerifyPasswordRequestBody) (*http.Request, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	req, err := requests.MakePOSTRequest(ctx, EndpointVerifyPassword,
		requests.NewURLParamOption(url.Values{"key": {appData.GoogleAPIKey}}),
		requests.NewPOSTRawOption(bytes.NewBuffer(body), HeaderValueMIMEJSON, int64(len(body))),
		requests.NewHeaderOption(DefaultHeadersFirebase(device, appData, false, false, true)),
	)
	if err != nil {
		return nil, fmt.Errorf("requests.MakePOSTRequest: %w", err)
	}
	return req, err
}

func SignUpNewUser(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, data *SignUpNewUserRequestBody) (*http.Request, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	req, err := requests.MakePOSTRequest(ctx, EndpointSignUpNewUser,
		requests.NewURLParamOption(url.Values{"key": {appData.GoogleAPIKey}}),
		requests.NewPOSTRawOption(bytes.NewBuffer(body), HeaderValueMIMEJSON, int64(len(body))),
		requests.NewHeaderOption(DefaultHeadersFirebase(device, appData, false, false, true)),
	)
	if err != nil {
		return nil, fmt.Errorf("requests.MakePOSTRequest: %w", err)
	}
	return req, err
}

func SetAccountInfoRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, data *SetAccountInfoRequestBody) (*http.Request, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	req, err := requests.MakePOSTRequest(ctx, EndpointSetAccountInto,
		requests.NewURLParamOption(url.Values{"key": {appData.GoogleAPIKey}}),
		requests.NewPOSTRawOption(bytes.NewBuffer(body), HeaderValueMIMEJSON, int64(len(body))),
		requests.NewHeaderOption(DefaultHeadersFirebase(device, appData, false, false, true)),
	)
	if err != nil {
		return nil, fmt.Errorf("requests.MakePOSTRequest: %w", err)
	}
	return req, err
}

func RefreshSecureTokenRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, data *RefreshSecureTokenRequestBody) (*http.Request, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	req, err := requests.MakePOSTRequest(ctx, EndpointRefreshSecureToken,
		requests.NewURLParamOption(url.Values{"key": {appData.GoogleAPIKey}}),
		requests.NewPOSTRawOption(bytes.NewBuffer(body), HeaderValueMIMEJSON, int64(len(body))),
		requests.NewHeaderOption(DefaultHeadersFirebase(device, appData, false, false, true)),
	)
	if err != nil {
		return nil, fmt.Errorf("requests.MakePOSTRequest: %w", err)
	}
	return req, err
}

func AuthRequest(ctx context.Context, device *andutils.Device, appData *FirebaseAppData, data url.Values, email, masterToken string) (*http.Request, error) {
	data["Email"] = []string{email}
	data["Token"] = []string{masterToken}

	data["androidId"] = []string{device.Id.ToHexString()}
	data["lang"] = []string{device.Locale.ToLocale("-", true)}
	data["device_country"] = []string{device.Locale.GetCountry(false)}
	data["sdk_version"] = []string{device.Version.ToAndroidSDK()}

	req, err := requests.MakePOSTRequest(ctx, EndpointRefreshSecureToken,
		requests.NewPOSTFormOption(data),
		requests.NewHeaderOption(DefaultHeadersAuth(device)),
	)
	if err != nil {
		return nil, fmt.Errorf("requests.MakePOSTRequest: %w", err)
	}
	return req, err
}

func CheckinAndroidRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData, digest, otaCert string, accountCookies ...string) (*http.Request, error) {
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
	if err != nil {
		return nil, fmt.Errorf("reqObj.MarshalVT: %w", err)
	}

	req, err := requests.MakePOSTRequest(ctx, EndpointAndroidCheckin,
		requests.NewPOSTRawOption(bytes.NewBuffer(reqBytes), "application/x-protobuffer", int64(len(reqBytes))),
		requests.NewHeaderOption(DefaultHeadersCheckin(device.Device)),
	)
	if err != nil {
		return nil, fmt.Errorf("requests.MakePOSTRequest: %w", err)
	}
	return req, err
}

func C2DMAndroidRegisterRequest(ctx context.Context, device *FirebaseDevice, appData *FirebaseAppData) (*http.Request, error) {
	installationData, ok := device.FirebaseInstallations[appData.PackageID]
	if !ok {
		return nil, errors.New("no installation available")
	}

	reqBody := url.Values{
		"sender":                             {appData.NotificationSenderID},
		"X-subtype":                          {appData.NotificationSenderID},
		"X-app_ver":                          {appData.AppVersionWithBuild},
		"X-osv":                              {device.Device.Version.ToAndroidSDK()},
		"X-cliv":                             {device.FirebaseClientVersion}, // {"fcm-22.0.0"},
		"X-gmsv":                             {device.GmsVersion},            // {"214815028"},
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
		"gcm_ver":                            {device.GmsVersion}, //{"214815028"},
		"plat":                               {"0"},
		"cert":                               {strings.ToLower(appData.PackageCertificate)},
		"target_ver":                         {device.Device.Version.ToAndroidSDK()}, // {"30"}
	}

	req, err := requests.MakePOSTRequest(ctx, EndpointAndroidRegister,
		requests.NewPOSTFormOption(reqBody),
		requests.NewHeaderOption(DefaultHeadersAndroidRegister(device)),
	)
	if err != nil {
		return nil, fmt.Errorf("requests.MakePOSTRequest: %w", err)
	}
	return req, err
}
