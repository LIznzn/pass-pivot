package geoip

import "fmt"

type GeoipProvider interface {
	LookupLocation(ipAddress string) (string, error)
}

func GetGeoipProvider(geoipType string) GeoipProvider {
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
