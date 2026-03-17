package service

import (
	externalgeoip "pass-pivot/external/geoip"
)

type GeoIPService struct {
	provider externalgeoip.GeoipProvider
}

func NewGeoIPService(dbPath string) *GeoIPService {
	return &GeoIPService{
		provider: externalgeoip.NewMaxmindGeoipProvider(dbPath),
	}
}

func (s *GeoIPService) Resolve(ipAddress string) string {
	if s == nil || s.provider == nil {
		return ""
	}
	location, _ := s.provider.LookupLocation(ipAddress)
	return location
}
