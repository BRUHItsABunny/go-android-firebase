package go_android_firebase

import (
	"fmt"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	"github.com/BRUHItsABunny/go-android-firebase/api"
	go_android_utils "github.com/BRUHItsABunny/go-android-utils"
)

type FireBaseClient struct {
	Client             *gokhttp.HttpClient
	Device             *go_android_utils.Device
	AndroidCertificate string
	AndroidPackage     string
	ProjectID          string
	APIKey             string
}

func NewFireBaseClient(options *gokhttp.HttpClientOptions, device *go_android_utils.Device, andCert, andPackage, projectId, apiKey string) *FireBaseClient {

	httpClient := gokhttp.GetHTTPClient(options)

	return &FireBaseClient{
		Client:             &httpClient,
		Device:             device,
		AndroidCertificate: andCert,
		AndroidPackage:     andPackage,
		ProjectID:          projectId,
		APIKey:             apiKey,
	}
}

func (c *FireBaseClient) NotifyInstallation(data *api.NotifyInstallationRequestBody) (*api.FireBaseInstallationResponse, error) {
	result := new(api.FireBaseInstallationResponse)
	userAgent := HeaderValueUserAgentPrefix + c.Device.GetUserAgent()
	fireBaseClient := c.Device.FormatUserAgent(HeaderValueFireBaseClient)

	req := api.NotifyInstallationRequest(data, &api.DefaultHeaders, c.ProjectID, c.AndroidPackage, c.AndroidCertificate, c.APIKey, fireBaseClient, "3", userAgent)
	resp, err := c.Client.Do(req)
	if err == nil {
		err = resp.Object(result)
		if err != nil {
			err = fmt.Errorf("resp.Object(result): %w", err)
		}
	} else {
		err = fmt.Errorf("c.Client.Do(req): %w", err)
	}
	return result, err
}
