package firebase

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	"github.com/BRUHItsABunny/gOkHttp/requests"
	"github.com/BRUHItsABunny/gOkHttp/responses"
	"github.com/BRUHItsABunny/go-android-firebase/api"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type FakeMTalk struct {
	buff *bytes.Buffer
}

func (c *FakeMTalk) readBytes(len int) ([]byte, error) {
	buf := make([]byte, len)
	var result []byte
	read, err := c.buff.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf(" c.buff.Read: %w", err)
	} else {
		err = nil
	}
	result = buf[:read]
	// fmt.Println(fmt.Sprintf("%s\tIO:BYTESIN:%s", time.Now().Format(time.RFC3339), hex.EncodeToString(result)))
	return result, err
}

func (c *FakeMTalk) readByte() (byte, error) {
	buf, err := c.readBytes(1)
	if err != nil {
		return 0, err
	}
	if len(buf) != 1 {
		return 0, errors.New("no data read")
	}
	return buf[0], nil
}

func (c *FakeMTalk) readVarInt() (int, error) {
	shift := uint(0)
	result := int64(0)
	for {
		b, err := c.readByte()
		if err != nil {
			return 0, fmt.Errorf("c.readByte: %w", err)
		}
		result |= int64(b&0x7f) << shift
		if (b & 0x80) != 0x80 {
			break
		}
		shift += 7
	}
	return int(result), nil
}

func NewFakeMTalk(fileName string) (*FakeMTalk, error) {
	fileBody, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return &FakeMTalk{
		buff: bytes.NewBuffer(fileBody),
	}, nil
}

func TestDecodeItem(t *testing.T) {
	c, err := NewFakeMTalk("_resources/samples/item.bin")
	if err != nil {
		log.Fatal(err)
	}
	tag, err := c.readByte()
	if err != nil {
		t.Fatal(fmt.Errorf("c.readByte: %w", err))
	}
	length, err := c.readVarInt()
	if err != nil {
		t.Fatal(fmt.Errorf("c.readVarInt: %w", err))
	}
	data, err := c.readBytes(length)
	if err != nil {
		t.Fatal(fmt.Errorf("c.readBytes data: %w", err))
	}

	fmt.Println(tag, length, string(data))
	fmt.Println(hex.EncodeToString(data))

	var result proto.Message
	switch firebase_api.MCSTag(int(tag)) {
	case firebase_api.MCSTag_MCS_HEARTBEAT_PING_TAG:
		result = &firebase_api.HeartbeatPing{}
		break
	case firebase_api.MCSTag_MCS_HEARTBEAT_ACK_TAG:
		result = &firebase_api.HeartbeatAck{}
		break
	case firebase_api.MCSTag_MCS_LOGIN_REQUEST_TAG:
		result = &firebase_api.LoginRequest{}
		break
	case firebase_api.MCSTag_MCS_LOGIN_RESPONSE_TAG:
		result = &firebase_api.LoginResponse{}
		break
	case firebase_api.MCSTag_MCS_CLOSE_TAG:
		result = &firebase_api.Close{}
		break
	case firebase_api.MCSTag_MCS_IQ_STANZA_TAG:
		result = &firebase_api.IqStanza{}
		break
	case firebase_api.MCSTag_MCS_DATA_MESSAGE_STANZA_TAG:
		result = &firebase_api.DataMessageStanza{}
		break
	default:
		t.Fatal(fmt.Errorf("unknown tag: %d", tag))
	}
	err = proto.Unmarshal(data, result)
	if err != nil {
		t.Fatal(fmt.Errorf("proto.Unmarshal[%x]: %w", data, err))
	}

	fmt.Println(spew.Sdump(result))
	// This succeeds so I know something is wrong in the registration/sending push notification part
}

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
		PackageID:            "fr.smarquis.fcm",
		PackageCertificate:   "FC674E2E5582B7BEA69EE5CA921FCEFAD2918452",
		GoogleAPIKey:         " AIzaSyDBHR45cWSsnJw-7inTYFDtK39-0TpjlhA",
		FirebaseProjectID:    "fir-cloudmessaging-4e2cd",
		GMPAppID:             "1:322141800886:android:7b41fd8ce1e97722",
		NotificationSenderID: "322141800886",
		AppVersion:           "1.9.0",
		AppVersionWithBuild:  "1090000",
		AuthVersion:          "FIS_v2",
		SdkVersion:           "a:17.1.3",
		AppNameHash:          "R1dAH9Ui7M-ynoznwBdw01tLxhI",
	}
	fDevice := &firebase_api.FirebaseDevice{
		Device:                device,
		CheckinAndroidID:      0,
		CheckinSecurityToken:  0,
		GmsVersion:            "241718022",
		FirebaseClientVersion: "fcm-23.1.2",
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
		PackageID:           "com.brave.browser",
		PackageCertificate:  "4b5d0914b118f51f30634a1523f96e020ab24fd2",
		AppVersion:          "1.75.180",
		AppVersionWithBuild: "427518024",
	}
	fDevice := &firebase_api.FirebaseDevice{
		Device:                device,
		CheckinAndroidID:      0,
		CheckinSecurityToken:  0,
		GmsVersion:            "250632029",
		FirebaseClientVersion: "fcm-22.0.0",
	}

	err := godotenv.Load(".env")
	hClient, err := gokhttp.TestHTTPClient()
	if err != nil {
		t.Error(err)
	}

	fClient := NewFirebaseClient(hClient, fDevice)

	checkinResult, err := fClient.Checkin(ctx, appData, "", "")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(fmt.Sprintf("AndroidID (checkin): %d\nSecurityToken: %d", checkinResult.AndroidId, checkinResult.SecurityToken))
	time.Sleep(time.Second * 5)

	sender := getNotificationDataWeb()
	uuidStr := strings.ToUpper(uuid.New().String())
	subType := "https://push.foo/#" + uuidStr[:len(uuidStr)-3]
	appid := "f1pdRYedASE" // TODO: IDK where this one comes from

	authNonceBytes := make([]byte, 16)
	_, err = rand.Read(authNonceBytes)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}

	curve := ecdh.P256()
	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}
	publicKey := privateKey.PublicKey()
	publicKeyStr := base64.RawURLEncoding.EncodeToString(publicKey.Bytes())
	fmt.Printf("My Public Key (base64): %s\n", publicKeyStr)
	// remotePubKeyBytes, err := base64.RawURLEncoding.DecodeString(sender)
	// if err != nil {
	// 	log.Fatalf("Failed to decode remote public key: %v", err)
	// }
	// remotePubKey, err := curve.NewPublicKey(remotePubKeyBytes)
	// if err != nil {
	// 	log.Fatalf("Failed to parse remote public key: %v", err)
	// }
	// // 3. Perform ECDH to compute the shared secret.
	// sharedSecret, err := privateKey.ECDH(remotePubKey)
	// if err != nil {
	// 	log.Fatalf("ECDH key agreement error: %v", err)
	// }

	result, err := fClient.C2DMRegisterWeb(ctx, appData, sender, subType, appid)
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
	err = sendNotificationWeb(hClient, result, publicKeyStr, base64.RawURLEncoding.EncodeToString(authNonceBytes))
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
	headerOpt := gokhttp_requests.NewHeaderOption(http.Header{})

	body := strings.ReplaceAll("{\"data\":{\"to\":\"$TOKEN\",\"ttl\":60,\"priority\":\"high\",\"data\":{\"ping\":{}}}}", "$TOKEN", token)
	req, err := gokhttp_requests.MakePOSTRequest(context.Background(), "https://us-central1-fir-cloudmessaging-4e2cd.cloudfunctions.net/send", headerOpt, gokhttp_requests.NewPOSTJSONOption([]byte(body), false))
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
	fmt.Println(respText)
	return nil
}

