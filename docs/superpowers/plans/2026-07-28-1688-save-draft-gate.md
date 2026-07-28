# 1688 Save-Draft Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` or implement task-by-task.

**Goal:** Wire Server `form/save-draft` → Agent `listing1688_seller_save_draft`, clear `draft_saved` on re-validate, and harden Publish to require `draft_verified` without the old blind-submit path.

**Architecture:** SyncSend for save-draft (like category inspect). Agent fills only verified title/images bindings then clicks save draft. Publish reuses open offer-new page and submits only when `draft_verified=true`.

**Tech Stack:** Go, Gin, Rod, existing listing1688 SyncSend helpers.

## Global Constraints

- Only verified bindings: title, images, save_draft, submit
- `confirm_save_draft` must be true for save-draft
- Publish requires Server `ready + draft_saved` and Agent `draft_verified=true`
- No remote deploy; no blind-fill of unbound attributes
- Spec: `docs/superpowers/specs/2026-07-28-1688-save-draft-gate-design.md`

### Task 1: Agent dispatcher + SaveDraft fill + Publish gate

**Files:**
- Modify: `commander-agent-.../backend/internal/dispatcher/default.go`
- Modify: `commander-agent-.../backend/internal/factory/alibaba/listing1688_seller.go`
- Modify: `commander-agent-.../backend/internal/factory/alibaba/listing1688_seller_form_test.go`

- [x] Register `listing1688_seller_save_draft` in dispatcher
- [x] SaveDraft: fill title/images from payload then click save
- [x] Publish: require `draft_verified`; use open category form; no new-page blind fill
- [x] Unit tests for draft_verified reject + draft success text

### Task 2: Server protocol + form/save-draft + validate clears draft_saved

**Files:**
- Modify: `commander-server-.../internal/constants/protocol.go`
- Modify: `commander-server-.../internal/router/register.go`
- Modify: `commander-server-.../internal/services/listing1688_publish_profile_handler.go`
- Modify: `commander-server-.../internal/services/listing1688_publish_profile.go` (helpers if needed)
- Test: `listing1688_publish_profile*_test.go` / new save-draft test

- [x] Add protocol constant
- [x] Route `POST form/save-draft`
- [x] Handler SyncSend + set `draft_saved`
- [x] `form/validate` sets `DraftSaved=false`
- [x] Unit tests for canSubmit + validate clears flag

### Task 3: Docs

- [x] Update `交接文档.md` banner + `开发文档.md` top entry
- [x] Mark design status as implemented
