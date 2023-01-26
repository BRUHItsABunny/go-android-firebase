package go_android_firebase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	gokhttpclient "github.com/BRUHItsABunny/gOkHttp/client"
	"github.com/BRUHItsABunny/gOkHttp/requests"
	"github.com/BRUHItsABunny/gOkHttp/responses"
	"github.com/BRUHItsABunny/go-android-firebase/api"
	firebaseclient "github.com/BRUHItsABunny/go-android-firebase/client"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

func testHTTPClient() (*http.Client, error) {
	hClient := http.DefaultClient
	err := godotenv.Load(".env")
	if err != nil {
		return hClient, err
	}

	opts := []gokhttpclient.Option{}
	if os.Getenv("USE_PROXY") == "true" {
		opts = append(opts, gokhttpclient.NewProxyOption(os.Getenv("PROXY_URL")))
	}

	hClient, err = gokhttpclient.NewHTTPClient(opts...)
	return hClient, err
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
		Device:                device,
		CheckinAndroidID:      0,
		CheckinSecurityToken:  0,
		GmsVersion:            "214815028",
		FirebaseClientVersion: "fcm-22.0.0",
	}

	client, err := testHTTPClient()
	if err != nil {
		t.Error(err)
	}

	fClient := firebaseclient.NewFirebaseClient(client, fDevice)
	authResult, err := fClient.NotifyInstallation(ctx, appData)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(spew.Sdump(authResult))

	checkinResult, err := fClient.Checkin(ctx, appData, "", "")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(fmt.Sprintf("AndroidID (checkin): %d\nSecurityToken: %d", checkinResult.AndroidId, checkinResult.SecurityToken))

	result, err := fClient.C2DMRegisterAndroid(ctx, appData)
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

func TestNativePushNotifications(t *testing.T) {
	ctx := context.Background()
	device := andutils.GetRandomDevice()
	appData := &api.FirebaseAppData{
		PackageID:            "com.debug.fcm",
		PackageCertificate:   "194324D4357EBB453DDB2A9F8FC8E86C27A35A14",
		GoogleAPIKey:         "AIzaSyBot7ALdoDk6RtUqNZbZ6Ik4ffqzaayY9I",
		FirebaseProjectID:    "debug-fcm",
		GMPAppID:             "1:1066350740658:android:4c54c351189dd709",
		NotificationSenderID: "1066350740658",
		AppVersion:           "1.5.6",
		AppVersionWithBuild:  "17",
		AuthVersion:          "FIS_v2",
		SdkVersion:           "a:16.3.2",
		AppNameHash:          "R1dAH9Ui7M-ynoznwBdw01tLxhI",
	}
	fDevice := &api.FirebaseDevice{
		Device:                device,
		CheckinAndroidID:      0,
		CheckinSecurityToken:  0,
		GmsVersion:            "214815028",
		FirebaseClientVersion: "fcm-22.0.0",
	}

	client, err := testHTTPClient()
	if err != nil {
		t.Error(err)
	}

	fClient := firebaseclient.NewFirebaseClient(client, fDevice)
	_, err = fClient.NotifyInstallation(ctx, appData)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second * 5)

	checkinResult, err := fClient.Checkin(ctx, appData, "", "")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(fmt.Sprintf("AndroidID (checkin): %d\nSecurityToken: %d", checkinResult.AndroidId, checkinResult.SecurityToken))
	time.Sleep(time.Second * 5)

	result, err := fClient.C2DMRegisterAndroid(ctx, appData)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("notificationToken: \n", result)

	time.Sleep(time.Second * 10) // it will error out if we don't wait, there is a latency between checkin credentials being registered with gcm/fcm and being registered with mtalk

	err = fClient.MTalk.Connect()
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second * 3)
	resultChan := make(chan *api.DataMessageStanza)
	fClient.MTalk.OnNotification = func(notification *api.DataMessageStanza) {
		resultChan <- notification
	}
	pre := time.Now()
	err = sendPushNotificationNative(fDevice, client, result)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("Waiting for message")
	msg := <-resultChan
	latency := time.Now().Sub(pre)
	fmt.Println(spew.Sdump(msg))
	fmt.Println("Latency: ", latency)
}

