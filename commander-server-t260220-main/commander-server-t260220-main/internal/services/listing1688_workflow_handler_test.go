package services

import (
	"bytes"
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/middleware"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/types"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type fakeListing1688Agents struct {
	byUUID map[string]*modules.TableSystemAgent
}

func (f *fakeListing1688Agents) GetByUUID(uuid string) (*modules.TableSystemAgent, error) {
	a, ok := f.byUUID[uuid]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return a, nil
}

type fakeListing1688Workflows struct {
	nextID uint64
	rows   map[uint]*modules.TableListing1688Workflows
}

func (f *fakeListing1688Workflows) Create(row *modules.TableListing1688Workflows) error {
	id := uint(atomic.AddUint64(&f.nextID, 1))
	cp := *row
	cp.ID = id
	f.rows[id] = &cp
	row.ID = id
	return nil
}

func (f *fakeListing1688Workflows) GetByIDForUser(id, userID uint) (*modules.TableListing1688Workflows, error) {
	row, ok := f.rows[id]
	if !ok || row.UserID != userID {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *row
	return &cp, nil
}

func (f *fakeListing1688Workflows) Update(id uint, updates map[string]interface{}) error {
	row, ok := f.rows[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if v, ok := updates["status"].(string); ok {
		row.Status = v
	}
	if v, ok := updates["offer_url"].(string); ok {
		row.OfferURL = v
	}
	if v, ok := updates["error_code"].(string); ok {
		row.ErrorCode = v
	}
	if v, ok := updates["message"].(string); ok {
		row.Message = v
	}
	if v, ok := updates["selected_image_idx"]; ok {
		switch t := v.(type) {
		case datatypes.JSON:
			row.SelectedImageIdx = t
		case []byte:
			row.SelectedImageIdx = datatypes.JSON(t)
		}
	}
	if v, ok := updates["offer_snapshot"]; ok {
		switch t := v.(type) {
		case datatypes.JSON:
			row.OfferSnapshot = t
		case []byte:
			row.OfferSnapshot = datatypes.JSON(t)
		}
	}
	if v, ok := updates["matched_template_offer_id"].(string); ok {
		row.MatchedTemplateOfferID = v
	}
	if v, ok := updates["matched_category_id"].(string); ok {
		row.MatchedCategoryID = v
	}
	if v, ok := updates["match_status"].(string); ok {
		row.MatchStatus = v
	}
	if v, ok := updates["similar_page_url"].(string); ok {
		row.SimilarPageURL = v
	}
	if v, ok := updates["match_candidates"]; ok {
		switch t := v.(type) {
		case datatypes.JSON:
			row.MatchCandidates = t
		case []byte:
			row.MatchCandidates = datatypes.JSON(t)
		}
	}
	f.rows[id] = row
	return nil
}

func newListing1688TestService(agents listing1688AgentFinder, wfs listing1688WorkflowStore) *Service {
	gin.SetMode(gin.TestMode)
	return &Service{
		response:                 utils.NewResponseUtils(utils.NewSilentLoggerUtils()),
		testListing1688Agents:    agents,
		testListing1688Workflows: wfs,
		testListing1688ImageJobs: newFakeListing1688ImageJobs(),
	}
}

func listing1688AuthCtx(userID uint) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	middleware.Set(c, &types.MiddlewareContext{
		UserInfo: &modules.TableSystemUser{Model: gorm.Model{ID: userID}},
	})
	return c, rec
}

func TestListing1688WorkflowCreateAndGet(t *testing.T) {
	agents := &fakeListing1688Agents{byUUID: map[string]*modules.TableSystemAgent{
		"agent-a": {UUID: "agent-a", Name: "A"},
	}}
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{}}
	s := newListing1688TestService(agents, wfs)

	c, rec := listing1688AuthCtx(7)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/listing1688/workflows", bytes.NewBufferString(`{"agent_id":"agent-a"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.Listing1688WorkflowCreate(c)

	var created utils.BaseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Code != 0 {
		t.Fatalf("create code=%v msg=%s", created.Code, created.Msg)
	}
	data, _ := created.Data.(map[string]any)
	if data == nil {
		// gin may decode numbers as float; re-decode via map
		raw, _ := json.Marshal(created.Data)
		_ = json.Unmarshal(raw, &data)
	}
	idVal, ok := data["id"].(float64)
	if !ok || idVal < 1 {
		t.Fatalf("unexpected create data: %#v", created.Data)
	}
	if data["status"] != string(constants.Listing1688StatusDraft) {
		t.Fatalf("status=%v", data["status"])
	}

	c2, rec2 := listing1688AuthCtx(7)
	c2.Params = gin.Params{{Key: "id", Value: "1"}}
	c2.Request = httptest.NewRequest(http.MethodGet, "/api/v1/listing1688/workflows/1", nil)
	s.Listing1688WorkflowGet(c2)

	var got utils.BaseResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != 0 {
		t.Fatalf("get code=%v msg=%s", got.Code, got.Msg)
	}
}

func TestListing1688WorkflowCreateInvalidAgent(t *testing.T) {
	agents := &fakeListing1688Agents{byUUID: map[string]*modules.TableSystemAgent{}}
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{}}
	s := newListing1688TestService(agents, wfs)

	c, rec := listing1688AuthCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/workflows", bytes.NewBufferString(`{"agent_id":"missing"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.Listing1688WorkflowCreate(c)

	var body utils.BaseResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Code != http.StatusForbidden {
		t.Fatalf("code=%v", body.Code)
	}
	if !bytes.Contains([]byte(body.Msg), []byte(constants.Listing1688ErrInvalidAgent)) {
		t.Fatalf("msg=%s", body.Msg)
	}
}

