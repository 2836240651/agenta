package services

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/middleware"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/repos"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type listing1688CreateWorkflowReq struct {
	AgentID string `json:"agent_id"`
}

// listing1688AgentFinder / listing1688WorkflowStore 便于 handler 单测注入假实现
type listing1688AgentFinder interface {
	GetByUUID(uuid string) (*modules.TableSystemAgent, error)
}

type listing1688WorkflowStore interface {
	Create(row *modules.TableListing1688Workflows) error
	GetByIDForUser(id, userID uint) (*modules.TableListing1688Workflows, error)
	Update(id uint, updates map[string]interface{}) error
}

func (s *Service) listing1688WorkflowRepo() *repos.Listing1688WorkflowRepo {
	return repos.NewListing1688WorkflowRepo(s.gormUtils.Get())
}

func (s *Service) listing1688Agents() listing1688AgentFinder {
	if s.testListing1688Agents != nil {
		return s.testListing1688Agents
	}
	return s.systemAgentRepo
}

func (s *Service) listing1688Workflows() listing1688WorkflowStore {
	if s.testListing1688Workflows != nil {
		return s.testListing1688Workflows
	}
	return s.listing1688WorkflowRepo()
}

func (s *Service) listing1688RequireUser(ctx *gin.Context) (uint, bool) {
	mc := middleware.Get(ctx)
	if mc == nil || mc.UserInfo == nil {
		s.response.Unauthorized(ctx, "未授权", nil)
		return 0, false
	}
	return mc.UserInfo.ID, true
}

func (s *Service) listing1688ParseWorkflowID(ctx *gin.Context) (uint, bool) {
	raw := strings.TrimSpace(ctx.Param("id"))
	id64, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id64 == 0 {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWorkflowNotFound, "无效 workflow id"), err)
		return 0, false
	}
	return uint(id64), true
}

// listing1688LoadOwnedWorkflow 校验 :id 属于当前用户；失败时已写响应
func (s *Service) listing1688LoadOwnedWorkflow(ctx *gin.Context) (*modules.TableListing1688Workflows, bool) {
	userID, ok := s.listing1688RequireUser(ctx)
	if !ok {
		return nil, false
	}
	id, ok := s.listing1688ParseWorkflowID(ctx)
	if !ok {
		return nil, false
	}
	row, err := s.listing1688Workflows().GetByIDForUser(id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWorkflowNotFound, "workflow 不存在"), err)
			return nil, false
		}
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": 查询失败", err)
		return nil, false
	}
	return row, true
}

// listing1688Stub 先校验 workflow 归属，再返回里程碑未实现（防越权探测）
func (s *Service) listing1688Stub(ctx *gin.Context, milestone string) {
	if _, ok := s.listing1688LoadOwnedWorkflow(ctx); !ok {
		return
	}
	s.response.Forbidden(ctx, string(constants.Listing1688ErrNotImplemented)+": 将在 "+milestone+" 实现", nil)
}

// Listing1688WorkflowCreate 选 Agent 后创建 draft Workflow（C2）
func (s *Service) Listing1688WorkflowCreate(ctx *gin.Context) {
	userID, ok := s.listing1688RequireUser(ctx)
	if !ok {
		return
	}
	var req listing1688CreateWorkflowReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrInvalidAgent, "请求体无效"), err)
		return
	}
	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrInvalidAgent, "agent_id 必填"), nil)
		return
	}
	agent, err := s.listing1688Agents().GetByUUID(agentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrInvalidAgent, "Agent 不存在"), err)
			return
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrInvalidAgent, "查询 Agent 失败"), err)
		return
	}

	row := &modules.TableListing1688Workflows{
		UserID:  userID,
		AgentID: agent.UUID,
		Status:  string(constants.Listing1688StatusDraft),
	}
	if err := s.listing1688Workflows().Create(row); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": 创建 workflow 失败", err)
		return
	}
	s.response.Success(ctx, gin.H{
		"id":       row.ID,
		"agent_id": row.AgentID,
		"status":   row.Status,
	})
}

// Listing1688WorkflowGet 读取当前用户的 Workflow
func (s *Service) Listing1688WorkflowGet(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if listing1688JobPhaseOK(row.Status) && len(row.OfferSnapshot) > 0 {
		_, _ = s.listing1688EnsureImageSlots(row)
	}
	jobs, err := s.listing1688ImageJobs().ListByWorkflow(row.ID)
	if err != nil {
		jobs = nil
	}
	s.response.Success(ctx, gin.H{
		"id":                        row.ID,
		"agent_id":                  row.AgentID,
		"status":                    row.Status,
		"offer_url":                 row.OfferURL,
		"offer_snapshot":            row.OfferSnapshot,
		"selected_image_idx":        row.SelectedImageIdx,
		"matched_template_offer_id": row.MatchedTemplateOfferID,
		"matched_category_id":       row.MatchedCategoryID,
		"match_status":              row.MatchStatus,
		"similar_page_url":          row.SimilarPageURL,
		"template_match_result":     listing1688WorkflowTemplateMatchResult(row),
		"publish_task_id":           row.PublishTaskID,
		"image_jobs":                listing1688JobsToMaps(jobs),
		"error_code":                row.ErrorCode,
		"message":                   row.Message,
		"created_at":                row.CreatedAt,
		"updated_at":                row.UpdatedAt,
	})
}
