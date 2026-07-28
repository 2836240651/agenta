package modules

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TableListing1688StorePresets struct {
	gorm.Model
	UserID   uint           `gorm:"not null;uniqueIndex:uidx_listing1688_store_preset;column:user_id" json:"user_id"`
	StoreKey string         `gorm:"not null;uniqueIndex:uidx_listing1688_store_preset;column:store_key" json:"store_key"`
	Fields   datatypes.JSON `gorm:"type:jsonb;column:fields" json:"fields"`
	IsActive bool           `gorm:"not null;default:true;column:is_active" json:"is_active"`
}

func (TableListing1688StorePresets) TableName() string { return "listing_1688_store_presets" }

type TableListing1688CategoryTemplates struct {
	gorm.Model
	UserID         uint           `gorm:"not null;uniqueIndex:uidx_listing1688_category_template;column:user_id" json:"user_id"`
	StoreKey       string         `gorm:"not null;uniqueIndex:uidx_listing1688_category_template;column:store_key" json:"store_key"`
	CategoryID     string         `gorm:"not null;uniqueIndex:uidx_listing1688_category_template;column:category_id" json:"category_id"`
	CategoryPath   string         `gorm:"type:text;column:category_path" json:"category_path"`
	ProductFamily  string         `gorm:"column:product_family" json:"product_family"`
	Fields         datatypes.JSON `gorm:"type:jsonb;column:fields" json:"fields"`
	RequiredFields datatypes.JSON `gorm:"type:jsonb;column:required_fields" json:"required_fields"`
	SkuRules       datatypes.JSON `gorm:"type:jsonb;column:sku_rules" json:"sku_rules"`
	DetailTemplate datatypes.JSON `gorm:"type:jsonb;column:detail_template" json:"detail_template"`
	IsActive       bool           `gorm:"not null;default:true;column:is_active" json:"is_active"`
}

func (TableListing1688CategoryTemplates) TableName() string { return "listing_1688_category_templates" }

type TableListing1688PublishDrafts struct {
	gorm.Model
	WorkflowID       uint           `gorm:"not null;uniqueIndex;column:workflow_id" json:"workflow_id"`
	UserID           uint           `gorm:"not null;index;column:user_id" json:"user_id"`
	StoreKey         string         `gorm:"column:store_key" json:"store_key"`
	CategoryID       string         `gorm:"index;column:category_id" json:"category_id"`
	FormData         datatypes.JSON `gorm:"type:jsonb;column:form_data" json:"form_data"`
	MissingFields    datatypes.JSON `gorm:"type:jsonb;column:missing_fields" json:"missing_fields"`
	ValidationStatus string         `gorm:"not null;index;column:validation_status" json:"validation_status"`
	DraftSaved       bool           `gorm:"not null;default:false;column:draft_saved" json:"draft_saved"`
	OfferID          string         `gorm:"index;column:offer_id" json:"offer_id"`
	PlatformStatus   string         `gorm:"column:platform_status" json:"platform_status"`
}

func (TableListing1688PublishDrafts) TableName() string { return "listing_1688_publish_drafts" }
