package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	. "github.com/BRUHItsABunny/go-android-firebase/constants"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"net/http"
	"net/url"
)

func NotifyInstallationRequest(ctx context.Context, device *FirebaseDevice, data *NotifyInstallationRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", fmt.Sprintf(EndpointInstallations, device.ProjectID), bytes.NewBuffer(body))
		if err == nil {
			req.Header = DefaultHeadersFirebase(device, true, true, false)
		}
	}

	return req, err
}

func VerifyPasswordRequest(ctx context.Context, device *FirebaseDevice, data *VerifyPasswordRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointVerifyPassword, bytes.NewBuffer(body))
		if err == nil {
			req.URL.RawQuery = url.Values{"key": {device.GoogleAPIKey}}.Encode()
			req.Header = DefaultHeadersFirebase(device, false, false, true)
		}
	}

	return req, err
}

func SignUpNewUser(ctx context.Context, device *FirebaseDevice, data *SignUpNewUserRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointSignUpNewUser, bytes.NewBuffer(body))
		if err == nil {
			req.URL.RawQuery = url.Values{"key": {device.GoogleAPIKey}}.Encode()
			req.Header = DefaultHeadersFirebase(device, false, false, true)
		}
	}

	return req, err
}

func SetAccountInto(ctx context.Context, device *FirebaseDevice, data *SetAccountInfoRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointSetAccountInto, bytes.NewBuffer(body))
		if err == nil {
			req.URL.RawQuery = url.Values{"key": {device.GoogleAPIKey}}.Encode()
			req.Header = DefaultHeadersFirebase(device, false, false, true)
		}
	}

	return req, err
}

func RefreshSecureTokenRequest(ctx context.Context, device *FirebaseDevice, data *RefreshSecureTokenRequestBody) (*http.Request, error) {
	var (
		body []byte
		req  *http.Request
		err  error
	)

	body, err = json.Marshal(data)
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", EndpointRefreshSecureToken, bytes.NewBuffer(body))
		if err == nil {
			req.URL.RawQuery = url.Values{"key": {device.GoogleAPIKey}}.Encode()
			req.Header = DefaultHeadersFirebase(device, false, false, true)
		}
	}

	return req, err
}

func Auth(ctx context.Context, device *andutils.Device, data url.Values, email, masterToken string) (*http.Request, error) {
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
