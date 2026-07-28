package modules

import (
	"gorm.io/gorm"
)

// TableListing1688ImageJobs 向导逐 Slot 图优任务（C1；禁止直调 produce_images）
type TableListing1688ImageJobs struct {
	gorm.Model
	WorkflowID   uint   `gorm:"not null;uniqueIndex:uidx_listing1688_wf_slot;column:workflow_id" json:"workflow_id"`
	UserID       uint   `gorm:"not null;index;column:user_id" json:"user_id"`
	Slot         int    `gorm:"not null;uniqueIndex:uidx_listing1688_wf_slot;column:slot" json:"slot"` // 0=主图, 1-9=副图
	SourceURL    string `gorm:"type:text;column:source_url" json:"source_url"`
	ResultURL    string `gorm:"type:text;column:result_url" json:"result_url"`
	Prompt       string `gorm:"type:text;column:prompt" json:"prompt"`
	Status       string `gorm:"not null;index;column:status" json:"status"` // pending|running|done|selected|use_original|failed
	ErrorCode    string `gorm:"column:error_code" json:"error_code"`
	Message      string `gorm:"type:text;column:message" json:"message"`
}

func (TableListing1688ImageJobs) TableName() string {
	return "listing_1688_image_jobs"
}
