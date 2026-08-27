package firebase_api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/BRUHItsABunny/go-android-firebase/constants"
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"github.com/google/uuid"
)

func DefaultHeadersFirebase(device *FirebaseDevice, appData *FirebaseAppData, includeAPIKey, includeFireBaseClient, includeClientVersion bool) http.Header {
	headers := http.Header{
		constants.HeaderKeyContentType:    {constants.HeaderValueMIMEJSON},
		constants.HeaderKeyAccept:         {constants.HeaderValueMIMEJSON},
		constants.HeaderKeyCacheControl:   {"no-cache"},
		constants.HeaderKeyAndroidCert:    {strings.ToUpper(appData.PackageCertificate)},
		constants.HeaderKeyAndroidPackage: {appData.PackageID},
		constants.HeaderKeyUserAgent:      {"Dalvik/2.1.0 " + device.Device.GetUserAgent()},
	}

	if includeAPIKey {
		headers[constants.HeaderKeyGoogAPIKey] = []string{appData.GoogleAPIKey}
	}
	if includeFireBaseClient {
		headers[constants.HeaderKeyFireBaseClient] = []string{device.Device.FormatUserAgent(constants.HeaderValueFireBaseClient)}
	}
	if includeClientVersion {
		// TODO: Constant or variable?
		headers[constants.HeaderKeyClientVersion] = []string{constants.HeaderValueClientVersion}
	}
	return headers
}

func DefaultHeadersAuth(device *andutils.Device) http.Header {
	headers := http.Header{
		constants.HeaderKeyContentType: {"application/x-www-form-urlencoded"},
		"app":                          {"com.google.android.gm"},
		"device":                       {device.Id.ToHexString()},
		constants.HeaderKeyUserAgent:   {device.FormatUserAgent(fmt.Sprintf("GoogleAuth/1.4 (%s %s); gzip", andutils.DeviceFormatKeyModel, andutils.DeviceFormatKeyBuild))},
	}

	return headers
}

func DefaultHeadersCheckin(device *andutils.Device) http.Header {
	headers := http.Header{
		constants.HeaderKeyContentType: {"application/x-protobuffer"},
		constants.HeaderKeyUserAgent:   {device.FormatUserAgent(fmt.Sprintf("Android-Checkin/2.0 (%s %s); gzip", andutils.DeviceFormatKeyDevice, andutils.DeviceFormatKeyBuild))},
	}

	return headers
}

func DefaultHeadersAndroidRegister(device *FirebaseDevice) http.Header {
	headers := http.Header{
		constants.HeaderKeyContentType: {"application/x-www-form-urlencoded"},
		constants.HeaderKeyUserAgent:   {device.Device.FormatUserAgent(fmt.Sprintf("Android-GCM/1.5 (%s %s)", andutils.DeviceFormatKeyDevice, andutils.DeviceFormatKeyBuild))},
		"Authorization":                {fmt.Sprintf("AidLogin %d:%d", device.CheckinAndroidID, device.CheckinSecurityToken)},
	}
	return headers
}

// FailSafeRandomAppFID generates a FID without going through the UUID bit dance.
// It is used as a fallback when RandomAppFID cannot obtain a random UUID.
func FailSafeRandomAppFID() (string, error) {
	// Mimic: https://firebase.google.com/docs/reference/android/com/google/firebase/installations/FirebaseInstallations#public-taskstring-getid
	// url-safe base64 of 128bit integer as bytes, our approach 16 random bytes
	fakeInt := make([]byte, 16)
	if _, err := rand.Read(fakeInt); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(fakeInt), nil
}

// GetLeastMostSignificantBits src: (Python) https://stackoverflow.com/a/40141219
func GetLeastMostSignificantBits(uuidStr string) (int64, int64) {
	uuidSplit := strings.Split(uuidStr, "-")
	leastStr, mostStr := "", ""
	for i, subStr := range uuidSplit {
		if i <= 2 {
			mostStr += subStr
		} else {
			leastStr += subStr
		}
	}

	if leastStr == "" || mostStr == "" {
		// Not a UUID, nothing sensible to derive from it.
		return 0, 0
	}

	subTractor := new(big.Int)
	subTractor.SetString("10000000000000000", 16)
	leastBitsB := new(big.Int)
	leastBitsB.SetString(leastStr, 16)
	mostBitsB := new(big.Int)
	mostBitsB.SetString(mostStr, 16)

	leastBitsSign, _ := strconv.ParseInt(string(leastStr[0]), 16, 64)
	mostBitsSign, _ := strconv.ParseInt(string(mostStr[0]), 16, 64)

	var (
		leastBits, mostBits int64
	)
	if leastBitsSign > 7 {
		leastPostB := big.NewInt(0).Sub(leastBitsB, subTractor)
		leastBits = leastPostB.Int64()
	} else {
		leastBits = leastBitsB.Int64()
	}

	if mostBitsSign > 7 {
		mostPostB := big.NewInt(0).Sub(mostBitsB, subTractor)
		mostBits = mostPostB.Int64()
	} else {
		mostBits = mostBitsB.Int64()
	}

	return leastBits, mostBits
}

// RandomAppFID generates a Firebase Installation ID the same way the Android SDK does.
// It only returns an error when no source of randomness is available at all, a failure
// to generate the UUID falls back to FailSafeRandomAppFID instead of failing the call.
func RandomAppFID() (string, error) {
	// Mimic: https://github.com/firebase/firebase-android-sdk/blob/9dec703a90767e1646466d2786ddda2b25201d84/firebase-installations/src/main/java/com/google/firebase/installations/RandomFidGenerator.java#L23
	// Fixes FID in != FID out, YAY!
	uuidObj, err := uuid.NewRandom()
	if err != nil {
		fid, fallbackErr := FailSafeRandomAppFID()
		if fallbackErr != nil {
			return "", fmt.Errorf("uuid.NewRandom: %w (fallback: %v)", err, fallbackErr)
		}
		return fid, nil
	}

	leastBits, mostBits := GetLeastMostSignificantBits(uuidObj.String())
	uuidBytes := make([]byte, 17)
	// src: https://docs.oracle.com/javase/7/docs/api/java/nio/ByteBuffer.html
	// "The order of a newly-created byte buffer is always BIG_ENDIAN."
	binary.BigEndian.PutUint64(uuidBytes[:8], uint64(mostBits))
	binary.BigEndian.PutUint64(uuidBytes[8:16], uint64(leastBits))
	uuidBytes[16] = uuidBytes[0]
	// Byte.parseByte("00001111", 2); = 15
	// Byte.parseByte("01110000", 2); = 112
	uuidBytes[0] = (15 & uuidBytes[0]) | 112
	return base64.URLEncoding.EncodeToString(uuidBytes)[:22], nil
}
