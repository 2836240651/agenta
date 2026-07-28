package modules

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TableListing1688Workflows 1688 竞品向导主状态机（与 v1 listing_1688_jobs 隔离）
type TableListing1688Workflows struct {
	gorm.Model
	UserID                 uint           `gorm:"not null;index;column:user_id" json:"user_id"`
	AgentID                string         `gorm:"not null;index;column:agent_id" json:"agent_id"`
	Status                 string         `gorm:"not null;index;column:status" json:"status"`
	OfferURL               string         `gorm:"type:text;column:offer_url" json:"offer_url"`
	OfferSnapshot          datatypes.JSON `gorm:"type:jsonb;column:offer_snapshot" json:"offer_snapshot"`
	SelectedImageIdx       datatypes.JSON `gorm:"type:jsonb;column:selected_image_idx" json:"selected_image_idx"`
	MatchedTemplateOfferID string         `gorm:"index;column:matched_template_offer_id" json:"matched_template_offer_id"`
	MatchedCategoryID      string         `gorm:"index;column:matched_category_id" json:"matched_category_id"`
	MatchCandidates        datatypes.JSON `gorm:"type:jsonb;column:match_candidates" json:"match_candidates"`
	MatchStatus            string         `gorm:"index;column:match_status" json:"match_status"`
	SimilarPageURL         string         `gorm:"type:text;column:similar_page_url" json:"similar_page_url"`
	PublishTaskID          string         `gorm:"index;column:publish_task_id" json:"publish_task_id"`
	ErrorCode              string         `gorm:"column:error_code" json:"error_code"`
	Message                string         `gorm:"type:text;column:message" json:"message"`
}

func (TableListing1688Workflows) TableName() string {
	return "listing_1688_workflows"
}
