package main

import (
	"context"
	"fmt"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	go_android_firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	go_android_firebase_client "github.com/BRUHItsABunny/go-android-firebase/client"
	andutilsdb "github.com/BRUHItsABunny/go-android-utils/database"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	opts := gokhttp.DefaultGOKHTTPOptions
	hClient := gokhttp.GetHTTPClient(opts)
	// _ = hClient.SetProxy("http://127.0.0.1:8888")
	device := &go_android_firebase_api.FirebaseDevice{
		Device:         andutilsdb.GetRandomDevice(),
		AndroidPackage: "com.barcodelookup",
		AndroidCert:    "526E7514F042F15966600565485F39F98288453F",
		GoogleAPIKey:   "",
		ProjectID:      "android-app-9d60d",
	}
	ctx := context.Background()
	client := go_android_firebase_client.NewFirebaseClient(hClient.Client, device)
	req := &go_android_firebase_api.NotifyInstallationRequestBody{
		FID:         "fYSwWtaFS7WBFO91hQx1g5",
		AppID:       "1:837055667328:android:897a139d2343863a6f1a65",
		AuthVersion: "FIS_v2",
		SDKVersion:  "a:16.3.3",
	}

	resp, err := client.NotifyInstallation(ctx, req)
	if err == nil {
		fmt.Println(spew.Sdump(resp))
	} else {
		fmt.Println(err)
	}
}
