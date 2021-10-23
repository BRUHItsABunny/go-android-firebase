package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	. "github.com/BRUHItsABunny/go-android-firebase/constants"
	"net/http"
	"net/url"
)

func NotifyInstallationRequest(ctx context.Context, device *FirebaseDevice, data *NotifyInstallationRequestBody) (*http.Request, error) {
	var (
		body []byte
		req *http.Request
		err error
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
		req *http.Request
		err error
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

func RefreshSecureTokenRequest(ctx context.Context, device *FirebaseDevice, data *RefreshSecureTokenRequestBody) (*http.Request, error) {
	var (
		body []byte
		req *http.Request
		err error
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
