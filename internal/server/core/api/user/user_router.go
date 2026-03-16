package user

import (
	"net/http"

	authhandler "pass-pivot/internal/server/auth/handler"
)

func RegisterRoutes(mux *http.ServeMux, user *Handler, authn authnHandler, passkey *authhandler.PasskeyHandler) {
	mux.HandleFunc("POST /api/user/v1/profile/query", user.GetCurrentUserProfile)
	mux.HandleFunc("POST /api/user/v1/profile/update", user.UpdateCurrentUserProfile)
	mux.HandleFunc("POST /api/user/v1/detail/query", user.GetCurrentUserDetail)
	mux.HandleFunc("POST /api/user/v1/setting/query", user.GetCurrentUserSetting)
	mux.HandleFunc("POST /api/user/v1/setting/update", user.UpdateCurrentUserSetting)
	mux.HandleFunc("POST /api/user/v1/device/untrust", user.UntrustCurrentDevice)
	mux.HandleFunc("POST /api/user/v1/mfa_method/update", user.UpdateCurrentUserMFAMethod)
	mux.HandleFunc("POST /api/user/v1/mfa_enrollment/delete", user.DeleteCurrentUserMFAEnrollment)
	mux.HandleFunc("POST /api/user/v1/passkey/delete", user.DeleteCurrentUserPasskey)
	mux.HandleFunc("POST /api/user/v1/external_identity_binding/create", user.CreateCurrentExternalIdentityBinding)
	mux.HandleFunc("POST /api/user/v1/external_identity_binding/delete", user.DeleteCurrentExternalIdentityBinding)
	mux.HandleFunc("POST /api/user/v1/totp/enroll", authn.EnrollPortalTOTP)
	mux.HandleFunc("POST /api/user/v1/totp/verify", authn.VerifyPortalTOTPEnrollment)
	mux.HandleFunc("POST /api/user/v1/recovery_code/generate", authn.GeneratePortalRecoveryCodes)
	mux.HandleFunc("POST /api/user/v1/passkey/register/begin", passkey.BeginPortalRegistration)
	mux.HandleFunc("POST /api/user/v1/passkey/register/finish", passkey.FinishRegistration)
	mux.HandleFunc("POST /api/user/v1/reset_ukid", authn.ResetUKID)
}

type authnHandler interface {
	EnrollPortalTOTP(http.ResponseWriter, *http.Request)
	VerifyPortalTOTPEnrollment(http.ResponseWriter, *http.Request)
	GeneratePortalRecoveryCodes(http.ResponseWriter, *http.Request)
	ResetUKID(http.ResponseWriter, *http.Request)
}
