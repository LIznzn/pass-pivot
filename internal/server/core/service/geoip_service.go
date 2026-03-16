package service

import (
	"net"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

type GeoIPService struct {
	once   sync.Once
	db     *maxminddb.Reader
	dbPath string
}

type geoIPCityRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	Country struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

func NewGeoIPService(dbPath string) *GeoIPService {
	return &GeoIPService{dbPath: dbPath}
}

func (s *GeoIPService) Resolve(ipAddress string) string {
	ip := parseGeoIPInput(ipAddress)
	if ip == nil {
		return ""
	}
	if fallback := classifyLocalIP(ip); fallback != "" {
		return fallback
	}
	db := s.load()
	if db == nil {
		return ""
	}
	var record geoIPCityRecord
	if err := db.Lookup(ip, &record); err != nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if city := resolveGeoName(record.City.Names); city != "" {
		parts = append(parts, city)
	}
	if len(record.Subdivisions) > 0 {
		if region := resolveGeoName(record.Subdivisions[0].Names); region != "" && !containsGeoPart(parts, region) {
			parts = append(parts, region)
		}
	}
	if country := resolveGeoName(record.Country.Names); country != "" && !containsGeoPart(parts, country) {
		parts = append(parts, country)
	}
	return strings.Join(parts, ", ")
}

func (s *GeoIPService) load() *maxminddb.Reader {
	s.once.Do(func() {
		if strings.TrimSpace(s.dbPath) == "" {
			s.dbPath = "external/ip/GeoLite2-City.mmdb"
		}
		reader, err := maxminddb.Open(filepath.Clean(s.dbPath))
		if err == nil {
			s.db = reader
		}
	})
	return s.db
}

func parseGeoIPInput(input string) net.IP {
	value := strings.TrimSpace(input)
	if value == "" {
		return nil
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	return net.ParseIP(value)
}

func resolveGeoName(names map[string]string) string {
	if len(names) == 0 {
		return ""
	}
	for _, key := range []string{"zh-CN", "en"} {
		if value := strings.TrimSpace(names[key]); value != "" {
			return value
		}
	}
	for _, value := range names {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func containsGeoPart(parts []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, part := range parts {
		if strings.EqualFold(strings.TrimSpace(part), target) {
			return true
		}
	}
	return false
}

func classifyLocalIP(ip net.IP) string {
	if ip.IsLoopback() {
		return "本机回环"
	}
	if ip.IsPrivate() {
		return "私有网络"
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return "链路本地"
	}
	if ip.IsUnspecified() {
		return "未指定地址"
	}
	if isUniqueLocalIPv6(ip) {
		return "本地 IPv6 网络"
	}
	return ""
}

func isUniqueLocalIPv6(ip net.IP) bool {
	ip = ip.To16()
	if ip == nil || ip.To4() != nil {
		return false
	}
	return ip[0]&0xfe == 0xfc
}
