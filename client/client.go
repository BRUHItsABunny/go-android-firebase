package client

import (
	"context"
	"fmt"
	"github.com/BRUHItsABunny/go-android-firebase/api"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type FireBaseClient struct {
	Client *http.Client
	Device *api.FirebaseDevice
	MTalk  *MTalkCon
}

func NewFirebaseClient(client *http.Client, device *api.FirebaseDevice) *FireBaseClient {
	if client == nil {
		client = http.DefaultClient
	}
	if device == nil {
		device = &api.FirebaseDevice{}
	}
	if device.Device == nil {
		device.Device = andutils.GetRandomDevice()
	}

	return &FireBaseClient{
		Client: client,
		Device: device,
		MTalk:  NewMTalkCon(device),
	}
}

func (c *FireBaseClient) NotifyInstallation(ctx context.Context, appData *api.FirebaseAppData) (*api.FireBaseInstallationResponse, error) {
	req, err := api.NotifyInstallationRequest(ctx, c.Device, appData)
	if err != nil {
		return nil, fmt.Errorf("api.NotifyInstallationRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := api.NotifyInstallationResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.NotifyInstallationResult: %w", err)
	}

	if c.Device.FirebaseInstallations == nil {
		c.Device.FirebaseInstallations = map[string]*api.FirebaseInstallationData{}
	}
	installation, ok := c.Device.FirebaseInstallations[appData.PackageID]
	if !ok {
		installation = &api.FirebaseInstallationData{NotificationData: &api.FirebaseNotificationData{}}
	}

	expiresIn, _ := strconv.Atoi(result.AuthToken.Expiration[:len(result.AuthToken.Expiration)-1])
	installation.InstallationAuthentication = &api.FirebaseAuthentication{
		AccessToken:  result.AuthToken.Token,
		Expires:      timestamppb.New(time.Now().Add(time.Duration(expiresIn) * time.Second)),
		RefreshToken: result.RefreshToken,
		IdToken:      "",
	}
	installation.FirebaseInstallationID = result.FID // FID in should equal FID out, but not always the case, override with FID out to be sure
	c.Device.FirebaseInstallations[appData.PackageID] = installation
	return result, err
}

func (c *FireBaseClient) VerifyPassword(ctx context.Context, data *api.VerifyPasswordRequestBody, appData *api.FirebaseAppData) (*api.GoogleVerifyPasswordResponse, error) {
	req, err := api.VerifyPasswordRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.VerifyPasswordRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := api.VerifyPasswordResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.VerifyPasswordResult: %w", err)
	}
	return result, err
}

func (c *FireBaseClient) SetAccountInfo(ctx context.Context, appData *api.FirebaseAppData, data *api.SetAccountInfoRequestBody) (*api.GoogleSetAccountInfoResponse, error) {
	req, err := api.SetAccountInfoRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.SetAccountInfoRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := api.SetAccountInfoResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.SetAccountInfoResult: %w", err)
	}
	return result, err
}

func (c *FireBaseClient) SignUpNewUser(ctx context.Context, appData *api.FirebaseAppData, data *api.SignUpNewUserRequestBody) (*api.GoogleSignUpNewUserResponse, error) {
	var (
		result = new(api.GoogleSignUpNewUserResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.SignUpNewUser(ctx, c.Device, appData, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.SignUpNewUserResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) RefreshSecureToken(ctx context.Context, appData *api.FirebaseAppData, data *api.RefreshSecureTokenRequestBody) (*api.SecureTokenRefreshResponse, error) {
	req, err := api.RefreshSecureTokenRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.RefreshSecureTokenRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := api.RefreshSecureTokenResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.RefreshSecureTokenResult: %w", err)
	}
	return result, err
}

func (c *FireBaseClient) Auth(ctx context.Context, appData *api.FirebaseAppData, data url.Values, email, masterToken string) (*api.AuthResponse, error) {
	req, err := api.AuthRequest(ctx, c.Device.Device, appData, data, email, masterToken)
	if err != nil {
		return nil, fmt.Errorf("api.AuthRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := api.AuthResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.AuthResult: %w", err)
	}
	return result, err
}

func (c *FireBaseClient) Checkin(ctx context.Context, appData *api.FirebaseAppData, digest, otaCert string, accountCookies ...string) (*api.CheckinResponse, error) {
	req, err := api.CheckinAndroidRequest(ctx, c.Device, appData, digest, otaCert, accountCookies...)
	if err != nil {
		return nil, fmt.Errorf("api.CheckinAndroidRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := api.CheckinResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.CheckinResult: %w", err)
	}
	c.Device.CheckinAndroidID = int64(*result.AndroidId)
	c.Device.CheckinSecurityToken = *result.SecurityToken
	return result, err
}

func (c *FireBaseClient) C2DMRegisterAndroid(ctx context.Context, appData *api.FirebaseAppData) (string, error) {
	req, err := api.C2DMAndroidRegisterRequest(ctx, c.Device, appData)
	if err != nil {
		return "", fmt.Errorf("api.C2DMAndroidRegisterRequest: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("c.Client.Do: %w", err)
	}

	result, err := api.AndroidRegisterResult(resp)
	if err != nil {
		return "", fmt.Errorf("api.AndroidRegisterResult: %w", err)
	}
	if c.Device.FirebaseInstallations == nil {
		c.Device.FirebaseInstallations = map[string]*api.FirebaseInstallationData{}
	}
	installation, ok := c.Device.FirebaseInstallations[appData.PackageID]
	if !ok {
		installation = &api.FirebaseInstallationData{NotificationData: &api.FirebaseNotificationData{}}
	}
	installation.NotificationData.NotificationToken = result
	c.Device.FirebaseInstallations[appData.PackageID] = installation
	return result, err
}
