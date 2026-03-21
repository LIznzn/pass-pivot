package fido

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"pass-pivot/internal/model"
)

func SyncCredentialEnrollments(ctx context.Context, db *gorm.DB, user model.User) error {
	if user.ID == "" || user.OrganizationID == "" {
		return errors.New("user and organization are required")
	}
	var passkeyCount int64
	if err := db.WithContext(ctx).Model(&model.SecureKey{}).
		Where("user_id = ? AND webauthn_enable = ? AND deleted_at IS NULL", user.ID, true).
		Count(&passkeyCount).Error; err != nil {
		return err
	}
	var u2fCount int64
	if err := db.WithContext(ctx).Model(&model.SecureKey{}).
		Where("user_id = ? AND u2f_enable = ? AND deleted_at IS NULL", user.ID, true).
		Count(&u2fCount).Error; err != nil {
		return err
	}
	if err := upsertCredentialEnrollment(ctx, db, user, []string{"webauthn"}, "webauthn", "通行密钥", passkeyCount > 0); err != nil {
		return err
	}
	return upsertCredentialEnrollment(ctx, db, user, []string{"u2f"}, "u2f", "安全密钥", u2fCount > 0)
}

func upsertCredentialEnrollment(ctx context.Context, db *gorm.DB, user model.User, lookupMethods []string, targetMethod, label string, enabled bool) error {
	var enrollment model.MFAEnrollment
	err := db.WithContext(ctx).
		Where("user_id = ? AND method IN ?", user.ID, lookupMethods).
		Order("created_at desc").
		First(&enrollment).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		status := "disabled"
		if enabled {
			status = "active"
		}
		enrollment = model.MFAEnrollment{
			OrganizationID: user.OrganizationID,
			UserID:         user.ID,
			Method:         targetMethod,
			Label:          label,
			Status:         status,
		}
		return db.WithContext(ctx).Create(&enrollment).Error
	}
	status := "disabled"
	if enabled {
		status = "active"
	}
	updates := map[string]any{
		"label":  label,
		"method": targetMethod,
		"status": status,
	}
	return db.WithContext(ctx).Model(&enrollment).Updates(updates).Error
}
