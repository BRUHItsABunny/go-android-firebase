package client

import (
	"context"
	"github.com/BRUHItsABunny/go-android-firebase/api"
	andutilsdb "github.com/BRUHItsABunny/go-android-utils/database"
	"net/http"
)

type FireBaseClient struct {
	Client             *http.Client
	Device             *api.FirebaseDevice
}

func NewFirebaseClient(client *http.Client, device *api.FirebaseDevice) *FireBaseClient {
	if client == nil {
		client = http.DefaultClient
	}
	if device == nil {
		device = &api.FirebaseDevice{Device: andutilsdb.GetRandomDevice()}
	}

	return &FireBaseClient{
		Client: client,
		Device: device,
	}
}

func (c *FireBaseClient) NotifyInstallation(ctx context.Context, data *api.NotifyInstallationRequestBody) (*api.FireBaseInstallationResponse, error) {
	var (
		result = new(api.FireBaseInstallationResponse)
		req *http.Request
		resp *http.Response
		err error
	)

	req, err = api.NotifyInstallationRequest(ctx, c.Device, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.NotifyInstallationResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) VerifyPassword(ctx context.Context, data *api.VerifyPasswordRequestBody) (*api.GoogleVerifyPasswordResponse, error) {
	var (
		result = new(api.GoogleVerifyPasswordResponse)
		req *http.Request
		resp *http.Response
		err error
	)

	req, err = api.VerifyPasswordRequest(ctx, c.Device, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.VerifyPasswordResult(resp)
		}
	}
	return result, err
}

func (c *FireBaseClient) RefreshSecureToken(ctx context.Context, data *api.RefreshSecureTokenRequestBody) (*api.SecureTokenRefreshResponse, error) {
	var (
		result = new(api.SecureTokenRefreshResponse)
		req *http.Request
		resp *http.Response
		err error
	)

	req, err = api.RefreshSecureTokenRequest(ctx, c.Device, data)
	if err == nil {
		resp, err = c.Client.Do(req)
		if err == nil {
			result, err = api.RefreshSecureTokenResult(resp)
		}
	}
	return result, err
}
