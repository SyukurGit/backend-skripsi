package domain

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	Email     string    `gorm:"type:varchar(191);uniqueIndex;not null"`
	Password  string    `gorm:"type:varchar(255);not null"`
	Role      string    `gorm:"type:varchar(20);index;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type UserProfile struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	UserID    uint64         `gorm:"uniqueIndex;not null"`
	Phone     string         `gorm:"type:varchar(30)"`
	Balance   int64          `gorm:"not null;default:0"`
	KYCData   datatypes.JSON `gorm:"type:json"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
}

type Ticket struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
	UserID       uint64    `gorm:"index;not null"`
	AssignedCSID *uint64   `gorm:"index"`
	Status       string    `gorm:"type:varchar(20);index;not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

type Message struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	TicketID  uint64    `gorm:"index;not null"`
	SenderID  uint64    `gorm:"index;not null"`
	Message   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type JITSession struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	CSID      uint64    `gorm:"index;not null"`
	TicketID  uint64    `gorm:"index;not null"`
	Feature   string    `gorm:"type:varchar(50);index;not null"`
	ExpiredAt time.Time `gorm:"index;not null"`
	IsActive  bool      `gorm:"index;not null"`
}

type AuditLog struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	UserID    uint64         `gorm:"index;not null"`
	Role      string         `gorm:"type:varchar(20);index;not null"`
	Level     string         `gorm:"type:enum('LOW','MEDIUM','HIGH');index;not null;default:LOW"`
	Action    string         `gorm:"type:varchar(100);index;not null"`
	TicketID  *uint64        `gorm:"index"`
	Metadata  datatypes.JSON `gorm:"type:json"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
}

// View DTO untuk kebijakan exposure data.
type UserProfileMaskedView struct {
	UserID         uint64         `json:"user_id"`
	Phone          string         `json:"phone"`
	Balance        int64          `json:"balance"`
	KYCData        datatypes.JSON `json:"kyc_data"`
	ExposureState  string         `json:"exposure_state"`
	PolicyNote     string         `json:"policy_note"`
	GrantedFeature string         `json:"granted_feature,omitempty"`
}
