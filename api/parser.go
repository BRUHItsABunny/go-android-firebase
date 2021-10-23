package api

import (
	"encoding/json"
	"io"
	"net/http"
)

func NotifyInstallationResult(resp *http.Response) (*FireBaseInstallationResponse, error) {
	result := new(FireBaseInstallationResponse)
	responseBody, err := io.ReadAll(resp.Body)
	if err == nil {
		err = json.Unmarshal(responseBody, result)
	}
	return result, err
}

func VerifyPasswordResult(resp *http.Response) (*GoogleVerifyPasswordResponse, error) {
	result := new(GoogleVerifyPasswordResponse)
	responseBody, err := io.ReadAll(resp.Body)
	if err == nil {
		err = json.Unmarshal(responseBody, result)
	}
	return result, err
}

func RefreshSecureTokenResult(resp *http.Response) (*SecureTokenRefreshResponse, error) {
	result := new(SecureTokenRefreshResponse)
	responseBody, err := io.ReadAll(resp.Body)
	if err == nil {
		err = json.Unmarshal(responseBody, result)
	}
	return result, err
}
