package api

import (
	"encoding/json"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"strconv"
	"time"
)

type NotifyInstallationRequestBody struct {
	FID         string `json:"fid"`
	AppID       string `json:"appId"`
	AuthVersion string `json:"authVersion"`
	SDKVersion  string `json:"sdkVersion"`
}

type FireBaseInstallationResponse struct {
	Name         string            `json:"name"`
	FID          string            `json:"fid"`
	RefreshToken string            `json:"refreshToken"`
	AuthToken    FireBaseAuthToken `json:"authToken"`
}

type FireBaseAuthToken struct {
	Token      string `json:"token"`
	Expiration string `json:"expiresin"`
}

type NotifyInstallationResponse struct {
	FID         string `json:"fid"`
	AppID       string `json:"appId"`
	AuthVersion string `json:"authVersion"`
	SDKVersion  string `json:"sdkVersion"`
}

type FirebaseDevice struct {
	Device *andutils.Device `json:"device"`
	// Other Firebase related constants...
	AndroidPackage string `json:"android_package"`
	AndroidCert string `json:"android_certificate"`
	GoogleAPIKey string `json:"google_api_key"`
	ProjectID string `json:"project_id"`
}

type FirebaseAuthentication struct {
	AccessToken string
	Expires time.Time
	RefreshToken string
	IDToken string
}

type auxFirebaseAuthentication struct {
	AccessToken string `json:"access_token"`
	ExpiresIn int64 `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken string `json:"id_token"`
}

func (fa *FirebaseAuthentication) MarshalJSON() ([]byte, error) {
	return json.Marshal(&auxFirebaseAuthentication{
		AccessToken:  fa.AccessToken,
		ExpiresIn:    fa.Expires.Unix(),
		RefreshToken: fa.RefreshToken,
		IDToken:      fa.IDToken,
	})
}

func (fa *FirebaseAuthentication) UnmarshalJSON(data []byte) error {
	aux := new(auxFirebaseAuthentication)
	err := json.Unmarshal(data, aux)
	if err == nil {
		fa.IDToken = aux.IDToken
		fa.RefreshToken = aux.RefreshToken
		if aux.ExpiresIn > 3600 { // If we store it
			fa.Expires = time.Unix(aux.ExpiresIn, 0)
		} else { // If we receive it from firebase
			fa.Expires = time.Now().Add(time.Duration(aux.ExpiresIn) * time.Second)
		}
	}
	return err
}

type SecureTokenRefreshResponse struct {
	AccessToken string
	Expires time.Time
	RefreshToken string
	IDToken string
	TokenType string
	UserID string
	ProjectID string
}

type auxSecureTokenRefreshResponse struct {
	AccessToken string
	ExpiresIn int64
	RefreshToken string
	IDToken string
	TokenType string
	UserID string
	ProjectID string
}

func (str *SecureTokenRefreshResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&auxSecureTokenRefreshResponse{
		AccessToken:  str.AccessToken,
		ExpiresIn:      str.Expires.Unix(),
		RefreshToken: str.RefreshToken,
		IDToken:      str.IDToken,
		TokenType:    str.TokenType,
		UserID:       str.UserID,
		ProjectID:    str.ProjectID,
	})
}

func (str *SecureTokenRefreshResponse) UnmarshalJSON(data []byte) error {
	aux := new(auxSecureTokenRefreshResponse)
	err := json.Unmarshal(data, aux)
	if err == nil {
		str.UserID = aux.UserID
		str.ProjectID = aux.ProjectID
		str.TokenType = aux.TokenType
		str.AccessToken = aux.AccessToken
		str.IDToken = aux.IDToken
		str.RefreshToken = aux.RefreshToken
		if aux.ExpiresIn > 3600 { // If we store it
			str.Expires = time.Unix(aux.ExpiresIn, 0)
		} else { // If we receive it from firebase
			str.Expires = time.Now().Add(time.Duration(aux.ExpiresIn) * time.Second)
		}
	}
	return err
}

type GoogleVerifyPasswordResponse struct {
	Expires time.Time
	RefreshToken string
	IDToken string
	Kind string
	Email string
	LocalID string
	DisplayName string
	Registered bool
}

type auxGoogleVerifyPasswordResponse struct {
	ExpiresIn string
	RefreshToken string
	IDToken string
	Kind string
	Email string
	LocalID string
	DisplayName string
	Registered bool
}

func (gvp *GoogleVerifyPasswordResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&auxGoogleVerifyPasswordResponse{
		ExpiresIn:      strconv.FormatInt(gvp.Expires.Unix(), 10),
		RefreshToken: gvp.RefreshToken,
		IDToken:      gvp.IDToken,
		Kind:         gvp.Kind,
		Email:        gvp.Email,
		LocalID:      gvp.LocalID,
		DisplayName:  gvp.DisplayName,
		Registered:   gvp.Registered,
	})
}

func (gvp *GoogleVerifyPasswordResponse) UnmarshalJSON(data []byte) error {
	var tmpExpiresIn int64
	aux := new(auxGoogleVerifyPasswordResponse)
	err := json.Unmarshal(data, aux)
	if err == nil {
		gvp.RefreshToken = aux.RefreshToken
		gvp.IDToken = aux.IDToken
		gvp.Kind = aux.Kind
		gvp.Email = aux.Email
		gvp.LocalID = aux.LocalID
		gvp.DisplayName = aux.DisplayName
		gvp.Registered = aux.Registered
		tmpExpiresIn, err = strconv.ParseInt(aux.ExpiresIn, 10, 64)
		if err == nil {
			if tmpExpiresIn > 3600 { // If we store it
				gvp.Expires = time.Unix(tmpExpiresIn, 0)
			} else { // If we receive it from firebase
				gvp.Expires = time.Now().Add(time.Duration(tmpExpiresIn) * time.Second)
			}
		}
	}
	return err
}

type VerifyPasswordRequestBody struct {
	Email string `json:"email"`
	Password string `json:"password"`
	ReturnSecureToken bool `json:"returnSecureToken"`
}

type RefreshSecureTokenRequestBody struct {
	GrantType string `json:"grantType"`
	RefreshToken string `json:"refreshToken"`
}
