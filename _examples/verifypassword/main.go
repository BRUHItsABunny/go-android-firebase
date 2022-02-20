package main

import (
	"context"
	"fmt"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	go_android_firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	go_android_firebase_client "github.com/BRUHItsABunny/go-android-firebase/client"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	opts := gokhttp.DefaultGOKHTTPOptions
	hClient := gokhttp.GetHTTPClient(opts)
	// _ = hClient.SetProxy("http://127.0.0.1:8888")
	device := &go_android_firebase_api.FirebaseDevice{
		Device: andutils.GetRandomDevice(),
	}
	appData := &go_android_firebase_api.FirebaseAppData{
		PackageID:          "com.barcodelooku",
		PackageCertificate: "526E7514F042F15966600565485F39F98288453F",
		GoogleAPIKey:       "",
		FirebaseProjectID:  "android-app-9d60d",
	}
	var (
		email    = ""
		password = ""
	)
	ctx := context.Background()
	client := go_android_firebase_client.NewFirebaseClient(hClient.Client, device, appData)

	req := &go_android_firebase_api.VerifyPasswordRequestBody{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}
	resp, err := client.VerifyPassword(ctx, req)
	if err == nil {
		fmt.Println(spew.Sdump(resp))
	} else {
		fmt.Println(err)
	}
}
