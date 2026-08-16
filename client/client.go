package firebase_client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FireBaseClient talks to the Firebase/GCM endpoints on behalf of a single (virtual) device.
//
// It is safe for concurrent use, the device state it mutates is guarded by a mutex.
type FireBaseClient struct {
	Client *http.Client
	Device *firebase_api.FirebaseDevice
	MTalk  *MTalkCon

	// deviceMutex guards the mutable parts of Device (installations, check-in credentials).
	deviceMutex sync.Mutex
	// ownsClient tracks whether Client was created here, we only ever mutate our own.
	ownsClient bool
	// logger is nil until WithLogger is used, use Logger() to get a usable one.
	logger *slog.Logger
	// logOptions control body logging.
	logOptions LogOptions
	// logCtx backs records that have no call context of their own, see WithLogContext.
	logCtx context.Context
}

// ClientOption customises a FireBaseClient at construction time.
type ClientOption func(*FireBaseClient)

// WithHTTPClient sets the *http.Client used for every call.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *FireBaseClient) {
		if client != nil {
			c.Client = client
		}
	}
}

// WithTimeout sets a timeout on the client's own *http.Client. It is ignored when the
// caller supplied their own client, so we never mutate something we do not own.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *FireBaseClient) {
		if c.ownsClient && timeout > 0 {
			c.Client.Timeout = timeout
		}
	}
}

// NewFirebaseClient builds a client for a device. A nil http client uses a sane default,
// a nil device gets a randomly generated one.
func NewFirebaseClient(client *http.Client, device *firebase_api.FirebaseDevice, opts ...ClientOption) (*FireBaseClient, error) {
	result := &FireBaseClient{}

	if client == nil {
		// http.DefaultClient has no timeout, a hung Google endpoint would hang the caller forever.
		result.Client = &http.Client{Timeout: DefaultHTTPTimeout}
		result.ownsClient = true
	} else {
		result.Client = client
	}

	if device == nil {
		device = &firebase_api.FirebaseDevice{}
	}
	if device.Device == nil {
		device.Device = andutils.GetRandomDevice()
	}
	device.ApplyDefaults()
	result.Device = device

	for _, opt := range opts {
		opt(result)
	}

	// Wire HTTP level logging in once every option has been seen. The caller's client is
	// copied rather than modified, we only ever own the transport we installed ourselves.
	if result.logger != nil {
		copied := *result.Client
		copied.Transport = &LoggingTransport{
			Base:    copied.Transport,
			Logger:  result.logger,
			Options: result.logOptions,
		}
		result.Client = &copied
	}

	mTalk, err := NewMTalkCon(device)
	if err != nil {
		return nil, fmt.Errorf("NewMTalkCon: %w", err)
	}
	mTalk.SetLogger(result.logger)
	mTalk.SetLogOptions(result.logOptions)
	mTalk.SetLogContext(result.logContext())
	result.MTalk = mTalk

	result.Logger().DebugContext(result.logContext(), "firebase client ready",
		logAttr("device", device),
		slog.Bool("log_bodies", result.logOptions.Bodies),
	)
	return result, nil
}

// DefaultHTTPTimeout is applied to the *http.Client this package creates for itself.
const DefaultHTTPTimeout = 30 * time.Second

// do performs a request and makes sure a transport error is reported with context.
func (c *FireBaseClient) do(req *http.Request) (*http.Response, error) {
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("c.Client.Do: %w", err)
	}
	return resp, nil
}

// installation returns the stored installation for a package, creating it when absent.
// The caller must hold deviceMutex.
func (c *FireBaseClient) installationLocked(packageID string) *firebase_api.FirebaseInstallationData {
	if c.Device.FirebaseInstallations == nil {
		c.Device.FirebaseInstallations = map[string]*firebase_api.FirebaseInstallationData{}
	}
	installation, ok := c.Device.FirebaseInstallations[packageID]
	if !ok || installation == nil {
		installation = &firebase_api.FirebaseInstallationData{NotificationData: &firebase_api.FirebaseNotificationData{}}
		c.Device.FirebaseInstallations[packageID] = installation
	}
	if installation.NotificationData == nil {
		installation.NotificationData = &firebase_api.FirebaseNotificationData{}
	}
	return installation
}

