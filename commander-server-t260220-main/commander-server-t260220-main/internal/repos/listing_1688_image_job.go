package repos

import (
	"commander-server-t260220/internal/modules"

	"gorm.io/gorm"
)

// Listing1688ImageJobRepo 向导 Slot 图优任务
type Listing1688ImageJobRepo struct {
	db *gorm.DB
}

func NewListing1688ImageJobRepo(db *gorm.DB) *Listing1688ImageJobRepo {
	return &Listing1688ImageJobRepo{db: db}
}

func (r *Listing1688ImageJobRepo) Create(row *modules.TableListing1688ImageJobs) error {
	return r.db.Create(row).Error
}

func (r *Listing1688ImageJobRepo) Update(id uint, updates map[string]interface{}) error {
	return r.db.Model(&modules.TableListing1688ImageJobs{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Listing1688ImageJobRepo) ListByWorkflow(workflowID uint) ([]modules.TableListing1688ImageJobs, error) {
	var rows []modules.TableListing1688ImageJobs
	err := r.db.Where("workflow_id = ?", workflowID).Order("slot ASC").Find(&rows).Error
	return rows, err
}

func (r *Listing1688ImageJobRepo) GetByWorkflowSlot(workflowID uint, slot int) (*modules.TableListing1688ImageJobs, error) {
	var row modules.TableListing1688ImageJobs
	err := r.db.Where("workflow_id = ? AND slot = ?", workflowID, slot).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
