# 1688 Listing Wizard M2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver M2 Slot image optimization for the 1688 wizard (Step3): per-slot generate / regenerate / select / use-original via `listing1688/.../images/*` → `AiEditImage` → grsai, persist `listing_1688_image_jobs`, reach `img_ready` only when all visible slots are selected (AC-03 / C1 / C5 / TC-I-*).

**Architecture:** Image work is **Server-only** (no new Agent protocol). Web calls existing `POST .../images/*`; Server seeds slots from `offer_snapshot.main_images` (cap: Slot0 main + ≤9 extras), downloads source bytes (`pkg.DecodeImageInputToBytes` / `DownloadImage`), calls `utils.AiEditImage` (same as v1 `RunListing1688` / AI produce path), writes `listing_1688_image_jobs`, updates workflow `img_editing` → `img_ready` and `selected_image_idx`. Wizard **must not** call `POST /api/v1/ai/produce_images`. Publish stays M3 stub.

**Tech Stack:** Go/Gin/GORM, existing `LyncrUtils`/grsai via `AiEditImage`, Vue3 wizard Step3 UI.

## Global Constraints

- Main battlefield: `D:\dev\workspace` only
- Spec authority: `docs/superpowers/specs/1688-listing-wizard-v2/README.md`
- C1: images API → `AiEditImage` → grsai; write `listing_1688_image_jobs`; **no** wizard direct `produce_images`
- C5: advance to Step4 only when every **visible** slot is `selected` (including after use-original)
- Slot cap: Slot0 + Slot1–9 → max **10** images (1 main + 9 extras); reject 11th (`TC-I-05`)
- Default prompt: `constants.Listing1688WizardDefaultPrompt`（可被单 Slot 请求覆盖）
- Aspect/size/model: align v1 img2img — `"1024:1024"`, `"1K"`, `config.Lyncr.ModelImages`
- Scene A / Miaoshou: **unchanged** — M2 must not wire publish to Miaoshou/店小秘/`ProductIssue`
- v1 `/1688-auto-upload/listing` untouched
- M1 collect failures stay retryable (`session_valid`); M2 slot failures stay per-slot (`failed`), workflow stays `img_editing` (do not lock whole workflow to `failed` on one slot fail)
- Prerequisite: workflow `status` in `collected` | `img_editing` | `img_ready` (and owned by user)

## Approach (locked)

| Topic | Choice |
|-------|--------|
| Where grsai runs | Server sync in request handler (or short internal helper); MVP **no** background queue |
| Seed slots | Lazy on first images API or explicit ensure: from `offer_snapshot.main_images` truncated to 1+9 |
| Job status | `pending` → `running` → `done` \| `failed`; `use_original`; `selected` |
| `use-original` | Set `result_url = source_url`, status `use_original` (or immediately allow select); **no** grsai call |
| `select` | Only from `done` or `use_original`; sets status `selected`; refresh `selected_image_idx` JSON |
| Unselect / re-edit | `regenerate` clears selection for that slot → status `running`→`done`; workflow drops to `img_editing` if was `img_ready` |
| `img_ready` | All seeded slots status == `selected` |
| `IMAGES_NOT_READY` | Any gate that requires Step4 / “all selected” while not `img_ready` |
| GET workflow | Extend response with `image_jobs: [...]` so Step3 can render without new list route |
| Agent | **No** M2 protocols |

## API contracts

All under `/api/v1/listing1688/workflows/:id/`；先 `listing1688LoadOwnedWorkflow`。

| Method | Body | Behavior |
|--------|------|----------|
| `POST images/generate` | `{ slot: int, prompt?: string }` | Ensure job row; download source; `AiEditImage`; write `result_url`; status `done`; workflow → `img_editing` |
| `POST images/regenerate` | `{ slot: int, prompt?: string }` | Same as generate for existing slot; only that slot updates (`TC-I-02`) |
| `POST images/select` | `{ slot: int }` | Mark selected; recompute `img_ready` / `selected_image_idx` |
| `POST images/use-original` | `{ slot: int }` | No grsai; `result_url=source_url`; status `use_original` (UI may then call select, or select is implied — **lock: use-original leaves `use_original`, Web calls select**; both allowed if select accepts `use_original`) |

**Responses (success):** `{ workflow_id, status, slot, job: { id, slot, source_url, result_url, prompt, status, error_code, message }, image_jobs: [...], selected_image_idx }`

**Errors:** ownership → `LISTING_1688_WORKFLOW_NOT_FOUND`; wrong phase → `COLLECT_*` / forbidden with clear msg; slot OOB → reject; grsai fail → job `failed` + message, HTTP Forbidden or 200 with job failed — **lock: HTTP Forbidden + job persisted failed**, workflow stays `img_editing`.

## File Map

| Path | Role |
|------|------|
| Server `internal/repos/listing_1688_image_job.go` | CRUD by workflow/slot |
| Server `internal/services/listing1688_images.go` | generate/regenerate/select/use-original + ensureSlots + ready gate |
| Server `internal/services/listing1688_workflow.go` | Remove image stubs; Get includes `image_jobs` |
| Server `internal/services/listing1688_images_test.go` | Unit tests: seed cap, select gate, use-original no AiEdit, ready transition |
| Server `internal/constants/listing_1688_wizard.go` | Already has prompt + `IMAGES_NOT_READY` / statuses — extend only if needed |
| Web `src/api/modules/listing1688.js` | Four image API helpers |
| Web `listing1688-wizard/index.vue` | Step3 Slot UI |
| Web locales `zh-CN.json` / `en.json` | Replace `m2Pending`; slot actions copy |
| `开发文档.md` | M2 entry |

