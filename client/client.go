package firebase_client

import (
	"context"
	"fmt"
	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type FireBaseClient struct {
	Client *http.Client
	Device *firebase_api.FirebaseDevice
	MTalk  *MTalkCon
}

func NewFirebaseClient(client *http.Client, device *firebase_api.FirebaseDevice) (*FireBaseClient, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if device == nil {
		device = &firebase_api.FirebaseDevice{}
	}
	if device.Device == nil {
		device.Device = andutils.GetRandomDevice()
	}
	mTalk, err := NewMTalkCon(device)
	if err != nil {
		err = fmt.Errorf("NewMTalkCon: %w", err)
		return nil, err
	}

	return &FireBaseClient{
		Client: client,
		Device: device,
		MTalk:  mTalk,
	}, nil
}

func (c *FireBaseClient) NotifyInstallation(ctx context.Context, appData *firebase_api.FirebaseAppData) (*firebase_api.FireBaseInstallationResponse, error) {
	req, err := firebase_api.NotifyInstallationRequest(ctx, c.Device, appData)
	if err != nil {
		return nil, fmt.Errorf("api.NotifyInstallationRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := firebase_api.NotifyInstallationResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.NotifyInstallationResult: %w", err)
	}

	if c.Device.FirebaseInstallations == nil {
		c.Device.FirebaseInstallations = map[string]*firebase_api.FirebaseInstallationData{}
	}
	installation, ok := c.Device.FirebaseInstallations[appData.PackageID]
	if !ok {
		installation = &firebase_api.FirebaseInstallationData{NotificationData: &firebase_api.FirebaseNotificationData{}}
	}

	expiresIn, _ := strconv.Atoi(result.AuthToken.Expiration[:len(result.AuthToken.Expiration)-1])
	installation.InstallationAuthentication = &firebase_api.FirebaseAuthentication{
		AccessToken:  result.AuthToken.Token,
		Expires:      timestamppb.New(time.Now().Add(time.Duration(expiresIn) * time.Second)),
		RefreshToken: result.RefreshToken,
		IdToken:      "",
	}
	installation.FirebaseInstallationID = result.FID // FID in should equal FID out, but not always the case, override with FID out to be sure
	c.Device.FirebaseInstallations[appData.PackageID] = installation
	return result, err
}

func (c *FireBaseClient) VerifyPassword(ctx context.Context, data *firebase_api.VerifyPasswordRequestBody, appData *firebase_api.FirebaseAppData) (*firebase_api.GoogleVerifyPasswordResponse, error) {
	req, err := firebase_api.VerifyPasswordRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.VerifyPasswordRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := firebase_api.VerifyPasswordResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.VerifyPasswordResult: %w", err)
	}
	return result, err
}

func (c *FireBaseClient) SetAccountInfo(ctx context.Context, appData *firebase_api.FirebaseAppData, data *firebase_api.SetAccountInfoRequestBody) (*firebase_api.GoogleSetAccountInfoResponse, error) {
	req, err := firebase_api.SetAccountInfoRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.SetAccountInfoRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := firebase_api.SetAccountInfoResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.SetAccountInfoResult: %w", err)
	}
	return result, err
}

func (c *FireBaseClient) SignUpNewUser(ctx context.Context, appData *firebase_api.FirebaseAppData, data *firebase_api.SignUpNewUserRequestBody) (*firebase_api.GoogleSignUpNewUserResponse, error) {
	var (
		result = new(firebase_api.GoogleSignUpNewUserResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = firebase_api.SignUpNewUser(ctx, c.Device, appData, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = firebase_api.SignUpNewUserResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) RefreshSecureToken(ctx context.Context, appData *firebase_api.FirebaseAppData, data *firebase_api.RefreshSecureTokenRequestBody) (*firebase_api.SecureTokenRefreshResponse, error) {
	req, err := firebase_api.RefreshSecureTokenRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.RefreshSecureTokenRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := firebase_api.RefreshSecureTokenResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.RefreshSecureTokenResult: %w", err)
	}
	return result, err
}

func (c *FireBaseClient) Auth(ctx context.Context, appData *firebase_api.FirebaseAppData, data url.Values, email, token string) (*firebase_api.AuthResponse, error) {

	if email != "" {
		data["Email"] = []string{email}

	}
	if token != "" {
		data["Token"] = []string{token}
	}

	req, err := firebase_api.AuthRequest(ctx, c.Device.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.AuthRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := firebase_api.AuthResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.AuthResult: %w", err)
	}
	return result, err
}

func (c *FireBaseClient) Checkin(ctx context.Context, appData *firebase_api.FirebaseAppData, digest, otaCert string, accountCookies ...string) (*firebase_api.CheckinResponse, error) {
	req, err := firebase_api.CheckinAndroidRequest(ctx, c.Device, appData, digest, otaCert, accountCookies...)
	if err != nil {
		return nil, fmt.Errorf("api.CheckinAndroidRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := firebase_api.CheckinResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.CheckinResult: %w", err)
	}
	c.Device.CheckinAndroidID = int64(*result.AndroidId)
	c.Device.CheckinSecurityToken = *result.SecurityToken
	return result, err
}

func (c *FireBaseClient) C2DMRegisterAndroid(ctx context.Context, appData *firebase_api.FirebaseAppData) (string, error) {
	req, err := firebase_api.C2DMAndroidRegisterRequest(ctx, c.Device, appData)
	if err != nil {
		return "", fmt.Errorf("api.C2DMAndroidRegisterRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := firebase_api.AndroidRegisterResult(resp)
	if err != nil {
		return "", fmt.Errorf("api.AndroidRegisterResult: %w", err)
	}
	if c.Device.FirebaseInstallations == nil {
		c.Device.FirebaseInstallations = map[string]*firebase_api.FirebaseInstallationData{}
	}
	installation, ok := c.Device.FirebaseInstallations[appData.PackageID]
	if !ok {
		installation = &firebase_api.FirebaseInstallationData{NotificationData: &firebase_api.FirebaseNotificationData{}}
	}
	installation.NotificationData.NotificationToken = result
	c.Device.FirebaseInstallations[appData.PackageID] = installation
	return result, err
}

func (c *FireBaseClient) C2DMRegisterWeb(ctx context.Context, appData *firebase_api.FirebaseAppData, sender, subtype, appid string) (string, error) {
	req, err := firebase_api.C2DMWebRegisterRequest(ctx, c.Device, appData, sender, subtype, appid)
	if err != nil {
		return "", fmt.Errorf("api.C2DMWebRegisterRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := firebase_api.AndroidRegisterResult(resp)
	if err != nil {
		return "", fmt.Errorf("api.AndroidRegisterResult: %w", err)
	}
	// TODO: Store subscription (notification token) somewhere?
	return result, err
}
