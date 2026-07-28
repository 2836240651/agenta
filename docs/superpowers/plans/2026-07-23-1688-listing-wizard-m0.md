# 1688 Listing Wizard M0 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver M0 empty shell for 1688 competitor listing wizard: DB tables, status/error enums, `/api/v1/listing1688/*` routes (create/get workflow real; others stub), and Web 4-step wizard shell.

**Architecture:** New Server module under `/api/v1/listing1688` physically isolated from v1 `/listing` Miaoshou path. Workflow rows own the state machine; Session/Collect/Images/Publish handlers return `LISTING_1688_NOT_IMPLEMENTED` until M1–M3. Web route `/1688-auto-upload/wizard` hosts a 4-step shell that can create a `draft` workflow.

**Tech Stack:** Go + Gin + GORM (Postgres), Vue3 + Vite + existing i18n/TDesign patterns.

## Global Constraints

- Main battlefield: `D:\dev\workspace` only
- Scene A: publish = own 1688 seller backend (M3); **never** call Miaoshou / 店小秘 / `alibaba_collect_box` as publish path
- Do not break v1 `/1688-auto-upload/listing`
- Wizard must not call `produce_images` directly
- Conflict authority: `docs/superpowers/specs/1688-listing-wizard-v2/README.md`

## File Map

| Path | Role |
|------|------|
| `commander-server-.../internal/constants/listing_1688_wizard.go` | Workflow statuses + wizard error codes |
| `commander-server-.../internal/modules/table_listing_1688_workflows.go` | Workflow table |
| `commander-server-.../internal/modules/table_listing_1688_sessions.go` | Session table |
| `commander-server-.../internal/modules/table_listing_1688_image_jobs.go` | Image job table |
| `commander-server-.../internal/repos/listing_1688_workflow.go` | Workflow repo |
| `commander-server-.../internal/services/listing1688_workflow.go` | Create/Get + stubs |
| `commander-server-.../internal/router/register.go` | Register module |
| `commander-server-.../internal/utils/gorm.go` | AutoMigrate 3 tables |
| `commander-web-.../src/api/modules/listing1688.js` | API client |
| `commander-web-.../src/views/.../listing1688-wizard/index.vue` | 4-step shell |
| `commander-web-.../src/router/index.js` | `/wizard` route |
| `commander-web-.../src/views/.../MatrixSection1688.vue` | Entry CTA |
| locales zh-CN / en | Wizard copy |

---

### Task 1: Server constants + tables + AutoMigrate

**Files:**
- Create: `commander-server-t260220-main/commander-server-t260220-main/internal/constants/listing_1688_wizard.go`
- Create: `.../internal/constants/listing_1688_wizard_test.go`
- Create: `.../internal/modules/table_listing_1688_workflows.go`
- Create: `.../internal/modules/table_listing_1688_sessions.go`
- Create: `.../internal/modules/table_listing_1688_image_jobs.go`
- Modify: `.../internal/utils/gorm.go` (add AutoMigrate)

- [x] **Step 1:** Add status/error constants and table structs; AutoMigrate; run `go test ./internal/constants/ -count=1`

### Task 2: Repo + Create/Get + stub handlers + routes

**Files:**
- Create: `.../internal/repos/listing_1688_workflow.go`
- Modify: `.../internal/repos/system_agent.go` (add `GetByUUID`)
- Create: `.../internal/services/listing1688_workflow.go`
- Modify: `.../internal/router/register.go`
- Create: `.../internal/services/listing1688_workflow_test.go` (parse/status helpers if any)

- [x] **Step 1:** Implement `POST /api/v1/listing1688/workflows` and `GET /api/v1/listing1688/workflows/:id`
- [x] **Step 2:** Stub session/collect/images/publish with `LISTING_1688_NOT_IMPLEMENTED`
- [x] **Step 3:** `go test ./internal/repos/ ./internal/services/ ./internal/constants/ -count=1` (or compile `go build ./...`)

### Task 3: Web API + wizard shell + matrix entry

**Files:**
- Create: `commander-web-.../src/api/modules/listing1688.js`
- Create: `commander-web-.../src/views/platform-entrance/listing1688-wizard/index.vue`
- Modify: `.../src/router/index.js`
- Modify: `.../src/views/platform-entrance/components/matrix/MatrixSection1688.vue`
- Modify: `.../src/locales/zh-CN.json`, `en.json`
- Append: `D:\dev\workspace\开发文档.md`

- [x] **Step 1:** Route + 4-step UI + create draft via API module
- [x] **Step 2:** Matrix entry link to wizard (keep v1 listing entry)
- [x] **Step 3:** Append 开发文档 entry for M0

---

## Self-Review

1. Spec M0 coverage: 表 / 枚举 / 路由空壳 / Web 四步壳 — all tasked
2. No Miaoshou publish wiring
3. v1 listing path untouched except matrix adds a second CTA