func TestWebPushNotifications(t *testing.T) {
	ctx := context.Background()
	device := andutils.GetRandomDevice()
	appData := &api.FirebaseAppData{
		PackageID:            "com.debug.fcm",
		PackageCertificate:   "194324D4357EBB453DDB2A9F8FC8E86C27A35A14",
		GoogleAPIKey:         "AIzaSyBot7ALdoDk6RtUqNZbZ6Ik4ffqzaayY9I",
		FirebaseProjectID:    "debug-fcm",
		GMPAppID:             "1:1066350740658:android:4c54c351189dd709",
		NotificationSenderID: "1066350740658",
		AppVersion:           "1.5.6",
		AppVersionWithBuild:  "17",
		AuthVersion:          "FIS_v2",
		SdkVersion:           "a:16.3.2",
		AppNameHash:          "R1dAH9Ui7M-ynoznwBdw01tLxhI",
	}
	fDevice := &api.FirebaseDevice{
		Device:                device,
		CheckinAndroidID:      0,
		CheckinSecurityToken:  0,
		GmsVersion:            "214815028",
		FirebaseClientVersion: "fcm-22.0.0",
	}

	client, err := testHTTPClient()
	if err != nil {
		t.Error(err)
	}

	fClient := firebaseclient.NewFirebaseClient(client, fDevice)
	_, err = fClient.NotifyInstallation(ctx, appData)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second * 5)

	checkinResult, err := fClient.Checkin(ctx, appData, "", "")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(fmt.Sprintf("AndroidID (checkin): %d\nSecurityToken: %d", checkinResult.AndroidId, checkinResult.SecurityToken))
	time.Sleep(time.Second * 5)

	result, err := fClient.C2DMRegisterAndroid(ctx, appData)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("notificationToken: \n", result)

	time.Sleep(time.Second * 10) // it will error out if we don't wait, there is a latency between checkin credentials being registered with gcm/fcm and being registered with mtalk

	err = fClient.MTalk.Connect()
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second * 3)
	resultChan := make(chan *api.DataMessageStanza)
	fClient.MTalk.OnNotification = func(notification *api.DataMessageStanza) {
		resultChan <- notification
	}
	pre := time.Now()
	err = sendPushNotificationNative(fDevice, client, result) // TODO: Web implementation
	if err != nil {
		t.Error(err)
	}
	fmt.Println("Waiting for message")
	msg := <-resultChan
	latency := time.Now().Sub(pre)
	fmt.Println(spew.Sdump(msg))
	fmt.Println("Latency: ", latency)
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

func sendPushNotificationNative(fDevice *api.FirebaseDevice, client *http.Client, token string) error {
	headerOpt := requests.NewHeaderOption(http.Header{"user-agent": []string{"okhttp/3.12.1"}})
	quotaParams := requests.NewPOSTFormOption(url.Values{"device_id": []string{fDevice.Device.Id.ToHexString()}, "credit_date": []string{time.Now().Format("2006-01-02")}, "type": []string{"1"}})
	req, err := requests.MakePOSTRequest(context.Background(), "https://api.sartajahmed.in/debug_fcm/v1/credit/addCredit", headerOpt, quotaParams)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	respText, err := responses.ResponseText(resp)
	if err != nil {
		return err
	}
	// fmt.Println(respText)

	sendParams := requests.NewPOSTFormOption(url.Values{"device_id": []string{fDevice.Device.Id.ToHexString()}, "push_device_token": []string{token}, "type": []string{"1"}, "push_limit": []string{"5"}})
	req, err = requests.MakePOSTRequest(context.Background(), "https://api.sartajahmed.in/debug_fcm/v1/send_push/sendSimplePushTest", headerOpt, sendParams)
	if err != nil {
		return err
	}
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	respText, err = responses.ResponseText(resp)
	if err != nil {
		return err
	}
	// fmt.Println(respText)
	fmt.Sprint(respText)
	return nil
}

func sendNotificationWeb() {
	// register

	// send the notification

	// decrypt
}
