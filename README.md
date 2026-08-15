# go-android-firebase
Library to interact with Firebase the Android way

### Motivation
As a reverse engineer there will be times when I need to interact with a backend the same way the official client would and unsurprisingly a lot of apps nowadays use Firebase for many things such as user authentication, notifications, database.

This was the main reason why I set out to create a generalized library for interacting with Firebase emulating Android apps.

### Features

This library is currently able to do the following things:
* Emulate an installation callback
* Authenticate and register users to Firebase authentication
* Get a notification token
* Being able to listen for push notifications (android apps)
* Being able to listen for push notifications (chromium web push)
* Google OAuth to log in with Google in Android apps.

In the future I hope to add the following:
* Cleaner codebase with more testing

### Quick start

```go
ctx := context.Background()

// nil HTTP client -> a default client with a timeout, nil device -> a random device
client, err := firebase.NewFirebaseClient(nil, nil)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// One call instead of NotifyInstallation + Checkin + C2DMRegisterAndroid with sleeps
// in between. Transient failures (PHONE_REGISTRATION_ERROR right after a check-in) are
// retried with a backoff.
token, err := client.RegisterForNotifications(ctx, appData, nil)
if err != nil {
    log.Fatal(err)
}

client.MTalk.OnNotification = func(notification *firebase_api.DataMessageStanza) {
    fmt.Println(string(notification.RawData))
}
client.MTalk.OnError = func(err error, fatal bool) {
    log.Printf("mtalk: %v (fatal=%t)", err, fatal)
}
if err = client.MTalk.Connect(); err != nil {
    log.Fatal(err)
}

<-client.MTalk.Closed() // the read loop stopped, client.MTalk.Err() says why
```

`client.Device` is a protobuf message, so it can be marshalled and stored to keep the same
virtual device (check-in credentials, installation IDs, push keys) across restarts.

### Error handling

Every call returns typed errors that can be inspected with `errors.Is` / `errors.As`:

| Error | Meaning |
| --- | --- |
| `firebase.ErrNilAppData`, `firebase.ErrNilDevice` | a required argument was nil |
| `*firebase_api.MissingFieldError` | the app data or device is missing fields the call needs, the error names them |
| `firebase.ErrNoCheckin` | `Checkin` has not run yet |
| `firebase.ErrNoInstallation`, `firebase.ErrNoInstallationAuth` | `NotifyInstallation` has not run yet (or its token expired) |
| `*firebase_api.RegisterError` | GCM refused the registration, e.g. `PHONE_REGISTRATION_ERROR` |
| `*firebase_api.AuthError` | the Google auth endpoint refused the login, e.g. `BadAuthentication` |
| `*firebase_api.GoogleAPIError`, `*firebase_api.HTTPError` | the endpoint answered with an error, body included |

`firebase.IsRetryable(err)` reports whether waiting and trying again is worthwhile
(propagation delays right after a check-in, 429s, 5xx). Note that the registration
endpoints answer with HTTP 200 even when they refuse the request, so checking the status
code is not enough - this library parses the body and reports the actual reason.

### Examples

Some examples can be found in the ``_examples`` directory. They will go over the basics of how to interact with this library.

Just in case though, here is a small overview of how to set this up.
You essentially just want to find an app's data

When you open an app for the first time after installing the app or wiping all it's data a bunch of initialization related requests are fired off.
For our use case we only care about request A and request B.

![request_list](https://github.com/BRUHItsABunny/go-android-firebase/raw/main/_resources/images/requests_list.jpg)

Request A:
The variables we care about:
* in green: is our `firebaseProjectID`
* in purple: x-android-cert is our `packageCertificate`
* in blue: x-goog-api-key is our `googleAPIKey`
* in orange: appId is our `GMPAppID`
* no color: x-android-package: is our `packageID`

So for the app below our app data would look like:

```
appData := &api.FirebaseAppData{
		PackageID:            "org.wikipedia",
		PackageCertificate:   "D21A6A91AA75C937C4253770A8F7025C6C2A8319",
		GoogleAPIKey:         "AIzaSyC7m9NhFXHiUPryquw7PecqFO0d9YPrVNE",
		FirebaseProjectID:    "pushnotifications-73c5e",
		GMPAppID:             "1:296120793014:android:34d2ba8d355ca9259a7317",
		AuthVersion:          "FIS_v2",
		SdkVersion:           "a:17.0.0",
	}
```
This would be enough to emulate Firebase login, registration and app install callbacks.

![req_a_request](https://github.com/BRUHItsABunny/go-android-firebase/raw/main/_resources/images/req_a_request.jpg)

Request B adds some more data, only needed if you want to also emulate notifications down the line:

This results in app data looking like this:
```
appData := &api.FirebaseAppData{
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
```

Now that we have our app data we can start issuing requests to Firebase as if we were whatever app and version you specified.

### Note  regarding push notifications
Implementation is experimental and thus subject to change

### Note to fellow app developers
This library's mere existence and the fact that it functions exactly the way I outlined above means that your Google Firebase credentials are *NOT* safe.

Where do these credentials come from? The answer is simple: your `google-services.json`

This also means using other Google API's, eg: Google Vision SDK, can also just be captured as shown above which means people really don't have to do a lot of `magical hacking` (although they would have to bypass ssl-pinning, but that is not difficult) to acquire and use your credentials - for free (on your dime).

I am unaware of a way to mitigate this as of right now. (and this has existed for years)