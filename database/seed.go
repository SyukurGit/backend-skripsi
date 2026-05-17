package database

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"support-backend/internal/domain"
	"support-backend/pkg/password"
)

func SeedIfEmpty(db *gorm.DB) error {
	var cnt int64
	if err := db.Model(&domain.User{}).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}

	adminPass, _ := password.Hash("admin123")
	csPass, _ := password.Hash("cs123")
	userPass, _ := password.Hash("user123")

	admin := domain.User{Email: "admin@example.com", Password: adminPass, Role: domain.RoleAdmin}
	cs := domain.User{Email: "cs@example.com", Password: csPass, Role: domain.RoleCS}
	user := domain.User{Email: "user@example.com", Password: userPass, Role: domain.RoleUser}

	if err := db.Create(&admin).Error; err != nil {
		return err
	}
	if err := db.Create(&cs).Error; err != nil {
		return err
	}
	if err := db.Create(&user).Error; err != nil {
		return err
	}

	adminProfile := domain.UserProfile{UserID: admin.ID, Phone: "080000000001", Balance: 0, KYCData: datatypes.JSON([]byte(`{"tier":"admin"}`)), CreatedAt: time.Now()}
	csProfile := domain.UserProfile{UserID: cs.ID, Phone: "080000000002", Balance: 0, KYCData: datatypes.JSON([]byte(`{"department":"support"}`)), CreatedAt: time.Now()}
	userProfile := domain.UserProfile{UserID: user.ID, Phone: "080000000003", Balance: 100000, KYCData: datatypes.JSON([]byte(`{"kyc_status":"PENDING","is_blocked":false,"pin_hash":"","full_name":"Syukur Ramadhan","nik":"3175091209990001","birth_date":"1999-09-12","place_of_birth":"Jakarta","mother_name":"Nur Aisyah","address":"Jl. Merdeka Selatan No. 18, Jakarta Pusat","province":"DKI Jakarta","occupation":"UI/UX Designer","monthly_income_range":"10-15 juta","last_login_city":"Jakarta Selatan","recent_device":"iPhone 14 Pro","account_limit":"Rp 25.000.000 / hari","risk_score":"LOW","linked_bank":"Bank Nusantara **** 1188","emergency_contact":"Rina - 0812xxxx7788"}`)), CreatedAt: time.Now()}

	if err := db.Create(&adminProfile).Error; err != nil {
		return err
	}
	if err := db.Create(&csProfile).Error; err != nil {
		return err
	}
	if err := db.Create(&userProfile).Error; err != nil {
		return err
	}

	// Ticket contoh untuk demo chat.
	ticket := domain.Ticket{UserID: user.ID, Status: domain.TicketStatusOpen, CreatedAt: time.Now()}
	if err := db.Create(&ticket).Error; err != nil {
		return err
	}

	return nil
}
