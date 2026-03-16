package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"pass-pivot/internal/model"
)

func syncWebAuthnMFAEnrollments(ctx context.Context, db *gorm.DB, user model.User) error {
	if user.ID == "" || user.OrganizationID == "" {
		return errors.New("user and organization are required")
	}
	var passkeyCount int64
	if err := db.WithContext(ctx).Model(&model.MFAPasskey{}).
		Where("user_id = ? AND is_passkey = ? AND deleted_at IS NULL", user.ID, true).
		Count(&passkeyCount).Error; err != nil {
		return err
	}
	var u2fCount int64
	if err := db.WithContext(ctx).Model(&model.MFAPasskey{}).
		Where("user_id = ? AND is_u2f = ? AND deleted_at IS NULL", user.ID, true).
		Count(&u2fCount).Error; err != nil {
		return err
	}
	if err := upsertWebAuthnMFAEnrollment(ctx, db, user, "passkey", "通行密钥", passkeyCount > 0); err != nil {
		return err
	}
	return upsertWebAuthnMFAEnrollment(ctx, db, user, "u2f", "安全密钥", u2fCount > 0)
}

func SyncWebAuthnMFAEnrollments(ctx context.Context, db *gorm.DB, user model.User) error {
	return syncWebAuthnMFAEnrollments(ctx, db, user)
}

func upsertWebAuthnMFAEnrollment(ctx context.Context, db *gorm.DB, user model.User, method, label string, enabled bool) error {
	var enrollment model.MFAEnrollment
	err := db.WithContext(ctx).
		Where("user_id = ? AND method = ?", user.ID, method).
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
			Method:         method,
			Label:          label,
			Status:         status,
		}
		return db.WithContext(ctx).Create(&enrollment).Error
	}
	updates := map[string]any{"label": label}
	if !enabled {
		updates["status"] = "disabled"
	}
	return db.WithContext(ctx).Model(&enrollment).Updates(updates).Error
}
