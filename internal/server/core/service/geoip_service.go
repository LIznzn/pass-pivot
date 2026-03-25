package service

import (
	providergeoip "pass-pivot/provider/geoip"
)

type GeoIPService struct {
	provider providergeoip.Provider
}

func NewGeoIPService(dbPath string) *GeoIPService {
	return &GeoIPService{
		provider: providergeoip.NewMaxmindGeoipProvider(dbPath),
	}
}

func (s *GeoIPService) Resolve(ipAddress string) string {
	if s == nil || s.provider == nil {
		return ""
	}
	location, _ := s.provider.LookupLocation(ipAddress)
	return location
}
