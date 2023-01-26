package go_android_firebase

import (
	"github.com/BRUHItsABunny/go-android-firebase/api"
	"github.com/BRUHItsABunny/go-android-firebase/client"
	"net/http"
)

func NewFirebaseClient(hClient *http.Client, device *api.FirebaseDevice) *client.FireBaseClient {
	return client.NewFirebaseClient(hClient, device)
}

func RandomAppFID() string {
	result, _ := api.RandomAppFID()
	return result
}
