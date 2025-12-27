package whatsapp

import (
	"math/rand"
	"time"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"google.golang.org/protobuf/proto"
)

type BrowserSignature struct {
	PlatformType waCompanionReg.DeviceProps_PlatformType
	OS           string
	OSVersion    [3]uint32
	Manufacturer string
	Device       string
}

var Signatures = []BrowserSignature{
	{
		PlatformType: waCompanionReg.DeviceProps_CHROME,
		OS:           "Windows",
		OSVersion:    [3]uint32{10, 0, 19045},
		Manufacturer: "Microsoft",
		Device:       "Windows",
	},
	{
		PlatformType: waCompanionReg.DeviceProps_EDGE,
		OS:           "Windows",
		OSVersion:    [3]uint32{11, 0, 22621},
		Manufacturer: "Microsoft",
		Device:       "Windows",
	},
	{
		PlatformType: waCompanionReg.DeviceProps_SAFARI,
		OS:           "macOS",
		OSVersion:    [3]uint32{17, 0, 0},
		Manufacturer: "Apple",
		Device:       "Mac",
	},
	{
		PlatformType: waCompanionReg.DeviceProps_CHROME,
		OS:           "macOS",
		OSVersion:    [3]uint32{14, 1, 0},
		Manufacturer: "Apple",
		Device:       "Mac",
	},
	{
		PlatformType: waCompanionReg.DeviceProps_FIREFOX,
		OS:           "Linux",
		OSVersion:    [3]uint32{120, 0, 0},
		Manufacturer: "Ubuntu",
		Device:       "Linux",
	},
	{
		PlatformType: waCompanionReg.DeviceProps_CHROME,
		OS:           "Linux",
		OSVersion:    [3]uint32{120, 0, 0},
		Manufacturer: "Debian",
		Device:       "Linux",
	},
	{
		PlatformType: waCompanionReg.DeviceProps_FIREFOX,
		OS:           "Windows",
		OSVersion:    [3]uint32{10, 0, 0},
		Manufacturer: "Microsoft",
		Device:       "Windows",
	},
	{
		PlatformType: waCompanionReg.DeviceProps_OPERA,
		OS:           "Windows",
		OSVersion:    [3]uint32{10, 0, 0},
		Manufacturer: "Microsoft",
		Device:       "Windows",
	},
}

// GetRandomSignature returns a random signature from the pool
func GetRandomSignature() BrowserSignature {
	return GetSignatureForSeed(time.Now().String())
}

// GetSignatureForSeed returns a consistent signature based on a seed string
func GetSignatureForSeed(seed string) BrowserSignature {
	if seed == "" {
		seed = time.Now().String()
	}

	// Create a simple hash from the seed
	var hash uint64
	for i := 0; i < len(seed); i++ {
		hash = uint64(seed[i]) + (hash << 6) + (hash << 16) - hash
	}

	r := rand.New(rand.NewSource(int64(hash)))
	return Signatures[r.Intn(len(Signatures))]
}

// ApplySignature applies the signature to the whatsmeow store
func ApplySignature(sig BrowserSignature) {
	store.DeviceProps.PlatformType = sig.PlatformType.Enum()
	store.DeviceProps.Os = proto.String(sig.OS)
	store.SetOSInfo(sig.OS, sig.OSVersion)

	store.BaseClientPayload.UserAgent.Manufacturer = proto.String(sig.Manufacturer)
	store.BaseClientPayload.UserAgent.Device = proto.String(sig.Device)
}
