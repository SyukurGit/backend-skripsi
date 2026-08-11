package database

import (
	"encoding/json"
	"log"

	"gorm.io/datatypes"
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
		&schemaMigration{},
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

	var profiles []domain.UserProfile
	if err := db.Find(&profiles).Error; err != nil {
		return err
	}
	for _, profile := range profiles {
		var obj map[string]any
		if len(profile.KYCData) > 0 {
			_ = json.Unmarshal(profile.KYCData, &obj)
		}
		if obj == nil {
			obj = map[string]any{}
		}
		if _, ok := obj["kyc_status"]; !ok {
			continue
		}
		defaults := map[string]any{
			"full_name":            "Nasabah DompetKu",
			"nik":                  "3175091209990001",
			"birth_date":           "1999-09-12",
			"place_of_birth":       "Jakarta",
			"mother_name":          "Nur Aisyah",
			"address":              "Jl. Merdeka Selatan No. 18, Jakarta Pusat",
			"province":             "DKI Jakarta",
			"occupation":           "UI/UX Designer",
			"monthly_income_range": "10-15 juta",
			"last_login_city":      "Jakarta Selatan",
			"recent_device":        "iPhone 14 Pro",
			"account_limit":        "Rp 25.000.000 / hari",
			"risk_score":           "LOW",
			"linked_bank":          "Bank Nusantara **** 1188",
			"emergency_contact":    "Rina - 0812xxxx7788",
		}
		changed := false
		for key, value := range defaults {
			if _, ok := obj[key]; !ok {
				obj[key] = value
				changed = true
			}
		}
		if changed {
			b, _ := json.Marshal(obj)
			if err := db.Model(&domain.UserProfile{}).Where("id = ?", profile.ID).Update("kyc_data", datatypes.JSON(b)).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
