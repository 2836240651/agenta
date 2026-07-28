package services

import (
	"bytes"
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type fakeListing1688ImageJobs struct {
	mu     sync.Mutex
	nextID uint64
	rows   map[uint]*modules.TableListing1688ImageJobs
}

func newFakeListing1688ImageJobs() *fakeListing1688ImageJobs {
	return &fakeListing1688ImageJobs{rows: map[uint]*modules.TableListing1688ImageJobs{}}
}

func (f *fakeListing1688ImageJobs) Create(row *modules.TableListing1688ImageJobs) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := uint(atomic.AddUint64(&f.nextID, 1))
	cp := *row
	cp.ID = id
	f.rows[id] = &cp
	row.ID = id
	return nil
}

func (f *fakeListing1688ImageJobs) Update(id uint, updates map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if v, ok := updates["status"].(string); ok {
		row.Status = v
	}
	if v, ok := updates["prompt"].(string); ok {
		row.Prompt = v
	}
	if v, ok := updates["result_url"].(string); ok {
		row.ResultURL = v
	}
	if v, ok := updates["error_code"].(string); ok {
		row.ErrorCode = v
	}
	if v, ok := updates["message"].(string); ok {
		row.Message = v
	}
	f.rows[id] = row
	return nil
}

func (f *fakeListing1688ImageJobs) ListByWorkflow(workflowID uint) ([]modules.TableListing1688ImageJobs, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]modules.TableListing1688ImageJobs, 0)
	for _, r := range f.rows {
		if r.WorkflowID == workflowID {
			out = append(out, *r)
		}
	}
	// slot asc
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Slot < out[i].Slot {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (f *fakeListing1688ImageJobs) GetByWorkflowSlot(workflowID uint, slot int) (*modules.TableListing1688ImageJobs, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.WorkflowID == workflowID && r.Slot == slot {
			cp := *r
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func TestListing1688TruncateSourceURLs_Max10(t *testing.T) {
	in := make([]string, 0, 15)
	for i := 0; i < 15; i++ {
		in = append(in, "http://img/"+string(rune('a'+i)))
	}
	// use unique numeric urls
	in = nil
	for i := 0; i < 15; i++ {
		in = append(in, "http://img/"+itoa(i))
	}
	out := listing1688TruncateSourceURLs(in)
	if len(out) != constants.Listing1688MaxImageSlots {
		t.Fatalf("want %d got %d", constants.Listing1688MaxImageSlots, len(out))
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func collectedWorkflowWithImages(userID uint, urls []string) *modules.TableListing1688Workflows {
	raw, _ := json.Marshal(map[string]any{"main_images": urls, "title": "t"})
	return &modules.TableListing1688Workflows{
		Model:         gorm.Model{ID: 1},
		UserID:        userID,
		AgentID:       "a",
		Status:        string(constants.Listing1688StatusCollected),
		OfferSnapshot: datatypes.JSON(raw),
	}
}

func newImagesTestService(wfs *fakeListing1688Workflows, jobs *fakeListing1688ImageJobs) *Service {
	gin.SetMode(gin.TestMode)
	s := &Service{
		response:                 utils.NewResponseUtils(utils.NewSilentLoggerUtils()),
		testListing1688Workflows: wfs,
		testListing1688ImageJobs: jobs,
		testListing1688DecodeImage: func(src string) ([]byte, error) {
			return []byte("img:" + src), nil
		},
		testListing1688ImageEdit: func(prompt string, ref [][]byte) (string, error) {
			return "https://cdn.example/edited.png?p=" + prompt, nil
		},
	}
	return s
}

func TestListing1688EnsureSlotsCapsAt10(t *testing.T) {
	urls := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		urls = append(urls, "http://x/"+itoa(i))
	}
	wf := collectedWorkflowWithImages(1, urls)
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{1: wf}}
	jobs := newFakeListing1688ImageJobs()
	s := newImagesTestService(wfs, jobs)
	got, err := s.listing1688EnsureImageSlots(wf)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("want 10 slots, got %d", len(got))
	}
}

func TestListing1688ImagesGenerateAndSelectReady(t *testing.T) {
	urls := []string{"http://a/1", "http://a/2"}
	wf := collectedWorkflowWithImages(1, urls)
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{1: wf}}
	jobs := newFakeListing1688ImageJobs()
	s := newImagesTestService(wfs, jobs)

	editCalls := 0
	s.testListing1688ImageEdit = func(prompt string, ref [][]byte) (string, error) {
		editCalls++
		return "https://cdn.example/r" + itoa(editCalls) + ".png", nil
	}

	post := func(path string, body string, h func(*gin.Context)) utils.BaseResponse {
		c, rec := listing1688AuthCtx(1)
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		c.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		h(c)
		var resp utils.BaseResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return resp
	}

	waitSettled := func(slot int) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			j, err := jobs.GetByWorkflowSlot(1, slot)
			if err == nil && j != nil && j.Status != constants.Listing1688ImageRunning {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("slot %d still running", slot)
	}

	g0 := post("/images/generate", `{"slot":0}`, s.Listing1688ImagesGenerate)
	if g0.Code != 0 {
		t.Fatalf("generate0: %v %s", g0.Code, g0.Msg)
	}
	waitSettled(0)
	g1 := post("/images/generate", `{"slot":1,"prompt":"custom"}`, s.Listing1688ImagesGenerate)
	if g1.Code != 0 {
		t.Fatalf("generate1: %v %s", g1.Code, g1.Msg)
	}
	waitSettled(1)
	if editCalls != 2 {
		t.Fatalf("editCalls=%d", editCalls)
	}

	// select pending rejected path: create third workflow state — slot already done
	selPending := post("/images/select", `{"slot":0}`, s.Listing1688ImagesSelect)
	if selPending.Code != 0 {
		t.Fatalf("select0 after done should ok: %s", selPending.Msg)
	}

	uo := post("/images/use-original", `{"slot":1}`, s.Listing1688ImagesUseOriginal)
	if uo.Code != 0 {
		t.Fatalf("use-original: %s", uo.Msg)
	}
	callsBefore := editCalls
	_ = post("/images/use-original", `{"slot":1}`, s.Listing1688ImagesUseOriginal)
	if editCalls != callsBefore {
		t.Fatal("use-original must not call AiEdit")
	}

	sel1 := post("/images/select", `{"slot":1}`, s.Listing1688ImagesSelect)
	if sel1.Code != 0 {
		t.Fatalf("select1: %s", sel1.Msg)
	}
	st := wfs.rows[1].Status
	if st != string(constants.Listing1688StatusImgReady) {
		t.Fatalf("want img_ready got %s", st)
	}
	if err := s.listing1688AssertImgReady(wfs.rows[1]); err != nil {
		t.Fatal(err)
	}

	// regenerate drops readiness
	rg := post("/images/regenerate", `{"slot":0,"prompt":"again"}`, s.Listing1688ImagesRegenerate)
	if rg.Code != 0 {
		t.Fatalf("regen: %s", rg.Msg)
	}
	if wfs.rows[1].Status != string(constants.Listing1688StatusImgEditing) {
		t.Fatalf("after regen want img_editing got %s", wfs.rows[1].Status)
	}
	if err := s.listing1688AssertImgReady(wfs.rows[1]); err == nil {
		t.Fatal("expected IMAGES_NOT_READY")
	}
}

func TestListing1688ImagesSelectFromPendingRejected(t *testing.T) {
	urls := []string{"http://a/1"}
	wf := collectedWorkflowWithImages(1, urls)
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{1: wf}}
	jobs := newFakeListing1688ImageJobs()
	s := newImagesTestService(wfs, jobs)
	_, _ = s.listing1688EnsureImageSlots(wf)

	c, rec := listing1688AuthCtx(1)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/images/select", bytes.NewBufferString(`{"slot":0}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.Listing1688ImagesSelect(c)
	var resp utils.BaseResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Code == 0 {
		t.Fatal("expected reject")
	}
	if !bytes.Contains([]byte(resp.Msg), []byte(constants.Listing1688ErrImagesNotReady)) {
		t.Fatalf("msg=%s", resp.Msg)
	}
}

