package go_android_firebase

import (
	"github.com/BRUHItsABunny/go-android-firebase/api"
	"github.com/BRUHItsABunny/go-android-firebase/client"
	"net/http"
)

func NewFirebaseClient(hClient *http.Client, device *api.FirebaseDevice, appData *api.FirebaseAppData) *client.FireBaseClient {
	return client.NewFirebaseClient(hClient, device, appData)
}

func RandomAppFID() string {
	return api.RandomAppFID()
}
