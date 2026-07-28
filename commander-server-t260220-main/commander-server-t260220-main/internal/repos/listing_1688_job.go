package repos

import (
	"commander-server-t260220/internal/modules"

	"gorm.io/gorm"
)

// Listing1688JobRepo 1688 采集图优上架审计仓库
type Listing1688JobRepo struct {
	db *gorm.DB
}

func NewListing1688JobRepo(db *gorm.DB) *Listing1688JobRepo {
	return &Listing1688JobRepo{db: db}
}

func (r *Listing1688JobRepo) Create(row *modules.TableListing1688Jobs) error {
	return r.db.Create(row).Error
}

func (r *Listing1688JobRepo) GetByTaskID(taskID string) (*modules.TableListing1688Jobs, error) {
	var row modules.TableListing1688Jobs
	err := r.db.Where("task_id = ?", taskID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Listing1688JobRepo) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&modules.TableListing1688Jobs{}).Where("id = ?", id).Updates(updates).Error
}
