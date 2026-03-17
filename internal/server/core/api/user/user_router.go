package user

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, user *Handler, authn authnHandler) {
	mux.HandleFunc("POST /api/user/v1/profile/query", user.GetCurrentUserProfile)
	mux.HandleFunc("POST /api/user/v1/profile/update", user.UpdateCurrentUserProfile)
	mux.HandleFunc("POST /api/user/v1/detail/query", user.GetCurrentUserDetail)
	mux.HandleFunc("POST /api/user/v1/setting/query", user.GetCurrentUserSetting)
	mux.HandleFunc("POST /api/user/v1/setting/update", user.UpdateCurrentUserSetting)
	mux.HandleFunc("POST /api/user/v1/device/untrust", user.UntrustCurrentDevice)
	mux.HandleFunc("POST /api/user/v1/mfa_method/update", user.UpdateCurrentUserMFAMethod)
	mux.HandleFunc("POST /api/user/v1/mfa_enrollment/delete", user.DeleteCurrentUserMFAEnrollment)
	mux.HandleFunc("POST /api/user/v1/securekey/delete", user.DeleteCurrentUserSecureKey)
	mux.HandleFunc("POST /api/user/v1/external_identity_binding/create", user.CreateCurrentExternalIdentityBinding)
	mux.HandleFunc("POST /api/user/v1/external_identity_binding/delete", user.DeleteCurrentExternalIdentityBinding)
	mux.HandleFunc("POST /api/user/v1/totp/enroll", authn.EnrollPortalTOTP)
	mux.HandleFunc("POST /api/user/v1/totp/verify", authn.VerifyPortalTOTPEnrollment)
	mux.HandleFunc("POST /api/user/v1/recovery_code/generate", authn.GeneratePortalRecoveryCodes)
	mux.HandleFunc("POST /api/user/v1/securekey/register/begin", user.BeginCurrentUserSecureKeyRegistration)
	mux.HandleFunc("POST /api/user/v1/securekey/register/finish", user.FinishCurrentUserSecureKeyRegistration)
	mux.HandleFunc("POST /api/user/v1/reset_ukid", authn.ResetUKID)
}

type authnHandler interface {
	EnrollPortalTOTP(http.ResponseWriter, *http.Request)
	VerifyPortalTOTPEnrollment(http.ResponseWriter, *http.Request)
	GeneratePortalRecoveryCodes(http.ResponseWriter, *http.Request)
	ResetUKID(http.ResponseWriter, *http.Request)
}
