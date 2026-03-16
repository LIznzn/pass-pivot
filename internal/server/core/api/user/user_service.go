package user

import (
	apimanage "pass-pivot/internal/server/core/api/manage"
)

type Service struct {
	platform *apimanage.Service
}

func NewService(platform *apimanage.Service) *Service {
	return &Service{platform: platform}
}
