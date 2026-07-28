package repos

import (
	"commander-server-t260220/internal/modules"

	"gorm.io/gorm"
)

type Listing1688StorePresetRepo struct{ db *gorm.DB }

func NewListing1688StorePresetRepo(db *gorm.DB) *Listing1688StorePresetRepo {
	return &Listing1688StorePresetRepo{db: db}
}

func (r *Listing1688StorePresetRepo) GetByUserStore(userID uint, storeKey string) (*modules.TableListing1688StorePresets, error) {
	var row modules.TableListing1688StorePresets
	if err := r.db.Where("user_id = ? AND store_key = ?", userID, storeKey).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Listing1688StorePresetRepo) Upsert(row *modules.TableListing1688StorePresets) error {
	var existing modules.TableListing1688StorePresets
	err := r.db.Where("user_id = ? AND store_key = ?", row.UserID, row.StoreKey).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(row).Error
	}
	if err != nil {
		return err
	}
	if err := r.db.Model(&existing).Updates(map[string]any{"fields": row.Fields, "is_active": row.IsActive}).Error; err != nil {
		return err
	}
	row.ID = existing.ID
	return nil
}

type Listing1688CategoryTemplateRepo struct{ db *gorm.DB }

func NewListing1688CategoryTemplateRepo(db *gorm.DB) *Listing1688CategoryTemplateRepo {
	return &Listing1688CategoryTemplateRepo{db: db}
}

func (r *Listing1688CategoryTemplateRepo) ListByUserStore(userID uint, storeKey string) ([]modules.TableListing1688CategoryTemplates, error) {
	var rows []modules.TableListing1688CategoryTemplates
	err := r.db.Where("user_id = ? AND store_key = ?", userID, storeKey).Order("category_id ASC").Find(&rows).Error
	return rows, err
}

func (r *Listing1688CategoryTemplateRepo) GetActiveByUserStoreCategory(userID uint, storeKey, categoryID string) (*modules.TableListing1688CategoryTemplates, error) {
	var row modules.TableListing1688CategoryTemplates
	if err := r.db.Where("user_id = ? AND store_key = ? AND category_id = ? AND is_active = ?", userID, storeKey, categoryID, true).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Listing1688CategoryTemplateRepo) Upsert(row *modules.TableListing1688CategoryTemplates) error {
	var existing modules.TableListing1688CategoryTemplates
	err := r.db.Where("user_id = ? AND store_key = ? AND category_id = ?", row.UserID, row.StoreKey, row.CategoryID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(row).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{"user_id": row.UserID, "category_path": row.CategoryPath, "product_family": row.ProductFamily, "fields": row.Fields, "required_fields": row.RequiredFields, "sku_rules": row.SkuRules, "detail_template": row.DetailTemplate, "is_active": row.IsActive}
	if err := r.db.Model(&existing).Updates(updates).Error; err != nil {
		return err
	}
	row.ID = existing.ID
	return nil
}

type Listing1688PublishDraftRepo struct{ db *gorm.DB }

func NewListing1688PublishDraftRepo(db *gorm.DB) *Listing1688PublishDraftRepo {
	return &Listing1688PublishDraftRepo{db: db}
}

func (r *Listing1688PublishDraftRepo) GetByWorkflow(workflowID uint) (*modules.TableListing1688PublishDrafts, error) {
	var row modules.TableListing1688PublishDrafts
	if err := r.db.Where("workflow_id = ?", workflowID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
func (r *Listing1688PublishDraftRepo) Upsert(row *modules.TableListing1688PublishDrafts) error {
	var existing modules.TableListing1688PublishDrafts
	err := r.db.Where("workflow_id = ?", row.WorkflowID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(row).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{"user_id": row.UserID, "store_key": row.StoreKey, "category_id": row.CategoryID, "form_data": row.FormData, "missing_fields": row.MissingFields, "validation_status": row.ValidationStatus, "draft_saved": row.DraftSaved, "offer_id": row.OfferID, "platform_status": row.PlatformStatus}
	if err := r.db.Model(&existing).Updates(updates).Error; err != nil {
		return err
	}
	row.ID = existing.ID
	return nil
}
