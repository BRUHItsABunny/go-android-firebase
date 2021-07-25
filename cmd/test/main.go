package main

import (
	"fmt"
	go_android_firebase "github.com/BRUHItsABunny/go-android-firebase"
	go_android_firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	go_android_db "github.com/BRUHItsABunny/go-android-utils/database"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	// Testing installation request for authToken
	var (
		project    = "api-project-829092136971"
		andCert    = "D06C0AACF652B0065DF272A4DDEC6255EB93CA06"
		andPackage = "com.kixeye.wcm"
		googAPIKey = "AIzaSyApp-r0nt0UCP8upiqL1_OXXXVi880dTOI"
	)

	client := go_android_firebase.NewFireBaseClient(nil, go_android_db.GetRandomDevice(), andCert, andPackage, project, googAPIKey)

	req := go_android_firebase_api.NotifyInstallationRequestBody{
		FID:         "ftyb0dRpS6io72kgwaNcW7",
		AppID:       "1:829092136971:android:5a518e9f9a6a1bd7",
		AuthVersion: "FIS_v2",
		SDKVersion:  "a:17.0.0",
	}

	fmt.Println(spew.Sdump(client.NotifyInstallation(&req)))
}

//{
//  "fid": "ftyb0dRpS6io72kgwaNcW8",
//  "appId": "1:829092136971:android:5a518e9f9a6a1bd8",
//  "authVersion": "FIS_v2",
//  "sdkVersion": "a:17.0.0"
//}