// NotifyInstallation registers the app installation with Firebase and stores the resulting
// installation ID and auth token on the device.
func (c *FireBaseClient) NotifyInstallation(ctx context.Context, appData *firebase_api.FirebaseAppData) (result *firebase_api.FireBaseInstallationResponse, err error) {
	start := time.Now()
	defer func() {
		c.logOperation(ctx, "NotifyInstallation", start, err, logAttr("app", appData), logAttr("result", result))
	}()

	req, err := firebase_api.NotifyInstallationRequest(ctx, c.Device, appData)
	if err != nil {
		return nil, fmt.Errorf("api.NotifyInstallationRequest: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	result, err = firebase_api.NotifyInstallationResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.NotifyInstallationResult: %w", err)
	}

	expiresIn, err := result.ExpiresIn()
	if err != nil {
		return nil, fmt.Errorf("api.NotifyInstallationResult: %w", err)
	}

	c.deviceMutex.Lock()
	defer c.deviceMutex.Unlock()

	installation := c.installationLocked(appData.PackageID)
	installation.InstallationAuthentication = &firebase_api.FirebaseAuthentication{
		AccessToken:  result.AuthToken.Token,
		Expires:      timestamppb.New(time.Now().Add(expiresIn)),
		RefreshToken: result.RefreshToken,
		IdToken:      "",
	}
	// FID in should equal FID out, but that is not always the case, override with FID out to be sure
	installation.FirebaseInstallationID = result.FID

	c.Logger().InfoContext(ctx, "firebase installation stored",
		slog.String("package_id", appData.PackageID),
		slog.String("fid", result.FID),
		slog.Duration("valid_for", expiresIn),
	)
	return result, nil
}

// InstallationExpired reports whether the stored installation auth token for a package is
// missing or (nearly) expired, and therefore whether NotifyInstallation should run again.
func (c *FireBaseClient) InstallationExpired(packageID string) bool {
	c.deviceMutex.Lock()
	defer c.deviceMutex.Unlock()

	installation, ok := c.Device.Installation(packageID)
	if !ok || installation.InstallationAuthentication == nil {
		return true
	}
	auth := installation.InstallationAuthentication
	if auth.AccessToken == "" || auth.Expires == nil {
		return true
	}
	// Treat a token that expires within a minute as expired, it would not survive the call.
	return time.Now().Add(time.Minute).After(auth.Expires.AsTime())
}

func (c *FireBaseClient) VerifyPassword(ctx context.Context, data *firebase_api.VerifyPasswordRequestBody, appData *firebase_api.FirebaseAppData) (result *firebase_api.GoogleVerifyPasswordResponse, err error) {
	start := time.Now()
	defer func() { c.logOperation(ctx, "VerifyPassword", start, err, logAttr("app", appData)) }()

	req, err := firebase_api.VerifyPasswordRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.VerifyPasswordRequest: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	result, err = firebase_api.VerifyPasswordResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.VerifyPasswordResult: %w", err)
	}
	return result, nil
}

func (c *FireBaseClient) SetAccountInfo(ctx context.Context, appData *firebase_api.FirebaseAppData, data *firebase_api.SetAccountInfoRequestBody) (result *firebase_api.GoogleSetAccountInfoResponse, err error) {
	start := time.Now()
	defer func() { c.logOperation(ctx, "SetAccountInfo", start, err, logAttr("app", appData)) }()

	req, err := firebase_api.SetAccountInfoRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.SetAccountInfoRequest: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	result, err = firebase_api.SetAccountInfoResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.SetAccountInfoResult: %w", err)
	}
	return result, nil
}

func (c *FireBaseClient) SignUpNewUser(ctx context.Context, appData *firebase_api.FirebaseAppData, data *firebase_api.SignUpNewUserRequestBody) (result *firebase_api.GoogleSignUpNewUserResponse, err error) {
	start := time.Now()
	defer func() { c.logOperation(ctx, "SignUpNewUser", start, err, logAttr("app", appData)) }()

	req, err := firebase_api.SignUpNewUser(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.SignUpNewUser: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	result, err = firebase_api.SignUpNewUserResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.SignUpNewUserResult: %w", err)
	}
	return result, nil
}

func (c *FireBaseClient) RefreshSecureToken(ctx context.Context, appData *firebase_api.FirebaseAppData, data *firebase_api.RefreshSecureTokenRequestBody) (result *firebase_api.SecureTokenRefreshResponse, err error) {
	start := time.Now()
	defer func() { c.logOperation(ctx, "RefreshSecureToken", start, err, logAttr("app", appData)) }()

	req, err := firebase_api.RefreshSecureTokenRequest(ctx, c.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.RefreshSecureTokenRequest: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	result, err = firebase_api.RefreshSecureTokenResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.RefreshSecureTokenResult: %w", err)
	}
	return result, nil
}

