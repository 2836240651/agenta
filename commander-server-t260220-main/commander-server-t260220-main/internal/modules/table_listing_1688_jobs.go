package modules

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TableListing1688Jobs 1688 采集图优上架审计表（platform=1688 专用，与 ai_images 语义隔离）
type TableListing1688Jobs struct {
	gorm.Model
	UserID          uint           `gorm:"not null;index;column:user_id" json:"user_id"`
	TaskID          string         `gorm:"not null;index;column:task_id" json:"task_id"`
	PeerURL         string         `gorm:"type:text;column:peer_url" json:"peer_url"`
	NumIid          string         `gorm:"column:num_iid" json:"num_iid"`
	SourceImages    datatypes.JSON `gorm:"type:jsonb;column:source_images" json:"source_images"`
	OptimizedImages datatypes.JSON `gorm:"type:jsonb;column:optimized_images" json:"optimized_images"`
	Status          string         `gorm:"not null;column:status" json:"status"`
	ErrorCode       string         `gorm:"column:error_code" json:"error_code"`
	Message         string         `gorm:"type:text;column:message" json:"message"`
	Raw             datatypes.JSON `gorm:"type:jsonb;column:raw" json:"raw"`
}

func (TableListing1688Jobs) TableName() string {
	return "listing_1688_jobs"
}
