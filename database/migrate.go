package database

import (
	"log"

	"gorm.io/gorm"

	"support-backend/internal/domain"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.UserProfile{},
		&domain.Ticket{},
		&domain.Message{},
		&domain.JITSession{},
		&domain.AuditLog{},
	); err != nil {
		return err
	}

	// Backfill untuk data lama: pastikan audit_logs.level tidak NULL.
	res := db.Model(&domain.AuditLog{}).Where("level IS NULL OR level = ''").Update("level", "LOW")
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		log.Printf("migrate: backfill audit_logs.level -> LOW, updated_rows=%d", res.RowsAffected)
	} else {
		log.Printf("migrate: backfill audit_logs.level -> LOW, updated_rows=0")
	}

	return nil
}