func TestListing1688ImagesGenerateFailRecomputesSelectedIdx(t *testing.T) {
	urls := []string{"http://a/1", "http://a/2"}
	wf := collectedWorkflowWithImages(1, urls)
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{1: wf}}
	jobs := newFakeListing1688ImageJobs()
	s := newImagesTestService(wfs, jobs)

	post := func(path string, body string, h func(*gin.Context)) utils.BaseResponse {
		c, rec := listing1688AuthCtx(1)
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		c.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		h(c)
		var resp utils.BaseResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return resp
	}

	_ = post("/images/generate", `{"slot":0}`, s.Listing1688ImagesGenerate)
	_ = post("/images/generate", `{"slot":1}`, s.Listing1688ImagesGenerate)
	waitSettled := func(slot int) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			j, err := jobs.GetByWorkflowSlot(1, slot)
			if err == nil && j != nil && j.Status != constants.Listing1688ImageRunning {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("slot %d still running", slot)
	}
	waitSettled(0)
	waitSettled(1)
	_ = post("/images/select", `{"slot":0}`, s.Listing1688ImagesSelect)
	_ = post("/images/select", `{"slot":1}`, s.Listing1688ImagesSelect)
	if wfs.rows[1].Status != string(constants.Listing1688StatusImgReady) {
		t.Fatalf("precondition img_ready, got %s", wfs.rows[1].Status)
	}

	s.testListing1688ImageEdit = func(prompt string, ref [][]byte) (string, error) {
		return "", fmt.Errorf("hyhacct down")
	}
	fail := post("/images/regenerate", `{"slot":0}`, s.Listing1688ImagesRegenerate)
	// 异步受理：HTTP 立即成功，失败写入 job
	if fail.Code != 0 {
		t.Fatalf("async regenerate should accept, got %s", fail.Msg)
	}
	waitSettled(0)
	j0, _ := jobs.GetByWorkflowSlot(1, 0)
	if j0 == nil || j0.Status != constants.Listing1688ImageFailed {
		t.Fatalf("want slot0 failed, got %+v", j0)
	}
	if wfs.rows[1].Status != string(constants.Listing1688StatusImgEditing) {
		t.Fatalf("want img_editing got %s", wfs.rows[1].Status)
	}
	var sel []listing1688SelectedImage
	_ = json.Unmarshal(wfs.rows[1].SelectedImageIdx, &sel)
	for _, item := range sel {
		if item.Slot == 0 {
			t.Fatalf("selected_image_idx still contains failed slot0: %+v", sel)
		}
	}
	if len(sel) != 1 || sel[0].Slot != 1 {
		t.Fatalf("want only slot1 selected, got %+v", sel)
	}
}

