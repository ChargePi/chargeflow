package ocpp

const (
	V15 = Version("1.5")
	V16 = Version("1.6")
	V20 = Version("2.0")
	V21 = Version("2.1")
)

type (
	// Version OCPP version of the central system or charge point
	Version string
)

func (p Version) String() string {
	return string(p)
}

func IsValidProtocolVersion(version Version) bool {
	switch version {
	case V15, V16,
		V20, V21:
		return true
	default:
		return false
	}
}
