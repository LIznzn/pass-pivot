package geoip

import (
	"net"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

const defaultMaxmindDBPath = "provider/geoip/resource/GeoLite2-City.mmdb"

type MaxmindGeoipProvider struct {
	once   sync.Once
	db     *maxminddb.Reader
	dbPath string
}

type maxmindCityRecord struct {
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

func NewMaxmindGeoipProvider(dbPath ...string) *MaxmindGeoipProvider {
	provider := &MaxmindGeoipProvider{}
	if len(dbPath) > 0 {
		provider.dbPath = dbPath[0]
	}
	return provider
}

func (provider *MaxmindGeoipProvider) LookupLocation(ipAddress string) (string, error) {
	ip := parseGeoIPInput(ipAddress)
	if ip == nil {
		return "", nil
	}
	if fallback := classifyLocalIP(ip); fallback != "" {
		return fallback, nil
	}
	db := provider.load()
	if db == nil {
		return "", nil
	}
	var record maxmindCityRecord
	if err := db.Lookup(ip, &record); err != nil {
		return "", nil
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
	return strings.Join(parts, ", "), nil
}

func (provider *MaxmindGeoipProvider) load() *maxminddb.Reader {
	provider.once.Do(func() {
		if strings.TrimSpace(provider.dbPath) == "" {
			provider.dbPath = defaultMaxmindDBPath
		}
		reader, err := maxminddb.Open(filepath.Clean(provider.dbPath))
		if err == nil {
			provider.db = reader
		}
	})
	return provider.db
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
