package main

import (
	"context"
	"fmt"
	gokhttp "github.com/BRUHItsABunny/gOkHttp"
	go_android_firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	go_android_firebase_client "github.com/BRUHItsABunny/go-android-firebase/client"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"github.com/davecgh/go-spew/spew"
	"net/url"
)

func main() {
	opts := gokhttp.DefaultGOKHTTPOptions
	hClient := gokhttp.GetHTTPClient(opts)
	_ = hClient.SetProxy("http://127.0.0.1:8888")
	device := &go_android_firebase_api.FirebaseDevice{
		Device: andutils.GetRandomDevice(),
	}

	appData := &go_android_firebase_api.FirebaseAppData{}

	var (
		email       = ""
		masterToken = ""
		data        = url.Values{}
	)
	ctx := context.Background()
	client := go_android_firebase_client.NewFirebaseClient(hClient.Client, device, appData)

	resp, err := client.Auth(ctx, data, email, masterToken)
	if err == nil {
		fmt.Println(spew.Sdump(resp))
	} else {
		fmt.Println(err)
	}
}