func TestListing1688WorkflowGetCrossUser(t *testing.T) {
	agents := &fakeListing1688Agents{byUUID: map[string]*modules.TableSystemAgent{
		"agent-a": {UUID: "agent-a"},
	}}
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{}}
	s := newListing1688TestService(agents, wfs)

	c, _ := listing1688AuthCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/workflows", bytes.NewBufferString(`{"agent_id":"agent-a"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.Listing1688WorkflowCreate(c)

	c2, rec2 := listing1688AuthCtx(99)
	c2.Params = gin.Params{{Key: "id", Value: "1"}}
	c2.Request = httptest.NewRequest(http.MethodGet, "/workflows/1", nil)
	s.Listing1688WorkflowGet(c2)

	var body utils.BaseResponse
	_ = json.Unmarshal(rec2.Body.Bytes(), &body)
	if body.Code != http.StatusForbidden {
		t.Fatalf("code=%v", body.Code)
	}
	if !bytes.Contains([]byte(body.Msg), []byte(constants.Listing1688ErrWorkflowNotFound)) {
		t.Fatalf("msg=%s", body.Msg)
	}
}

func TestListing1688StubRequiresOwnedWorkflow(t *testing.T) {
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{
		1: {Model: gorm.Model{ID: 1}, UserID: 1, AgentID: "a", Status: "draft"},
	}}
	s := newListing1688TestService(&fakeListing1688Agents{byUUID: map[string]*modules.TableSystemAgent{}}, wfs)

	// M3 publish 已实现；用 stub 辅助仍校验归属：对未知 stub 里程碑用 listing1688Stub 直接测
	c2, rec2 := listing1688AuthCtx(2)
	c2.Params = gin.Params{{Key: "id", Value: "1"}}
	c2.Request = httptest.NewRequest(http.MethodPost, "/workflows/1/publish", nil)
	s.listing1688Stub(c2, "M4")
	var other utils.BaseResponse
	_ = json.Unmarshal(rec2.Body.Bytes(), &other)
	if !bytes.Contains([]byte(other.Msg), []byte(constants.Listing1688ErrWorkflowNotFound)) {
		t.Fatalf("cross-user stub msg=%s", other.Msg)
	}

	c, rec := listing1688AuthCtx(1)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/workflows/1/x", nil)
	s.listing1688Stub(c, "M4")
	var owned utils.BaseResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &owned)
	if !bytes.Contains([]byte(owned.Msg), []byte(constants.Listing1688ErrNotImplemented)) {
		t.Fatalf("owned stub msg=%s", owned.Msg)
	}
}
