package modules

import (
	"time"

	"gorm.io/gorm"
)

// TableListing1688Sessions 采集/商家会话（Cookie 加密落库；前端永不回传明文）
type TableListing1688Sessions struct {
	gorm.Model
	WorkflowID     uint       `gorm:"not null;index;column:workflow_id" json:"workflow_id"`
	UserID         uint       `gorm:"not null;index;column:user_id" json:"user_id"`
	SessionType    string     `gorm:"not null;index;column:session_type" json:"session_type"` // collect | seller
	CookieEnc      string     `gorm:"type:text;column:cookie_enc" json:"-"`
	ProbeOK        bool       `gorm:"column:probe_ok" json:"probe_ok"`
	ProbeMessage   string     `gorm:"type:text;column:probe_message" json:"probe_message"`
	LastValidatedAt *time.Time `gorm:"column:last_validated_at" json:"last_validated_at"`
}

func (TableListing1688Sessions) TableName() string {
	return "listing_1688_sessions"
}
