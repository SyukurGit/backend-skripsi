package database

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"support-backend/internal/domain"
	"support-backend/pkg/password"
)

type schemaMigration struct {
	Version   string    `gorm:"type:varchar(191);primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}

func (schemaMigration) TableName() string {
	return "schema_migrations"
}

type versionedMigration struct {
	version string
	apply   func(*gorm.DB) error
}

var versionedMigrations = []versionedMigration{
	{
		version: "2026081101_sync_demo_accounts",
		apply:   syncDemoAccounts,
	},
}

func runVersionedMigrations(db *gorm.DB) error {
	return db.Connection(func(conn *gorm.DB) error {
		var lockAcquired int
		if err := conn.Raw("SELECT GET_LOCK(?, 30)", "support_platform_schema_migrations").Scan(&lockAcquired).Error; err != nil {
			return fmt.Errorf("acquire migration lock: %w", err)
		}
		if lockAcquired != 1 {
			return errors.New("acquire migration lock: timeout")
		}
		defer func() {
			var lockReleased int
			if err := conn.Raw("SELECT RELEASE_LOCK(?)", "support_platform_schema_migrations").Scan(&lockReleased).Error; err != nil {
				log.Printf("migrate: gagal release migration lock: %v", err)
			}
		}()

		for _, migration := range versionedMigrations {
			var count int64
			if err := conn.Model(&schemaMigration{}).Where("version = ?", migration.version).Count(&count).Error; err != nil {
				return fmt.Errorf("check migration %s: %w", migration.version, err)
			}
			if count > 0 {
				continue
			}

			if err := conn.Transaction(func(tx *gorm.DB) error {
				if err := migration.apply(tx); err != nil {
					return err
				}
				return tx.Create(&schemaMigration{Version: migration.version, AppliedAt: time.Now()}).Error
			}); err != nil {
				return fmt.Errorf("apply migration %s: %w", migration.version, err)
			}

			log.Printf("migrate: applied %s", migration.version)
		}

		return nil
	})
}

func syncDemoAccounts(tx *gorm.DB) error {
	var userCount int64
	if err := tx.Model(&domain.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	// Database kosong akan diisi oleh SeedIfEmpty setelah proses migration selesai.
	if userCount == 0 {
		return nil
	}

	renames := []struct {
		from string
		to   string
	}{
		{from: "admin@example.com", to: "admin@test.com"},
		{from: "cs@example.com", to: "cs01@test.com"},
		{from: "user@example.com", to: "syukur@gmail.com"},
		{from: "syukurkursyu@gmail.com", to: "syukur@gmail.com"},
	}

	for _, rename := range renames {
		var targetCount int64
		if err := tx.Model(&domain.User{}).Where("email = ?", rename.to).Count(&targetCount).Error; err != nil {
			return err
		}
		if targetCount > 0 {
			continue
		}
		if err := tx.Model(&domain.User{}).Where("email = ?", rename.from).Update("email", rename.to).Error; err != nil {
			return err
		}
	}

	adminPassword, err := password.Hash("admin123")
	if err != nil {
		return err
	}
	csPassword, err := password.Hash("cs123")
	if err != nil {
		return err
	}
	userPassword, err := password.Hash("user123")
	if err != nil {
		return err
	}

	accounts := []domain.User{
		{Email: "cs01@test.com", Password: csPassword, Role: domain.RoleCS},
		{Email: "cs02@test.com", Password: csPassword, Role: domain.RoleCS},
		{Email: "syukur@gmail.com", Password: userPassword, Role: domain.RoleUser},
		{Email: "admin@test.com", Password: adminPassword, Role: domain.RoleAdmin},
	}

	for _, account := range accounts {
		var existing domain.User
		err := tx.Where("email = ?", account.Email).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := tx.Model(&existing).Updates(map[string]any{
			"password": account.Password,
			"role":     account.Role,
		}).Error; err != nil {
			return err
		}
	}

	var cs02 domain.User
	if err := tx.Where("email = ?", "cs02@test.com").First(&cs02).Error; err != nil {
		return err
	}
	var cs02ProfileCount int64
	if err := tx.Model(&domain.UserProfile{}).Where("user_id = ?", cs02.ID).Count(&cs02ProfileCount).Error; err != nil {
		return err
	}
	if cs02ProfileCount == 0 {
		profile := domain.UserProfile{
			UserID:    cs02.ID,
			Phone:     "080000000004",
			Balance:   0,
			KYCData:   datatypes.JSON([]byte(`{"department":"support"}`)),
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&profile).Error; err != nil {
			return err
		}
	}

	return nil
}
