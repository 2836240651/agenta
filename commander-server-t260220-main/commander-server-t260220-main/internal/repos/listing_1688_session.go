package repos

import (
	"commander-server-t260220/internal/modules"
	"fmt"

	"gorm.io/gorm"
)

// Listing1688SessionRepo 向导采集/商家会话
type Listing1688SessionRepo struct {
	db *gorm.DB
}

func NewListing1688SessionRepo(db *gorm.DB) *Listing1688SessionRepo {
	return &Listing1688SessionRepo{db: db}
}

func (r *Listing1688SessionRepo) Create(row *modules.TableListing1688Sessions) error {
	return r.db.Create(row).Error
}

func (r *Listing1688SessionRepo) GetLatestByWorkflow(workflowID uint, sessionType string) (*modules.TableListing1688Sessions, error) {
	var row modules.TableListing1688Sessions
	err := r.db.Where("workflow_id = ? AND session_type = ?", workflowID, sessionType).
		Order("id DESC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Listing1688SessionRepo) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&modules.TableListing1688Sessions{}).Where("id = ?", id).Updates(updates).Error
}

// UpsertByWorkflowType 同 workflow+session_type 只保留一行：存在则更新，否则创建
func (r *Listing1688SessionRepo) UpsertByWorkflowType(row *modules.TableListing1688Sessions) error {
	if row == nil {
		return fmt.Errorf("session row is nil")
	}
	existing, err := r.GetLatestByWorkflow(row.WorkflowID, row.SessionType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.Create(row)
		}
		return err
	}
	updates := map[string]interface{}{
		"cookie_enc":        row.CookieEnc,
		"probe_ok":          row.ProbeOK,
		"probe_message":     row.ProbeMessage,
		"last_validated_at": row.LastValidatedAt,
		"user_id":           row.UserID,
	}
	if err := r.Update(existing.ID, updates); err != nil {
		return err
	}
	row.ID = existing.ID
	return nil
}