func (c *FireBaseClient) Auth(ctx context.Context, appData *firebase_api.FirebaseAppData, data url.Values, email, token string) (result *firebase_api.AuthResponse, err error) {
	start := time.Now()
	defer func() {
		c.logOperation(ctx, "Auth", start, err,
			logAttr("app", appData),
			slog.String("service", data.Get("service")),
			logAttr("result", result),
		)
	}()

	if err = c.Device.Validate(); err != nil {
		return nil, err
	}
	if data == nil {
		data = url.Values{}
	}
	if email != "" {
		data["Email"] = []string{email}
	}
	if token != "" {
		data["Token"] = []string{token}
	}

	req, err := firebase_api.AuthRequest(ctx, c.Device.Device, appData, data)
	if err != nil {
		return nil, fmt.Errorf("api.AuthRequest: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	result, err = firebase_api.AuthResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.AuthResult: %w", err)
	}
	return result, nil
}

// Checkin performs an Android check-in and stores the resulting credentials on the device.
func (c *FireBaseClient) Checkin(ctx context.Context, appData *firebase_api.FirebaseAppData, digest, otaCert string, accountCookies ...string) (result *firebase_api.CheckinResponse, err error) {
	start := time.Now()
	defer func() { c.logOperation(ctx, "Checkin", start, err) }()

	req, err := firebase_api.CheckinAndroidRequest(ctx, c.Device, appData, digest, otaCert, accountCookies...)
	if err != nil {
		return nil, fmt.Errorf("api.CheckinAndroidRequest: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}

	// CheckinResult guarantees a non-nil android id and security token.
	result, err = firebase_api.CheckinResult(resp)
	if err != nil {
		return nil, fmt.Errorf("api.CheckinResult: %w", err)
	}

	c.deviceMutex.Lock()
	c.Device.CheckinAndroidID = int64(*result.AndroidId)
	c.Device.CheckinSecurityToken = *result.SecurityToken
	c.deviceMutex.Unlock()

	c.Logger().InfoContext(ctx, "device checked in",
		slog.Int64("android_id", int64(*result.AndroidId)),
		slog.String("security_token", strconv.FormatUint(*result.SecurityToken, 10)),
	)
	return result, nil
}

// C2DMRegisterAndroid obtains an FCM notification token for the app and stores it on the device.
func (c *FireBaseClient) C2DMRegisterAndroid(ctx context.Context, appData *firebase_api.FirebaseAppData) (result string, err error) {
	start := time.Now()
	defer func() { c.logOperation(ctx, "C2DMRegisterAndroid", start, err, logAttr("app", appData)) }()

	req, err := firebase_api.C2DMAndroidRegisterRequest(ctx, c.Device, appData)
	if err != nil {
		return "", fmt.Errorf("api.C2DMAndroidRegisterRequest: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return "", err
	}

	result, err = firebase_api.AndroidRegisterResult(resp)
	if err != nil {
		return "", fmt.Errorf("api.AndroidRegisterResult: %w", err)
	}

	c.deviceMutex.Lock()
	c.installationLocked(appData.PackageID).NotificationData.NotificationToken = result
	c.deviceMutex.Unlock()

	c.Logger().InfoContext(ctx, "notification token obtained",
		slog.String("package_id", appData.PackageID),
		slog.String("token", result),
	)
	return result, nil
}

// C2DMRegisterWeb obtains a web push subscription token and stores it under the package ID.
func (c *FireBaseClient) C2DMRegisterWeb(ctx context.Context, appData *firebase_api.FirebaseAppData, sender, subtype, appid string) (result string, err error) {
	start := time.Now()
	defer func() {
		c.logOperation(ctx, "C2DMRegisterWeb", start, err, logAttr("app", appData), slog.String("subtype", subtype))
	}()

	req, err := firebase_api.C2DMWebRegisterRequest(ctx, c.Device, appData, sender, subtype, appid)
	if err != nil {
		return "", fmt.Errorf("api.C2DMWebRegisterRequest: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return "", err
	}

	result, err = firebase_api.AndroidRegisterResult(resp)
	if err != nil {
		return "", fmt.Errorf("api.AndroidRegisterResult: %w", err)
	}

	c.deviceMutex.Lock()
	c.installationLocked(appData.PackageID).NotificationData.NotificationToken = result
	c.deviceMutex.Unlock()
	return result, nil
}

// RegisterOptions tunes RegisterForNotifications.
type RegisterOptions struct {
	// Retries is how often a retryable registration failure is retried. Right after a
	// check-in, GCM commonly answers PHONE_REGISTRATION_ERROR for a few seconds.
	Retries int
	// RetryDelay is the wait between attempts, it doubles on every retry.
	RetryDelay time.Duration
	// ForceCheckin performs a check-in even when the device already has credentials.
	ForceCheckin bool
}

// DefaultRegisterOptions are the values RegisterForNotifications uses when passed nil.
func DefaultRegisterOptions() *RegisterOptions {
	return &RegisterOptions{Retries: 4, RetryDelay: 3 * time.Second}
}

// RegisterForNotifications runs the full "give me a push token" flow: it installs the app
// with Firebase, checks the device in when needed, and registers with GCM, retrying the
// registration while the failure is a transient one.
//
// This replaces having to call NotifyInstallation, Checkin and C2DMRegisterAndroid by hand
// in the right order with the right sleeps in between.
func (c *FireBaseClient) RegisterForNotifications(ctx context.Context, appData *firebase_api.FirebaseAppData, opts *RegisterOptions) (token string, err error) {
	start := time.Now()
	defer func() { c.logOperation(ctx, "RegisterForNotifications", start, err, logAttr("app", appData)) }()

	if opts == nil {
		opts = DefaultRegisterOptions()
	}
	if err = appData.Validate(); err != nil {
		return "", err
	}

	logger := c.Logger()

	if c.InstallationExpired(appData.PackageID) {
		logger.DebugContext(ctx, "installation missing or expired, installing", slog.String("package_id", appData.PackageID))
		if _, err := c.NotifyInstallation(ctx, appData); err != nil {
			return "", fmt.Errorf("c.NotifyInstallation: %w", err)
		}
	}

	if opts.ForceCheckin || !c.Device.HasCheckin() {
		logger.DebugContext(ctx, "checking device in", slog.Bool("forced", opts.ForceCheckin))
		if _, err := c.Checkin(ctx, appData, "", ""); err != nil {
			return "", fmt.Errorf("c.Checkin: %w", err)
		}
	}

	delay := opts.RetryDelay
	if delay <= 0 {
		delay = DefaultRegisterOptions().RetryDelay
	}

	var lastErr error
	for attempt := 0; attempt <= opts.Retries; attempt++ {
		if attempt > 0 {
			logger.WarnContext(ctx, "registration failed, retrying",
				slog.Int("attempt", attempt),
				slog.Int("of", opts.Retries+1),
				slog.Duration("waiting", delay),
				slog.String("error", lastErr.Error()),
			)
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("registration aborted after %d attempt(s): %w (last error: %v)", attempt, ctx.Err(), lastErr)
			case <-time.After(delay):
			}
			delay *= 2
		}

		token, err := c.C2DMRegisterAndroid(ctx, appData)
		if err == nil {
			return token, nil
		}
		lastErr = err
		if !firebase_api.IsRetryable(err) {
			return "", err
		}

		// Rejected credentials will keep being rejected, a fresh check-in is the only
		// thing that can turn this attempt into a successful one.
		registerErr := new(firebase_api.RegisterError)
		if errors.As(err, &registerErr) && registerErr.Code == firebase_api.RegisterErrorAuthentication {
			logger.WarnContext(ctx, "check-in credentials rejected, checking in again",
				logAttr("register_error", registerErr))
			if _, checkinErr := c.Checkin(ctx, appData, "", ""); checkinErr != nil {
				return "", fmt.Errorf("c.Checkin (after %w): %w", err, checkinErr)
			}
		}
	}
	return "", fmt.Errorf("registration failed after %d attempt(s): %w", opts.Retries+1, lastErr)
}

// NotificationToken returns the stored notification token for a package, if any.
func (c *FireBaseClient) NotificationToken(packageID string) (string, bool) {
	c.deviceMutex.Lock()
	defer c.deviceMutex.Unlock()
	return c.Device.NotificationToken(packageID)
}

// Close releases the resources the client holds, currently the MTalk connection.
func (c *FireBaseClient) Close() error {
	if c.MTalk == nil {
		return nil
	}
	err := c.MTalk.Close()
	if errors.Is(err, firebase_api.ErrNotConnected) {
		return nil
	}
	return err
}
