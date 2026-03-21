package service

import "strings"

const (
	ApplicationMetadataDisplayName     = "displayName"
	ApplicationMetadataDisplayNameEN   = "displayName.en"
	ApplicationMetadataDisplayNameJA   = "displayName.ja"
	ApplicationMetadataDisplayNameZHS  = "displayName.zhs"
	ApplicationMetadataDisplayNameZHT  = "displayName.zht"
)

func defaultApplicationMetadata() map[string]string {
	return map[string]string{
		ApplicationMetadataDisplayName:    "",
		ApplicationMetadataDisplayNameEN:  "",
		ApplicationMetadataDisplayNameJA:  "",
		ApplicationMetadataDisplayNameZHS: "",
		ApplicationMetadataDisplayNameZHT: "",
	}
}

func NormalizeApplicationMetadata(candidate map[string]string, fallback map[string]string) map[string]string {
	result := defaultApplicationMetadata()
	for key, value := range fallback {
		if strings.TrimSpace(key) == "" {
			continue
		}
		result[key] = value
	}
	for key, value := range candidate {
		if strings.TrimSpace(key) == "" {
			continue
		}
		result[key] = value
	}
	return result
}

func BuildApplicationDisplayNameMap(metadata map[string]string) map[string]string {
	normalized := NormalizeApplicationMetadata(metadata, nil)
	return map[string]string{
		"default": ApplicationDisplayNameForLocale(normalized, "", ""),
		"en":      ApplicationDisplayNameForLocale(normalized, "en", ""),
		"ja":      ApplicationDisplayNameForLocale(normalized, "ja", ""),
		"zhs":     ApplicationDisplayNameForLocale(normalized, "zhs", ""),
		"zht":     ApplicationDisplayNameForLocale(normalized, "zht", ""),
	}
}

func ApplicationDisplayNameForLocale(metadata map[string]string, locale string, fallback string) string {
	normalized := NormalizeApplicationMetadata(metadata, nil)
	switch strings.TrimSpace(locale) {
	case "en", "en-US":
		if value := strings.TrimSpace(normalized[ApplicationMetadataDisplayNameEN]); value != "" {
			return value
		}
	case "ja", "ja-JP":
		if value := strings.TrimSpace(normalized[ApplicationMetadataDisplayNameJA]); value != "" {
			return value
		}
	case "zhs", "zh-CN":
		if value := strings.TrimSpace(normalized[ApplicationMetadataDisplayNameZHS]); value != "" {
			return value
		}
	case "zht", "zh-TW", "zh-HK":
		if value := strings.TrimSpace(normalized[ApplicationMetadataDisplayNameZHT]); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(normalized[ApplicationMetadataDisplayName]); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}