func TestListing1688ImagesSlotOOB(t *testing.T) {
	urls := []string{"http://a/1"}
	wf := collectedWorkflowWithImages(1, urls)
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{1: wf}}
	jobs := newFakeListing1688ImageJobs()
	s := newImagesTestService(wfs, jobs)

	c, rec := listing1688AuthCtx(1)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/images/generate", bytes.NewBufferString(`{"slot":10}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.Listing1688ImagesGenerate(c)
	var resp utils.BaseResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Code == 0 {
		t.Fatal("expected OOB reject")
	}
}

func TestListing1688ImagesCrossUser(t *testing.T) {
	urls := []string{"http://a/1"}
	wf := collectedWorkflowWithImages(1, urls)
	wfs := &fakeListing1688Workflows{rows: map[uint]*modules.TableListing1688Workflows{1: wf}}
	jobs := newFakeListing1688ImageJobs()
	s := newImagesTestService(wfs, jobs)

	c, rec := listing1688AuthCtx(2)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/images/generate", bytes.NewBufferString(`{"slot":0}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.Listing1688ImagesGenerate(c)
	var resp utils.BaseResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !bytes.Contains([]byte(resp.Msg), []byte(constants.Listing1688ErrWorkflowNotFound)) {
		t.Fatalf("msg=%s", resp.Msg)
	}
}
