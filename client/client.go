package client

import (
	"context"
	"github.com/BRUHItsABunny/go-android-firebase/api"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type FireBaseClient struct {
	Client  *http.Client
	Device  *api.FirebaseDevice
	AppData *api.FirebaseAppData
}

func NewFirebaseClient(client *http.Client, device *api.FirebaseDevice, appData *api.FirebaseAppData) *FireBaseClient {
	if client == nil {
		client = http.DefaultClient
	}
	if device == nil {
		device = &api.FirebaseDevice{}
	}
	if device.Device == nil {
		device.Device = andutils.GetRandomDevice()
	}
	if appData == nil {
		appData = &api.FirebaseAppData{}
	}

	return &FireBaseClient{
		Client:  client,
		Device:  device,
		AppData: appData,
	}
}

func (c *FireBaseClient) NotifyInstallation(ctx context.Context) (*api.FireBaseInstallationResponse, error) {
	var (
		result = new(api.FireBaseInstallationResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.NotifyInstallationRequest(ctx, c.Device, c.AppData)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.NotifyInstallationResult(resp)
			if err == nil {
				if c.Device.FirebaseInstallations == nil {
					c.Device.FirebaseInstallations = map[string]*api.FirebaseInstallationData{}
				}
				installation, ok := c.Device.FirebaseInstallations[c.AppData.PackageID]
				if !ok {
					installation = &api.FirebaseInstallationData{}
				}

				expiresIn, _ := strconv.Atoi(result.AuthToken.Expiration[:len(result.AuthToken.Expiration)-1])
				installation.InstallationAuthentication = &api.FirebaseAuthentication{
					AccessToken:  result.AuthToken.Token,
					Expires:      timestamppb.New(time.Now().Add(time.Duration(expiresIn) * time.Second)),
					RefreshToken: result.RefreshToken,
					IdToken:      "",
				}
				installation.FirebaseInstallationID = result.FID // FID in should equal FID out, but not always the case, override with FID out to be sure
				c.Device.FirebaseInstallations[c.AppData.PackageID] = installation
			}
		}
	}
	return result, err
}

func (c *FireBaseClient) VerifyPassword(ctx context.Context, data *api.VerifyPasswordRequestBody) (*api.GoogleVerifyPasswordResponse, error) {
	var (
		result = new(api.GoogleVerifyPasswordResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.VerifyPasswordRequest(ctx, c.Device, c.AppData, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.VerifyPasswordResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) SetAccountInfo(ctx context.Context, data *api.SetAccountInfoRequestBody) (*api.GoogleSetAccountInfoResponse, error) {
	var (
		result = new(api.GoogleSetAccountInfoResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.SetAccountInto(ctx, c.Device, c.AppData, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.SetAccountInfoResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) SignUpNewUser(ctx context.Context, data *api.SignUpNewUserRequestBody) (*api.GoogleSignUpNewUserResponse, error) {
	var (
		result = new(api.GoogleSignUpNewUserResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.SignUpNewUser(ctx, c.Device, c.AppData, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.SignUpNewUserResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) RefreshSecureToken(ctx context.Context, data *api.RefreshSecureTokenRequestBody) (*api.SecureTokenRefreshResponse, error) {
	var (
		result = new(api.SecureTokenRefreshResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.RefreshSecureTokenRequest(ctx, c.Device, c.AppData, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.RefreshSecureTokenResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) Auth(ctx context.Context, data url.Values, email, masterToken string) (*api.AuthResponse, error) {
	var (
		result = new(api.AuthResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.Auth(ctx, c.Device.Device, c.AppData, data, email, masterToken)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.AuthResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) Checkin(ctx context.Context, digest, otaCert string, accountCookies ...string) (*api.CheckinResponse, error) {
	var (
		result = new(api.CheckinResponse)
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.CheckinAndroidRequest(ctx, c.Device, c.AppData, digest, otaCert, accountCookies...)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.CheckinResult(resp)
			if err == nil {
				c.Device.CheckinAndroidID = int64(*result.AndroidId)
				c.Device.CheckinSecurityToken = *result.SecurityToken
			}
		}
	}
	return result, err
}

func (c *FireBaseClient) C2DMRegisterAndroid(ctx context.Context) (string, error) {
	var (
		result = ""
		req    *http.Request
		resp   *http.Response
		err    error
	)

	req, err = api.C2DMAndroidRegisterRequest(ctx, c.Device, c.AppData)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.AndroidRegisterResult(resp)
			if err == nil {
				if c.Device.FirebaseInstallations == nil {
					c.Device.FirebaseInstallations = map[string]*api.FirebaseInstallationData{}
				}
				installation, ok := c.Device.FirebaseInstallations[c.AppData.PackageID]
				if !ok {
					installation = &api.FirebaseInstallationData{}
				}
				installation.NotificationToken = result
				c.Device.FirebaseInstallations[c.AppData.PackageID] = installation
			}
		}
	}
	return result, err
}
