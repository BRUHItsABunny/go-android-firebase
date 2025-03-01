package firebase_api

import (
	"bytes"
	"fmt"
	gokhttp_responses "github.com/BRUHItsABunny/gOkHttp/responses"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NotifyInstallationResult(resp *http.Response) (*FireBaseInstallationResponse, error) {
	result := new(FireBaseInstallationResponse)
	err := gokhttp_responses.CheckHTTPCode(resp, http.StatusOK)
	if err != nil {
		err = fmt.Errorf("gokhttp_responses.CheckHTTPCode: %w", err)
		return nil, err
	}
	err = gokhttp_responses.ResponseJSON(resp, result)
	if err != nil {
		return nil, fmt.Errorf("gokhttp_responses.ResponseJSON: %w", err)
	}
	return result, nil
}

func VerifyPasswordResult(resp *http.Response) (*GoogleVerifyPasswordResponse, error) {
	result := new(GoogleVerifyPasswordResponse)
	err := gokhttp_responses.ResponseJSON(resp, result)
	if err != nil {
		return nil, fmt.Errorf("gokhttp_responses.ResponseJSON: %w", err)
	}
	return result, nil
}

func SignUpNewUserResult(resp *http.Response) (*GoogleSignUpNewUserResponse, error) {
	result := new(GoogleSignUpNewUserResponse)
	err := gokhttp_responses.ResponseJSON(resp, result)
	if err != nil {
		return nil, fmt.Errorf("gokhttp_responses.ResponseJSON: %w", err)
	}
	return result, nil
}

func SetAccountInfoResult(resp *http.Response) (*GoogleSetAccountInfoResponse, error) {
	result := new(GoogleSetAccountInfoResponse)
	err := gokhttp_responses.ResponseJSON(resp, result)
	if err != nil {
		return nil, fmt.Errorf("gokhttp_responses.ResponseJSON: %w", err)
	}
	return result, nil
}

func RefreshSecureTokenResult(resp *http.Response) (*SecureTokenRefreshResponse, error) {
	result := new(SecureTokenRefreshResponse)
	err := gokhttp_responses.ResponseJSON(resp, result)
	if err != nil {
		return nil, fmt.Errorf("gokhttp_responses.ResponseJSON: %w", err)
	}
	return result, nil
}

func AuthResult(resp *http.Response) (*AuthResponse, error) {
	result := new(AuthResponse)
	responseBody, err := gokhttp_responses.ResponseBytes(resp)
	if err != nil {
		return nil, fmt.Errorf("gokhttp_responses.ResponseBytes: %w", err)
	}
	var timeStamp int64
	for _, entryBytes := range bytes.Split(responseBody, []byte("\n")) {
		entryParts := bytes.Split(entryBytes, []byte("="))
		switch string(entryParts[0]) {
		case "Expiry":
			timeStamp, err = strconv.ParseInt(string(entryParts[1]), 10, 64)
			result.Expires = time.Unix(timeStamp, 0)
			break
		case "grantedScopes":
			result.Scopes = strings.Split(string(entryParts[1]), " ")
			break
		case "itMetadata":
			result.Metadata = string(entryParts[1])
			break
		case "it":
			result.Token = string(entryParts[1])
			break
		case "Auth":
			result.Token = string(entryParts[1])
			break
		default:
			continue
		}
		if err != nil {
			break
		}
	}
	return result, err
}

func CheckinResult(resp *http.Response) (*CheckinResponse, error) {
	result := new(CheckinResponse)
	responseBody, err := gokhttp_responses.ResponseBytes(resp)
	if err != nil {
		return nil, fmt.Errorf("gokhttp_responses.ResponseBytes: %w", err)
	}
	err = result.UnmarshalVT(responseBody)
	if err != nil {
		return nil, fmt.Errorf("result.UnmarshalVT: %w", err)
	}
	return result, nil
}

func AndroidRegisterResult(resp *http.Response) (string, error) {
	responseBody, err := gokhttp_responses.ResponseBytes(resp)
	if err != nil {
		return "", fmt.Errorf("gokhttp_responses.ResponseBytes: %w", err)
	}
	return string(responseBody[6:]), nil
}
