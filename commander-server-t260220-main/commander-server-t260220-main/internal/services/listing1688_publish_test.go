package services

import (
	"bytes"
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type fakeListing1688Sessions struct {
	mu     sync.Mutex
	nextID uint64
	rows   map[uint]*modules.TableListing1688Sessions
}

func newFakeListing1688Sessions() *fakeListing1688Sessions {
	return &fakeListing1688Sessions{rows: map[uint]*modules.TableListing1688Sessions{}}
}

func (f *fakeListing1688Sessions) Create(row *modules.TableListing1688Sessions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := uint(atomic.AddUint64(&f.nextID, 1))
	cp := *row
	cp.ID = id
	f.rows[id] = &cp
	row.ID = id
	return nil
}

func (f *fakeListing1688Sessions) GetLatestByWorkflow(workflowID uint, sessionType string) (*modules.TableListing1688Sessions, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var best *modules.TableListing1688Sessions
	for _, r := range f.rows {
		if r.WorkflowID == workflowID && r.SessionType == sessionType {
			if best == nil || r.ID > best.ID {
				cp := *r
				best = &cp
			}
		}
	}
	if best == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return best, nil
}

func (f *fakeListing1688Sessions) Update(id uint, updates map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if v, ok := updates["cookie_enc"].(string); ok {
		row.CookieEnc = v
	}
	if v, ok := updates["probe_ok"].(bool); ok {
		row.ProbeOK = v
	}
	if v, ok := updates["probe_message"].(string); ok {
		row.ProbeMessage = v
	}
	if v, ok := updates["last_validated_at"].(time.Time); ok {
		row.LastValidatedAt = &v
	}
	if v, ok := updates["last_validated_at"].(*time.Time); ok {
		row.LastValidatedAt = v
	}
	f.rows[id] = row
	return nil
}

func (f *fakeListing1688Sessions) UpsertByWorkflowType(row *modules.TableListing1688Sessions) error {
	existing, err := f.GetLatestByWorkflow(row.WorkflowID, row.SessionType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return f.Create(row)
		}
		return err
	}
	return f.Update(existing.ID, map[string]interface{}{
		"cookie_enc":        row.CookieEnc,
		"probe_ok":          row.ProbeOK,
		"probe_message":     row.ProbeMessage,
		"last_validated_at": row.LastValidatedAt,
	})
}

func (f *fakeListing1688Sessions) countType(sessionType string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.rows {
		if r.SessionType == sessionType {
			n++
		}
	}
	return n
}

// TC-P-05：仅有 collect 会话不能发品
func TestListing1688Publish_CollectOnlyCannotPublish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sel, _ := json.Marshal([]listing1688SelectedImage{{Slot: 0, ResultURL: "https://cdn.example/a.png"}})
	wf := &modules.TableListing1688Workflows{
		Model:            gorm.Model{ID: 1},
		UserID:           1,
		AgentID:          "agent-a",
		Status:           string(constants.Listing1688StatusImgReady),
		SelectedImageIdx: datatypes.JSON(sel),
		OfferSnapshot:    datatypes.JSON([]byte(`{"title":"t1"}`)),
	}
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{1: wf}}
	jobs := newFakeListing1688ImageJobs()
	_ = jobs.Create(&modules.TableListing1688ImageJobs{
		WorkflowID: 1, Slot: 0, Status: constants.Listing1688ImageSelected, ResultURL: "https://cdn.example/a.png",
	})
	sessions := newFakeListing1688Sessions()
	_ = sessions.Create(&modules.TableListing1688Sessions{
		WorkflowID: 1, UserID: 1, SessionType: constants.Listing1688SessionCollect,
		CookieEnc: "enc-collect", ProbeOK: true,
	})

	s := &Service{
		response:                 utils.NewResponseUtils(utils.NewSilentLoggerUtils()),
		testListing1688Workflows: wfs,
		testListing1688ImageJobs: jobs,
		testListing1688Sessions:  sessions,
		// agentManager nil → online check fails first；先测 RequireSellerSession 门禁本身
	}

	_, err := s.listing1688RequireSellerSession(wf)
	if err == nil {
		t.Fatal("collect-only must not satisfy seller session")
	}
	if !bytes.Contains([]byte(err.Error()), []byte(constants.Listing1688ErrSellerSessionInvalid)) {
		t.Fatalf("err=%v", err)
	}

	c, rec := listing1688AuthCtx(1)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/publish", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.Listing1688Publish(c)
	var resp utils.BaseResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Code == 0 {
		t.Fatal("publish must fail without seller session / agent")
	}
	if !bytes.Contains([]byte(resp.Msg), []byte(constants.Listing1688ErrWizardAgentOffline)) &&
		!bytes.Contains([]byte(resp.Msg), []byte(constants.Listing1688ErrSellerSessionInvalid)) {
		t.Fatalf("unexpected msg=%s", resp.Msg)
	}
}

func TestListing1688SellerSessionUpsert_NoDuplicateRows(t *testing.T) {
	sessions := newFakeListing1688Sessions()
	now := time.Now()
	row := &modules.TableListing1688Sessions{
		WorkflowID: 9, UserID: 1, SessionType: constants.Listing1688SessionSeller,
		CookieEnc: "a", ProbeOK: true, LastValidatedAt: &now,
	}
	if err := sessions.UpsertByWorkflowType(row); err != nil {
		t.Fatal(err)
	}
	row2 := &modules.TableListing1688Sessions{
		WorkflowID: 9, UserID: 1, SessionType: constants.Listing1688SessionSeller,
		CookieEnc: "b", ProbeOK: true, LastValidatedAt: &now,
	}
	if err := sessions.UpsertByWorkflowType(row2); err != nil {
		t.Fatal(err)
	}
	if sessions.countType(constants.Listing1688SessionSeller) != 1 {
		t.Fatalf("want 1 seller row, got %d", sessions.countType(constants.Listing1688SessionSeller))
	}
	got, err := sessions.GetLatestByWorkflow(9, constants.Listing1688SessionSeller)
	if err != nil {
		t.Fatal(err)
	}
	if got.CookieEnc != "b" {
		t.Fatalf("cookie_enc=%s", got.CookieEnc)
	}
}
