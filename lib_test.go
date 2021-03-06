package go_android_firebase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/BRUHItsABunny/go-android-firebase/api"
	client2 "github.com/BRUHItsABunny/go-android-firebase/client"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"github.com/davecgh/go-spew/spew"
	"net/http"
	"net/url"
	"testing"
)

func testHTTPClient(proxy string) *http.Client {
	client := http.DefaultClient
	client.Transport = http.DefaultTransport
	if len(proxy) > 0 {
		proxyURL, _ := url.Parse(proxy)
		client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyURL)
	}
	return client
}

func TestRegister3(t *testing.T) {
	ctx := context.Background()
	device := andutils.GetRandomDevice()
	appData := &api.FirebaseAppData{
		PackageID:            "org.wikipedia",
		PackageCertificate:   "D21A6A91AA75C937C4253770A8F7025C6C2A8319",
		GoogleAPIKey:         "AIzaSyC7m9NhFXHiUPryquw7PecqFO0d9YPrVNE",
		FirebaseProjectID:    "pushnotifications-73c5e",
		GMPAppID:             "1:296120793014:android:34d2ba8d355ca9259a7317",
		NotificationSenderID: "296120793014",
		AppVersion:           "2.7.50394-r-2022-02-10",
		AppVersionWithBuild:  "50394",
		AuthVersion:          "FIS_v2",
		SdkVersion:           "a:17.0.0",
		AppNameHash:          "R1dAH9Ui7M-ynoznwBdw01tLxhI",
	}
	fDevice := &api.FirebaseDevice{
		Device:               device,
		CheckinAndroidID:     0,
		CheckinSecurityToken: 0,
	}

	client := testHTTPClient("http://127.0.0.1:8888")
	fClient := client2.NewFirebaseClient(client, fDevice, appData)
	authResult, err := fClient.NotifyInstallation(ctx)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(spew.Sdump(authResult))

	checkinResult, err := fClient.Checkin(ctx, "", "")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(fmt.Sprintf("AndroidID (checkin): %d\nSecurityToken: %d", checkinResult.AndroidId, checkinResult.SecurityToken))

	result, err := fClient.C2DMRegisterAndroid(ctx)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("notificationToken: \n", result)

	// Check if fDevice was updated with the new information returned by the api calls
	prettyBytes, err := json.MarshalIndent(fDevice, "", "    ")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(prettyBytes))
}

func TestRandomAppFID(t *testing.T) {
	fmt.Println(RandomAppFID())
}

func TestBits(t *testing.T) {
	fmt.Println(api.GetLeastMostSignificantBits("a316b044-0157-1000-efe6-40fc5d2f0036"))
}

func TestConvert(t *testing.T) {
	fmt.Println(base64.StdEncoding.DecodeString("cA=="))
}
