package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
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

func SignUpNewUserResult(resp *http.Response) (*GoogleSignUpNewUserResponse, error) {
	result := new(GoogleSignUpNewUserResponse)
	responseBody, err := io.ReadAll(resp.Body)
	if err == nil {
		err = json.Unmarshal(responseBody, result)
	}
	return result, err
}

func SetAccountInfoResult(resp *http.Response) (*GoogleSetAccountInfoResponse, error) {
	result := new(GoogleSetAccountInfoResponse)
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

func AuthResult(resp *http.Response) (*AuthResponse, error) {
	result := new(AuthResponse)
	responseBody, err := io.ReadAll(resp.Body)
	if err == nil {
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
	}
	return result, err
}

func CheckinResult(resp *http.Response) (*CheckinResponse, error) {
	result := new(CheckinResponse)
	responseBody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err == nil {
		err = result.UnmarshalVT(responseBody)
	}
	return result, err
}

func AndroidRegisterResult(resp *http.Response) (string, error) {
	result := ""
	responseBody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err == nil {
		result = string(responseBody[6:])
	}
	return result, err
}
