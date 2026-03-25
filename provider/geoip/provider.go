package geoip

import "fmt"

type Provider interface {
	LookupLocation(ipAddress string) (string, error)
}

func GetGeoipProvider(geoipType string) Provider {
	switch geoipType {
	default:
		return nil
	case "MaxMind GeoLite":
		return NewMaxmindGeoipProvider()
	}
}

func VerifyCaptchaByCaptchaType(geoipType, ipAddress string) (string, error) {
	provider := GetGeoipProvider(geoipType)
	if provider == nil {
		return "", fmt.Errorf("invalid geoip provider: %s", geoipType)
	}

	return provider.LookupLocation(ipAddress)
}
