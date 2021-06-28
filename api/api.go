package api

import (
	"bytes"
	"encoding/json"
	go_android_firebase "go-android-firebase"
	"net/http"
)

type NotifyInstallationRequestBody struct {
	FID         string `json:"fid"`
	AppID       string `json:"appId"`
	AuthVersion string `json:"authVersion"`
	SDKVersion  string `json:"sdkVersion"`
}

type FireBaseInstallationResponse struct {
	Name         string            `json:"name"`
	FID          string            `json:"fid"`
	RefreshToken string            `json:"refreshToken"`
	AuthToken    FireBaseAuthToken `json:"authToken"`
}

type FireBaseAuthToken struct {
	Token      string `json:"token"`
	Expiration string `json:"expiresin"`
}

type NotifyInstallationResponse struct {
	FID         string `json:"fid"`
	AppID       string `json:"appId"`
	AuthVersion string `json:"authVersion"`
	SDKVersion  string `json:"sdkVersion"`
}

type HeaderFiller interface {
	Fill(*http.Request) *http.Request
}

type DefaultHeadersFiller struct {
	Headers map[string]string
}

func (filler *DefaultHeadersFiller) Fill(req *http.Request) *http.Request {

	for key, val := range filler.Headers {
		req.Header[key] = []string{val}
	}

	return req
}

var DefaultHeaders = DefaultHeadersFiller{
	Headers: map[string]string{
		go_android_firebase.HeaderKeyContentType:  go_android_firebase.HeaderValueMIMEJSON,
		go_android_firebase.HeaderKeyAccept:       go_android_firebase.HeaderValueMIMEJSON,
		go_android_firebase.HeaderKeyCacheControl: "no-cache",
	},
}

func NotifyInstallationRequest(data *NotifyInstallationRequestBody, filler HeaderFiller, ProjectID, AndroidPackage, AndroidCertificate, GoogAPIKey, FireBaseClient, FireBaseLogType, UserAgent string) *http.Request {
	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", go_android_firebase.Protocol+go_android_firebase.Host+go_android_firebase.EndpointProjects+ProjectID+go_android_firebase.SubEndpointInstallations, bytes.NewBuffer(body))

	req.Header[go_android_firebase.HeaderKeyAndroidCert] = []string{AndroidCertificate}
	req.Header[go_android_firebase.HeaderKeyAndroidPackage] = []string{AndroidPackage}
	req.Header[go_android_firebase.HeaderKeyFireBaseClient] = []string{FireBaseClient}
	req.Header[go_android_firebase.HeaderKeyGoogAPIKey] = []string{GoogAPIKey}
	req.Header[go_android_firebase.HeaderKeyFireBaseLogType] = []string{FireBaseLogType}
	req.Header[go_android_firebase.HeaderKeyUserAgent] = []string{UserAgent}

	req = filler.Fill(req)
	return req
}
