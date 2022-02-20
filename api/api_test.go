package api

import (
	"context"
	"fmt"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"github.com/davecgh/go-spew/spew"
	"net/http"
	"net/url"
	"testing"
)

func testHTTPClient(proxy string) *http.Client {
	client := http.DefaultClient
	client.Transport = http.DefaultTransport
	proxyURL, _ := url.Parse(proxy)
	client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyURL)
	return client
}

func TestCheckinRequest(t *testing.T) {
	ctx := context.Background()
	device, _ := andutils.GetDBDevice("oneplus9pro")
	fmt.Println("device: \n", spew.Sdump(device))
	req, err := CheckinRequest(ctx, &FirebaseDevice{Device: device}, "", "")
	if err != nil {
		t.Error(err)
	}

	client := testHTTPClient("http://127.0.0.1:8888")

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
	}
	result, err := CheckinResult(resp)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("androidID: ", *result.AndroidId)
	fmt.Println("securityToken: ", *result.SecurityToken)
}

func TestRegister3(t *testing.T) {
	ctx := context.Background()
	device, _ := andutils.GetDBDevice("oneplus9pro")
	fDevice := &FirebaseDevice{
		Device:                   device,
		AndroidPackage:           "jp.naver.line.android",
		AndroidCert:              "89396DC419292473972813922867E6973D6F5C50",
		GoogleAPIKey:             "AIzaSyBGRb2sEaaXjsKH6ea6f2xSiUeG4D8vaCY",
		ProjectID:                "jp-naver-line",
		AppNameHash:              "R1dAH9Ui7M-ynoznwBdw01tLxh",
		NotificationSender:       "4586549225",
		AppVersion:               "11.22.2",
		AppVersionWithBuild:      "112220115",
		FirebaseInstallationAuth: nil,
		FirebaseInstallationID:   RandomAppFID(),
		CheckinAndroidID:         0,
		CheckinSecurityToken:     0,
	}

	client := testHTTPClient("http://127.0.0.1:8888")

	req, err := C2DMAndroidRegisterRequest(ctx, fDevice)
	if err != nil {
		t.Error(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
	}

	result, err := AndroidRegisterResult(resp)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("notificationToken: \n", result)
}

func TestRandomAppFID(t *testing.T) {
	fmt.Println(RandomAppFID())
}
