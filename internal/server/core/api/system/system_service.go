package system

import (
	"context"

	"pass-pivot/internal/model"
	apimanage "pass-pivot/internal/server/core/api/manage"
	coreservice "pass-pivot/internal/server/core/service"
)

type Service struct {
	manage *apimanage.Service
}

func NewService(manage *apimanage.Service) *Service {
	return &Service{manage: manage}
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, accessToken string) (*coreservice.AccessTokenIdentity, error) {
	return s.manage.AuthenticateAccessToken(ctx, accessToken)
}

func (s *Service) ValidateConsoleAccessToken(ctx context.Context, accessToken string) (*coreservice.AccessTokenIdentity, error) {
	return s.manage.ValidateConsoleAccessToken(ctx, accessToken)
}

func (s *Service) IntrospectToken(ctx context.Context, token string) (map[string]any, error) {
	return s.manage.IntrospectToken(ctx, token)
}

func (s *Service) ListPublicExternalIDPsByApplication(ctx context.Context, applicationID string) ([]model.ExternalIDP, error) {
	return s.manage.ListPublicExternalIDPsByApplication(ctx, applicationID)
}

func (s *Service) GetLoginTarget(ctx context.Context, applicationID string) (*coreservice.LoginTarget, error) {
	return s.manage.GetLoginTarget(ctx, applicationID)
}
