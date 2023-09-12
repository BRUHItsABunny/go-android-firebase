package firebase_api

import (
	andutils "github.com/BRUHItsABunny/go-android-utils"
	"google.golang.org/protobuf/proto"
	"math/rand"
	"time"
)

func NewCheckinRequest(device *andutils.Device) *CheckinRequest {
	currentTimeStamp := proto.Int64(time.Now().UnixMilli())
	hni := "000000"
	if len(device.SimSlots) > 0 {
		hni = device.SimSlots[0].GetHNI()
	}
	result := &CheckinRequest{
		AndroidId: proto.Int64(0),
		Digest:    proto.String("1-929a0dca0eee55513280171a8585da7dcd3700f8"), // INITIAL_DIGEST src: https://github.com/microg/GmsCore/blob/4a5c98491bcfe4754b3efcfed20f3ada75a6ebec/play-services-base-core/src/main/kotlin/org/microg/gms/settings/SettingsContract.kt
		Checkin: &CheckinRequest_Checkin{
			Build: &CheckinRequest_Checkin_Build{
				Fingerprint:  proto.String(device.GetFingerprint()),
				Brand:        proto.String(device.Manufacturer),
				Bootloader:   proto.String(device.IncrementalVersion), // Technically inaccurate
				ClientId:     proto.String("android-google"),
				Time:         currentTimeStamp,
				Device:       proto.String(device.Device),
				SdkVersion:   proto.Int32(int32(device.Version)),
				Model:        proto.String(device.Model),
				Manufacturer: proto.String(device.Manufacturer),
				Product:      proto.String(device.Product),
				OtaInstalled: proto.Bool(false),
			},
			LastCheckinMs: proto.Int64(0),
			Event: []*CheckinRequest_Checkin_Event{{
				Tag:    proto.String("event_log_start"),
				TimeMs: currentTimeStamp,
			}},
			CellOperator: &hni,
			SimOperator:  &hni,
			Roaming:      proto.String("mobile-notroaming"),
			UserNumber:   proto.Int32(0),
		},
		Locale:        proto.String(device.Locale.ToLocale("_", true)),
		LoggingId:     proto.Int64(rand.Int63()), // Randomize?
		MacAddress:    []string{device.MacAddress.Address},
		Meid:          proto.String(device.SimSlots[0].Imei.Imei), // TODO: use MEID instead
		AccountCookie: []string{""},
		TimeZone:      proto.String(device.Timezone.GetName()),
		Version:       proto.Int32(3),
		OtaCert:       []string{"--no-output--"}, // Randomize? 18 bytes base64 no padding or "--no-output--"
		DeviceConfiguration: &CheckinRequest_DeviceConfig{
			TouchScreen:          proto.Int32(3),
			KeyboardType:         proto.Int32(1),
			Navigation:           proto.Int32(1),
			ScreenLayout:         proto.Int32(2),
			HasHardKeyboard:      proto.Bool(false),
			HasFiveWayNavigation: proto.Bool(false),
			DensityDpi:           proto.Int32(device.DPI),
			GlEsVersion:          proto.Int32(196610), // Make configurable?
			NativePlatform:       device.AbiList,
			WidthPixels:          proto.Int32(device.ResolutionHorizontal),
			HeightPixels:         proto.Int32(device.ResolutionVertical),
			Locale:               nil,
			GlExtension:          nil,
			SharedLibrary:        nil,
		},
		MacAddressType: []string{"wifi"},
		Fragment:       proto.Int32(0),
	}

	return result
}
