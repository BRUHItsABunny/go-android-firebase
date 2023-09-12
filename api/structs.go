package firebase_api

import (
	"encoding/json"
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

/*
type FirebaseDevice struct {
	Device *andutils.Device `json:"device"`
	// Other Firebase related constants...
	AndroidPackage string `json:"android_package"`
	AndroidCert    string `json:"android_certificate"`
	GoogleAPIKey   string `json:"google_api_key"`
	ProjectID      string `json:"project_id"`
	GMPAppID       string `json:"gmp_app_id"`
	// Also app specific constants but might be harder to find
	AppNameHash         string `json:"app_name_hash"`
	NotificationSender  string `json:"notification_sender"`
	AppVersion          string `json:"app_version"`
	AppVersionWithBuild string `json:"app_version_with_build"`
	// Firebase Installation persistent variables
	FirebaseInstallationID   string `json:"firebase_installation_id"`
	FirebaseInstallationAuth *FireBaseAuthToken `json:"firebase_installation_auth"`
	// Checkin related persistent variables
	CheckinAndroidID     int64  `json:"checkin_android_id"`
	CheckinSecurityToken uint64 `json:"checkin_security_token"`
}

type FirebaseAuthentication struct {
	AccessToken  string
	Expires      time.Time
	RefreshToken string
	IDToken      string
}

type auxFirebaseAuthentication struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
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
*/

type SecureTokenRefreshResponse struct {
	AccessToken  string
	Expires      time.Time
	RefreshToken string
	IDToken      string
	TokenType    string
	UserID       string
	ProjectID    string
}

type AuthResponse struct {
	Token    string
	Expires  time.Time
	Metadata string
	Scopes   []string
}

type auxSecureTokenRefreshResponse struct {
	AccessToken  string
	ExpiresIn    int64
	RefreshToken string
	IDToken      string
	TokenType    string
	UserID       string
	ProjectID    string
}

func (str *SecureTokenRefreshResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&auxSecureTokenRefreshResponse{
		AccessToken:  str.AccessToken,
		ExpiresIn:    str.Expires.Unix(),
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

type ProviderUserInfo struct {
	ProviderID  string `json:"providerId"`
	DisplayName string `json:"displayName"`
	FederateID  string `json:"federateId"`
	Email       string `json:"email,omitempty"`
	RawID       string `json:"rawId"`
}

type GoogleSetAccountInfoResponse struct {
	Kind             string              `json:"kind"`
	Email            string              `json:"email"`
	LocalID          string              `json:"localId"`
	DisplayName      string              `json:"displayName"`
	ProviderUserInfo []*ProviderUserInfo `json:"providerUserInfo"`
	PasswordHash     string              `json:"passwordHash"`
	EmailVerified    bool                `json:"emailVerified"`
}

type GoogleVerifyPasswordResponse struct {
	Expires      time.Time
	RefreshToken string
	IDToken      string
	Kind         string
	Email        string
	LocalID      string
	DisplayName  string
	Registered   bool
}

type auxGoogleVerifyPasswordResponse struct {
	ExpiresIn    string `json:"expiresIn"`
	RefreshToken string `json:"refreshToken"`
	IDToken      string `json:"idToken"`
	Kind         string `json:"kind"`
	Email        string `json:"email"`
	LocalID      string `json:"localId"`
	DisplayName  string `json:"displayName"`
	Registered   bool   `json:"registered"`
}

func (gvp *GoogleVerifyPasswordResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&auxGoogleVerifyPasswordResponse{
		ExpiresIn:    strconv.FormatInt(gvp.Expires.Unix(), 10),
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

type GoogleSignUpNewUserResponse struct {
	Expires      time.Time
	RefreshToken string
	IDToken      string
	Kind         string
	Email        string
	LocalID      string
}

type auxGoogleSignUpNewUserResponse struct {
	ExpiresIn    string
	RefreshToken string
	IDToken      string
	Kind         string
	Email        string
	LocalID      string
}

func (gsp *GoogleSignUpNewUserResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&auxGoogleSignUpNewUserResponse{
		ExpiresIn:    strconv.FormatInt(gsp.Expires.Unix(), 10),
		RefreshToken: gsp.RefreshToken,
		IDToken:      gsp.IDToken,
		Kind:         gsp.Kind,
		Email:        gsp.Email,
		LocalID:      gsp.LocalID,
	})
}

func (gsp *GoogleSignUpNewUserResponse) UnmarshalJSON(data []byte) error {
	var tmpExpiresIn int64
	aux := new(auxGoogleSignUpNewUserResponse)
	err := json.Unmarshal(data, aux)
	if err == nil {
		gsp.RefreshToken = aux.RefreshToken
		gsp.IDToken = aux.IDToken
		gsp.Kind = aux.Kind
		gsp.Email = aux.Email
		gsp.LocalID = aux.LocalID
		tmpExpiresIn, err = strconv.ParseInt(aux.ExpiresIn, 10, 64)
		if err == nil {
			if tmpExpiresIn > 3600 { // If we store it
				gsp.Expires = time.Unix(tmpExpiresIn, 0)
			} else { // If we receive it from firebase
				gsp.Expires = time.Now().Add(time.Duration(tmpExpiresIn) * time.Second)
			}
		}
	}
	return err
}

type FirebaseRequest struct {
	ReturnSecureToken bool   `json:"returnSecureToken"`
	AuthToken         string `json:"idToken,omitempty"`
}

type SetAccountInfoRequestBody struct {
	FirebaseRequest
	Data map[string]interface{}
}

func (sai *SetAccountInfoRequestBody) MarshalJSON() ([]byte, error) {
	targetMap := make(map[string]interface{})
	for k, v := range sai.Data {
		targetMap[k] = v
	}
	if len(sai.AuthToken) > 0 {
		targetMap["idToken"] = sai.AuthToken
	}
	targetMap["returnSecureToken"] = sai.ReturnSecureToken

	return json.Marshal(&targetMap)
}

func (sai *SetAccountInfoRequestBody) UnmarshalJSON(data []byte) error {
	targetMap := make(map[string]interface{})
	err := json.Unmarshal(data, &targetMap)
	if err == nil {
		returnSecureToken, ok := targetMap["returnSecureToken"]
		if ok {
			switch value := returnSecureToken.(type) {
			case bool:
				sai.ReturnSecureToken = value
				break
			}
		}

		idToken, ok := targetMap["idToken"]
		if ok {
			switch value := idToken.(type) {
			case string:
				sai.AuthToken = value
				break
			}
		}
		delete(targetMap, "idToken")
		delete(targetMap, "returnSecureToken")
		sai.Data = targetMap
	}
	return err
}

type VerifyPasswordRequestBody struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type SignUpNewUserRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshSecureTokenRequestBody struct {
	GrantType    string `json:"grantType"`
	RefreshToken string `json:"refreshToken"`
}
