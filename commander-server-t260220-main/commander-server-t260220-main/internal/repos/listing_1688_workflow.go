package repos

import (
	"commander-server-t260220/internal/modules"

	"gorm.io/gorm"
)

// Listing1688WorkflowRepo 1688 向导 Workflow 仓库
type Listing1688WorkflowRepo struct {
	db *gorm.DB
}

func NewListing1688WorkflowRepo(db *gorm.DB) *Listing1688WorkflowRepo {
	return &Listing1688WorkflowRepo{db: db}
}

func (r *Listing1688WorkflowRepo) Create(row *modules.TableListing1688Workflows) error {
	return r.db.Create(row).Error
}

func (r *Listing1688WorkflowRepo) GetByIDForUser(id, userID uint) (*modules.TableListing1688Workflows, error) {
	var row modules.TableListing1688Workflows
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Listing1688WorkflowRepo) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&modules.TableListing1688Workflows{}).Where("id = ?", id).Updates(updates).Error
}
