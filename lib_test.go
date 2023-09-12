package firebase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	"github.com/BRUHItsABunny/gOkHttp/requests"
	"github.com/BRUHItsABunny/gOkHttp/responses"
	"github.com/BRUHItsABunny/go-android-firebase/api"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestAuthLogin(t *testing.T) {
	err := godotenv.Load(".env")
	hClient, err := gokhttp.TestHTTPClient()
	if err != nil {
		t.Error(err)
		panic(err)
	}

	device := &firebase_api.FirebaseDevice{
		Device: andutils.GetRandomDevice(),
	}
	appData := &firebase_api.FirebaseAppData{
		PackageID:          os.Getenv("AUTH_LOGIN_PACKAGE_ID"),
		PackageCertificate: os.Getenv("AUTH_LOGIN_PACKAGE_CERTIFICATE"),
		GoogleAPIKey:       os.Getenv("AUTH_LOGIN_GOOGLE_API_KEY"),
		FirebaseProjectID:  os.Getenv("AUTH_LOGIN_FIREBASE_PROJECT_ID"),
	}

	values := url.Values{
		"add_account":                  {"1"},
		"get_accountid":                {"1"},
		"google_play_services_version": {"220217000"},
		"ACCESS_TOKEN":                 {"1"},
		"operatorCountry":              {"us"},
		"service":                      {"ac2dm"},
	}

	ctx := context.Background()
	client := NewFirebaseClient(hClient, device)

	resp, err := client.Auth(ctx, appData, values, os.Getenv("AUTH_LOGIN_EMAIL"), os.Getenv("AUTH_LOGIN_OAUTH_TOKEN"))
	if err == nil {
		fmt.Println(spew.Sdump(resp))
	} else {
		fmt.Println(err)
	}
}

func TestAuthOAUTH(t *testing.T) {
	err := godotenv.Load(".env")
	hClient, err := gokhttp.TestHTTPClient()
	if err != nil {
		t.Error(err)
		panic(err)
	}
	device := &firebase_api.FirebaseDevice{
		Device: andutils.GetRandomDevice(),
	}
	appData := &firebase_api.FirebaseAppData{
		PackageID:          os.Getenv("AUTH_OAUTH_PACKAGE_ID"),
		PackageCertificate: os.Getenv("AUTH_OAUTH_PACKAGE_CERTIFICATE"),
		GoogleAPIKey:       os.Getenv("AUTH_OAUTH_GOOGLE_API_KEY"),
		FirebaseProjectID:  os.Getenv("AUTH_OAUTH_FIREBASE_PROJECT_ID"),
	}

	values := url.Values{
		"add_account":                  {"1"},
		"get_accountid":                {"1"},
		"google_play_services_version": {"220217000"},
		"ACCESS_TOKEN":                 {"1"},
		"operatorCountry":              {"us"},
		"it_caveat_types":              {"1"},
		"oauth2_foreground":            {"1"},
		"has_permission":               {"1"},
		"token_request_options":        {"CAA4AVAB"},
		"check_email":                  {"1"},
		"service":                      {"oauth2:https://www.googleapis.com/auth/accounts.reauth https://www.googleapis.com/auth/youtube.force-ssl https://www.googleapis.com/auth/youtube https://www.googleapis.com/auth/identity.lateimpersonation https://www.googleapis.com/auth/assistant-sdk-prototype"},
		"system_partition":             {"1"},
	}

	ctx := context.Background()
	client := NewFirebaseClient(hClient, device)

	resp, err := client.Auth(ctx, appData, values, os.Getenv("AUTH_OAUTH_EMAIL"), os.Getenv("AUTH_OAUTH_MASTER_TOKEN"))
	if err == nil {
		fmt.Println(spew.Sdump(resp))
	} else {
		fmt.Println(err)
	}
}

func TestNotify(t *testing.T) {
	err := godotenv.Load(".env")
	hClient, err := gokhttp.TestHTTPClient()
	if err != nil {
		t.Error(err)
		panic(err)
	}
	device := &firebase_api.FirebaseDevice{
		Device: andutils.GetRandomDevice(),
	}
	appData := &firebase_api.FirebaseAppData{
		PackageID:          os.Getenv("NOTIFY_PACKAGE_ID"),
		PackageCertificate: os.Getenv("NOTIFY_PACKAGE_CERTIFICATE"),
		GoogleAPIKey:       os.Getenv("NOTIFY_GOOGLE_API_KEY"),
		FirebaseProjectID:  os.Getenv("NOTIFY_FIREBASE_PROJECT_ID"),
	}

	ctx := context.Background()
	client := NewFirebaseClient(hClient, device)

	resp, err := client.NotifyInstallation(ctx, appData)
	if err == nil {
		fmt.Println(spew.Sdump(resp))
	} else {
		fmt.Println(err)
	}
}

