package auditlog

import (
	"pass-pivot/internal/model"
)

type FieldChange struct {
	Field  string `json:"field"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

type Detail struct {
	Request  *RequestMeta   `json:"request,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Changes  []FieldChange  `json:"changes,omitempty"`
}

type RequestMeta struct {
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

func BuildRequestMeta(method, path, ipAddress, userAgent string) *RequestMeta {
	if method == "" && path == "" && ipAddress == "" && userAgent == "" {
		return nil
	}
	return &RequestMeta{
		Method:    method,
		Path:      path,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
}

func InferTargetName(eventTargetName string, metadata map[string]any, target model.AuditLog) string {
	if eventTargetName != "" {
		return eventTargetName
	}
	for _, key := range []string{"name", "displayName", "username", "email", "identifier", "host", "subject"} {
		if value, ok := metadata[key]; ok {
			if text, ok := value.(string); ok && text != "" {
				return text
			}
		}
	}
	return target.TargetID
}
