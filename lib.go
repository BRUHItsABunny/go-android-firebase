package firebase

import (
	"github.com/BRUHItsABunny/go-android-firebase/api"
	"github.com/BRUHItsABunny/go-android-firebase/client"
	"net/http"
)

func NewFirebaseClient(hClient *http.Client, device *firebase_api.FirebaseDevice) (*firebase_client.FireBaseClient, error) {
	return firebase_client.NewFirebaseClient(hClient, device)
}

func RandomAppFID() string {
	result, _ := firebase_api.RandomAppFID()
	return result
}