func TestVerifyPassword(t *testing.T) {
	err := godotenv.Load(".env")
	hClient, err := gokhttp.TestHTTPClient()
	if err != nil {
		t.Error(err)
		panic(err)
	}
	device := &firebase_api.FirebaseDevice{
		Device: andutils.GetRandomDevice(),
	}
	appData := &firebase_api.FirebaseAppData{
		PackageID:          os.Getenv("VERIFY_PASSWORD_PACKAGE_ID"),
		PackageCertificate: os.Getenv("VERIFY_PASSWORD_PACKAGE_CERTIFICATE"),
		GoogleAPIKey:       os.Getenv("VERIFY_PASSWORD_GOOGLE_API_KEY"),
		FirebaseProjectID:  os.Getenv("VERIFY_PASSWORD_FIREBASE_PROJECT_ID"),
	}
	var (
		email    = os.Getenv("VERIFY_PASSWORD_USERNAME")
		password = os.Getenv("VERIFY_PASSWORD_PASSWORD")
	)
	ctx := context.Background()
	client := NewFirebaseClient(hClient, device)

	req := &firebase_api.VerifyPasswordRequestBody{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}
	resp, err := client.VerifyPassword(ctx, req, appData)
	if err == nil {
		fmt.Println(spew.Sdump(resp))
	} else {
		fmt.Println(err)
	}
}

func TestRegister3(t *testing.T) {
	ctx := context.Background()
	device := andutils.GetRandomDevice()
	appData := &firebase_api.FirebaseAppData{
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
	fDevice := &firebase_api.FirebaseDevice{
		Device:                device,
		CheckinAndroidID:      0,
		CheckinSecurityToken:  0,
		GmsVersion:            "214815028",
		FirebaseClientVersion: "fcm-22.0.0",
	}

	err := godotenv.Load(".env")
	hClient, err := gokhttp.TestHTTPClient()
	if err != nil {
		t.Error(err)
	}

	fClient := NewFirebaseClient(hClient, fDevice)
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
	appData := &firebase_api.FirebaseAppData{
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
	fDevice := &firebase_api.FirebaseDevice{
		Device:                device,
		CheckinAndroidID:      0,
		CheckinSecurityToken:  0,
		GmsVersion:            "214815028",
		FirebaseClientVersion: "fcm-22.0.0",
	}

	err := godotenv.Load(".env")
	hClient, err := gokhttp.TestHTTPClient()
	if err != nil {
		t.Error(err)
	}

	fClient := NewFirebaseClient(hClient, fDevice)
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
	resultChan := make(chan *firebase_api.DataMessageStanza)
	fClient.MTalk.OnNotification = func(notification *firebase_api.DataMessageStanza) {
		resultChan <- notification
	}
	pre := time.Now()
	err = sendPushNotificationNative(fDevice, hClient, result)
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
	appData := &firebase_api.FirebaseAppData{
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
	fDevice := &firebase_api.FirebaseDevice{
		Device:                device,
		CheckinAndroidID:      0,
		CheckinSecurityToken:  0,
		GmsVersion:            "214815028",
		FirebaseClientVersion: "fcm-22.0.0",
	}

	err := godotenv.Load(".env")
	hClient, err := gokhttp.TestHTTPClient()
	if err != nil {
		t.Error(err)
	}

	fClient := NewFirebaseClient(hClient, fDevice)
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
	resultChan := make(chan *firebase_api.DataMessageStanza)
	fClient.MTalk.OnNotification = func(notification *firebase_api.DataMessageStanza) {
		resultChan <- notification
	}
	pre := time.Now()
	err = sendPushNotificationNative(fDevice, hClient, result) // TODO: Web implementation
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
	fmt.Println(firebase_api.GetLeastMostSignificantBits("a316b044-0157-1000-efe6-40fc5d2f0036"))
}

func TestConvert(t *testing.T) {
	fmt.Println(base64.StdEncoding.DecodeString("cA=="))
}

func sendPushNotificationNative(fDevice *firebase_api.FirebaseDevice, client *http.Client, token string) error {
	headerOpt := gokhttp_requests.NewHeaderOption(http.Header{"user-agent": []string{"okhttp/3.12.1"}})
	quotaParams := gokhttp_requests.NewPOSTFormOption(url.Values{"device_id": []string{fDevice.Device.Id.ToHexString()}, "credit_date": []string{time.Now().Format("2006-01-02")}, "type": []string{"1"}})
	req, err := gokhttp_requests.MakePOSTRequest(context.Background(), "https://api.sartajahmed.in/debug_fcm/v1/credit/addCredit", headerOpt, quotaParams)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	respText, err := gokhttp_responses.ResponseText(resp)
	if err != nil {
		return err
	}
	// fmt.Println(respText)

	sendParams := gokhttp_requests.NewPOSTFormOption(url.Values{"device_id": []string{fDevice.Device.Id.ToHexString()}, "push_device_token": []string{token}, "type": []string{"1"}, "push_limit": []string{"5"}})
	req, err = gokhttp_requests.MakePOSTRequest(context.Background(), "https://api.sartajahmed.in/debug_fcm/v1/send_push/sendSimplePushTest", headerOpt, sendParams)
	if err != nil {
		return err
	}
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	respText, err = gokhttp_responses.ResponseText(resp)
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