func sendNotificationWeb(client *http.Client, token, publicKey, authNonce string) error {
	headerOpt := gokhttp_requests.NewHeaderOption(http.Header{})

	body := strings.ReplaceAll("{\"pushSubscription\":{\"endpoint\":\"https://fcm.googleapis.com/fcm/send/$TOKEN\",\"expirationTime\":null,\"keys\":{\"p256dh\":\"$PUBLIC_KEY\",\"auth\":\"$AUTH_NONCE\"}},\"notification\":{\"title\":\"Push.Foo Notification Title\",\"actions\":[{\"action\":\"open_project_repo\",\"title\":\"Show source code\"},{\"action\":\"open_author_twitter\",\"title\":\"Author on Twitter\"},{\"action\":\"open_author_linkedin\",\"title\":\"Author on LinkedIn\"},{\"action\":\"open_url\",\"title\":\"Open custom URL\"}],\"body\":\"Test notification body\",\"dir\":\"auto\",\"image\":\"https://push.foo/images/social.png\",\"icon\":\"https://push.foo/images/logo.jpg\",\"badge\":\"https://push.foo/images/logo-mask.png\",\"lang\":\"en-US\",\"renotify\":false,\"requireInteraction\":true,\"silent\":false,\"tag\":\"Custom tag\",\"timestamp\":1740333526775,\"data\":{\"dateOfArrival\":1740333526775,\"updateInAppCounter\":true,\"updateIconBadgeCounter\":true,\"author\":{\"name\":\"Maxim Salnikov\",\"github\":\"https://github.com/webmaxru\",\"twitter\":\"https://twitter.com/webmaxru\",\"linkedin\":\"https://www.linkedin.com/in/webmax/\"},\"project\":{\"github\":\"https://github.com/webmaxru/push.foo\"},\"action\":{\"url\":\"https://push.foo\"}}}}", "$TOKEN", token)
	body = strings.ReplaceAll(body, "$PUBLIC_KEY", publicKey)
	body = strings.ReplaceAll(body, "$AUTH_NONCE", authNonce)

	req, err := gokhttp_requests.MakePOSTRequest(context.Background(), "https://push.foo/api/quick-notification", headerOpt, gokhttp_requests.NewPOSTJSONOption([]byte(body), false))
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
	fmt.Println(respText)
	return nil
}

func getNotificationDataWeb() string {
	// Use https://push.foo
	// Return publicKey (sender)
	return "BDweuGCGNzjleeyQYPvtFLEbMG4BX9rc_M9Abtx16NvaR_Jpo5i08WAJUll2Hn6ZiErbSjkzxWdpKjus_qO2cMw"
}