**Do not modify for M2:** Agent factory/protocols; `RunListing1688` v1 path; publish handlers (remain M3 stub).

---

### Task 1: Image job repo + ensureSlots + generate/regenerate

**Files:**
- Create: `commander-server-t260220-main/commander-server-t260220-main/internal/repos/listing_1688_image_job.go`
- Create: `.../internal/services/listing1688_images.go`
- Create: `.../internal/services/listing1688_images_test.go`
- Modify: `.../internal/services/listing1688_workflow.go` (wire handlers; drop image stubs)

**Interfaces:**
- Consumes: `utils.AiEditImage`, `pkg.DecodeImageInputToBytes` (or `DownloadImage`), `row.OfferSnapshot` JSON with `main_images`, `s.lyncrUtils` / config model (same access pattern as AI produce / Task)
- Produces: `Listing1688ImagesGenerate|Regenerate` real handlers; `listing1688EnsureImageSlots(wf) ([]jobs, error)`; job statuses as above

- [x] **Step 1:** Repo `ListByWorkflow`, `GetByWorkflowSlot`, `Create`, `Update`
- [x] **Step 2:** Failing tests: ensureSlots truncates to 10; slot≥10 rejected; generate with fake AiEdit dependency or interface seam sets `done` + result URL; regenerate updates only that slot
- [x] **Step 3:** Implement ensure + generate/regenerate (sync AiEditImage); set workflow `img_editing`; persist prompt
- [x] **Step 4:** `go test ./internal/services/ ./internal/repos/ -count=1 -run Listing1688`

---

### Task 2: select / use-original / img_ready + GET image_jobs

**Files:**
- Modify: `.../internal/services/listing1688_images.go`
- Modify: `.../internal/services/listing1688_workflow.go` (`Listing1688WorkflowGet`)
- Extend: `.../internal/services/listing1688_images_test.go`

**Interfaces:**
- Produces: `listing1688RecomputeImageReady(wfID) (status, selectedIdx JSON)`; select/use-original handlers; Get returns `image_jobs`

- [x] **Step 1:** Tests: use-original does not call edit; select from pending rejected; all selected → `img_ready`; regenerate after ready → `img_editing`; partial select → not ready (`IMAGES_NOT_READY` helper for future publish gate — export small `listing1688AssertImgReady` used later by M3)
- [x] **Step 2:** Implement select / use-original / recompute; write `selected_image_idx` as JSON array of selected slot ints (or map slot→result_url — **lock: JSON array of `{slot, result_url}` for selected only**)
- [x] **Step 3:** Extend Get response with `image_jobs`
- [x] **Step 4:** Update stub ownership test: owned generate no longer returns `NOT_IMPLEMENTED` (adjust `TestListing1688StubRequiresOwnedWorkflow` to only cover publish M3 stubs, or add new images ownership tests)

---

### Task 3: Web Step3 Slot UI + API module

**Files:**
- Modify: `commander-web-t260220-main/commander-web-t260220-main/src/api/modules/listing1688.js`
- Modify: `.../src/views/platform-entrance/listing1688-wizard/index.vue`
- Modify: `.../src/locales/zh-CN.json`, `en.json`
- Append: `D:\dev\workspace\开发文档.md`

**UI rules:**
- Enter Step3 only when status ∈ `collected|img_editing|img_ready` (browse OK earlier; actions gated)
- For each job: show source thumb, result thumb (if any), prompt input, buttons 生成/重生成/沿用原图/选定
- Disable「下一步 / 发品」until `status === 'img_ready'`（Step4 still M3 stub but gate copy uses `IMAGES_NOT_READY`）
- Map API errors via existing `listing1688Error.js`
- Remove `m2Pending` placeholder

- [x] **Step 1:** API helpers: `listing1688ImagesGenerate|Regenerate|Select|UseOriginal`
- [x] **Step 2:** Step3 UI wired to Get + four actions; refresh jobs after each call
- [x] **Step 3:** i18n + 开发文档 M2 entry
- [x] **Step 4:** Manual smoke checklist (below)

---

## Manual smoke (TC-I mapping)

| Case | Expect |
|------|--------|
| TC-I-01 | Slot0 generate → result URL ≠ empty; job `done` |
| TC-I-02 | Change prompt regenerate Slot0 → only Slot0 `result_url` changes |
| TC-I-03 | use-original → no grsai traffic; can select |
| TC-I-04 | Next/publish gate while not all selected → blocked + `IMAGES_NOT_READY` |
| TC-I-05 | Ensure from snapshot with >10 URLs → only 10 jobs; no 11th generate |
| TC-I-06 | Network: wizard only hits `/api/v1/listing1688/.../images/*`; no `/ai/produce_images` |

Local needs: Server `ENV=dev` with working `Lyncr`/grsai config (same as AI 生图页); workflow already `collected` with snapshot images.

---

## Self-Review

1. Spec M2 / AC-03 / C1 / C5 / TC-I-* — covered in tasks
2. No Agent protocols; no Miaoshou publish
3. No wizard `produce_images`; reuse `AiEditImage`
4. v1 listing path untouched
5. Publish remains M3 stub; only readiness helper prepared
6. Out of scope: M3 seller publish, Onebound, batch auto-img-all-slots (MVP is per-slot explicit generate)
